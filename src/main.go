package main

import (
	// #include "def.h"
	"C"

	"github.com/Mrs4s/MiraiGo/client"
)

// export login
func login(uin C.longlong, pw *C.char) bool {
	cli := client.NewClient(int64(uin), C.GoString(pw))
	_, err := cli.Login()
	return err != nil
}

func main() {
}
