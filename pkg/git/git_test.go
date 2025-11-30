package git

import (
	"testing"
)

func TestParseRefNames_Empty(t *testing.T) {
	got := parseRefNames("")
	if len(got) != 0 {
		t.Fatalf("expected empty slice, got %v", got)
	}
}

func TestParseRefNames_Mixed(t *testing.T) {
	input := "HEAD -> main, tag: v1.0.0, origin/HEAD -> origin/main, origin/main, master"
	got := parseRefNames(input)
	if len(got) != 5 {
		t.Fatalf("expected 5 entries, got %d (%v)", len(got), got)
	}

	// 1: HEAD pointer
	if got[0].Kind != RefKindHEAD || got[0].Name != "HEAD" || got[0].Target != "main" {
		t.Errorf("unexpected HEAD entry: %+v", got[0])
	}
	// 2: tag
	if got[1].Kind != RefKindTag || got[1].Name != "v1.0.0" || got[1].Target != "" {
		t.Errorf("unexpected Tag entry: %+v", got[1])
	}
	// 3: remote HEAD pointer
	if got[2].Kind != RefKindRemoteHEAD || got[2].Name != "origin/HEAD" || got[2].Target != "origin/main" {
		t.Errorf("unexpected RemoteHEAD entry: %+v", got[2])
	}
	// 4: remote branch
	if got[3].Kind != RefKindRemote || got[3].Name != "origin/main" || got[3].Target != "" {
		t.Errorf("unexpected Remote entry: %+v", got[3])
	}
	// 5: local branch
	if got[4].Kind != RefKindBranch || got[4].Name != "master" || got[4].Target != "" {
		t.Errorf("unexpected Branch entry: %+v", got[4])
	}
}

func TestParseRefNames_Singles(t *testing.T) {
	cases := []struct {
		in     string
		kind   RefKind
		name   string
		target string
	}{
		{"tag: v2", RefKindTag, "v2", ""},
		{"main", RefKindBranch, "main", ""},
		{"origin/dev", RefKindRemote, "origin/dev", ""},
		{"origin/HEAD -> origin/main", RefKindRemoteHEAD, "origin/HEAD", "origin/main"},
	}
	for _, c := range cases {
		got := parseRefNames(c.in)
		if len(got) != 1 {
			t.Fatalf("%q: expected 1 entry, got %d (%v)", c.in, len(got), got)
		}
		if got[0].Kind != c.kind || got[0].Name != c.name || got[0].Target != c.target {
			t.Errorf("%q: unexpected entry: %+v", c.in, got[0])
		}
	}
}
