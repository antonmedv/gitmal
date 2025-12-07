package main

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/antonmedv/gitmal/pkg/git"
	"github.com/antonmedv/gitmal/pkg/links"
	"github.com/antonmedv/gitmal/pkg/templates"
)

func generateIndex(files []git.Blob, params Params) error {
	// Build directory indexes
	type dirInfo struct {
		subdirs map[string]struct{}
		files   []git.Blob
	}
	dirs := map[string]*dirInfo{}

	ensureDir := func(p string) *dirInfo {
		if di, ok := dirs[p]; ok {
			return di
		}
		di := &dirInfo{subdirs: map[string]struct{}{}, files: []git.Blob{}}
		dirs[p] = di
		return di
	}

	dirsSet := links.BuildDirSet(files)
	filesSet := links.BuildFileSet(files)

	for _, b := range files {
		// Normalize to forward slash paths for URL construction
		p := b.Path
		parts := strings.Split(p, "/")
		// walk directories
		cur := ""
		for i := 0; i < len(parts)-1; i++ {
			child := parts[i]
			ensureDir(cur).subdirs[child] = struct{}{}
			if cur == "" {
				cur = child
			} else {
				cur = cur + "/" + child
			}
			ensureDir(cur) // ensure it exists
		}
		ensureDir(cur).files = append(ensureDir(cur).files, b)
	}

	di := dirs[""] // root

	outDir := params.OutputDir
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}

	// Build entries
	dirNames := make([]string, 0, len(di.subdirs))
	for name := range di.subdirs {
		dirNames = append(dirNames, name)
	}
	// Sort for stable output
	sort.Strings(dirNames)
	sort.Slice(di.files, func(i, j int) bool {
		return di.files[i].FileName < di.files[j].FileName
	})

	subdirEntries := make([]templates.ListEntry, 0, len(dirNames))
	for _, name := range dirNames {
		subdirEntries = append(subdirEntries, templates.ListEntry{
			Name:  name + "/",
			Href:  "blob/" + params.Ref.DirName() + "/" + name + "/index.html",
			IsDir: true,
		})
	}

	fileEntries := make([]templates.ListEntry, 0, len(di.files))
	for _, b := range di.files {
		fileEntries = append(fileEntries, templates.ListEntry{
			Name: b.FileName + "",
			Href: "blob/" + params.Ref.DirName() + "/" + b.FileName + ".html",
			Mode: b.Mode,
			Size: humanizeSize(b.Size),
		})
	}

	// Title and current path label
	title := params.Name

	f, err := os.Create(filepath.Join(outDir, "index.html"))
	if err != nil {
		return err
	}

	rootHref := "./"
	readmeHTML := readme(di.files, dirsSet, filesSet, params, rootHref)

	err = templates.ListTemplate.ExecuteTemplate(f, "layout.gohtml", templates.ListParams{
		LayoutParams: templates.LayoutParams{
			Title:         title,
			Name:          params.Name,
			Dark:          params.Dark,
			CSSMarkdown:   cssMarkdown(params.Dark),
			RootHref:      rootHref,
			CurrentRefDir: params.Ref.DirName(),
			Selected:      "code",
		},
		HeaderParams: templates.HeaderParams{
			Ref:         params.Ref,
			Breadcrumbs: breadcrumbs(params.Name, "", false),
		},
		Ref:    params.Ref,
		Dirs:   subdirEntries,
		Files:  fileEntries,
		Readme: readmeHTML,
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
