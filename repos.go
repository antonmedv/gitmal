package main

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/antonmedv/gitmal/pkg/templates"
)

func generatePortalIndex(repos []templates.RepoSummary, outputRoot string, owner string, dark bool) error {
	outputDir, err := filepath.Abs(outputRoot)
	if err != nil {
		return err
	}

	sorted := make([]templates.RepoSummary, len(repos))
	copy(sorted, repos)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].DisplayName < sorted[j].DisplayName
	})

	heading := "Repositories"
	title := "Repositories"
	headerName := "Repositories"
	if owner != "" {
		heading = owner
		title = owner + " / Repositories"
		headerName = owner
	}

	f, err := os.Create(filepath.Join(outputDir, "index.html"))
	if err != nil {
		return err
	}

	err = templates.ReposTemplate.ExecuteTemplate(f, "layout.gohtml", templates.ReposParams{
		LayoutParams: templates.LayoutParams{
			Title:    title,
			Name:     headerName,
			Dark:     dark,
			RootHref: "./",
		},
		Heading: heading,
		Repos:   sorted,
		Total:   len(sorted),
	})
	if err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}

	return nil
}

func repoDisplayName(owner, name string) string {
	if owner != "" {
		return owner + "/" + name
	}
	return name
}

func buildRepoSummary(params Params, defaultBranch string, branches int, tags int, details templates.CommitDetails) templates.RepoSummary {
	summary := templates.RepoSummary{
		Name:          params.Name,
		Owner:         params.Owner,
		DisplayName:   repoDisplayName(params.Owner, params.Name),
		Href:          filepath.ToSlash(filepath.Join(params.Name, "index.html")),
		DefaultBranch: defaultBranch,
		BranchCount:   branches,
		TagCount:      tags,
	}
	if details.TotalCommits > 0 {
		summary.TotalCommits = details.TotalCommits
		summary.LastCommitDate = details.LastCommitDate
		summary.LastCommitSubject = strings.TrimSpace(details.LastCommit.Subject)
	}
	return summary
}
