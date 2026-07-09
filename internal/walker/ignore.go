package walker

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

// rule is a single compiled gitignore-style pattern.
type rule struct {
	pattern  string // cleaned pattern, no leading/trailing slash
	negate   bool   // prefixed with !
	dirOnly  bool   // suffixed with /
	anchored bool   // contains a slash in the middle -> match from root of .gitignore's dir
}

// IgnoreSet holds rules loaded from one or more .gitignore files plus built-in defaults.
type IgnoreSet struct {
	rules []rule
}

// defaultIgnores are always-skip directories
// even before consulting .gitignore.
var defaultIgnores = []string{".git"}

// NewIgnoreSet builds an IgnoreSet seeded with built-in defaults.
func NewIgnoreSet() *IgnoreSet {
	is := &IgnoreSet{}
	for _, d := range defaultIgnores {
		is.rules = append(is.rules, rule{pattern: d, dirOnly: true})
	}
	return is
}

// LoadGitignore parses a .gitignore file at the given path and appends its rules.
// Missing files are silently ignored (not every directory has one).
func (is *IgnoreSet) LoadGitignore(path string) error {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		r := rule{}
		if strings.HasPrefix(trimmed, "!") {
			r.negate = true
			trimmed = trimmed[1:]
		}
		if strings.HasSuffix(trimmed, "/") {
			r.dirOnly = true
			trimmed = strings.TrimSuffix(trimmed, "/")
		}
		if strings.Contains(strings.TrimPrefix(trimmed, "/"), "/") {
			r.anchored = true
		}
		trimmed = strings.TrimPrefix(trimmed, "/")
		r.pattern = trimmed
		is.rules = append(is.rules, r)
	}
	return scanner.Err()
}

// Match reports whether relPath (relative to the walk root, using forward slashes)
// should be ignored. isDir indicates whether relPath refers to a directory.
func (is *IgnoreSet) Match(relPath string, isDir bool) bool {
	base := filepath.Base(relPath)
	ignored := false
	for _, r := range is.rules {
		if r.dirOnly && !isDir {
			continue
		}
		var matched bool
		var err error
		if r.anchored {
			matched, err = filepath.Match(r.pattern, relPath)
		} else {
			matched, err = filepath.Match(r.pattern, base)
		}
		if err != nil {
			continue
		}
		if matched {
			ignored = !r.negate
		}
	}
	return ignored
}

// Clone returns a shallow copy of the rule set, used when descending into a
// subdirectory that may add its own nested .gitignore rules without mutating
// the parent's rule slice (avoids aliasing issues from append reuse).
func (is *IgnoreSet) Clone() *IgnoreSet {
	cp := &IgnoreSet{rules: make([]rule, len(is.rules))}
	copy(cp.rules, is.rules)
	return cp
}
