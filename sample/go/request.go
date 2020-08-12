// package name: request
package main

/*
#include <stdlib.h>
#include <stddef.h>
#include "def.h"
*/
import "C"
import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"unsafe"
)

//export RequestC
func RequestC(urlC *C.char, cb C.ByteCallback, ctx C.uintptr_t) {
	url := C.GoString(urlC)
	go func() {
		defer errorHandler(cb, ctx)
		resp := request(url)
		b, err := json.Marshal(resp)
		if err != nil {
			panic(err)
		}

		C.InvokeByteCallback(cb, ctx, unsafe.Pointer(&b[0]), nil, C.size_t(len(b)))
	}()
}

func errorHandler(cb C.ByteCallback, ctx C.uintptr_t) {
	if e := recover(); e != nil {
		var b []byte
		var jsonErr error
		switch err := e.(type) {
		case error:
			b, jsonErr = json.Marshal(GenericError{
				Message: err.Error(),
				Detail:  err,
			})
			if jsonErr != nil {
				b, _ = json.Marshal(GenericError{
					Message: err.Error(),
				})
			}
		case string:
			b, _ = json.Marshal(GenericError{
				Message: err,
			})
		default:
			b, jsonErr = json.Marshal(GenericError{
				Message: "UNKNOWN",
				Detail:  err,
			})
			if jsonErr != nil {
				b, _ = json.Marshal(GenericError{
					Message: "UNKNOWN",
				})
			}
		}
		C.InvokeByteCallback(cb, ctx, nil, unsafe.Pointer(&b[0]), C.size_t(len(b)))
	}
}

type GenericError struct {
	Message string
	Detail  interface{}
}

type ResponseT struct {
	Response   string
	StatusCode int
}

func request(url string) ResponseT {
	resp, err := http.Get(url)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	return ResponseT{
		StatusCode: resp.StatusCode,
		Response:   fmt.Sprintf("%s", body),
	}
}

func main() {
}
