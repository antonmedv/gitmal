package gitdiff

import (
	"bytes"
	"errors"
	"io"
	"io/ioutil"
	"path/filepath"
	"testing"
)

func TestApplyTextFragment(t *testing.T) {
	tests := map[string]applyTest{
		"createFile": {Files: getApplyFiles("text_fragment_new")},
		"deleteFile": {Files: getApplyFiles("text_fragment_delete_all")},

		"addStart":    {Files: getApplyFiles("text_fragment_add_start")},
		"addMiddle":   {Files: getApplyFiles("text_fragment_add_middle")},
		"addEnd":      {Files: getApplyFiles("text_fragment_add_end")},
		"addEndNoEOL": {Files: getApplyFiles("text_fragment_add_end_noeol")},

		"changeStart":       {Files: getApplyFiles("text_fragment_change_start")},
		"changeMiddle":      {Files: getApplyFiles("text_fragment_change_middle")},
		"changeEnd":         {Files: getApplyFiles("text_fragment_change_end")},
		"changeEndEOL":      {Files: getApplyFiles("text_fragment_change_end_eol")},
		"changeExact":       {Files: getApplyFiles("text_fragment_change_exact")},
		"changeSingleNoEOL": {Files: getApplyFiles("text_fragment_change_single_noeol")},

		"errorShortSrcBefore": {
			Files: applyFiles{
				Src:   "text_fragment_error.src",
				Patch: "text_fragment_error_short_src_before.patch",
			},
			Err: &Conflict{},
		},
		"errorShortSrc": {
			Files: applyFiles{
				Src:   "text_fragment_error.src",
				Patch: "text_fragment_error_short_src.patch",
			},
			Err: &Conflict{},
		},
		"errorContextConflict": {
			Files: applyFiles{
				Src:   "text_fragment_error.src",
				Patch: "text_fragment_error_context_conflict.patch",
			},
			Err: &Conflict{},
		},
		"errorDeleteConflict": {
			Files: applyFiles{
				Src:   "text_fragment_error.src",
				Patch: "text_fragment_error_delete_conflict.patch",
			},
			Err: &Conflict{},
		},
		"errorNewFile": {
			Files: applyFiles{
				Src:   "text_fragment_error.src",
				Patch: "text_fragment_error_new_file.patch",
			},
			Err: &Conflict{},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			test.run(t, func(dst io.Writer, src io.ReaderAt, file *File) error {
				if len(file.TextFragments) != 1 {
					t.Fatalf("patch should contain exactly one fragment, but it has %d", len(file.TextFragments))
				}
				applier := NewTextApplier(dst, src)
				return applier.ApplyFragment(file.TextFragments[0])
			})
		})
	}
}

func TestApplyBinaryFragment(t *testing.T) {
	tests := map[string]applyTest{
		"literalCreate":    {Files: getApplyFiles("bin_fragment_literal_create")},
		"literalModify":    {Files: getApplyFiles("bin_fragment_literal_modify")},
		"deltaModify":      {Files: getApplyFiles("bin_fragment_delta_modify")},
		"deltaModifyLarge": {Files: getApplyFiles("bin_fragment_delta_modify_large")},

		"errorIncompleteAdd": {
			Files: applyFiles{
				Src:   "bin_fragment_delta_error.src",
				Patch: "bin_fragment_delta_error_incomplete_add.patch",
			},
			Err: "incomplete add",
		},
		"errorIncompleteCopy": {
			Files: applyFiles{
				Src:   "bin_fragment_delta_error.src",
				Patch: "bin_fragment_delta_error_incomplete_copy.patch",
			},
			Err: "incomplete copy",
		},
		"errorSrcSize": {
			Files: applyFiles{
				Src:   "bin_fragment_delta_error.src",
				Patch: "bin_fragment_delta_error_src_size.patch",
			},
			Err: &Conflict{},
		},
		"errorDstSize": {
			Files: applyFiles{
				Src:   "bin_fragment_delta_error.src",
				Patch: "bin_fragment_delta_error_dst_size.patch",
			},
			Err: "insufficient or extra data",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			test.run(t, func(dst io.Writer, src io.ReaderAt, file *File) error {
				applier := NewBinaryApplier(dst, src)
				return applier.ApplyFragment(file.BinaryFragment)
			})
		})
	}
}

func TestApplyFile(t *testing.T) {
	tests := map[string]applyTest{
		"textModify": {
			Files: applyFiles{
				Src:   "file_text.src",
				Patch: "file_text_modify.patch",
				Out:   "file_text_modify.out",
			},
		},
		"textDelete": {
			Files: applyFiles{
				Src:   "file_text.src",
				Patch: "file_text_delete.patch",
				Out:   "file_text_delete.out",
			},
		},
		"textErrorPartialDelete": {
			Files: applyFiles{
				Src:   "file_text.src",
				Patch: "file_text_error_partial_delete.patch",
			},
			Err: &Conflict{},
		},
		"binaryModify": {
			Files: getApplyFiles("file_bin_modify"),
		},
		"modeChange": {
			Files: getApplyFiles("file_mode_change"),
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			test.run(t, func(dst io.Writer, src io.ReaderAt, file *File) error {
				return Apply(dst, src, file)
			})
		})
	}
}

type applyTest struct {
	Files applyFiles
	Err   interface{}
}

func (at applyTest) run(t *testing.T, apply func(io.Writer, io.ReaderAt, *File) error) {
	src, patch, out := at.Files.Load(t)

	files, _, err := Parse(bytes.NewReader(patch))
	if err != nil {
		t.Fatalf("failed to parse patch file: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("patch should contain exactly one file, but it has %d", len(files))
	}

	var dst bytes.Buffer
	err = apply(&dst, bytes.NewReader(src), files[0])
	if at.Err != nil {
		assertError(t, at.Err, err, "applying fragment")
		return
	}
	if err != nil {
		var aerr *ApplyError
		if errors.As(err, &aerr) {
			t.Fatalf("unexpected error applying: at %d: fragment %d at %d: %v", aerr.Line, aerr.Fragment, aerr.FragmentLine, err)
		} else {
			t.Fatalf("unexpected error applying: %v", err)
		}
	}

	if !bytes.Equal(out, dst.Bytes()) {
		t.Errorf("incorrect result after apply\nexpected:\n%q\nactual:\n%q", out, dst.Bytes())
	}
}

type applyFiles struct {
	Src   string
	Patch string
	Out   string
}

func getApplyFiles(name string) applyFiles {
	return applyFiles{
		Src:   name + ".src",
		Patch: name + ".patch",
		Out:   name + ".out",
	}
}

func (f applyFiles) Load(t *testing.T) (src []byte, patch []byte, out []byte) {
	load := func(name, kind string) []byte {
		d, err := ioutil.ReadFile(filepath.Join("testdata", "apply", name))
		if err != nil {
			t.Fatalf("failed to read %s file: %v", kind, err)
		}
		return d
	}

	if f.Src != "" {
		src = load(f.Src, "source")
	}
	if f.Patch != "" {
		patch = load(f.Patch, "patch")
	}
	if f.Out != "" {
		out = load(f.Out, "output")
	}
	return
}
