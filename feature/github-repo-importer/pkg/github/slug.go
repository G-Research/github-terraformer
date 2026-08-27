package github

import (
	"strings"
	"unicode"

	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

func slugifyTeamName(name string) string {
	folded, _, err := transform.String(
		transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC),
		name,
	)
	if err != nil {
		folded = name
	}

	var b strings.Builder
	separated := false
	for _, r := range strings.ToLower(folded) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '_':
			b.WriteRune(r)
			separated = false
		default:
			if !separated && b.Len() > 0 {
				b.WriteByte('-')
				separated = true
			}
		}
	}

	return strings.TrimRight(b.String(), "-")
}
