package templates

import (
	"embed"
	_ "embed"
	. "html/template"
	"path/filepath"
	"time"

	"github.com/antonmedv/gitmal/pkg/git"
)

var funcs = FuncMap{
	"BaseName": filepath.Base,
	"FormatDate": func(date time.Time) string {
		return date.Format("2006-01-02 15:04:05")
	},
	"ShortHash": func(hash string) string {
		return hash[:7]
	},
	"FileTreeParams": func(node []*FileTree) FileTreeParams {
		return FileTreeParams{Nodes: node}
	},
}

//go:embed css/markdown_light.css
var CSSMarkdownLight string

//go:embed css/markdown_dark.css
var CSSMarkdownDark string

//go:embed layout.gohtml header.gohtml file_tree.gohtml svg.gohtml
var layoutContent embed.FS
var layout = Must(New("layout").Funcs(funcs).ParseFS(layoutContent, "*.gohtml"))

//go:embed blob.gohtml
var blobContent string
var BlobTemplate = Must(Must(layout.Clone()).Parse(blobContent))

//go:embed markdown.gohtml
var markdownContent string
var MarkdownTemplate = Must(Must(layout.Clone()).Parse(markdownContent))

//go:embed list.gohtml
var listContent string
var ListTemplate = Must(Must(layout.Clone()).Parse(listContent))

//go:embed branches.gohtml
var branchesContent string
var BranchesTemplate = Must(Must(layout.Clone()).Parse(branchesContent))

//go:embed tags.gohtml
var tagsContent string
var TagsTemplate = Must(Must(layout.Clone()).Parse(tagsContent))

//go:embed commits_list.gohtml
var commitsListContent string
var CommitsListTemplate = Must(Must(layout.Clone()).Parse(commitsListContent))

//go:embed commit.gohtml
var commitContent string
var CommitTemplate = Must(Must(layout.Clone()).Parse(commitContent))

//go:embed preview.gohtml
var previewContent string
var PreviewTemplate = Must(New("preview").Parse(previewContent))

type LayoutParams struct {
	Title       string
	Name        string
	Dark        bool
	CSSMarkdown CSS
	RootHref    string
	CurrentRef  git.Ref
	Selected    string
}

type HeaderParams struct {
	Ref         git.Ref
	Header      string
	Breadcrumbs []Breadcrumb
}

type Breadcrumb struct {
	Name  string
	Href  string
	IsDir bool
}

type BlobParams struct {
	LayoutParams
	HeaderParams
	CSS      CSS
	Blob     git.Blob
	IsImage  bool
	IsBinary bool
	Content  HTML
}

type MarkdownParams struct {
	LayoutParams
	HeaderParams
	Blob    git.Blob
	Content HTML
}

type ListParams struct {
	LayoutParams
	HeaderParams
	Ref        git.Ref
	ParentHref string
	Dirs       []ListEntry
	Files      []ListEntry
	Readme     HTML
}

type ListEntry struct {
	Name  string
	Href  string
	IsDir bool
	Mode  string
	Size  string
}

type BranchesParams struct {
	LayoutParams
	Branches []BranchEntry
}

type BranchEntry struct {
	Name        string
	Href        string
	IsDefault   bool
	CommitsHref string
}

type TagsParams struct {
	LayoutParams
	Tags []git.Tag
}

type Pagination struct {
	Page       int
	TotalPages int
	PrevHref   string
	NextHref   string
	FirstHref  string
	LastHref   string
}

type CommitsListParams struct {
	LayoutParams
	HeaderParams
	Ref     git.Ref
	Commits []git.Commit
	Page    Pagination
}

type CommitParams struct {
	LayoutParams
	Commit    git.Commit
	DiffCSS   CSS
	FileTree  []*FileTree
	FileViews []FileView
}

type FileTreeParams struct {
	Nodes []*FileTree
}

// FileTree represents a directory or file in a commit's changed files tree.
// For directories, IsDir is true and Children contains nested nodes.
// For files, status flags indicate the type of change.
type FileTree struct {
	Name     string
	Path     string
	IsDir    bool
	Children []*FileTree

	// File status (applicable when IsDir == false)
	IsNew    bool
	IsDelete bool
	IsRename bool
	OldName  string
	NewName  string
	// Anchor id for this file (empty for directories)
	Anchor string
}

// FileView represents a single file section on the commit page with its
// pre-rendered HTML diff and metadata used by the template.
type FileView struct {
	Path     string
	OldName  string
	NewName  string
	IsNew    bool
	IsDelete bool
	IsRename bool
	HTML     HTML // pre-rendered HTML for diff of this file
}

type PreviewCard struct {
	Name string
	Tone string
	HTML HTML
}

type PreviewParams struct {
	Count  int
	Themes []PreviewCard
}
