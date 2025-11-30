package gitdiff

import (
	"testing"
)

func TestParsePatchIdentity(t *testing.T) {
	tests := map[string]struct {
		Input  string
		Output PatchIdentity
		Err    interface{}
	}{
		"simple": {
			Input: "Morton Haypenny <mhaypenny@example.com>",
			Output: PatchIdentity{
				Name:  "Morton Haypenny",
				Email: "mhaypenny@example.com",
			},
		},
		"extraWhitespace": {
			Input: "\t  Morton Haypenny  \r\n<mhaypenny@example.com>  ",
			Output: PatchIdentity{
				Name:  "Morton Haypenny",
				Email: "mhaypenny@example.com",
			},
		},
		"trailingCharacters": {
			Input: "Morton Haypenny <mhaypenny@example.com> II",
			Output: PatchIdentity{
				Name:  "Morton Haypenny II",
				Email: "mhaypenny@example.com",
			},
		},
		"onlyEmail": {
			Input: "mhaypenny@example.com",
			Output: PatchIdentity{
				Name:  "mhaypenny@example.com",
				Email: "mhaypenny@example.com",
			},
		},
		"onlyEmailInBrackets": {
			Input: "<mhaypenny@example.com>",
			Output: PatchIdentity{
				Name:  "mhaypenny@example.com",
				Email: "mhaypenny@example.com",
			},
		},
		"rfc5322SpecialCharacters": {
			Input: `"dependabot[bot]" <12345+dependabot[bot]@users.noreply.github.com>`,
			Output: PatchIdentity{
				Name:  "dependabot[bot]",
				Email: "12345+dependabot[bot]@users.noreply.github.com",
			},
		},
		"rfc5322QuotedPairs": {
			Input: `"Morton \"Old-Timer\" Haypenny" <"mhaypenny\+[1900]"@example.com> (III \(PhD\))`,
			Output: PatchIdentity{
				Name:  `Morton "Old-Timer" Haypenny (III (PhD))`,
				Email: "mhaypenny+[1900]@example.com",
			},
		},
		"rfc5322QuotedPairsOutOfContext": {
			Input: `Morton \\Backslash Haypenny <mhaypenny@example.com>`,
			Output: PatchIdentity{
				Name:  `Morton \\Backslash Haypenny`,
				Email: "mhaypenny@example.com",
			},
		},
		"emptyEmail": {
			Input: "Morton Haypenny <>",
			Output: PatchIdentity{
				Name:  "Morton Haypenny",
				Email: "",
			},
		},
		"unclosedEmail": {
			Input: "Morton Haypenny <mhaypenny@example.com",
			Output: PatchIdentity{
				Name:  "Morton Haypenny",
				Email: "mhaypenny@example.com",
			},
		},
		"bogusEmail": {
			Input: "Morton Haypenny <mhaypenny>",
			Output: PatchIdentity{
				Name:  "Morton Haypenny",
				Email: "mhaypenny",
			},
		},
		"bogusEmailWithWhitespace": {
			Input: "Morton Haypenny <  mhaypenny  >",
			Output: PatchIdentity{
				Name:  "Morton Haypenny",
				Email: "mhaypenny",
			},
		},
		"missingEmail": {
			Input: "Morton Haypenny",
			Err:   "invalid identity",
		},
		"missingNameAndEmptyEmail": {
			Input: "<>",
			Err:   "invalid identity",
		},
		"empty": {
			Input: "",
			Err:   "invalid identity",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			id, err := ParsePatchIdentity(test.Input)
			if test.Err != nil {
				assertError(t, test.Err, err, "parsing identity")
				return
			}
			if err != nil {
				t.Fatalf("unexpected error parsing identity: %v", err)
			}

			if test.Output != id {
				t.Errorf("incorrect identity: expected %#v, actual %#v", test.Output, id)
			}
		})
	}
}
