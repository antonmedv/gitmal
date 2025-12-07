package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/antonmedv/gitmal/pkg/git"
	"github.com/antonmedv/gitmal/pkg/templates"
)

const dot = "·"

func echo(a ...any) {
	_, _ = fmt.Fprintln(os.Stderr, a...)
}

func breadcrumbs(rootName string, path string, isFile bool) []templates.Breadcrumb {
	// Root list
	if path == "" {
		return []templates.Breadcrumb{
			{
				Name:  rootName,
				Href:  "./index.html",
				IsDir: true,
			},
		}
	}

	// Paths from git are already with '/'
	parts := strings.Split(path, "/")

	// Build breadcrumbs relative to the file location so links work in static output
	// Example: for a/b/c.txt, at /blob/<ref>/a/b/c.txt.html
	// - root: ../../index.html
	// - a: ../index.html
	// - b: index.html
	// - c.txt: (no link)
	d := len(parts)

	// current directory depth relative to ref
	if isFile {
		d -= 1
	}

	crumbs := make([]templates.Breadcrumb, 0, len(parts))

	// root
	crumbs = append(crumbs, templates.Breadcrumb{
		Name:  rootName,
		Href:  "./" + strings.Repeat("../", d) + "index.html",
		IsDir: true,
	})

	// intermediate directories
	for i := 0; i < len(parts)-1; i++ {
		name := parts[i]
		// target directory depth t = i+1
		up := d - (i + 1)
		href := "./" + strings.Repeat("../", up) + "index.html"
		crumbs = append(crumbs, templates.Breadcrumb{
			Name:  name,
			Href:  href,
			IsDir: true,
		})
	}

	// final file (no link)
	crumbs = append(crumbs, templates.Breadcrumb{
		Name:  parts[len(parts)-1],
		IsDir: !isFile,
	})

	return crumbs
}

func humanizeSize(size int64) string {
	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%d B", size)
	}
	div, exp := int64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(size)/float64(div), "KMGTPE"[exp])
}

func isMarkdown(path string) bool {
	lower := strings.ToLower(path)
	if strings.HasSuffix(lower, ".md") || strings.HasSuffix(lower, ".markdown") || strings.HasSuffix(lower, ".mdown") || strings.HasSuffix(lower, ".mkd") || strings.HasSuffix(lower, ".mkdown") {
		return true
	}
	return false
}

func isImage(path string) bool {
	switch filepath.Ext(path) {
	case ".png", ".jpg", ".jpeg", ".gif", ".webp":
		return true
	default:
		return false
	}
}

func containsBranch(branches []git.Ref, branch string) bool {
	for _, b := range branches {
		if b.String() == branch {
			return true
		}
	}
	return false
}
