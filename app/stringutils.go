package main

import (
	"strings"
	"unicode"
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
