package git

import (
	"fmt"
	"strconv"
)

// ParseFileMode converts a git-style file mode (e.g. "100644")
// into a human-readable string like "rw-r--r--".
func ParseFileMode(modeStr string) (string, error) {
	// Git modes are typically 6 digits. The last 3 represent permissions.
	// e.g. 100644 → 644
	if len(modeStr) < 3 {
		return "", fmt.Errorf("invalid mode: %s", modeStr)
	}

	permStr := modeStr[len(modeStr)-3:]
	permVal, err := strconv.Atoi(permStr)
	if err != nil {
		return "", err
	}

	return numericPermToLetters(permVal), nil
}

func numericPermToLetters(perm int) string {
	// Map each octal digit to rwx letters
	lookup := map[int]string{
		0: "---",
		1: "--x",
		2: "-w-",
		3: "-wx",
		4: "r--",
		5: "r-x",
		6: "rw-",
		7: "rwx",
	}

	u := perm / 100       // user
	g := (perm / 10) % 10 // group
	o := perm % 10        // others

	return lookup[u] + lookup[g] + lookup[o]
}

// IsBinary performs a heuristic check to determine if data is binary.
// Rules:
//   - Any NUL byte => binary
//   - Consider only a sample (up to 8 KiB). If >30% of bytes are control characters
//     outside the common whitespace/newline range, treat as binary.
func IsBinary(b []byte) bool {
	n := len(b)
	if n == 0 {
		return false
	}
	if n > 8192 {
		n = 8192
	}
	sample := b[:n]
	bad := 0
	for _, c := range sample {
		if c == 0x00 {
			return true
		}
		// Allow common whitespace and control: tab(9), LF(10), CR(13)
		if c == 9 || c == 10 || c == 13 {
			continue
		}
		// Count other control chars and DEL as non-text
		if c < 32 || c == 127 {
			bad++
		}
	}
	// If more than 30% of sampled bytes are non-text, consider binary
	return bad*100 > n*30
}
