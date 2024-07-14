package main

import "unicode"

func CamelCaseToSnakeCase(input string) string {
	snakeCaseString := ""
	for _, r := range input {
		if unicode.IsUpper(r) {
			snakeCaseString += "_" + string(unicode.ToLower(r))
			continue
		}

		snakeCaseString += string(r)
	}
	return snakeCaseString
}
