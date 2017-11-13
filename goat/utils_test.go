package goat

import (
	"testing"
	"reflect"
)

func TestTupleEncoding(t *testing.T) {
    InitSend()
    t1 := NewTuple(7, "abc", true)
    if z1, _, _ := unescapeWithType(escapeWithType(t1, false),0); !reflect.DeepEqual(z1, t1) {
        t.Fail()
    }
    
    t2 := NewTuple(NewTuple("abc"), "abc", NewTuple(), NewTuple(7))
    if z2, _, _ := unescapeWithType(escapeWithType(t2, false),0); !reflect.DeepEqual(z2, t2) {
        t.Fail()
    }
}
