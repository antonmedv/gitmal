package links

import (
	"strings"
	"testing"

	"github.com/antonmedv/gitmal/pkg/git"
)

func buildTestSets(blobs []git.Blob) (Set, Set) {
	dirs := BuildDirSet(blobs)
	files := BuildFileSet(blobs)
	return dirs, files
}

func TestResolve_Links(t *testing.T) {
	blobs := []git.Blob{
		{Path: "README.md"},
		{Path: "docs/intro.md"},
		{Path: "docs/getting-started.md"},
		{Path: "docs/faq.md"},
		{Path: "docs/tutorial/step1.md"},
	}

	dirs, files := buildTestSets(blobs)

	currentPath := "docs/intro.md"
	rootHref := "../../"
	ref := "master"

	tests := []struct {
		name        string
		content     string
		wantContain []string
		notContain  []string
	}{
		{
			name:    "relative link to existing .md file gets .html appended (per current transformHref)",
			content: `<a href="getting-started.md">Getting started</a>`,
			wantContain: []string{
				`href="../../blob/master/docs/getting-started.md.html"`,
			},
			notContain: []string{
				`href="../../blob/master/docs/getting-started.md"`,
			},
		},
		{
			name:    "relative link without extension to existing .md file gets .html appended",
			content: `<a href="faq">FAQ</a>`,
			wantContain: []string{
				`href="../../blob/master/docs/faq.md.html"`,
			},
			notContain: []string{
				`href="faq">`,
			},
		},
		{
			name:    "relative link to directory without trailing slash goes to /index.html",
			content: `<a href="tutorial">Tutorial</a>`,
			wantContain: []string{
				`href="../../blob/master/docs/tutorial/index.html"`,
			},
			notContain: []string{
				`href="tutorial">`,
			},
		},
		{
			name:    "relative link to directory with trailing slash goes to /index.html",
			content: `<a href="tutorial/">Tutorial</a>`,
			wantContain: []string{
				`href="../../blob/master/docs/tutorial/index.html"`,
			},
			notContain: []string{
				`href="tutorial/">`,
			},
		},
		{
			name:    "absolute http URL unchanged",
			content: `<a href="https://example.org">External</a>`,
			wantContain: []string{
				`href="https://example.org"`,
			},
		},
		{
			name:    "root-relative link to repo file is resolved",
			content: `<a href="/README.md">FAQ</a>`,
			wantContain: []string{
				`href="../../blob/master/README.md.html"`,
			},
		},
		{
			name:    "fragment-only href unchanged",
			content: `<a href="#section1">Jump</a>`,
			wantContain: []string{
				`href="#section1"`,
			},
		},
		{
			name:    "mailto href unchanged",
			content: `<a href="mailto:test@example.com">Mail</a>`,
			wantContain: []string{
				`href="mailto:test@example.com"`,
			},
		},
		{
			name:    "unknown relative path left untouched",
			content: `<a href="unknown.md">Unknown</a>`,
			wantContain: []string{
				`href="unknown.md"`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Resolve(tt.content, currentPath, rootHref, ref, dirs, files)

			for _, want := range tt.wantContain {
				if !strings.Contains(got, want) {
					t.Errorf("expected output to contain %q, got:\n%s", want, got)
				}
			}
			for _, notWant := range tt.notContain {
				if strings.Contains(got, notWant) {
					t.Errorf("expected output NOT to contain %q, got:\n%s", notWant, got)
				}
			}
		})
	}
}

func TestResolve_Images(t *testing.T) {
	// For image behavior we only care about currentPath/rootHref/ref, not dirs/files.
	blobs := []git.Blob{
		{Path: "foo/bar/readme.md"},
	}
	dirs, files := buildTestSets(blobs)

	rootHref := "../../"
	ref := "master"

	t.Run("relative image src rewritten to raw URL with ref", func(t *testing.T) {
		// current blob: foo/bar/readme.md
		// img src:      ../images/pic.png
		// repoPath:     foo/images/pic.png
		// final:        rootHref + "/raw/master/foo/images/pic.png"
		content := `<p><img src="../images/pic.png" alt="Pic"></p>`

		got := Resolve(content, "foo/bar/readme.md", rootHref, ref, dirs, files)

		expected := `src="../../raw/master/foo/images/pic.png"`
		if !strings.Contains(got, expected) {
			t.Fatalf("expected output to contain %q, got:\n%s", expected, got)
		}
	})

	t.Run("relative image src with ./ prefix", func(t *testing.T) {
		content := `<p><img src="./img/logo.png"></p>`

		got := Resolve(content, "foo/readme.md", rootHref, ref, dirs, files)

		// repoPath: "foo/img/logo.png"
		expected := `src="../../raw/master/foo/img/logo.png"`
		if !strings.Contains(got, expected) {
			t.Fatalf("expected output to contain %q, got:\n%s", expected, got)
		}
	})

	t.Run("absolute image src unchanged, root-relative resolved from repo root", func(t *testing.T) {
		content := `
<p>
  <img src="https://cdn.example.com/img.png">
  <img src="/static/logo.png">
</p>`

		got := Resolve(content, "docs/intro.md", rootHref, ref, dirs, files)

		if !strings.Contains(got, `src="https://cdn.example.com/img.png"`) {
			t.Errorf("expected absolute src to be unchanged, got:\n%s", got)
		}
		// root-relative should now point to raw/ref/... from repo root
		if !strings.Contains(got, `src="../../raw/master/static/logo.png"`) {
			t.Errorf("expected root-relative src to be resolved to raw URL, got:\n%s", got)
		}
	})

	t.Run("image with query and fragment preserves them", func(t *testing.T) {
		content := `<img src="../images/pic.png?size=large#anchor">`

		got := Resolve(content, "foo/bar/readme.md", rootHref, ref, dirs, files)

		if !strings.Contains(got, `src="../../raw/master/foo/images/pic.png?size=large#anchor"`) {
			t.Errorf("expected src to be rewritten and keep query+fragment, got:\n%s", got)
		}
	})
}

func TestBuildDirSet(t *testing.T) {
	blobs := []git.Blob{
		{Path: "a/b/c.md"},
		{Path: "a/d/e.md"},
		{Path: "x.md"},
	}

	dirs := BuildDirSet(blobs)

	wantDirs := []string{"a", "a/b", "a/d"}
	for _, d := range wantDirs {
		if _, ok := dirs[d]; !ok {
			t.Errorf("expected dirs to contain %q", d)
		}
	}
}

func TestBuildFileSet(t *testing.T) {
	blobs := []git.Blob{
		{Path: "docs/intro.md"},
		{Path: "README.md"},
	}

	files := BuildFileSet(blobs)

	for _, p := range []string{"docs/intro.md", "README.md"} {
		if _, ok := files[p]; !ok {
			t.Errorf("expected files to contain %q", p)
		}
	}
}
