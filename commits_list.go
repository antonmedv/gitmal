package main

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"github.com/antonmedv/gitmal/pkg/git"
	"github.com/antonmedv/gitmal/pkg/progress_bar"
	"github.com/antonmedv/gitmal/pkg/templates"
)

const commitsPerPage = 100

func generateLogForBranch(allCommits []git.Commit, params Params) error {
	total := len(allCommits)
	totalPages := (total + commitsPerPage - 1) / commitsPerPage

	// RootHref from commits/<branch>/... => ../../
	rootHref := "../../"
	outBase := filepath.Join(params.OutputDir, "commits", params.Ref.DirName())
	if err := os.MkdirAll(outBase, 0o755); err != nil {
		return err
	}

	p := progress_bar.NewProgressBar("commits for "+params.Ref.String(), totalPages)

	page := 1
	for pageCommits := range slices.Chunk(allCommits, commitsPerPage) {
		for i := range pageCommits {
			pageCommits[i].Href = filepath.ToSlash(filepath.Join(rootHref, "commit", pageCommits[i].Hash+".html"))
		}

		fileName := "index.html"
		if page > 1 {
			fileName = fmt.Sprintf("page-%d.html", page)
		}

		outPath := filepath.Join(outBase, fileName)
		f, err := os.Create(outPath)
		if err != nil {
			return err
		}

		var prevHref, nextHref, firstHref, lastHref string
		if page > 1 {
			if page-1 == 1 {
				prevHref = "index.html"
			} else {
				prevHref = fmt.Sprintf("page-%d.html", page-1)
			}
			firstHref = "index.html"
		}

		if page < totalPages {
			nextHref = fmt.Sprintf("page-%d.html", page+1)
			if totalPages > 1 {
				lastHref = fmt.Sprintf("page-%d.html", totalPages)
			}
		}

		err = templates.CommitsListTemplate.ExecuteTemplate(f, "layout.gohtml", templates.CommitsListParams{
			LayoutParams: templates.LayoutParams{
				Title:         fmt.Sprintf("Commits %s %s", dot, params.Name),
				Name:          params.Name,
				Dark:          params.Dark,
				RootHref:      rootHref,
				CurrentRefDir: params.Ref.DirName(),
				Selected:      "commits",
			},
			HeaderParams: templates.HeaderParams{
				Header: "Commits",
			},
			Ref:     params.Ref,
			Commits: pageCommits,
			Page: templates.Pagination{
				Page:       page,
				TotalPages: totalPages,
				PrevHref:   prevHref,
				NextHref:   nextHref,
				FirstHref:  firstHref,
				LastHref:   lastHref,
			},
		})
		if err != nil {
			_ = f.Close()
			return err
		}
		if err := f.Close(); err != nil {
			return err
		}

		page++
		p.Inc()
	}

	p.Done()

	return nil
}
