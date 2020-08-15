package main

import (
	// #include "../bind/def.h"
	"C"
	"encoding/json"
	"unsafe"
)

var botRegistry map[int64]*CQBot = make(map[int64]*CQBot)

//export GoFree
func GoFree(p unsafe.Pointer) {
	C.free(p)
}

type GenericError struct {
	Message string
	Detail  interface{}
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

//export _setupDevice
func _setupDevice(infoC *C.char) *C.char {
	info := C.GoString(infoC)
	SetupDevice(info)
	return C.CString("null")
}

//export _newCQBot
func _newCQBot(uin int64, pwC *C.char) uintptr {
	pw := C.GoString(pwC)
	ptr := NewCQBot(uin, pw)
	botRegistry[uin] = ptr
	return uintptr(unsafe.Pointer(ptr))
}

//export _login
func _login(botC unsafe.Pointer, cb C.ByteCallback, ctx C.uintptr_t) {
	bot := (*CQBot)(botC)
	go func() {
		defer errorHandler(cb, ctx)
		rsp := bot.Login()
		b, _ := json.Marshal(rsp)
		C.InvokeByteCallback(cb, ctx, unsafe.Pointer(&b[0]), nil, C.size_t(len(b)))
	}()
}

//export onEvent
func onEvent(botC unsafe.Pointer, cb C.ByteCallback, ctx C.uintptr_t) {
	bot := (*CQBot)(botC)
	bot.onEvent(func(msg MSG) {
		b, _ := json.Marshal(msg)
		C.InvokeByteCallback(cb, ctx, unsafe.Pointer(&b[0]), nil, C.size_t(len(b)))
	})
}

//export _sendPrivateMessage
func _sendPrivateMessage(botC unsafe.Pointer, target int64, contentC *C.char, cb C.ByteCallback, ctx C.uintptr_t) {
	bot := (*CQBot)(botC)
	content := C.GoString(contentC)

	go func() {
		defer errorHandler(cb, ctx)
		resp := bot.SendPrivateMessage(target, content)
		b, _ := json.Marshal(resp)
		C.InvokeByteCallback(cb, ctx, unsafe.Pointer(&b[0]), nil, C.size_t(len(b)))
	}()
}

//export getFriendList
func getFriendList(botC unsafe.Pointer) *C.char {
	bot := (*CQBot)(botC)
	fs := bot.GetFriendList()
	b, _ := json.Marshal(fs)
	return C.CString(string(b))
}

//export getGroupList
func getGroupList(botC unsafe.Pointer) *C.char {
	bot := (*CQBot)(botC)
	gs := bot.GetGroupList()
	b, _ := json.Marshal(gs)
	return C.CString(string(b))
}

//export getGroupInfo
func getGroupInfo(botC unsafe.Pointer, groupId int64) *C.char {
	bot := (*CQBot)(botC)
	info := bot.GetGroupInfo(groupId)
	b, _ := json.Marshal(info)
	return C.CString(string(b))
}

//export getGroupMemberList
func getGroupMemberList(botC unsafe.Pointer, groupId int64) *C.char {
	bot := (*CQBot)(botC)
	members := bot.GetGroupMemberList(groupId)
	b, _ := json.Marshal(members)
	return C.CString(string(b))
}

func main() {
}
