package engine

import (
	"compress/gzip"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"zin-engine/model"
	"zin-engine/utils"
)

func ConnTimeOut(conn net.Conn) {
	status := 408
	conn.Write([]byte(fmt.Sprintf("HTTP/1.1 %d %s\r\n", status, http.StatusText(status))))
	conn.Write([]byte("Content-Type: text/plain\r\n"))
	conn.Write([]byte("Content-Length: 0\r\n"))
	conn.Write([]byte("Parser: " + zinVersion + "\r\n"))
	conn.Write([]byte("Connection: close\r\n"))
	conn.Write([]byte("\r\n"))
	conn.Write([]byte(fmt.Sprintf("%d %s", status, http.StatusText(status))))
}

func PrintErrorOnClient(conn net.Conn, status int, path string, content string) {
	// Get final content to print on client
	content = utils.GetStatusCodeFileContent(status, rootDir, content)

	// Write the HTTP response
	conn.Write([]byte(fmt.Sprintf("HTTP/1.1 %d %s\r\n", status, http.StatusText(status))))
	conn.Write([]byte("Content-Type: text/html\r\n"))
	conn.Write([]byte(fmt.Sprintf("Content-Length: %d\r\n", len(content))))
	conn.Write([]byte("Parser: " + zinVersion + "\r\n"))
	conn.Write([]byte("\r\n"))
	conn.Write([]byte(content))

}

func Favicon(conn net.Conn, rootDir string) {
	path := utils.GetFaviconIconPath(rootDir, "/favicon.ico")

	// Open the favicon file
	f, err := os.Open(path)
	if err != nil {
		PrintErrorOnClient(conn, 404, path, "Error: Favicon icon not found")
		return
	}
	defer f.Close()

	// Write a minimal HTTP response header for the favicon
	conn.Write([]byte("HTTP/1.1 200 OK\r\n"))
	conn.Write([]byte("Content-Type: image/x-icon\r\n"))
	conn.Write([]byte("Parser: " + zinVersion + "\r\n"))
	conn.Write([]byte("Cache-Control: max-age=86400\r\n"))
	conn.Write([]byte("\r\n"))

	// Copy the file contents to the connection
	io.Copy(conn, f)
}

func SendRawFile(conn net.Conn, ctx *model.RequestContext) {

	path := ctx.ContentSource
	contentType := ctx.ContentType
	fmt.Printf(">> Resolve: %s\n>> ContentType: %s\n", path, contentType)

	// Open the file
	f, err := os.Open(path)
	if err != nil {
		PrintErrorOnClient(conn, 404, path, fmt.Sprintf("Error: Unable to find file `%s`. %s", path, err.Error()))
		return
	}
	defer f.Close()

	// Get file info (to read size)
	info, err := f.Stat()
	if err != nil || info.IsDir() {
		PrintErrorOnClient(conn, 404, path, fmt.Sprintf("Error: A directory can't be renders on clint. %s", err.Error()))
		return
	}

	// Try compression first
	if ctx.GzipCompression {
		err := trySendCompressed(conn, f, contentType)
		if err == nil {
			return
		}
		// Reset file reader
		f.Seek(0, io.SeekStart)
	}

	// Send uncompressed
	sendUncompressed(conn, f, contentType, info.Size())
}

func JsonResponse(conn net.Conn, status int, content string) {
	conn.Write([]byte(fmt.Sprintf("HTTP/1.1 %d %s\r\n", status, http.StatusText(status))))
	conn.Write([]byte("Content-Type: application/json\r\n"))
	conn.Write([]byte(fmt.Sprintf("Content-Length: %d\r\n", len(content))))
	conn.Write([]byte("Parser: " + zinVersion + "\r\n"))
	conn.Write([]byte("Connection: close\r\n"))
	conn.Write([]byte("\r\n"))
	conn.Write([]byte(content))
}

func Redirect(conn net.Conn, status int, location string) {
	if status < 300 || status > 399 {
		status = 302 // Default to temporary redirect
	}

	conn.Write([]byte(fmt.Sprintf("HTTP/1.1 %d %s\r\n", status, http.StatusText(status))))
	conn.Write([]byte("Location: " + location + "\r\n"))
	conn.Write([]byte("Content-Length: 0\r\n"))
	conn.Write([]byte("Connection: close\r\n"))
	conn.Write([]byte("Parser: " + zinVersion + "\r\n"))
	conn.Write([]byte("\r\n"))
}

func SendPageContent(conn net.Conn, content string, ctx *model.RequestContext) {

	// Send uncompressed is not requested
	if !ctx.GzipCompression {
		writePlainContent(conn, content)
		return
	}

	// Try-sending gzip-compressed content
	err := writeGzipContent(conn, content)
	if err != nil {
		// Send fallback uncompressed response
		writePlainContent(conn, content)
	}

}

func writeGzipContent(conn net.Conn, content string) error {
	// Headers
	conn.Write([]byte("HTTP/1.1 200 OK\r\n"))
	conn.Write([]byte("Content-Type: text/html\r\n"))
	conn.Write([]byte("Content-Encoding: gzip\r\n"))
	conn.Write([]byte("Parser: " + zinVersion + "\r\n"))
	conn.Write([]byte("Connection: close\r\n"))
	conn.Write([]byte("\r\n"))

	// Body
	gz := gzip.NewWriter(conn)
	_, err := gz.Write([]byte(content))
	if err != nil {
		return fmt.Errorf("error writing gzip content: %w", err)
	}

	err = gz.Close() // must flush footer
	if err != nil {
		return fmt.Errorf("error closing gzip writer: %w", err)
	}

	return nil
}

func writePlainContent(conn net.Conn, content string) {
	conn.Write([]byte("HTTP/1.1 200 OK\r\n"))
	conn.Write([]byte("Content-Type: text/html\r\n"))
	conn.Write([]byte(fmt.Sprintf("Content-Length: %d\r\n", len(content))))
	conn.Write([]byte("Connection: close\r\n"))
	conn.Write([]byte("Parser: " + zinVersion + "\r\n"))
	conn.Write([]byte("\r\n"))
	conn.Write([]byte(content))
}

func sendUncompressed(conn net.Conn, f *os.File, contentType string, contentLength int64) {
	conn.Write([]byte("HTTP/1.1 200 OK\r\n"))
	conn.Write([]byte("Content-Type: " + contentType + "\r\n"))
	conn.Write([]byte(fmt.Sprintf("Content-Length: %d\r\n", contentLength)))
	conn.Write([]byte("Connection: close\r\n"))
	conn.Write([]byte("Parser: " + zinVersion + "\r\n"))
	conn.Write([]byte("\r\n"))
	io.Copy(conn, f)
}

func trySendCompressed(conn net.Conn, f *os.File, contentType string) error {
	// Headers
	conn.Write([]byte("HTTP/1.1 200 OK\r\n"))
	conn.Write([]byte("Content-Type: " + contentType + "\r\n"))
	conn.Write([]byte("Content-Encoding: gzip\r\n"))
	conn.Write([]byte("Parser: " + zinVersion + "\r\n"))
	conn.Write([]byte("Connection: close\r\n"))
	conn.Write([]byte("\r\n"))

	gz := gzip.NewWriter(conn)
	defer gz.Close()

	// Try flushing header
	if err := gz.Flush(); err != nil {
		return fmt.Errorf("failed to flush gzip header: %w", err)
	}

	if _, err := io.Copy(gz, f); err != nil {
		return fmt.Errorf("gzip copy failed:: %w", err)
	}

	return nil
}
