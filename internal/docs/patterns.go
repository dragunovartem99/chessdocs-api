package docs

import "regexp"

// The patterns the API contract is written in. Each one is copied verbatim from
// the matching schema in openapi.yaml, and contract_test.go re-reads that file
// to prove the copies are still exact.
var (
	// sourcePathPattern keeps submissions inside the docs tree: the leading
	// language directory is fixed and a dot may only appear in the ".md"
	// suffix, so no accepted path can traverse out of it.
	sourcePathPattern = regexp.MustCompile(`^[a-z]{2}/[\w/-]+\.md$`)
	langPattern       = regexp.MustCompile(`^[a-z]{2}$`)
	contactPattern    = regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]+$`)
)
