// package name: calculator
package main

import "C"
import (
	"github.com/JJ/pigo"
)

//export CalculatePI
func CalculatePI(x int64) *C.char {
	return C.CString(pigo.Pi(x))
}

func main() {
}
