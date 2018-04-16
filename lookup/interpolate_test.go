package lookup

import (
	"strings"
	"regexp"
	"fmt"
)

func ExampleInterpolate() {
	str := `hello %{a}, %{ b }.`
	str = regexp.MustCompile(`%\{[^\}]*\}`).ReplaceAllStringFunc(str, func (match string) string {
		match = strings.TrimSpace(match[2:len(match)-1])
		return match
	})
	fmt.Println(str)
	// Output: hello a, b.
}

