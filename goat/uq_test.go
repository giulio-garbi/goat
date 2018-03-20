// component_test.go
package goat

import (
	"testing"
//	"fmt"
)


func TestComponentEmpty(t *testing.T) {
    q := newUnboundChanInt()
    for i := 0; i < 10000; i++ {
        q.In <- i
    }
    for i := 0; i < 10000; i++ {
        d := <- q.Out
        if d!= i {
            t.Fail()
        }
    }
}

