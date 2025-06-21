package config

import (
	"encoding/xml"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
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
func ComposeRobomap(root string, host string) {

	robotsPath := filepath.Join(root, robotsFileName)
	sitemapPath := filepath.Join(root, sitemapFileName)

	isReGenRequired := needsUpdate(robotsPath) || needsUpdate(sitemapPath)

	if !isReGenRequired {
		return
	}

	ignored := loadIgnoreList(root)

	files := []string{}
	filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}

		rel, _ := filepath.Rel(root, path)
		rel = filepath.ToSlash(rel)

		if rel == robotsFileName || rel == sitemapFileName || rel == zinignoreFile || rel == ".env" {
			return nil
		}

		for _, ignore := range ignored {
			if strings.HasPrefix(rel, ignore) {
				return nil
			}
		}

		files = append(files, rel)
		return nil
	})

	writeRobotsTxt(root, ignored, host)
	writeSitemapXml(root, files, host)
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

	ignorePath := filepath.Join(root, zinignoreFile)
	data, err := os.ReadFile(ignorePath)
	if err == nil {
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line != "" {
				ignoreSet[line] = true
			}
		}
	}

	ignoreList := []string{}
	for k := range ignoreSet {
		ignoreList = append(ignoreList, k)
	}
	return ignoreList
}

func writeRobotsTxt(root string, ignored []string, host string) {
	robots := "User-agent: *\n"
	for _, path := range ignored {
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
