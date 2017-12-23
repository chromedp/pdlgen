package types

import (
	"regexp"
	"strings"

	"github.com/client9/misspell"
)

func init() {
	misspellReplacer.Compile()
}

// replacers.
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
