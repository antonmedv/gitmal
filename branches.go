package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/antonmedv/gitmal/pkg/git"
	"github.com/antonmedv/gitmal/pkg/templates"
)

// generateBranches creates a branches.html page at the root of the output
// that lists all branches and links to their root directory listings.
func generateBranches(branches []git.Ref, defaultBranch string, params Params) error {
	outDir := params.OutputDir
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}

	entries := make([]templates.BranchEntry, 0, len(branches))
	for _, b := range branches {
		entries = append(entries, templates.BranchEntry{
			Name:        string(b),
			Href:        filepath.ToSlash(filepath.Join("blob", string(b)) + "/index.html"),
			IsDefault:   string(b) == defaultBranch,
			CommitsHref: filepath.ToSlash(filepath.Join("commits", string(b), "index.html")),
		})
	}

	// Ensure default branch is shown at the top of the list.
	// Keep remaining branches sorted alphabetically for determinism.
	sort.SliceStable(entries, func(i, j int) bool {
		if entries[i].IsDefault != entries[j].IsDefault {
			return entries[i].IsDefault && !entries[j].IsDefault
		}
		return entries[i].Name < entries[j].Name
	})

	f, err := os.Create(filepath.Join(outDir, "branches.html"))
	if err != nil {
		return err
	}
	defer f.Close()

	// RootHref from root page is just ./
	rootHref := "./"

	err = templates.BranchesTemplate.ExecuteTemplate(f, "layout.gohtml", templates.BranchesParams{
		LayoutParams: templates.LayoutParams{
			Title:      fmt.Sprintf("Branches %s %s", dot, params.Name),
			Name:       params.Name,
			Dark:       params.Dark,
			RootHref:   rootHref,
			CurrentRef: params.DefaultRef,
			Selected:   "branches",
		},
		Branches: entries,
	})
	if err != nil {
		return err
	}

	return nil
}
