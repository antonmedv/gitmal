package main

import (
	"context"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"

	"github.com/antonmedv/gitmal/pkg/git"
	"github.com/antonmedv/gitmal/pkg/links"
	"github.com/antonmedv/gitmal/pkg/progress_bar"
	"github.com/antonmedv/gitmal/pkg/templates"
)

func generateLists(files []git.Blob, params Params) error {
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

	// Prepare jobs slice to have stable iteration order (optional)
	type job struct {
		dirPath string
		di      *dirInfo
	}
	jobsSlice := make([]job, 0, len(dirs))
	for dp, di := range dirs {
		jobsSlice = append(jobsSlice, job{dirPath: dp, di: di})
	}
	// Sort by dirPath for determinism
	sort.Slice(jobsSlice, func(i, j int) bool { return jobsSlice[i].dirPath < jobsSlice[j].dirPath })

	// Worker pool similar to generateBlobs
	workers := runtime.NumCPU()
	if workers < 1 {
		workers = 1
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	jobCh := make(chan job)
	errCh := make(chan error, 1)
	var wg sync.WaitGroup

	p := progress_bar.NewProgressBar("lists for "+params.Ref.String(), len(jobsSlice))

	check := func(err error) bool {
		if err != nil {
			select {
			case errCh <- err:
				cancel()
			default:
			}
			return true
		}
		return false
	}

	workerFn := func() {
		defer wg.Done()
		for {
			select {
			case <-ctx.Done():
				return
			case jb, ok := <-jobCh:
				if !ok {
					return
				}
				func() {
					dirPath := jb.dirPath
					di := jb.di

					outDir := filepath.Join(params.OutputDir, "blob", params.Ref.DirName())
					if dirPath != "" {
						// convert forward slash path into OS path
						outDir = filepath.Join(outDir, filepath.FromSlash(dirPath))
					}
					if err := os.MkdirAll(outDir, 0o755); check(err) {
						return
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
							Href:  name + "/index.html",
							IsDir: true,
						})
					}

					fileEntries := make([]templates.ListEntry, 0, len(di.files))
					for _, b := range di.files {
						fileEntries = append(fileEntries, templates.ListEntry{
							Name: b.FileName + "",
							Href: b.FileName + ".html",
							Mode: b.Mode,
							Size: humanizeSize(b.Size),
						})
					}

					// Title and current path label
					title := fmt.Sprintf("%s/%s at %s", params.Name, dirPath, params.Ref)
					if dirPath == "" {
						title = fmt.Sprintf("%s at %s", params.Name, params.Ref)
					}

					f, err := os.Create(filepath.Join(outDir, "index.html"))
					if check(err) {
						return
					}
					defer func() {
						_ = f.Close()
					}()

					// parent link is not shown for root
					parent := "../index.html"
					if dirPath == "" {
						parent = ""
					}

					depth := 0
					if dirPath != "" {
						depth = len(strings.Split(dirPath, "/"))
					}
					rootHref := strings.Repeat("../", depth+2)

					readmeHTML := readme(di.files, dirsSet, filesSet, params, rootHref)
					var CSSMarkdown template.CSS
					if readmeHTML != "" {
						CSSMarkdown = cssMarkdown(params.Dark)
					}

					err = templates.ListTemplate.ExecuteTemplate(f, "layout.gohtml", templates.ListParams{
						LayoutParams: templates.LayoutParams{
							Title:       title,
							Name:        params.Name,
							Dark:        params.Dark,
							CSSMarkdown: CSSMarkdown,
							RootHref:    rootHref,
							CurrentRef:  params.Ref,
							Selected:    "code",
						},
						HeaderParams: templates.HeaderParams{
							Ref:         params.Ref,
							Breadcrumbs: breadcrumbs(params.Name, dirPath, false),
						},
						Ref:        params.Ref,
						ParentHref: parent,
						Dirs:       subdirEntries,
						Files:      fileEntries,
						Readme:     readmeHTML,
					})
					if check(err) {
						return
					}
				}()

				p.Inc()
			}
		}
	}

	// Start workers
	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go workerFn()
	}

	// Feed jobs
	go func() {
		defer close(jobCh)
		for _, jb := range jobsSlice {
			select {
			case <-ctx.Done():
				return
			case jobCh <- jb:
			}
		}
	}()

	// Wait for workers or first error
	doneCh := make(chan struct{})
	go func() {
		wg.Wait()
		close(doneCh)
	}()

	var runErr error
	select {
	case runErr = <-errCh:
		<-doneCh
	case <-doneCh:
	}

	p.Done()

	return runErr
}
