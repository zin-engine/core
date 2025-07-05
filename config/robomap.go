package config

import (
	"encoding/xml"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
	"zin-engine/model"
)

const (
	robotsFileName  = "robots.txt"
	sitemapFileName = "sitemap.xml"
	zinignoreFile   = ".zinignore"
)

// Sitemap XML struct
type UrlSet struct {
	XMLName xml.Name `xml:"urlset"`
	Xmlns   string   `xml:"xmlns,attr"`
	Urls    []Url    `xml:"url"`
}

type Url struct {
	Loc     string `xml:"loc"`
	LastMod string `xml:"lastmod"`
}

// Call this function to generated fresh sitemap.xml + robots.txt
func ComposeRobomap(ctx *model.RequestContext) {

	if ctx.Path != "/robots.txt" && ctx.Path != "/sitemap.xml" {
		return
	}

	robotsPath := filepath.Join(ctx.Root, robotsFileName)
	sitemapPath := filepath.Join(ctx.Root, sitemapFileName)

	isReGenRequired := needsUpdate(robotsPath) || needsUpdate(sitemapPath)

	if !isReGenRequired {
		return
	}

	ignored := loadIgnoreList(ctx.Root)
	files, err := collectFiles(ctx, ignored)
	if err != nil {
		return
	}

	writeRobotsTxt(ctx.Root, ignored, ctx.Host)
	writeSitemapXml(ctx.Root, files, ctx.Host)
}

func needsUpdate(path string) bool {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return true
	}
	return time.Since(info.ModTime()) > 24*time.Hour
}

func loadIgnoreList(root string) []string {
	ignoreSet := map[string]bool{
		".env":        true,
		zinignoreFile: true,
	}

	ignorePath := filepath.Join(root, ".zinignore")
	data, err := os.ReadFile(ignorePath)
	if err == nil {
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue // Skip blank lines and comments
			}

			// Normalize slashes
			line = filepath.ToSlash(line)

			// If ends with /* or / treat as directory prefix
			if strings.HasSuffix(line, "/*") {
				ignoreSet[strings.TrimSuffix(line, "/*")+"/"] = true
			} else if strings.HasSuffix(line, "/") {
				ignoreSet[line] = true
			} else {
				ignoreSet[line] = true
			}
		}
	}

	ignoreList := make([]string, 0, len(ignoreSet))
	for k := range ignoreSet {
		ignoreList = append(ignoreList, k)
	}
	return ignoreList
}

func writeRobotsTxt(root string, ignored []string, host string) {
	robots := "User-agent: *\n"
	for _, path := range ignored {
		path = cleanFilePath(path)
		if strings.HasSuffix(path, "/") {
			robots += fmt.Sprintf("Disallow: /%s\n", path)
		} else {
			robots += fmt.Sprintf("Disallow: /%s\n", strings.TrimSuffix(path, "/"))
		}
	}
	robots += "Allow: /\n"
	robots += fmt.Sprintf("Sitemap: https://%s/%s\n", host, sitemapFileName)

	_ = os.WriteFile(filepath.Join(root, robotsFileName), []byte(robots), 0644)
}

func writeSitemapXml(root string, files []string, host string) {
	urls := []Url{}
	now := time.Now().Format("2006-01-02")

	for _, file := range files {
		file = cleanFilePath(file)
		urls = append(urls, Url{
			Loc:     fmt.Sprintf("https://%s/%s", host, file),
			LastMod: now,
		})
	}

	sitemap := UrlSet{
		Xmlns: "http://www.sitemaps.org/schemas/sitemap/0.9",
		Urls:  urls,
	}

	output, _ := xml.MarshalIndent(sitemap, "", "  ")
	xmlContent := []byte(xml.Header + string(output))
	_ = os.WriteFile(filepath.Join(root, sitemapFileName), xmlContent, 0644)
}

// cleanFilePath strips 'index.html' or any '*.html' file from the end of a path
func cleanFilePath(input string) string {
	base := path.Base(input)

	// Remove the file part
	if strings.HasSuffix(base, ".html") {
		input = strings.TrimSuffix(input, ".html")
	}

	// Remove trailing slash (except for root "/")
	if input != "/" && strings.HasSuffix(input, "/") {
		input = strings.TrimRight(input, "/")
	}

	return input
}

func collectFiles(ctx *model.RequestContext, ignored []string) ([]string, error) {
	var files []string

	err := filepath.WalkDir(ctx.Root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}

		rel, _ := filepath.Rel(ctx.Root, path)
		rel = filepath.ToSlash(rel)

		// Skip dotfiles (e.g., .env, .DS_Store, .gitignore, etc.)
		if strings.HasPrefix(filepath.Base(rel), ".") {
			return nil
		}

		// Skip specific files
		switch strings.ToLower(filepath.Base(rel)) {
		case "robots.txt", "sitemap.txt":
			return nil
		}

		// Skip file extensions like .csv, .json, .config
		for _, ext := range []string{".csv", ".json", ".config"} {
			if strings.HasSuffix(rel, ext) {
				return nil
			}
		}

		// Skip based on ignored prefixes
		for _, ignore := range ignored {
			if strings.HasPrefix(rel, ignore) {
				return nil
			}
		}

		files = append(files, rel)
		return nil
	})

	return files, err
}
