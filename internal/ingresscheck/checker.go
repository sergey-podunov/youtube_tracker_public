package ingresscheck

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// Rule is a single ingress HTTP path rule.
type Rule struct {
	// Path is the rule's path; a start-anchored regex when PathType is ImplementationSpecific.
	Path string
	// PathType is one of "Exact", "Prefix" or "ImplementationSpecific".
	PathType string
}

var paramPattern = regexp.MustCompile(`\{[^}]*}`)

// UncoveredPaths reports which OpenAPI paths are not matched by any ingress rule.
//
// Template parameters in apiPaths (e.g. /youtube/video/{id}) are substituted with
// a concrete segment before matching. Rules are interpreted the way ingress-nginx
// does: Exact as literal equality, Prefix on /-segment boundaries, and
// ImplementationSpecific as a start-anchored regex (use-regex annotation).
// Paths equal to an exception entry or nested under it are skipped.
//
// The result is sorted. An error is returned for an ImplementationSpecific rule
// whose path is not a valid regex.
func UncoveredPaths(apiPaths []string, rules []Rule, exceptions []string) ([]string, error) {
	matchers := make([]func(string) bool, 0, len(rules))
	for _, rule := range rules {
		match, err := rule.matcher()
		if err != nil {
			return nil, err
		}
		matchers = append(matchers, match)
	}

	var uncovered []string
	for _, apiPath := range apiPaths {
		if isException(apiPath, exceptions) {
			continue
		}
		sample := paramPattern.ReplaceAllString(apiPath, "123")
		covered := false
		for _, match := range matchers {
			if match(sample) {
				covered = true
				break
			}
		}
		if !covered {
			uncovered = append(uncovered, apiPath)
		}
	}
	sort.Strings(uncovered)
	return uncovered, nil
}

func (r Rule) matcher() (func(string) bool, error) {
	switch r.PathType {
	case "Exact":
		return func(p string) bool { return p == r.Path }, nil
	case "Prefix":
		prefix := strings.TrimSuffix(r.Path, "/")
		return func(p string) bool { return p == prefix || strings.HasPrefix(p, prefix+"/") }, nil
	default: // ImplementationSpecific
		re, err := regexp.Compile("^" + r.Path)
		if err != nil {
			return nil, fmt.Errorf("invalid regex in ingress rule %q: %w", r.Path, err)
		}
		return re.MatchString, nil
	}
}

func isException(apiPath string, exceptions []string) bool {
	for _, prefix := range exceptions {
		if apiPath == prefix || strings.HasPrefix(apiPath, prefix+"/") {
			return true
		}
	}
	return false
}
