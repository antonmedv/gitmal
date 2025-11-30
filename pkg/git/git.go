package git

import (
	"bufio"
	"fmt"
	"io"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

func Branches(repoDir string, filter *regexp.Regexp, defaultBranch string) ([]Ref, error) {
	cmd := exec.Command("git", "for-each-ref", "--format=%(refname:short)", "refs/heads/")
	if repoDir != "" {
		cmd.Dir = repoDir
	}
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list branches: %w", err)
	}
	lines := strings.Split(string(out), "\n")
	branches := make([]Ref, 0, len(lines))
	for _, line := range lines {
		if line == "" {
			continue
		}

		if filter != nil && !filter.MatchString(line) && line != defaultBranch {
			continue
		}
		branches = append(branches, Ref(line))
	}
	return branches, nil
}

func Tags(repoDir string) ([]Tag, error) {
	format := []string{
		"%(refname:short)",    // tag name
		"%(creatordate:unix)", // creation date
		"%(objectname)",       // commit hash for lightweight tags
		"%(*objectname)",      // peeled object => commit hash
	}
	args := []string{
		"for-each-ref",
		"--sort=-creatordate",
		"--format=" + strings.Join(format, "%00"),
		"refs/tags",
	}
	cmd := exec.Command("git", args...)
	if repoDir != "" {
		cmd.Dir = repoDir
	}
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list tags: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	tags := make([]Tag, 0, len(lines))

	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.Split(line, "\x00")
		if len(parts) != len(format) {
			continue
		}
		name, timestamp, objectName, commitHash := parts[0], parts[1], parts[2], parts[3]
		timestampInt, err := strconv.Atoi(timestamp)
		if err != nil {
			return nil, fmt.Errorf("failed to parse tag creation date: %w", err)
		}
		if commitHash == "" {
			commitHash = objectName // tag is lightweight
		}
		tags = append(tags, Tag{
			Name:       name,
			Date:       time.Unix(int64(timestampInt), 0),
			CommitHash: commitHash,
		})
	}

	return tags, nil
}

func Files(ref Ref, repoDir string) ([]Blob, error) {
	if ref == "" {
		ref = "HEAD"
	}

	// -r: recurse into subtrees
	// -l: include blob size
	cmd := exec.Command("git", "ls-tree", "--full-tree", "-r", "-l", string(ref))
	if repoDir != "" {
		cmd.Dir = repoDir
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to get stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to get stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start git ls-tree: %w", err)
	}

	files := make([]Blob, 0, 256)

	// Read stdout line by line; each line is like:
	// <mode> <type> <object> <size>\t<path>
	// Example: "100644 blob e69de29... 12\tREADME.md"
	scanner := bufio.NewScanner(stdout)

	// Allow long paths by increasing the scanner buffer limit
	scanner.Buffer(make([]byte, 0, 64*1024), 2*1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		// Split header and path using the tab delimiter
		// to preserve spaces in file names
		tab := strings.IndexByte(line, '\t')
		if tab == -1 {
			return nil, fmt.Errorf("expected tab delimiter in ls-tree output: %s", line)
		}
		header := line[:tab]
		path := line[tab+1:]

		// header fields: mode, type, object, size
		parts := strings.Fields(header)
		if len(parts) < 4 {
			return nil, fmt.Errorf("unexpected ls-tree output format: %s", line)
		}
		modeNumber := parts[0]
		typ := parts[1]
		// object := parts[2]
		sizeStr := parts[3]

		if typ != "blob" {
			// We only care about files (blobs)
			continue
		}

		// Size could be "-" for non-blobs in some forms;
		// for blobs it should be a number.
		size, err := strconv.ParseInt(sizeStr, 10, 64)
		if err != nil {
			return nil, err
		}

		mode, err := ParseFileMode(modeNumber)
		if err != nil {
			return nil, err
		}

		files = append(files, Blob{
			Ref:      ref,
			Mode:     mode,
			Path:     path,
			FileName: filepath.Base(path),
			Size:     size,
		})
	}

	if err := scanner.Err(); err != nil {
		// Drain stderr to include any git error message
		_ = cmd.Wait()
		b, _ := io.ReadAll(stderr)
		if len(b) > 0 {
			return nil, fmt.Errorf("failed to read ls-tree output: %v: %s", err, string(b))
		}
		return nil, fmt.Errorf("failed to read ls-tree output: %w", err)
	}

	// Ensure the command completed successfully
	if err := cmd.Wait(); err != nil {
		b, _ := io.ReadAll(stderr)
		if len(b) > 0 {
			return nil, fmt.Errorf("git ls-tree %q failed: %v: %s", ref, err, string(b))
		}
		return nil, fmt.Errorf("git ls-tree %q failed: %w", ref, err)
	}

	return files, nil
}

func BlobContent(ref Ref, path string, repoDir string) ([]byte, bool, error) {
	if ref == "" {
		ref = "HEAD"
	}
	// Use `git show ref:path` to get the blob content at that ref
	cmd := exec.Command("git", "show", string(ref)+":"+path)
	if repoDir != "" {
		cmd.Dir = repoDir
	}
	out, err := cmd.Output()
	if err != nil {
		// include stderr if available
		if ee, ok := err.(*exec.ExitError); ok {
			return nil, false, fmt.Errorf("git show failed: %v: %s", err, string(ee.Stderr))
		}
		return nil, false, fmt.Errorf("git show failed: %w", err)
	}
	return out, IsBinary(out), nil
}

func Commits(ref Ref, repoDir string) ([]Commit, error) {
	format := []string{
		"%H",  // commit hash
		"%h",  // abbreviated commit hash
		"%s",  // subject
		"%b",  // body
		"%an", // author name
		"%ae", // author email
		"%ad", // author date
		"%P",  // parent hashes
		"%D",  // ref names without the "(", ")" wrapping.
	}

	args := []string{
		"log",
		"--date=unix",
		"--pretty=format:" + strings.Join(format, "\x1F"),
		"-z", // Separate the commits with NULs instead of newlines
		string(ref),
	}

	cmd := exec.Command("git", args...)
	if repoDir != "" {
		cmd.Dir = repoDir
	}

	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(out), "\x00")
	commits := make([]Commit, 0, len(lines))
	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.Split(line, "\x1F")
		if len(parts) != len(format) {
			return nil, fmt.Errorf("unexpected commit format: %s", line)
		}
		full, short, subject, body, author, email, date, parents, refs :=
			parts[0], parts[1], parts[2], parts[3], parts[4], parts[5], parts[6], parts[7], parts[8]
		timestamp, err := strconv.Atoi(date)
		if err != nil {
			return nil, fmt.Errorf("failed to parse commit date: %w", err)
		}
		commits = append(commits, Commit{
			Hash:      full,
			ShortHash: short,
			Subject:   subject,
			Body:      body,
			Author:    author,
			Email:     email,
			Date:      time.Unix(int64(timestamp), 0),
			Parents:   strings.Fields(parents),
			RefNames:  parseRefNames(refs),
		})
	}
	return commits, nil
}

func parseRefNames(refNames string) []RefName {
	refNames = strings.TrimSpace(refNames)
	if refNames == "" {
		return nil
	}

	parts := strings.Split(refNames, ", ")
	out := make([]RefName, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}

		// tag: v1.2.3
		if strings.HasPrefix(p, "tag: ") {
			out = append(out, RefName{
				Kind: RefKindTag,
				Name: strings.TrimSpace(strings.TrimPrefix(p, "tag: ")),
			})
			continue
		}

		// HEAD -> main
		if strings.HasPrefix(p, "HEAD -> ") {
			out = append(out, RefName{
				Kind:   RefKindHEAD,
				Name:   "HEAD",
				Target: strings.TrimSpace(strings.TrimPrefix(p, "HEAD -> ")),
			})
			continue
		}

		// origin/HEAD -> origin/main
		if strings.Contains(p, " -> ") && strings.HasSuffix(strings.SplitN(p, " -> ", 2)[0], "/HEAD") {
			leftRight := strings.SplitN(p, " -> ", 2)
			out = append(out, RefName{
				Kind:   RefKindRemoteHEAD,
				Name:   strings.TrimSpace(leftRight[0]),
				Target: strings.TrimSpace(leftRight[1]),
			})
			continue
		}

		// Remote branch like origin/main
		if strings.Contains(p, "/") {
			out = append(out, RefName{
				Kind: RefKindRemote,
				Name: p,
			})
			continue
		}

		// Local branch
		out = append(out, RefName{
			Kind: RefKindBranch,
			Name: p,
		})
	}
	return out
}

func CommitDiff(hash, repoDir string) (string, error) {
	// unified diff without a commit header
	cmd := exec.Command("git", "show", "--pretty=format:", "--patch", hash)
	if repoDir != "" {
		cmd.Dir = repoDir
	}
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}
