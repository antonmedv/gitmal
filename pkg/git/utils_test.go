package git_test

import (
	"strings"
	"testing"

	"github.com/antonmedv/gitmal/pkg/git"
)

func TestParseFileMode(t *testing.T) {
	tests := []struct {
		mode     string
		expected string
	}{
		{"100644", "rw-r--r--"},
		{"100755", "rwxr-xr-x"},
		{"100600", "rw-------"},
		{"100400", "r--------"},
	}

	for _, tt := range tests {
		t.Run(tt.mode, func(t *testing.T) {
			result, err := git.ParseFileMode(tt.mode)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("got %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestParseFileModeInvalid(t *testing.T) {
	tests := []string{
		"",
		"12",
		"abc",
		"10070x",
	}

	for _, mode := range tests {
		t.Run(mode, func(t *testing.T) {
			result, err := git.ParseFileMode(mode)
			if err == nil {
				t.Error("expected error, got nil")
			}
			if result != "" {
				t.Errorf("expected empty result, got %q", result)
			}
			if !strings.Contains(err.Error(), "invalid mode") && !strings.Contains(err.Error(), "strconv.Atoi") {
				t.Errorf("unexpected error message: %v", err)
			}
		})
	}
}
