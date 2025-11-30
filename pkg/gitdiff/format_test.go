package gitdiff

import (
	"strings"
	"testing"
)

func TestFormatter_WriteQuotedName(t *testing.T) {
	tests := []struct {
		Input    string
		Expected string
	}{
		{"noquotes.txt", `noquotes.txt`},
		{"no quotes.txt", `no quotes.txt`},
		{"new\nline", `"new\nline"`},
		{"escape\x1B null\x00", `"escape\033 null\000"`},
		{"snowman \u2603 snowman", `"snowman \342\230\203 snowman"`},
		{"\"already quoted\"", `"\"already quoted\""`},
	}

	for _, test := range tests {
		var b strings.Builder
		newFormatter(&b).WriteQuotedName(test.Input)
		if b.String() != test.Expected {
			t.Errorf("expected %q, got %q", test.Expected, b.String())
		}
	}
}
