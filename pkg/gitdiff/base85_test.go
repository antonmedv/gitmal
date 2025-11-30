package gitdiff

import (
	"bytes"
	"testing"
)

func TestBase85Decode(t *testing.T) {
	tests := map[string]struct {
		Input  string
		Output []byte
		Err    bool
	}{
		"twoBytes": {
			Input:  "%KiWV",
			Output: []byte{0xCA, 0xFE},
		},
		"fourBytes": {
			Input:  "007GV",
			Output: []byte{0x0, 0x0, 0xCA, 0xFE},
		},
		"sixBytes": {
			Input:  "007GV%KiWV",
			Output: []byte{0x0, 0x0, 0xCA, 0xFE, 0xCA, 0xFE},
		},
		"invalidCharacter": {
			Input: "00'GV",
			Err:   true,
		},
		"underpaddedSequence": {
			Input: "007G",
			Err:   true,
		},
		"dataUnderrun": {
			Input:  "007GV",
			Output: make([]byte, 8),
			Err:    true,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			dst := make([]byte, len(test.Output))
			err := base85Decode(dst, []byte(test.Input))
			if test.Err {
				if err == nil {
					t.Fatalf("expected error decoding base85 data, but got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error decoding base85 data: %v", err)
			}
			for i, b := range test.Output {
				if dst[i] != b {
					t.Errorf("incorrect byte at index %d: expected 0x%X, actual 0x%X", i, b, dst[i])
				}
			}
		})
	}
}

func TestBase85Encode(t *testing.T) {
	tests := map[string]struct {
		Input  []byte
		Output string
	}{
		"zeroBytes": {
			Input:  []byte{},
			Output: "",
		},
		"twoBytes": {
			Input:  []byte{0xCA, 0xFE},
			Output: "%KiWV",
		},
		"fourBytes": {
			Input:  []byte{0x0, 0x0, 0xCA, 0xFE},
			Output: "007GV",
		},
		"sixBytes": {
			Input:  []byte{0x0, 0x0, 0xCA, 0xFE, 0xCA, 0xFE},
			Output: "007GV%KiWV",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			dst := make([]byte, len(test.Output))
			base85Encode(dst, test.Input)
			for i, b := range test.Output {
				if dst[i] != byte(b) {
					t.Errorf("incorrect character at index %d: expected '%c', actual '%c'", i, b, dst[i])
				}
			}
		})
	}
}

func FuzzBase85Roundtrip(f *testing.F) {
	f.Add([]byte{0x2b, 0x0d})
	f.Add([]byte{0xbc, 0xb4, 0x3f})
	f.Add([]byte{0xfa, 0x62, 0x05, 0x83, 0x24, 0x39, 0xd5, 0x25})
	f.Add([]byte{0x31, 0x59, 0x02, 0xa0, 0x61, 0x12, 0xd9, 0x43, 0xb8, 0x23, 0x1a, 0xb4, 0x02, 0xae, 0xfa, 0xcc, 0x22, 0xad, 0x41, 0xb9, 0xb8})

	f.Fuzz(func(t *testing.T, in []byte) {
		n := len(in)
		dst := make([]byte, base85Len(n))
		out := make([]byte, n)

		base85Encode(dst, in)
		if err := base85Decode(out, dst); err != nil {
			t.Fatalf("unexpected error decoding base85 data: %v", err)
		}
		if !bytes.Equal(in, out) {
			t.Errorf("decoded data differed from input data:\n   input: %x\n  output: %x\nencoding: %s\n", in, out, string(dst))
		}
	})
}
