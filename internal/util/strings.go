package util

import "strings"

func RemoveNonAlphabetChars(s string) string {
	s = strings.ToLower(s)

	var builder strings.Builder
	for _, r := range s {
		if 'a' <= r && r <= 'z' {
			builder.WriteRune(r)
		}
	}

	return builder.String()
}
