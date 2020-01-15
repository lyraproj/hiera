package api

import (
	"fmt"
)

// JSONNOtHash creates an error with a descriptive text and returns it.
func JSONNOtHash(path string) error {
	return fmt.Errorf(`file '%s' does not contain a JSON object`, path)
}

// MissingRequiredOption creates an error with a descriptive text and returns it.
func MissingRequiredOption(option string) error {
	return fmt.Errorf(`missing required provider option '%s'`, option)
}

// MissingRequiredEnvironmentVariable creates an error with a descriptive text and returns it.
func MissingRequiredEnvironmentVariable(name string) error {
	return fmt.Errorf(`missing required environment variable '%s'`, name)
}

// YamlNotHash creates an error with a descriptive text and returns it.
func YamlNotHash(path string) error {
	return fmt.Errorf(`file '%s' does not contain a YAML hash`, path)
}
