package lookup

import (
	"os"
	"fmt"
)

func ExampleLocation_resolve() {
	loc := &glob{`**/*_test.go`}
	pwd, _ := os.Getwd()
	fmt.Println(loc.resolve(nil, pwd))
	// Output: ss
}
