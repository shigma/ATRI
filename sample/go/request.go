// package name: request
package main

/*
#include "def.h"
*/
import "C"
import (
	"fmt"
	"io/ioutil"
	"net/http"
)

//export Request
func Request(url_ *C.char, ctx C.size_t, cb C.Callback) {
	url := C.GoString(url_)
	go func() {
		resp, err := http.Get(url)
		if err != nil {
			C.InvokeCallback(cb, ctx, C.CString(err.Error()))
			return
		}
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			C.InvokeCallback(cb, ctx, C.CString(err.Error()))
			return
		}
		C.InvokeCallback(cb, ctx, C.CString(fmt.Sprintf("%s", body)))
	}()
}

func main() {
}
