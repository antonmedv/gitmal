package git

import (
	"time"
)

type Ref struct {
	ref     string
	dirName string
}

func NewRef(ref string) Ref {
	return Ref{
		ref:     ref,
		dirName: RefToFileName(ref),
	}
}

func (r Ref) IsEmpty() bool {
	return r.ref == ""
}

func (r Ref) String() string {
	return r.ref
}

func (r Ref) DirName() string {
	return r.dirName
}

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
