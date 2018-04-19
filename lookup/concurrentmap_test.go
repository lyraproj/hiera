package lookup

import (
	"fmt"
	"testing"
	"time"
)

func TestConcurrentMap_EnsureSet(t *testing.T) {
	c := NewConcurrentMap(7)
	done := make(chan bool, 10)
	for o := 0; o < 1000; o++ {
		for i := 0; i < 10; i++ {
			go func(ix, ox int) {
				time.Sleep(time.Duration(ox) * time.Nanosecond)
				c.EnsureSet(fmt.Sprintf(`hello%d`, ox), func() interface{} {
					return ix
				})
				done <- true
			}(i, o)
		}
		for i := 0; i < 10; i++ {
			<-done
		}
	}
	fmt.Println(c.Get(`hello567`))
}
