package common

import (
	"strings"
	"unicode"
)

// ToPascalCase returns a pascal-cased (e.g. SomeValueLikeThis) out of a string
func ToPascalCase(value string) string {
	b := strings.Builder{}

	var toUpper bool

	for i, rune := range value {
		// Always upper the first character
		if i == 0 {
			toUpper = true
		}
		// Always upper the character after non-letter/non-digit skipping the character
		if !unicode.IsLetter(rune) && !unicode.IsDigit(rune) {
			toUpper = true
			continue
		}
		// If the flag was set by one of the previous steps
		if toUpper {
			rune = unicode.ToUpper(rune)
			toUpper = false
		}

		b.WriteRune(rune)
	}

	return b.String()
}

// ToUnderscoreCase returns a underscore-cased (e.g. some_value_like_this) out of a string
func ToUnderscoreCase(value string) string {
	b := strings.Builder{}

	for _, rune := range value {
		// Always upper the character after non-letter/non-digit skipping the character
		if !unicode.IsLetter(rune) && !unicode.IsDigit(rune) {
			b.WriteByte('_')
			continue
		}

		// convert pascal or camel case into underscore case
		if unicode.IsUpper(rune) {
			b.WriteByte('_')
		}

		b.WriteRune(unicode.ToLower(rune))
	}

	return strings.Trim(b.String(), "_")
}
