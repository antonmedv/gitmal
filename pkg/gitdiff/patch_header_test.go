package gitdiff

import (
	"testing"
	"time"
)

func TestParsePatchDate(t *testing.T) {
	expected := time.Date(2020, 4, 9, 8, 7, 6, 0, time.UTC)

	tests := map[string]struct {
		Input  string
		Output time.Time
		Err    interface{}
	}{
		"default": {
			Input:  "Thu Apr 9 01:07:06 2020 -0700",
			Output: expected,
		},
		"defaultLocal": {
			Input:  "Thu Apr 9 01:07:06 2020",
			Output: time.Date(2020, 4, 9, 1, 7, 6, 0, time.Local),
		},
		"iso": {
			Input:  "2020-04-09 01:07:06 -0700",
			Output: expected,
		},
		"isoStrict": {
			Input:  "2020-04-09T01:07:06-07:00",
			Output: expected,
		},
		"rfc": {
			Input:  "Thu, 9 Apr 2020 01:07:06 -0700",
			Output: expected,
		},
		"short": {
			Input:  "2020-04-09",
			Output: time.Date(2020, 4, 9, 0, 0, 0, 0, time.Local),
		},
		"raw": {
			Input:  "1586419626 -0700",
			Output: expected,
		},
		"unix": {
			Input:  "1586419626",
			Output: expected,
		},
		"unknownFormat": {
			Input: "4/9/2020 01:07:06 PDT",
			Err:   "unknown date format",
		},
		"empty": {
			Input: "",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			d, err := ParsePatchDate(test.Input)
			if test.Err != nil {
				assertError(t, test.Err, err, "parsing date")
				return
			}
			if err != nil {
				t.Fatalf("unexpected error parsing date: %v", err)
			}
			if !test.Output.Equal(d) {
				t.Errorf("incorrect parsed date: expected %v, actual %v", test.Output, d)
			}
		})
	}
}

func TestParsePatchHeader(t *testing.T) {
	expectedSHA := "61f5cd90bed4d204ee3feb3aa41ee91d4734855b"
	expectedIdentity := &PatchIdentity{
		Name:  "Morton Haypenny",
		Email: "mhaypenny@example.com",
	}
	expectedDate := time.Date(2020, 04, 11, 15, 21, 23, 0, time.FixedZone("PDT", -7*60*60))
	expectedTitle := "A sample commit to test header parsing"
	expectedEmojiOneLineTitle := "🤖 Enabling auto-merging"
	expectedEmojiMultiLineTitle := "[IA64] Put ia64 config files on the Uwe Kleine-König diet"
	expectedBody := "The medium format shows the body, which\nmay wrap on to multiple lines.\n\nAnother body line."
	expectedBodyAppendix := "CC: Joe Smith <joe.smith@company.com>"

	tests := map[string]struct {
		Input   string
		Options []PatchHeaderOption
		Header  PatchHeader
		Err     interface{}
	}{
		"prettyShort": {
			Input: `commit 61f5cd90bed4d204ee3feb3aa41ee91d4734855b
Author: Morton Haypenny <mhaypenny@example.com>

    A sample commit to test header parsing
`,
			Header: PatchHeader{
				SHA:    expectedSHA,
				Author: expectedIdentity,
				Title:  expectedTitle,
			},
		},
		"prettyMedium": {
			Input: `commit 61f5cd90bed4d204ee3feb3aa41ee91d4734855b
Author: Morton Haypenny <mhaypenny@example.com>
Date:   Sat Apr 11 15:21:23 2020 -0700

    A sample commit to test header parsing

    The medium format shows the body, which
    may wrap on to multiple lines.

    Another body line.
`,
			Header: PatchHeader{
				SHA:        expectedSHA,
				Author:     expectedIdentity,
				AuthorDate: expectedDate,
				Title:      expectedTitle,
				Body:       expectedBody,
			},
		},
		"prettyFull": {
			Input: `commit 61f5cd90bed4d204ee3feb3aa41ee91d4734855b
Author: Morton Haypenny <mhaypenny@example.com>
Commit: Morton Haypenny <mhaypenny@example.com>

    A sample commit to test header parsing

    The medium format shows the body, which
    may wrap on to multiple lines.

    Another body line.
`,
			Header: PatchHeader{
				SHA:       expectedSHA,
				Author:    expectedIdentity,
				Committer: expectedIdentity,
				Title:     expectedTitle,
				Body:      expectedBody,
			},
		},
		"prettyFuller": {
			Input: `commit 61f5cd90bed4d204ee3feb3aa41ee91d4734855b
Author:     Morton Haypenny <mhaypenny@example.com>
AuthorDate: Sat Apr 11 15:21:23 2020 -0700
Commit:     Morton Haypenny <mhaypenny@example.com>
CommitDate: Sat Apr 11 15:21:23 2020 -0700

    A sample commit to test header parsing

    The medium format shows the body, which
    may wrap on to multiple lines.

    Another body line.
`,
			Header: PatchHeader{
				SHA:           expectedSHA,
				Author:        expectedIdentity,
				AuthorDate:    expectedDate,
				Committer:     expectedIdentity,
				CommitterDate: expectedDate,
				Title:         expectedTitle,
				Body:          expectedBody,
			},
		},
		"prettyAppendix": {
			Input: `commit 61f5cd90bed4d204ee3feb3aa41ee91d4734855b
Author:     Morton Haypenny <mhaypenny@example.com>
AuthorDate: Sat Apr 11 15:21:23 2020 -0700
Commit:     Morton Haypenny <mhaypenny@example.com>
CommitDate: Sat Apr 11 15:21:23 2020 -0700

    A sample commit to test header parsing

    The medium format shows the body, which
    may wrap on to multiple lines.

    Another body line.
    ---
    CC: Joe Smith <joe.smith@company.com>
`,
			Header: PatchHeader{
				SHA:           expectedSHA,
				Author:        expectedIdentity,
				AuthorDate:    expectedDate,
				Committer:     expectedIdentity,
				CommitterDate: expectedDate,
				Title:         expectedTitle,
				Body:          expectedBody + "\n---\n" + expectedBodyAppendix,
			},
		},
		"mailbox": {
			Input: `From 61f5cd90bed4d204ee3feb3aa41ee91d4734855b Mon Sep 17 00:00:00 2001
From: Morton Haypenny <mhaypenny@example.com>
Date: Sat, 11 Apr 2020 15:21:23 -0700
Subject: [PATCH] A sample commit to test header parsing

The medium format shows the body, which
may wrap on to multiple lines.

Another body line.
`,
			Header: PatchHeader{
				SHA:        expectedSHA,
				Author:     expectedIdentity,
				AuthorDate: expectedDate,
				Title:      expectedTitle,
				Body:       expectedBody,
			},
		},
		"mailboxPatchOnly": {
			Input: `From 61f5cd90bed4d204ee3feb3aa41ee91d4734855b Mon Sep 17 00:00:00 2001
From: Morton Haypenny <mhaypenny@example.com>
Date: Sat, 11 Apr 2020 15:21:23 -0700
Subject: [PATCH] [BUG-123] A sample commit to test header parsing

The medium format shows the body, which
may wrap on to multiple lines.

Another body line.
`,
			Options: []PatchHeaderOption{
				WithSubjectCleanMode(SubjectCleanPatchOnly),
			},
			Header: PatchHeader{
				SHA:        expectedSHA,
				Author:     expectedIdentity,
				AuthorDate: expectedDate,
				Title:      "[BUG-123] " + expectedTitle,
				Body:       expectedBody,
			},
		},
		"mailboxEmojiOneLine": {
			Input: `From 61f5cd90bed4d204ee3feb3aa41ee91d4734855b Mon Sep 17 00:00:00 2001
From: Morton Haypenny <mhaypenny@example.com>
Date: Sat, 11 Apr 2020 15:21:23 -0700
Subject: [PATCH] =?UTF-8?q?=F0=9F=A4=96=20Enabling=20auto-merging?=

The medium format shows the body, which
may wrap on to multiple lines.

Another body line.
`,
			Header: PatchHeader{
				SHA:        expectedSHA,
				Author:     expectedIdentity,
				AuthorDate: expectedDate,
				Title:      expectedEmojiOneLineTitle,
				Body:       expectedBody,
			},
		},
		"mailboxEmojiMultiLine": {
			Input: `From 61f5cd90bed4d204ee3feb3aa41ee91d4734855b Mon Sep 17 00:00:00 2001
From: Morton Haypenny <mhaypenny@example.com>
Date: Sat, 11 Apr 2020 15:21:23 -0700
Subject: [PATCH] =?UTF-8?q?[IA64]=20Put=20ia64=20config=20files=20on=20the=20?=
 =?UTF-8?q?Uwe=20Kleine-K=C3=B6nig=20diet?=

The medium format shows the body, which
may wrap on to multiple lines.

Another body line.
`,
			Header: PatchHeader{
				SHA:        expectedSHA,
				Author:     expectedIdentity,
				AuthorDate: expectedDate,
				Title:      expectedEmojiMultiLineTitle,
				Body:       expectedBody,
			},
		},
		"mailboxRFC5322SpecialCharacters": {
			Input: `From 61f5cd90bed4d204ee3feb3aa41ee91d4734855b Mon Sep 17 00:00:00 2001
From: "dependabot[bot]" <12345+dependabot[bot]@users.noreply.github.com>
Date: Sat, 11 Apr 2020 15:21:23 -0700
Subject: [PATCH] A sample commit to test header parsing

The medium format shows the body, which
may wrap on to multiple lines.

Another body line.
`,
			Header: PatchHeader{
				SHA: expectedSHA,
				Author: &PatchIdentity{
					Name:  "dependabot[bot]",
					Email: "12345+dependabot[bot]@users.noreply.github.com",
				},
				AuthorDate: expectedDate,
				Title:      expectedTitle,
				Body:       expectedBody,
			},
		},
		"mailboxAppendix": {
			Input: `From 61f5cd90bed4d204ee3feb3aa41ee91d4734855b Mon Sep 17 00:00:00 2001
From: Morton Haypenny <mhaypenny@example.com>
Date: Sat, 11 Apr 2020 15:21:23 -0700
Subject: [PATCH] A sample commit to test header parsing

The medium format shows the body, which
may wrap on to multiple lines.

Another body line.
---
CC: Joe Smith <joe.smith@company.com>
`,
			Header: PatchHeader{
				SHA:          expectedSHA,
				Author:       expectedIdentity,
				AuthorDate:   expectedDate,
				Title:        expectedTitle,
				Body:         expectedBody,
				BodyAppendix: expectedBodyAppendix,
			},
		},
		"mailboxMinimalNoName": {
			Input: `From: <mhaypenny@example.com>
Subject: [PATCH] A sample commit to test header parsing

The medium format shows the body, which
may wrap on to multiple lines.

Another body line.
`,
			Header: PatchHeader{
				Author: &PatchIdentity{expectedIdentity.Email, expectedIdentity.Email},
				Title:  expectedTitle,
				Body:   expectedBody,
			},
		},
		"mailboxMinimal": {
			Input: `From: Morton Haypenny <mhaypenny@example.com>
Subject: [PATCH] A sample commit to test header parsing

The medium format shows the body, which
may wrap on to multiple lines.

Another body line.
`,
			Header: PatchHeader{
				Author: expectedIdentity,
				Title:  expectedTitle,
				Body:   expectedBody,
			},
		},
		"unwrapTitle": {
			Input: `commit 61f5cd90bed4d204ee3feb3aa41ee91d4734855b
Author: Morton Haypenny <mhaypenny@example.com>
Date:   Sat Apr 11 15:21:23 2020 -0700

    A sample commit to test header parsing with a long
	title that is wrapped.
`,
			Header: PatchHeader{
				SHA:        expectedSHA,
				Author:     expectedIdentity,
				AuthorDate: expectedDate,
				Title:      expectedTitle + " with a long title that is wrapped.",
			},
		},
		"normalizeBodySpace": {
			Input: `commit 61f5cd90bed4d204ee3feb3aa41ee91d4734855b
Author: Morton Haypenny <mhaypenny@example.com>
Date:   Sat Apr 11 15:21:23 2020 -0700

    A sample commit to test header parsing


    The medium format shows the body, which
    may wrap on to multiple lines.


    Another body line.


`,
			Header: PatchHeader{
				SHA:        expectedSHA,
				Author:     expectedIdentity,
				AuthorDate: expectedDate,
				Title:      expectedTitle,
				Body:       expectedBody,
			},
		},
		"ignoreLeadingBlankLines": {
			Input: `

` + "    " + `
commit 61f5cd90bed4d204ee3feb3aa41ee91d4734855b
Author: Morton Haypenny <mhaypenny@example.com>

    A sample commit to test header parsing
`,
			Header: PatchHeader{
				SHA:    expectedSHA,
				Author: expectedIdentity,
				Title:  expectedTitle,
			},
		},
		"emptyHeader": {
			Input:  "",
			Header: PatchHeader{},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			h, err := ParsePatchHeader(test.Input, test.Options...)
			if test.Err != nil {
				assertError(t, test.Err, err, "parsing patch header")
				return
			}
			if err != nil {
				t.Fatalf("unexpected error parsing patch header: %v", err)
			}
			if h == nil {
				t.Fatalf("expected non-nil header, but got nil")
			}

			exp := test.Header
			act := *h

			if exp.SHA != act.SHA {
				t.Errorf("incorrect parsed SHA: expected %q, actual %q", exp.SHA, act.SHA)
			}

			assertPatchIdentity(t, "author", exp.Author, act.Author)
			if !exp.AuthorDate.Equal(act.AuthorDate) {
				t.Errorf("incorrect parsed author date: expected %v, but got %v", exp.AuthorDate, act.AuthorDate)
			}

			assertPatchIdentity(t, "committer", exp.Committer, act.Committer)
			if !exp.CommitterDate.Equal(act.CommitterDate) {
				t.Errorf("incorrect parsed committer date: expected %v, but got %v", exp.CommitterDate, act.CommitterDate)
			}

			if exp.Title != act.Title {
				t.Errorf("incorrect parsed title:\n  expected: %q\n    actual: %q", exp.Title, act.Title)
			}
			if exp.Body != act.Body {
				t.Errorf("incorrect parsed body:\n  expected: %q\n    actual: %q", exp.Body, act.Body)
			}
			if exp.BodyAppendix != act.BodyAppendix {
				t.Errorf("incorrect parsed body appendix:\n  expected: %q\n    actual: %q",
					exp.BodyAppendix, act.BodyAppendix)
			}
		})
	}
}

func assertPatchIdentity(t *testing.T, kind string, exp, act *PatchIdentity) {
	switch {
	case exp == nil && act == nil:
	case exp == nil && act != nil:
		t.Errorf("incorrect parsed %s: expected nil, but got %+v", kind, act)
	case exp != nil && act == nil:
		t.Errorf("incorrect parsed %s: expected %+v, but got nil", kind, exp)
	case exp.Name != act.Name || exp.Email != act.Email:
		t.Errorf("incorrect parsed %s, expected %+v, bot got %+v", kind, exp, act)
	}
}

func TestCleanSubject(t *testing.T) {
	expectedSubject := "A sample commit to test header parsing"

	tests := map[string]struct {
		Input   string
		Mode    SubjectCleanMode
		Prefix  string
		Subject string
	}{
		"CleanAll/noPrefix": {
			Input:   expectedSubject,
			Mode:    SubjectCleanAll,
			Subject: expectedSubject,
		},
		"CleanAll/patchPrefix": {
			Input:   "[PATCH] " + expectedSubject,
			Mode:    SubjectCleanAll,
			Prefix:  "[PATCH] ",
			Subject: expectedSubject,
		},
		"CleanAll/patchPrefixNoSpace": {
			Input:   "[PATCH]" + expectedSubject,
			Mode:    SubjectCleanAll,
			Prefix:  "[PATCH]",
			Subject: expectedSubject,
		},
		"CleanAll/patchPrefixContent": {
			Input:   "[PATCH 3/7] " + expectedSubject,
			Mode:    SubjectCleanAll,
			Prefix:  "[PATCH 3/7] ",
			Subject: expectedSubject,
		},
		"CleanAll/spacePrefix": {
			Input:   "   " + expectedSubject,
			Mode:    SubjectCleanAll,
			Subject: expectedSubject,
		},
		"CleanAll/replyLowerPrefix": {
			Input:   "re: " + expectedSubject,
			Mode:    SubjectCleanAll,
			Prefix:  "re: ",
			Subject: expectedSubject,
		},
		"CleanAll/replyMixedPrefix": {
			Input:   "Re: " + expectedSubject,
			Mode:    SubjectCleanAll,
			Prefix:  "Re: ",
			Subject: expectedSubject,
		},
		"CleanAll/replyCapsPrefix": {
			Input:   "RE: " + expectedSubject,
			Mode:    SubjectCleanAll,
			Prefix:  "RE: ",
			Subject: expectedSubject,
		},
		"CleanAll/replyDoublePrefix": {
			Input:   "Re: re: " + expectedSubject,
			Mode:    SubjectCleanAll,
			Prefix:  "Re: re: ",
			Subject: expectedSubject,
		},
		"CleanAll/noPrefixSubjectHasRe": {
			Input:   "Reimplement parsing",
			Mode:    SubjectCleanAll,
			Subject: "Reimplement parsing",
		},
		"CleanAll/patchPrefixSubjectHasRe": {
			Input:   "[PATCH 1/2] Reimplement parsing",
			Mode:    SubjectCleanAll,
			Prefix:  "[PATCH 1/2] ",
			Subject: "Reimplement parsing",
		},
		"CleanAll/unclosedPrefix": {
			Input:   "[Just to annoy people",
			Mode:    SubjectCleanAll,
			Subject: "[Just to annoy people",
		},
		"CleanAll/multiplePrefix": {
			Input:   " Re:Re: [PATCH 1/2][DRAFT] " + expectedSubject + "  ",
			Mode:    SubjectCleanAll,
			Prefix:  "Re:Re: [PATCH 1/2][DRAFT] ",
			Subject: expectedSubject,
		},
		"CleanPatchOnly/patchPrefix": {
			Input:   "[PATCH] " + expectedSubject,
			Mode:    SubjectCleanPatchOnly,
			Prefix:  "[PATCH] ",
			Subject: expectedSubject,
		},
		"CleanPatchOnly/mixedPrefix": {
			Input:   "[PATCH] [TICKET-123] " + expectedSubject,
			Mode:    SubjectCleanPatchOnly,
			Prefix:  "[PATCH] ",
			Subject: "[TICKET-123] " + expectedSubject,
		},
		"CleanPatchOnly/multiplePrefix": {
			Input:   "Re:Re: [PATCH 1/2][DRAFT] " + expectedSubject,
			Mode:    SubjectCleanPatchOnly,
			Prefix:  "Re:Re: [PATCH 1/2]",
			Subject: "[DRAFT] " + expectedSubject,
		},
		"CleanWhitespace/leadingSpace": {
			Input:   "    [PATCH] " + expectedSubject,
			Mode:    SubjectCleanWhitespace,
			Subject: "[PATCH] " + expectedSubject,
		},
		"CleanWhitespace/trailingSpace": {
			Input:   "[PATCH] " + expectedSubject + "   ",
			Mode:    SubjectCleanWhitespace,
			Subject: "[PATCH] " + expectedSubject,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			prefix, subject := cleanSubject(test.Input, test.Mode)
			if prefix != test.Prefix {
				t.Errorf("incorrect prefix: expected %q, actual %q", test.Prefix, prefix)
			}
			if subject != test.Subject {
				t.Errorf("incorrect subject: expected %q, actual %q", test.Subject, subject)
			}
		})
	}
}
