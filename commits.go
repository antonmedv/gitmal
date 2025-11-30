package main

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"

	"github.com/antonmedv/gitmal/pkg/git"
	"github.com/antonmedv/gitmal/pkg/gitdiff"
	"github.com/antonmedv/gitmal/pkg/progress_bar"
	"github.com/antonmedv/gitmal/pkg/templates"
)

func generateCommits(commits map[string]git.Commit, params Params) error {
	outDir := filepath.Join(params.OutputDir, "commit")
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}

	list := make([]git.Commit, 0, len(commits))
	for _, c := range commits {
		list = append(list, c)
	}

	workers := runtime.NumCPU()
	if workers < 1 {
		workers = 1
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	jobs := make(chan git.Commit)
	errCh := make(chan error, 1)
	var wg sync.WaitGroup

	p := progress_bar.NewProgressBar("commits", len(list))

	workerFn := func() {
		defer wg.Done()
		for {
			select {
			case <-ctx.Done():
				return
			case c, ok := <-jobs:
				if !ok {
					return
				}
				if err := generateCommitPage(c, params); err != nil {
					select {
					case errCh <- err:
						cancel()
					default:
					}
					return
				}
				p.Inc()
			}
		}
	}

	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go workerFn()
	}

	go func() {
		defer close(jobs)
		for _, c := range list {
			select {
			case <-ctx.Done():
				return
			case jobs <- c:
			}
		}
	}()

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	var err error
	select {
	case err = <-errCh:
		cancel()
		<-done
	case <-done:
	}

	p.Done()
	return err
}

func generateCommitPage(commit git.Commit, params Params) error {
	diff, err := git.CommitDiff(commit.Hash, params.RepoDir)
	if err != nil {
		return err
	}

	files, _, err := gitdiff.Parse(strings.NewReader(diff))
	if err != nil {
		return err
	}

	style := styles.Get(params.Style)
	if style == nil {
		return fmt.Errorf("unknown style: %s", params.Style)
	}

	formatter := html.New(
		html.WithClasses(true),
		html.WithCSSComments(false),
		html.WithCustomCSS(map[chroma.TokenType]string{
			chroma.GenericInserted: "display: block;",
			chroma.GenericDeleted:  "display: block;",
		}),
	)

	var cssBuf bytes.Buffer
	if err := formatter.WriteCSS(&cssBuf, style); err != nil {
		return err
	}

	lexer := lexers.Get("diff")
	if lexer == nil {
		return fmt.Errorf("failed to get lexer for diff")
	}

	outPath := filepath.Join(params.OutputDir, "commit", commit.Hash+".html")

	f, err := os.Create(outPath)
	if err != nil {
		return err
	}
	rootHref := filepath.ToSlash("../")

	fileTree := buildFileTree(files)

	// Create a stable order for files that matches the file tree traversal
	// so that the per-file views appear in the same order as the sidebar tree.
	fileOrder := make(map[string]int)
	{
		// Preorder traversal (dirs first, then files), respecting sortNode ordering
		var idx int
		var walk func(nodes []*templates.FileTree)
		walk = func(nodes []*templates.FileTree) {
			for _, n := range nodes {
				if n.IsDir {
					// Children are already sorted by sortNode
					walk(n.Children)
					continue
				}
				if n.Path == "" {
					continue
				}
				if _, ok := fileOrder[n.Path]; !ok {
					fileOrder[n.Path] = idx
					idx++
				}
			}
		}
		walk(fileTree)
	}

	// Prepare per-file views
	var filesViews []templates.FileView
	for _, f := range files {
		path := f.NewName
		if f.IsDelete {
			path = f.OldName
		}
		if path == "" {
			continue
		}

		var fileDiff strings.Builder
		for _, frag := range f.TextFragments {
			fileDiff.WriteString(frag.String())
		}

		it, err := lexer.Tokenise(nil, fileDiff.String())
		if err != nil {
			return err
		}
		var buf bytes.Buffer
		if err := formatter.Format(&buf, style, it); err != nil {
			return err
		}

		filesViews = append(filesViews, templates.FileView{
			Path:     path,
			OldName:  f.OldName,
			NewName:  f.NewName,
			IsNew:    f.IsNew,
			IsDelete: f.IsDelete,
			IsRename: f.IsRename,
			HTML:     template.HTML(buf.String()),
		})
	}

	// Sort file views to match the file tree order. If for some reason a path
	// is missing in the order map (shouldn't happen), fall back to case-insensitive
	// alphabetical order by full path.
	sort.Slice(filesViews, func(i, j int) bool {
		oi, iok := fileOrder[filesViews[i].Path]
		oj, jok := fileOrder[filesViews[j].Path]
		if iok && jok {
			return oi < oj
		}
		if iok != jok {
			return iok // known order first
		}
		return filesViews[i].Path < filesViews[j].Path
	})

	err = templates.CommitTemplate.ExecuteTemplate(f, "layout.gohtml", templates.CommitParams{
		LayoutParams: templates.LayoutParams{
			Title:      fmt.Sprintf("%s — %s", params.Name, commit.ShortHash),
			Name:       params.Name,
			Dark:       params.Dark,
			RootHref:   rootHref,
			CurrentRef: params.Ref,
			Selected:   "commits",
		},
		Commit:    commit,
		DiffCSS:   template.CSS(cssBuf.String()),
		FileTree:  fileTree,
		FileViews: filesViews,
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

func buildFileTree(files []*gitdiff.File) []*templates.FileTree {
	// Use a synthetic root (not rendered), collect top-level nodes in a map first.
	root := &templates.FileTree{IsDir: true, Name: "", Path: "", Children: nil}

	for _, f := range files {
		path := f.NewName
		if f.IsDelete {
			path = f.OldName
		}

		path = filepath.ToSlash(strings.TrimPrefix(path, "./"))
		if path == "" {
			continue
		}
		parts := strings.Split(path, "/")

		parent := root
		accum := ""
		if len(parts) > 1 {
			for i := 0; i < len(parts)-1; i++ {
				if accum == "" {
					accum = parts[i]
				} else {
					accum = accum + "/" + parts[i]
				}
				parent = findOrCreateDir(parent, parts[i], accum)
			}
		}

		fileName := parts[len(parts)-1]
		node := &templates.FileTree{
			Name:     fileName,
			Path:     path,
			IsDir:    false,
			IsNew:    f.IsNew,
			IsDelete: f.IsDelete,
			IsRename: f.IsRename,
			OldName:  f.OldName,
			NewName:  f.NewName,
		}
		parent.Children = append(parent.Children, node)
	}

	sortNode(root)
	return root.Children
}

func findOrCreateDir(parent *templates.FileTree, name, path string) *templates.FileTree {
	for _, ch := range parent.Children {
		if ch.IsDir && ch.Name == name {
			return ch
		}
	}
	node := &templates.FileTree{IsDir: true, Name: name, Path: path}
	parent.Children = append(parent.Children, node)
	return node
}

func sortNode(n *templates.FileTree) {
	if len(n.Children) == 0 {
		return
	}
	sort.Slice(n.Children, func(i, j int) bool {
		a, b := n.Children[i], n.Children[j]
		if a.IsDir != b.IsDir {
			return a.IsDir && !b.IsDir // dirs first
		}
		return strings.ToLower(a.Name) < strings.ToLower(b.Name)
	})
	for _, ch := range n.Children {
		if ch.IsDir {
			sortNode(ch)
		}
	}
}
