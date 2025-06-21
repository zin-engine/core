package engine

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"net/http"
	"path/filepath"
	"strings"
	"zin-engine/config"
	"zin-engine/controller"
	"zin-engine/directives"
	"zin-engine/model"
	"zin-engine/utils"
)

func HandleConnection(ctx context.Context, conn net.Conn) {
	defer conn.Close()

	reader := bufio.NewReader(conn)
	req, err := http.ReadRequest(reader)
	if err != nil {
		PrintErrorOnClient(conn, 500, req.URL.Path, fmt.Sprintf("Invalid request: %s", err.Error()))
		return
	}

	// Log Request
	fmt.Printf("\n----->>>> HTTP/1.1 %s [%s] %s\n", req.Method, conn.RemoteAddr().String(), req.URL.Path)

	// Handle Request
	select {
	case <-ctx.Done():
		// Graceful exit if timeout occurs
		ConnTimeOut(conn)
		return
	default:
		// Read the X-Root-Dir header
		rootDir := req.Header.Get("X-Root-Dir")
		if rootDir == "" {
			PrintErrorOnClient(conn, 500, req.URL.Path, "ERROR: Missing X-Root-Dir header")
			return
		}

		// Only GET & POST req.methods are allowed
		if req.Method != "GET" && req.Method != "POST" {
			PrintErrorOnClient(conn, 405, req.URL.Path, "ERROR: Method Not Allowed")
			return
		}

		// Favicon icon request
		if req.URL.Path == "/favicon.ico" {
			Favicon(conn, rootDir)
			return
		}

		// Get file to serve & check if listed in .zinignore
		path := GetFilePathFromURI(req.URL.Path)
		if config.CheckZinIgnore(rootDir, path) {
			PrintErrorOnClient(conn, 403, path, "Error: You do not have permission to access this file")
			return
		}

		// Compose session content
		ctx := ComposeSessionContent(conn, req, rootDir)

		// Handle form submission
		if req.Method == http.MethodPost && strings.HasPrefix(path, "/bro-form") {
			statusCode, content := controller.FormSubmit(req, &ctx)
			JsonResponse(conn, statusCode, content)
			return
		}

		// If mine-type is not text/html just return the content as it is
		if !strings.HasPrefix(ctx.ContentType, "text/html") {
			SendRawFile(conn, ctx.ContentSource, ctx.ContentType)
			return
		}

		// Let's handle source file rendering along with re-write checks
		HandleSourceRender(conn, req, &ctx)

	}
}

func ComposeSessionContent(conn net.Conn, req *http.Request, rootDir string) model.RequestContext {
	ctx := model.RequestContext{
		ClientIp:      conn.RemoteAddr().String(),
		Method:        req.Method,
		Host:          req.Host,
		Path:          req.URL.Path,
		Root:          rootDir,
		ContentType:   "text/plain",
		ContentSource: req.URL.Path,
		ServerError:   "",
		Query:         req.URL.Query(),
		Headers:       make(map[string]string),
		CustomVar: model.CustomVar{
			Raw:  make(map[string]string),
			JSON: make(map[string]map[string]any),
			LIST: make(map[string][]any),
		},
		ENV:      config.LoadEnvironmentVars(rootDir),
		LocalVar: make(map[string]string),
		SqlConn:  nil,
	}

	// Update context according to request data
	for name, values := range req.Header {
		ctx.Headers[name] = values[0]
	}

	// Set content source & type
	path := GetFilePathFromURI(req.URL.Path)
	ctx.ContentSource = filepath.Join(rootDir, filepath.FromSlash(path))
	ctx.ContentType = utils.GetMineTypeFromPath(path)

	return ctx
}

func GetFilePathFromURI(path string) string {

	if path == "/" {
		path = "/index.html"
	}

	ext := filepath.Ext(path)
	if ext == "" {
		return path + ".html"
	}

	return path
}

func HandleSourceRender(conn net.Conn, req *http.Request, ctx *model.RequestContext) {

	if HandleExistenceAndRedirect(conn, req, ctx) {
		return
	}

	// Compose page content wrapped inside template.html - conditionally
	content, err := GetPageContent(ctx.Root, req.URL.Path, ctx.ContentSource)
	if err != nil {
		PrintErrorOnClient(conn, 500, req.URL.Path, fmt.Sprintf("Template Parsing Error: %s", err.Error()))
		return
	}

	// ToDo: Parse & Apply Directives
	content = directives.ParseAndApply(content, ctx)

	// Finally Load Page Content
	SendPageContent(conn, content)
}

func HandleExistenceAndRedirect(conn net.Conn, req *http.Request, ctx *model.RequestContext) bool {
	if !utils.FileExists(ctx.ContentSource) {
		route, err := config.GetReWriteTarget(ctx.Root, req.URL.Path)
		if err != nil {
			PrintErrorOnClient(conn, 404, req.URL.Path, fmt.Sprintf("Error: Unable to find file at `%s`.", req.URL.Path))
			return true
		}

		// Redirect if target is external HTTP/S URI
		if route.Type == "external" {
			Redirect(conn, 302, route.Path)
			return true
		}

		// Check file existence for the one last time XD
		if !utils.FileExists(route.Path) {
			PrintErrorOnClient(conn, 404, req.URL.Path, fmt.Sprintf("Error: Unable to find file at `%s`.", req.URL.Path))
			return true
		}

		// If not a HTML file the render it as raw
		routeMimeType := utils.GetMineTypeFromPath(route.Path)
		if !strings.HasPrefix(routeMimeType, "text/html") {
			SendRawFile(conn, route.Path, routeMimeType)
			return true
		}

		// Update context-value
		ctx.ContentType = routeMimeType
		ctx.ContentSource = route.Path
	}

	return false
}
