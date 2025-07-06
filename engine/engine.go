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

var (
	rootDir    string
	zinVersion string
)

func HandleConnection(ctx context.Context, conn net.Conn, root string, version string) {
	defer conn.Close()

	zinVersion = version
	reader := bufio.NewReader(conn)
	req, err := http.ReadRequest(reader)
	if err != nil {
		PrintErrorOnClient(conn, 500, "<untracked>", fmt.Sprintf("Invalid request: %v", err))
		return
	}

	// Log Request
	fmt.Printf("\n----->>>> HTTP/1.1 %s [%s] %s\n", req.Method, conn.RemoteAddr().String(), req.URL.Path)
	fmt.Printf(">> Host: %s\n", req.Host)

	// Handle Request
	select {
	case <-ctx.Done():
		// Graceful exit if timeout occurs
		ConnTimeOut(conn)
		return
	default:
		// Read the X-Root-Dir header
		rootDir = req.Header.Get("X-Root-Dir")

		// Reset rootDir from configured directory
		if rootDir == "" && root != "" {
			rootDir = root
		}

		// In case client is visiting 127.0.0.*:900* & root is not configured
		if rootDir == "" && req.URL.Path == "/" && (strings.Contains(req.Host, "127.0.0") || strings.Contains(req.Host, "host:900")) {
			PrintErrorOnClient(conn, 200, "/", "")
			return
		}

		// Handle other route when root-dir is not configured
		if rootDir == "" {
			PrintErrorOnClient(conn, 500, req.URL.Path, "Missing X-Root-Dir on nginx server configuration.")
			return
		}

		// Only GET & POST req.methods are allowed
		if req.Method != "GET" && req.Method != "POST" {
			PrintErrorOnClient(conn, 405, req.URL.Path, "Oops! Method Not Allowed")
			return
		}

		// Get file to serve & check if listed in .zinignore
		path := utils.GetFilePathFromURI(req.URL.Path)
		if config.CheckZinIgnore(rootDir, path) {
			PrintErrorOnClient(conn, 403, path, "Forbidden — You do not have permission to access this file")
			return
		}

		// Compose session content
		ctx := ComposeSessionContext(conn, req)

		// Handle zin-default paths
		if HandleDefaultLoads(conn, req, &ctx) {
			return
		}

		// Handle form submission
		if req.Method == http.MethodPost && strings.HasPrefix(path, "/zin-form") {
			statusCode, content := controller.HandleFormSubmission(req, &ctx)
			JsonResponse(conn, statusCode, content)
			return
		}

		// Robot.txt & Sitemap configuration -auto generate if not available or older then 24hr
		config.ComposeRobomap(&ctx)

		// If mine-type is not text/html just return the content as it is
		if !strings.HasPrefix(ctx.ContentType, "text/html") {
			SendRawFile(conn, &ctx)
			return
		}

		// Let's handle source file rendering along with re-write checks
		HandleSourceRender(conn, req, &ctx)

	}
}

func getClientIP(conn net.Conn) string {
	ip, _, err := net.SplitHostPort(conn.RemoteAddr().String())
	if err != nil {
		return conn.RemoteAddr().String()
	}
	return ip
}

func ComposeSessionContext(conn net.Conn, req *http.Request) model.RequestContext {
	ctx := model.RequestContext{
		ClientIp:      getClientIP(conn),
		Method:        req.Method,
		Host:          req.Host,
		Path:          req.URL.Path,
		Root:          rootDir,
		ContentType:   "text/plain",
		ContentSource: req.URL.Path,
		ServerVersion: zinVersion,
		ServerError:   make(map[string]string),
		Query:         req.URL.Query(),
		Headers:       make(map[string]string),
		CustomVar: model.CustomVar{
			Raw:  make(map[string]string),
			JSON: make(map[string]map[string]any),
			LIST: make(map[string][]any),
		},
		ENV:             config.LoadEnvironmentVars(rootDir),
		LocalVar:        make(map[string]string),
		SqlConn:         nil,
		GzipCompression: false,
	}

	// Update context according to request data
	for name, values := range req.Header {
		ctx.Headers[name] = values[0]
	}

	// Set content source & type
	path := utils.GetFilePathFromURI(req.URL.Path)
	ctx.ContentSource = filepath.Join(rootDir, filepath.FromSlash(path))
	ctx.ContentType = utils.GetMineTypeFromPath(path)

	// Check for gzip support
	ctx.GzipCompression = strings.Contains(ctx.Headers["Accept-Encoding"], "gzip")

	return ctx
}

func HandleDefaultLoads(conn net.Conn, req *http.Request, ctx *model.RequestContext) bool {
	// Favicon icon request
	if req.URL.Path == "/favicon.ico" {
		Favicon(conn, rootDir)
		return true
	}

	if req.URL.Path == "/zin-assets/engine.css" {
		ctx.Path = utils.GetExeAssetPath("engine.css")
		SendRawFile(conn, ctx)
		return true
	}

	if req.URL.Path == "/zin-assets/engine.js" {
		ctx.Path = utils.GetExeAssetPath("engine.js")
		SendRawFile(conn, ctx)
		return true
	}

	return false
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

	// Parse zin-tags & Apply Directives
	content = directives.ParseAndApply(content, ctx)

	// Check for errors during directive parsing
	if len(ctx.ServerError) > 0 {
		content := ComposeServerErrorContent(ctx)
		PrintErrorOnClient(conn, 500, req.URL.Path, content)
		return
	}

	// Inject Zin-Assets In Case SHOW_ERROR is ON
	if utils.GetValue(ctx, "SHOW_ERRORS", "OFF", true) == "ON" {
		content = utils.InjectZinScriptAndStyle(content)
	}

	// Finally Load Page Content
	SendPageContent(conn, content, ctx)
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
			SendRawFile(conn, ctx)
			return true
		}

		// Update context-value
		ctx.ContentType = routeMimeType
		ctx.ContentSource = route.Path
	}

	return false
}

func ComposeServerErrorContent(ctx *model.RequestContext) string {

	config := utils.GetValue(ctx, "SHOW_ERRORS", "OFF", true)
	if config != "ON" {
		return ctx.ServerError["title"]
	}

	// Display detailed error
	content := fmt.Sprintf(`<h4>%s</h4>`, ctx.ServerError["title"])
	content += fmt.Sprintf(`<p>%s</p>`, utils.SanitizeHTML(ctx.ServerError["reason"]))
	content += fmt.Sprintf(`<br><details><summary>View Code Block</summary><code>%s</code></details>`, utils.SanitizeHTML(ctx.ServerError["code"]))

	return content
}
