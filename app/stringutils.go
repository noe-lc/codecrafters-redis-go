package main

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

func CamelCaseToSnakeCase(input string) string {
	snakeCaseString := ""
	for _, r := range input {
		if unicode.IsUpper(r) {
			snakeCaseString += "_" + string(r)
			continue
		}

		snakeCaseString += string(r)
	}
	return strings.ToLower(snakeCaseString)
}

func CapitalizeFirstCharacter(s string) string {
	if s == "" {
		return s
	}

	_, size := utf8.DecodeRuneInString(s)
	return strings.ToUpper(s[:size]) + s[size:]
}
