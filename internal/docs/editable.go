package docs

import "regexp"

// Mirrors chessdocs' modules/edit/utils/paths.ts: a page refuses edits when its
// frontmatter sets `editLink: false` or `dev: true`. The two implementations
// are kept identical so the site and the API never disagree about a page.
var (
	frontmatter   = regexp.MustCompile(`(?s)\A---\r?\n(.*?)\r?\n---`)
	editLinkFalse = regexp.MustCompile(`(?m)^editLink:[ \t]*false[ \t]*$`)
	devOnly       = regexp.MustCompile(`(?m)^dev:[ \t]*true[ \t]*$`)
)

// Editable reports whether a page's markdown source accepts reader edits. Only
// the frontmatter block is consulted, so prose that merely mentions the flags
// cannot lock a page.
func Editable(source string) bool {
	block := frontmatter.FindStringSubmatch(source)
	if block == nil {
		return true
	}
	return !editLinkFalse.MatchString(block[1]) && !devOnly.MatchString(block[1])
}
