package normalize

import (
	"strings"
	"unicode"
)

func Query(raw string) (string, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", false
	}

	var builder strings.Builder
	builder.Grow(len(raw))

	previousWasSpace := false

	for _, r := range raw {
		if unicode.IsSpace(r) {
			if builder.Len() > 0 && !previousWasSpace {
				builder.WriteRune(' ')
				previousWasSpace = true
			}

			continue
		}

		if unicode.IsControl(r) {
			continue
		}

		builder.WriteRune(unicode.ToLower(r))
		previousWasSpace = false
	}

	normalized := strings.TrimSpace(builder.String())
	if normalized == "" {
		return "", false
	}

	return normalized, true
}
