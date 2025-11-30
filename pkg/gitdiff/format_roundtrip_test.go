package gitdiff

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestFormatRoundtrip(t *testing.T) {
	patches := []struct {
		File            string
		SkipTextCompare bool
	}{
		{File: "copy.patch"},
		{File: "copy_modify.patch"},
		{File: "delete.patch"},
		{File: "mode.patch"},
		{File: "mode_modify.patch"},
		{File: "modify.patch"},
		{File: "new.patch"},
		{File: "new_empty.patch"},
		{File: "new_mode.patch"},
		{File: "rename.patch"},
		{File: "rename_modify.patch"},

		// Due to differences between Go's 'encoding/zlib' package and the zlib
		// C library, binary patches cannot be compared directly as the patch
		// data is slightly different when re-encoded by Go.
		{File: "binary_modify.patch", SkipTextCompare: true},
		{File: "binary_new.patch", SkipTextCompare: true},
		{File: "binary_modify_nodata.patch"},
	}

	for _, patch := range patches {
		t.Run(patch.File, func(t *testing.T) {
			b, err := os.ReadFile(filepath.Join("testdata", "string", patch.File))
			if err != nil {
				t.Fatalf("failed to read patch: %v", err)
			}

			original := assertParseSingleFile(t, b, "patch")
			str := original.String()

			if !patch.SkipTextCompare {
				if string(b) != str {
					t.Errorf("incorrect patch text\nexpected: %q\n  actual: %q\n", string(b), str)
				}
			}

			reparsed := assertParseSingleFile(t, []byte(str), "formatted patch")
			assertFilesEqual(t, original, reparsed)
		})
	}
}

func assertParseSingleFile(t *testing.T, b []byte, kind string) *File {
	files, _, err := Parse(bytes.NewReader(b))
	if err != nil {
		t.Fatalf("failed to parse %s: %v", kind, err)
	}
	if len(files) != 1 {
		t.Fatalf("expected %s to contain a single files, but found %d", kind, len(files))
	}
	return files[0]
}

func assertFilesEqual(t *testing.T, expected, actual *File) {
	assertEqual(t, expected.OldName, actual.OldName, "OldName")
	assertEqual(t, expected.NewName, actual.NewName, "NewName")

	assertEqual(t, expected.IsNew, actual.IsNew, "IsNew")
	assertEqual(t, expected.IsDelete, actual.IsDelete, "IsDelete")
	assertEqual(t, expected.IsCopy, actual.IsCopy, "IsCopy")
	assertEqual(t, expected.IsRename, actual.IsRename, "IsRename")

	assertEqual(t, expected.OldMode, actual.OldMode, "OldMode")
	assertEqual(t, expected.NewMode, actual.NewMode, "NewMode")

	assertEqual(t, expected.OldOIDPrefix, actual.OldOIDPrefix, "OldOIDPrefix")
	assertEqual(t, expected.NewOIDPrefix, actual.NewOIDPrefix, "NewOIDPrefix")
	assertEqual(t, expected.Score, actual.Score, "Score")

	if len(expected.TextFragments) == len(actual.TextFragments) {
		for i := range expected.TextFragments {
			prefix := fmt.Sprintf("TextFragments[%d].", i)
			ef := expected.TextFragments[i]
			af := actual.TextFragments[i]

			assertEqual(t, ef.Comment, af.Comment, prefix+"Comment")

			assertEqual(t, ef.OldPosition, af.OldPosition, prefix+"OldPosition")
			assertEqual(t, ef.OldLines, af.OldLines, prefix+"OldLines")

			assertEqual(t, ef.NewPosition, af.NewPosition, prefix+"NewPosition")
			assertEqual(t, ef.NewLines, af.NewLines, prefix+"NewLines")

			assertEqual(t, ef.LinesAdded, af.LinesAdded, prefix+"LinesAdded")
			assertEqual(t, ef.LinesDeleted, af.LinesDeleted, prefix+"LinesDeleted")

			assertEqual(t, ef.LeadingContext, af.LeadingContext, prefix+"LeadingContext")
			assertEqual(t, ef.TrailingContext, af.TrailingContext, prefix+"TrailingContext")

			if !slices.Equal(ef.Lines, af.Lines) {
				t.Errorf("%sLines: expected %#v, actual %#v", prefix, ef.Lines, af.Lines)
			}
		}
	} else {
		t.Errorf("TextFragments: expected length %d, actual length %d", len(expected.TextFragments), len(actual.TextFragments))
	}

	assertEqual(t, expected.IsBinary, actual.IsBinary, "IsBinary")

	if expected.BinaryFragment != nil {
		if actual.BinaryFragment == nil {
			t.Errorf("BinaryFragment: expected non-nil, actual is nil")
		} else {
			ef := expected.BinaryFragment
			af := expected.BinaryFragment

			assertEqual(t, ef.Method, af.Method, "BinaryFragment.Method")
			assertEqual(t, ef.Size, af.Size, "BinaryFragment.Size")

			if !slices.Equal(ef.Data, af.Data) {
				t.Errorf("BinaryFragment.Data: expected %#v, actual %#v", ef.Data, af.Data)
			}
		}
	} else if actual.BinaryFragment != nil {
		t.Errorf("BinaryFragment: expected nil, actual is non-nil")
	}

	if expected.ReverseBinaryFragment != nil {
		if actual.ReverseBinaryFragment == nil {
			t.Errorf("ReverseBinaryFragment: expected non-nil, actual is nil")
		} else {
			ef := expected.ReverseBinaryFragment
			af := expected.ReverseBinaryFragment

			assertEqual(t, ef.Method, af.Method, "ReverseBinaryFragment.Method")
			assertEqual(t, ef.Size, af.Size, "ReverseBinaryFragment.Size")

			if !slices.Equal(ef.Data, af.Data) {
				t.Errorf("ReverseBinaryFragment.Data: expected %#v, actual %#v", ef.Data, af.Data)
			}
		}
	} else if actual.ReverseBinaryFragment != nil {
		t.Errorf("ReverseBinaryFragment: expected nil, actual is non-nil")
	}
}

func assertEqual[T comparable](t *testing.T, expected, actual T, name string) {
	if expected != actual {
		t.Errorf("%s: expected %#v, actual %#v", name, expected, actual)
	}
}
