// Package go contains the valyala/quicktemplate based code generation
// templates for Go used by cdproto-gen.
package genutil

import (
	"regexp"
	"strings"
	"unicode"

	"github.com/client9/misspell"
	"github.com/knq/snaker"

	"github.com/chromedp/cdproto-gen/pdl"
)

// Comment consts.
const (
	CommentWidth  = 80
	CommentPrefix = `// `
)

// KeepUpper are names to keep in upper case.
var KeepUpper = map[string]bool{
	"DOM": true,
	"X":   true,
	"Y":   true,
}

// Keep are names to maintain exact spelling.
var Keep = map[string]bool{
	"JavaScript": true,
}

// FormatComment formats a comment.
func FormatComment(s, chop, newstr string) string {
	s = strings.TrimPrefix(s, chop)
	s = strings.TrimSpace(CleanDesc(s))

	l := len(s)
	if newstr != "" && l > 0 {
		if i := strings.IndexFunc(s, unicode.IsSpace); i != -1 {
			firstWord, remaining := s[:i], s[i:]
			if snaker.IsInitialism(firstWord) || KeepUpper[firstWord] {
				s = strings.ToUpper(firstWord)
			} else if Keep[firstWord] {
				s = firstWord
			} else {
				s = strings.ToLower(firstWord[:1]) + firstWord[1:]
			}
			s += remaining
		}
	}
	s = newstr + strings.TrimSuffix(s, ".")
	if l < 1 {
		s += "[no description]"
	}
	s += "."

	return Wrap(s, CommentWidth-len(CommentPrefix), CommentPrefix)
}

// Wrap wraps a line of text to the specified width, and adding the prefix to
// each wrapped line.
func Wrap(s string, width int, prefix string) string {
	words := strings.Fields(strings.TrimSpace(s))
	if len(words) == 0 {
		return s
	}

	wrapped := prefix + words[0]
	spaceLeft := width - len(wrapped)
	for _, word := range words[1:] {
		if len(word)+1 > spaceLeft {
			wrapped += "\n" + prefix + word
			spaceLeft = width - len(word)
		} else {
			wrapped += " " + word
			spaceLeft -= 1 + len(word)
		}
	}

	return wrapped
}

func init() {
	misspellReplacer.Compile()
}

// description replacers.
var (
	misspellReplacer = misspell.New()
	codeRE           = regexp.MustCompile(`<\/?code>`)
	descReplacer     = strings.NewReplacer(
		"&lt;", "<",
		"&gt;", ">",
		"&gt", ">",
		"`", "",
		"\n", " ",
	)
)

// CleanDesc cleans comments / descriptions of "<code>" and "</code>" strings
// and "`" characters, and fixes common misspellings.
func CleanDesc(s string) string {
	s, _ = misspellReplacer.Replace(codeRE.ReplaceAllString(s, ""))
	return descReplacer.Replace(s)
}

// PackageName returns the package name to use for a domain.
func PackageName(d *pdl.Domain) string {
	return strings.ToLower(d.Domain.String())
}
