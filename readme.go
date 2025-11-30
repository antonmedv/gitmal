package main

import (
	"bytes"
	"html/template"
	"strings"

	"github.com/antonmedv/gitmal/pkg/git"
	"github.com/antonmedv/gitmal/pkg/links"
)

func readme(files []git.Blob, params Params, rootHref string) template.HTML {
	var readmeHTML template.HTML

	dirsSet := links.BuildDirSet(files)
	filesSet := links.BuildFileSet(files)

	md := createMarkdown(params.Style)

	for _, b := range files {
		nameLower := strings.ToLower(b.FileName)
		if strings.HasPrefix(nameLower, "readme") && isMarkdown(b.Path) {
			data, isBin, err := git.BlobContent(params.Ref, b.Path, params.RepoDir)
			if err != nil || isBin {
				break
			}
			var buf bytes.Buffer
			if err := md.Convert(data, &buf); err != nil {
				break
			}

			// Fix links/images relative to README location
			htmlStr := links.Resolve(buf.String(), b.Path, rootHref, string(params.Ref), dirsSet, filesSet)

			readmeHTML = template.HTML(htmlStr)
			break
		}
	}

	return readmeHTML
}
