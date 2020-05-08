package db2struct

import (
	"strings"
)

func snake2camel(s string) string {
	var (
		sb       strings.Builder
		segments = strings.Split(s, "_")
	)
	sb.Grow(len(s))

	for _, seg := range segments {
		sb.WriteString(uppercaseFirstLetter(seg))
	}

	return sb.String()
}

func snake2uncapCamel(s string) string {
	s = snake2camel(s)
	return uncapFirstLetter(s)
}

func uppercaseFirstLetter(s string) string {
	return strings.ToUpper(string(s[0])) + s[1:]
}

func uncapFirstLetter(s string) string {
	return strings.ToLower(string(s[0])) + s[1:]
}
