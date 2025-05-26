package utils

import (
	"fmt"
	"strings"
	"unicode"
)

// EscapeTOMLString escapes a string to be safely embedded in a TOML file.
func EscapeTOMLString(s string) string {
	// 1. Try Multi-line Literal String ('''...''')
	// This is the cleanest if it doesn't contain the delimiter.
	if !strings.Contains(s, "'''") {
		return fmt.Sprintf("'''%s'''", s)
	}

	// 2. Fallback to Multi-line Basic String ("""...""")
	// This requires escaping internal " and \ characters, and control characters.
	var sb strings.Builder
	sb.WriteString(`"""`)
	for _, r := range s {
		switch r {
		case '\\':
			sb.WriteString(`\\`)
		case '"':
			sb.WriteString(`\"`)
		case '\b':
			sb.WriteString(`\b`)
		case '\t':
			sb.WriteString(`\t`)
		case '\n':
			// TOML spec: "All other Unicode characters are allowed." for multi-line basic strings.
			// So, writing raw '\n' is technically allowed for readability.
			sb.WriteRune(r)
		case '\f':
			sb.WriteString(`\f`)
		case '\r':
			sb.WriteString(`\r`)
		default:
			// Handle other non-printable ASCII characters or characters that might confuse parsers
			// (e.g., ASCII control characters U+0000-U+001F, U+007F)
			if (r >= '\x00' && r <= '\x1f') || r == '\x7f' || !unicode.IsPrint(r) {
				// Use \uXXXX for common Unicode escapes
				fmt.Fprintf(&sb, "\\u%04x", r)
			} else {
				sb.WriteRune(r)
			}
		}
	}

	sb.WriteString(`"""`) // End TOML basic string delimiter
	return sb.String()
}
