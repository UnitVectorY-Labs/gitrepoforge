package engine

import (
	"fmt"
	"path"
	"path/filepath"
	"strings"
)

// FileFilter filters managed file paths using include and exclude glob patterns.
type FileFilter struct {
	Include []string
	Exclude []string
}

// NewFileFilter constructs and validates a FileFilter from include and exclude
// glob patterns. Patterns support standard path globs (*, ?, [...]) as well as
// ** for recursive directory matching.
func NewFileFilter(include, exclude []string) (FileFilter, error) {
	normInclude, err := normalizeFilterPatterns(include, "--file")
	if err != nil {
		return FileFilter{}, err
	}
	normExclude, err := normalizeFilterPatterns(exclude, "--exclude-file")
	if err != nil {
		return FileFilter{}, err
	}
	return FileFilter{
		Include: normInclude,
		Exclude: normExclude,
	}, nil
}

// IsActive reports whether any include or exclude filters are configured.
func (f FileFilter) IsActive() bool {
	return len(f.Include) > 0 || len(f.Exclude) > 0
}

// Matches reports whether filePath passes the include and exclude filters.
func (f FileFilter) Matches(filePath string) bool {
	if !f.IsActive() {
		return true
	}

	normalized := normalizeFilterPath(filePath)
	if len(f.Include) > 0 && !matchesAnyPattern(f.Include, normalized) {
		return false
	}
	if len(f.Exclude) > 0 && matchesAnyPattern(f.Exclude, normalized) {
		return false
	}
	return true
}

// MatchesManifest reports whether the generated managed-files manifest should
// be evaluated for the current filter.
//
// When no filter is active, the manifest is always evaluated. When a filter is
// active, the manifest is only evaluated if it is explicitly matched by an
// include pattern (and not excluded).
func (f FileFilter) MatchesManifest(manifestPath string) bool {
	if !f.IsActive() {
		return true
	}
	if len(f.Include) == 0 {
		return false
	}
	return f.Matches(manifestPath)
}

func normalizeFilterPatterns(patterns []string, flagName string) ([]string, error) {
	if len(patterns) == 0 {
		return nil, nil
	}

	normalized := make([]string, 0, len(patterns))
	for _, raw := range patterns {
		p := normalizeFilterPath(raw)
		if p == "" || p == "." {
			return nil, fmt.Errorf("%s pattern must not be empty", flagName)
		}
		if err := validateGlobPattern(p); err != nil {
			return nil, fmt.Errorf("invalid %s pattern %q: %w", flagName, raw, err)
		}
		normalized = append(normalized, p)
	}
	return normalized, nil
}

func normalizeFilterPath(p string) string {
	trimmed := strings.TrimSpace(p)
	if trimmed == "" {
		return ""
	}
	slashed := filepath.ToSlash(trimmed)
	for strings.HasPrefix(slashed, "./") {
		slashed = strings.TrimPrefix(slashed, "./")
	}
	slashed = strings.TrimPrefix(slashed, "/")
	return slashed
}

func validateGlobPattern(pattern string) error {
	for _, seg := range collapseDoubleStars(strings.Split(pattern, "/")) {
		if seg == "**" {
			continue
		}
		if _, err := path.Match(seg, ""); err != nil {
			return err
		}
	}
	return nil
}

func matchesAnyPattern(patterns []string, target string) bool {
	for _, pattern := range patterns {
		if matchGlob(pattern, target) {
			return true
		}
	}
	return false
}

func collapseDoubleStars(segs []string) []string {
	out := make([]string, 0, len(segs))
	for _, seg := range segs {
		if seg == "**" && len(out) > 0 && out[len(out)-1] == "**" {
			continue
		}
		out = append(out, seg)
	}
	return out
}

func matchGlob(pattern, target string) bool {
	if target == "" {
		return false
	}
	patSegs := collapseDoubleStars(strings.Split(pattern, "/"))
	targetSegs := strings.Split(target, "/")

	memo := make(map[[2]int]bool)
	var match func(i, j int) bool
	match = func(i, j int) bool {
		key := [2]int{i, j}
		if res, ok := memo[key]; ok {
			return res
		}

		var res bool
		switch {
		case i == len(patSegs) && j == len(targetSegs):
			res = true
		case i == len(patSegs):
			res = false
		case patSegs[i] == "**":
			if i == len(patSegs)-1 && i > 0 {
				// Trailing "/**" matches one or more remaining segments.
				res = j < len(targetSegs)
			} else {
				// Standalone "**" or intermediate "**/" matches zero or more segments.
				res = match(i+1, j) || (j < len(targetSegs) && match(i, j+1))
			}
		case j == len(targetSegs):
			res = false
		default:
			matched, err := path.Match(patSegs[i], targetSegs[j])
			res = err == nil && matched && match(i+1, j+1)
		}

		memo[key] = res
		return res
	}

	return match(0, 0)
}
