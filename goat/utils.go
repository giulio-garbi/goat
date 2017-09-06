package goat

import (
    "fmt"
    "strconv"
)

func itoa(n int) string {
    return fmt.Sprintf("%d", n)
}

func atoi(s string) int {
    n, _ := strconv.Atoi(s)
    return n
}
