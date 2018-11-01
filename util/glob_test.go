package util

import (
	"fmt"
	"github.com/bmatcuk/doublestar"
)

func ExampleSplit() {
	matches, err := doublestar.Glob(`**/*_test.go`)
	if err != nil {
		fmt.Println(err)
	} else {
		for _, m := range matches {
			fmt.Println(m)
		}
	}
	// Output: glob_test.go
}