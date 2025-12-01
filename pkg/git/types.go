package git

import (
	"time"
)

type Ref string

type Blob struct {
	Ref      Ref
	Mode     string
	Path     string
	FileName string
	Size     int64
}

type Commit struct {
	Hash      string
	ShortHash string
	Subject   string
	Body      string
	Author    string
	Email     string
	Date      time.Time
	Parents   []string
	Branch    Ref
	RefNames  []RefName
	Href      string
}

type RefKind string

const (
	RefKindHEAD       RefKind = "HEAD"
	RefKindRemoteHEAD RefKind = "RemoteHEAD"
	RefKindBranch     RefKind = "Branch"
	RefKindRemote     RefKind = "Remote"
	RefKindTag        RefKind = "Tag"
)

type RefName struct {
	Kind   RefKind
	Name   string // Name is the primary name of the ref as shown by `git log %D` token (left side for pointers)
	Target string // Target is set for symbolic refs like "HEAD -> main" or "origin/HEAD -> origin/main"
}

type Tag struct {
	Name       string
	Date       time.Time
	CommitHash string
}
