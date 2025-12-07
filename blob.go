package main

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"

	"github.com/antonmedv/gitmal/pkg/git"
	"github.com/antonmedv/gitmal/pkg/links"
	"github.com/antonmedv/gitmal/pkg/progress_bar"
	"github.com/antonmedv/gitmal/pkg/templates"
)

func generateBlobs(files []git.Blob, params Params) error {
	// Prepare shared, read-only resources
	var css strings.Builder
	style := styles.Get(params.Style)
	if style == nil {
		return fmt.Errorf("unknown style: %s", params.Style)
	}

	formatterOptions := []html.Option{
		html.WithLineNumbers(true),
		html.WithLinkableLineNumbers(true, "L"),
		html.WithClasses(true),
		html.WithCSSComments(false),
	}

	// Use a temporary formatter to render CSS once
	if err := html.New(formatterOptions...).WriteCSS(&css, style); err != nil {
		return err
	}

	dirsSet := links.BuildDirSet(files)
	filesSet := links.BuildFileSet(files)

	// Bounded worker pool
	workers := runtime.NumCPU()
	if workers < 1 {
		workers = 1
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	jobs := make(chan git.Blob)
	errCh := make(chan error, 1)
	var wg sync.WaitGroup

	p := progress_bar.NewProgressBar("blobs for "+params.Ref.String(), len(files))

	workerFn := func() {
		defer wg.Done()

		// Per-worker instances
		md := createMarkdown(params.Style)
		formatter := html.New(formatterOptions...)

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

		for {
			select {
			case <-ctx.Done():
				return
			case blob, ok := <-jobs:
				if !ok {
					return
				}
				func() {
					var content string
					data, isBin, err := git.BlobContent(params.Ref, blob.Path, params.RepoDir)
					if check(err) {
						return
					}

					isImg := isImage(blob.Path)
					if !isBin {
						content = string(data)
					}

					outPath := filepath.Join(params.OutputDir, "blob", params.Ref.DirName(), blob.Path) + ".html"
					if err := os.MkdirAll(filepath.Dir(outPath), 0o755); check(err) {
						return
					}

					f, err := os.Create(outPath)
					if check(err) {
						return
					}
					defer func() {
						_ = f.Close()
					}()

					depth := 0
					if strings.Contains(blob.Path, "/") {
						depth = len(strings.Split(blob.Path, "/")) - 1
					}
					rootHref := strings.Repeat("../", depth+2)

					if isMarkdown(blob.Path) {
						var b bytes.Buffer
						if err := md.Convert([]byte(content), &b); check(err) {
							return
						}

						contentHTML := links.Resolve(
							b.String(),
							blob.Path,
							rootHref,
							params.Ref.DirName(),
							dirsSet,
							filesSet,
						)

						err = templates.MarkdownTemplate.ExecuteTemplate(f, "layout.gohtml", templates.MarkdownParams{
							LayoutParams: templates.LayoutParams{
								Title:       fmt.Sprintf("%s/%s at %s", params.Name, blob.Path, params.Ref),
								Dark:        params.Dark,
								CSSMarkdown: cssMarkdown(params.Dark),
								Name:        params.Name,
								RootHref:    rootHref,
								CurrentRef:  params.Ref,
								Selected:    "code",
							},
							HeaderParams: templates.HeaderParams{
								Ref:         params.Ref,
								Breadcrumbs: breadcrumbs(params.Name, blob.Path, true),
							},
							Blob:    blob,
							Content: template.HTML(contentHTML),
						})
						if check(err) {
							return
						}

					} else {

						var contentHTML template.HTML
						if !isBin {
							var b bytes.Buffer
							lx := lexers.Match(blob.Path)
							if lx == nil {
								lx = lexers.Fallback
							}
							iterator, _ := lx.Tokenise(nil, content)
							if err := formatter.Format(&b, style, iterator); check(err) {
								return
							}
							contentHTML = template.HTML(b.String())

						} else if isImg {

							rawPath := filepath.Join(params.OutputDir, "raw", params.Ref.DirName(), blob.Path)
							if err := os.MkdirAll(filepath.Dir(rawPath), 0o755); check(err) {
								return
							}

							rf, err := os.Create(rawPath)
							if check(err) {
								return
							}
							defer func() {
								_ = rf.Close()
							}()

							if _, err := rf.Write(data); check(err) {
								return
							}

							relativeRawPath := filepath.Join(rootHref, "raw", params.Ref.DirName(), blob.Path)
							contentHTML = template.HTML(fmt.Sprintf(`<img src="%s" alt="%s" />`, relativeRawPath, blob.FileName))
						}

						err = templates.BlobTemplate.ExecuteTemplate(f, "layout.gohtml", templates.BlobParams{
							LayoutParams: templates.LayoutParams{
								Title:      fmt.Sprintf("%s/%s at %s", params.Name, blob.Path, params.Ref),
								Dark:       params.Dark,
								Name:       params.Name,
								RootHref:   rootHref,
								CurrentRef: params.Ref,
								Selected:   "code",
							},
							HeaderParams: templates.HeaderParams{
								Ref:         params.Ref,
								Breadcrumbs: breadcrumbs(params.Name, blob.Path, true),
							},
							CSS:      template.CSS(css.String()),
							Blob:     blob,
							IsBinary: isBin,
							IsImage:  isImg,
							Content:  contentHTML,
						})
						if check(err) {
							return
						}
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
		defer close(jobs)
		for _, b := range files {
			select {
			case <-ctx.Done():
				return
			case jobs <- b:
			}
		}
	}()

	// Wait for workers
	doneCh := make(chan struct{})
	go func() {
		wg.Wait()
		close(doneCh)
	}()

	var runErr error
	select {
	case runErr = <-errCh:
		// error occurred, wait workers to finish
		<-doneCh
	case <-doneCh:
	}

	p.Done()
	return runErr
}
