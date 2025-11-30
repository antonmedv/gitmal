package links

import (
	"bytes"
	"net/url"
	"path"
	"strings"

	"golang.org/x/net/html"

	"github.com/antonmedv/gitmal/pkg/git"
)

type Set map[string]struct{}

func BuildDirSet(files []git.Blob) Set {
	dirs := make(Set)
	for _, f := range files {
		dir := path.Dir(f.Path)
		for dir != "." && dir != "/" {
			if _, ok := dirs[dir]; ok {
				break
			}
			dirs[dir] = struct{}{}
			if i := strings.LastIndex(dir, "/"); i != -1 {
				dir = dir[:i]
			} else {
				break
			}
		}
	}
	return dirs
}

func BuildFileSet(files []git.Blob) Set {
	filesSet := make(Set)
	for _, f := range files {
		filesSet[f.Path] = struct{}{}
	}
	return filesSet
}

func Resolve(content, currentPath, rootHref, ref string, dirs, files Set) string {
	doc, err := html.Parse(strings.NewReader(content))
	if err != nil {
		return content
	}

	baseDir := path.Dir(currentPath)

	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode {
			switch n.Data {
			case "a":
				for i, attr := range n.Attr {
					if attr.Key == "href" {
						newHref := transformHref(attr.Val, baseDir, rootHref, ref, files, dirs)
						n.Attr[i].Val = newHref
						break
					}
				}
			case "img":
				for i, attr := range n.Attr {
					if attr.Key == "src" {
						newSrc := transformImgSrc(attr.Val, baseDir, rootHref, ref)
						n.Attr[i].Val = newSrc
						break
					}
				}
			}
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)

	var buf bytes.Buffer
	if err := html.Render(&buf, doc); err != nil {
		return content
	}

	return buf.String()
}

func transformHref(href, baseDir, rootHref, ref string, files, dirs Set) string {
	if href == "" {
		return href
	}
	if strings.HasPrefix(href, "#") {
		return href
	}

	u, err := url.Parse(href)
	if err != nil {
		return href
	}

	// Absolute URLs are left untouched
	if u.IsAbs() {
		return href
	}

	// Skip mailto:, javascript:, data: etc. (url.Parse sets Scheme)
	if u.Scheme != "" {
		return href
	}

	// Resolve against the directory of the current file
	relPath := u.Path
	if relPath == "" {
		return href
	}

	var repoPath string

	if strings.HasPrefix(relPath, "/") {
		// Root-relative repo path
		relPath = strings.TrimPrefix(relPath, "/")
		repoPath = path.Clean(relPath)
	} else {
		// Relative to current file directory
		repoPath = path.Clean(path.Join(baseDir, relPath))
	}

	// Decide if this is a file or a directory in the repo
	var newPath string

	// 1) Exact file match
	if _, ok := files[repoPath]; ok {
		newPath = repoPath + ".html"
	} else if _, ok := files[repoPath+".md"]; ok {
		// 2) Maybe the link omitted ".md" but the repo has it
		newPath = repoPath + ".md.html"
	} else if _, ok := dirs[repoPath]; ok {
		// 3) Directory: add /index.html
		newPath = path.Join(repoPath, "index.html")
	} else {
		// Unknown target, leave as-is
		return href
	}

	// Link from the root href
	newPath = path.Join(rootHref, "blob", ref, newPath)

	// Preserve any query/fragment if they existed
	u.Path = newPath
	return u.String()
}

func transformImgSrc(src, baseDir, rootHref, ref string) string {
	u, err := url.Parse(src)
	if err != nil {
		return src
	}

	if u.IsAbs() {
		return src
	}

	relPath := u.Path

	var repoPath string
	if strings.HasPrefix(relPath, "/") {
		// Root-relative: drop leading slash
		repoPath = strings.TrimPrefix(relPath, "/")
	} else {
		// Resolve against current file directory
		repoPath = path.Clean(path.Join(baseDir, relPath))
	}

	final := path.Join(rootHref, "raw", ref, repoPath)

	// Preserve any query/fragment if they existed
	u.Path = final
	return u.String()
}
