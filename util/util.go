package util

import (
	"regexp"
	"strings"
	"unicode"

	"github.com/client9/misspell"
	"github.com/kenshaw/snaker"
)

// DefaultCommentFormatter is a default comment formatter.
var DefaultCommentFormatter = &CommentFormatter{
	Width:  80,
	Prefix: `// `,
	Empty:  "[no description]",
	Pre:    CleanDesc,
	KeepUpper: map[string]bool{
		"DOM": true,
		"X":   true,
		"Y":   true,
		"UTC": true,
	},
	Keep: map[string]bool{
		"JavaScript": true,
	},
}

// CommentFormatter is a comment formatter.
type CommentFormatter struct {
	Width  int
	Prefix string
	Empty  string
	Pre    func(string) string
	// KeepUpper are names to keep in upper case.
	KeepUpper map[string]bool
	// Keep are names to maintain exact spelling.
	Keep map[string]bool
}

// NewCommentFormatter creates a new comment formatter.
func NewCommentFormatter(width int, prefix string, empty string) *CommentFormatter {
	return &CommentFormatter{Width: width, Prefix: prefix, Empty: empty}
}

// Format formats a comment, choping the passed prefix
func (cf *CommentFormatter) Format(s, chop, newstr string) string {
	s = strings.TrimPrefix(s, chop)
	if cf.Pre != nil {
		s = cf.Pre(s)
	}
	s = strings.TrimSpace(s)
	l := len(s)
	if newstr != "" && l > 0 {
		if i := strings.IndexFunc(s, unicode.IsSpace); i != -1 {
			firstWord, remaining := s[:i], s[i:]
			if snaker.IsInitialism(firstWord) || cf.KeepUpper[firstWord] {
				s = strings.ToUpper(firstWord)
			} else if cf.Keep[firstWord] {
				s = firstWord
			} else {
				s = strings.ToLower(firstWord[:1]) + firstWord[1:]
			}
			s += remaining
		}
	}
	s = newstr + strings.TrimSuffix(s, ".")
	if l < 1 {
		s += cf.Empty
	}
	s += "."
	var w string
	for i := strings.Index(s, "\n\n"); i != -1; i = strings.Index(s, "\n\n") {
		w += Wrap(s[:i], cf.Width-len(cf.Prefix), cf.Prefix) + "\n" + cf.Prefix + "\n"
		s = s[i+2:]
	}
	return w + Wrap(s, cf.Width-len(cf.Prefix), cf.Prefix)
}

// Wrap wraps a line of text to the specified width, adding the specified
// prefix to each wrapped line.
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

// CleanDesc cleans comments / descriptions of "<code>" and "</code>" strings
// and "`" characters, and fixes common misspellings.
func CleanDesc(s string) string {
	s, _ = misspellReplacer.Replace(codeRE.ReplaceAllString(s, ""))
	s = descReplacer.Replace(s)
	s = pStartRE.ReplaceAllString(s, "\n\n")
	s = pEndRE.ReplaceAllString(s, "")
	return s
}

// description replacers.
var (
	misspellReplacer = misspell.New()
	codeRE           = regexp.MustCompile(`(?i)<\/?code>`)
	pStartRE         = regexp.MustCompile(`(?i)<p>`)
	pEndRE           = regexp.MustCompile(`(?i)</p>`)
	descReplacer     = strings.NewReplacer(
		"&lt;", "<",
		"&gt;", ">",
		"&gt", ">",
		"`", "",
		"\n", " ",
	)
)

func init() {
	misspellReplacer.Compile()
}
