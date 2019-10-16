package hieraapi

import (
	"fmt"
)

func JSONNOtHash(path string) error {
	return fmt.Errorf(`file '%s' does not contain a JSON object`, path)
}
func MissingRequiredOption(option string) error {
	return fmt.Errorf(`missing required provider option '%s'`, option)
}
func MissingRequiredEnvironmentVariable(name string) error {
	return fmt.Errorf(`missing required environment variable '%s'`, name)
}
func YamlNotHash(path string) error {
	return fmt.Errorf(`file '%s' does not contain a YAML hash`, path)
}
