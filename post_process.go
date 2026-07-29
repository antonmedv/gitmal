package main

import (
	"bytes"
	"compress/gzip"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/tdewolff/minify/v2"
	"github.com/tdewolff/minify/v2/css"
	"github.com/tdewolff/minify/v2/html"
	"github.com/tdewolff/minify/v2/svg"

	"github.com/antonmedv/gitmal/pkg/progress_bar"
)

func postProcessHTMLFile(path string, doMinify bool, doGzip bool) error {
	return postProcessHTMLFiles([]string{path}, doMinify, doGzip)
}

func postProcessHTML(root string, doMinify bool, doGzip bool) error {
	// 1) Collect all HTML files first
	var files []string
	if root != "" {
		if err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			if strings.HasSuffix(d.Name(), ".html") {
				files = append(files, path)
			}
			return nil
		}); err != nil {
			return err
		}
	}

	return postProcessHTMLFiles(files, doMinify, doGzip)
}

func postProcessHTMLFiles(files []string, doMinify bool, doGzip bool) error {

	if len(files) == 0 {
		return nil
	}

	// 2) Setup progress bar
	labels := []string{}
	if doMinify {
		labels = append(labels, "minify")
	}
	if doGzip {
		labels = append(labels, "gzip")
	}
	pb := progress_bar.NewProgressBar(strings.Join(labels, " + "), len(files))
	defer pb.Done()

	// 3) Worker pool
	workers := runtime.NumCPU()
	if workers < 1 {
		workers = 1
	}
	jobs := make(chan string, workers*2)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var firstErr error

	workerFn := func() {
		defer wg.Done()
		var m *minify.M
		if doMinify {
			m = minify.New()
			m.AddFunc("text/html", html.Minify)
			m.AddFunc("text/css", css.Minify)
			m.AddFunc("image/svg+xml", svg.Minify)
		}
		for path := range jobs {
			data, err := os.ReadFile(path)
			if err == nil && doMinify {
				if md, e := m.Bytes("text/html", data); e == nil {
					data = md
				} else {
					err = e
				}
			}
			if err == nil {
				if doGzip {
					// write to file.html.gz
					gzPath := path + ".gz"
					if e := writeGzip(gzPath, data); e != nil {
						err = e
					} else if e := os.Remove(path); e != nil { // remove original .html
						err = e
					}
				} else {
					if e := os.WriteFile(path, data, 0o644); e != nil {
						err = e
					}
				}
			}

			if err != nil {
				mu.Lock()
				if firstErr == nil {
					firstErr = err
				}
				mu.Unlock()
			}
			pb.Inc()
		}
	}

	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go workerFn()
	}
	for _, f := range files {
		jobs <- f
	}
	close(jobs)
	wg.Wait()

	return firstErr
}

func writeGzip(path string, data []byte) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	gw := gzip.NewWriter(f)
	gw.Name = filepath.Base(strings.TrimSuffix(path, ".gz"))
	if _, err := io.Copy(gw, bytes.NewReader(data)); err != nil {
		_ = gw.Close()
		return err
	}
	if err := gw.Close(); err != nil {
		return err
	}
	return nil
}
