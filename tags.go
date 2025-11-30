package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/antonmedv/gitmal/pkg/git"
	"github.com/antonmedv/gitmal/pkg/templates"
)

func generateTags(entries []git.Tag, params Params) error {
	outDir := params.OutputDir
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}

	f, err := os.Create(filepath.Join(outDir, "tags.html"))
	if err != nil {
		return err
	}
	defer f.Close()

	rootHref := "./"

	return templates.TagsTemplate.ExecuteTemplate(f, "layout.gohtml", templates.TagsParams{
		LayoutParams: templates.LayoutParams{
			Title:      fmt.Sprintf("Tags %s %s", dot, params.Name),
			Name:       params.Name,
			Dark:       params.Dark,
			RootHref:   rootHref,
			CurrentRef: params.DefaultRef,
			Selected:   "tags",
		},
		Tags: entries,
	})
}
