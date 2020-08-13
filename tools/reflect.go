package main

import (
	"bytes"
	"fmt"
	"reflect"
	"strings"

	"github.com/Mrs4s/MiraiGo/message"
)

func main() {
	ForwardMessageT := reflect.TypeOf((*message.ForwardMessage)(nil)).Elem()
	symbols := make(map[string]reflect.Type)
	println(generate(ForwardMessageT, symbols))
	// LoginResponseT := reflect.TypeOf((*pack.CQBot)(nil)).Elem()
	// println(generate(LoginResponseT))
}

var test int
var intLength = reflect.TypeOf(test).Size()
var intEquiv string = map[bool]string{true: "Int64", false: "Int32"}[intLength == 8]
var uintEquiv string = map[bool]string{true: "UInt64", false: "UInt32"}[intLength == 8]

var typeMap = map[reflect.Kind]string{
	reflect.Bool:    "boolean",
	reflect.Int8:    "number",
	reflect.Int16:   "number", // consider typedef?
	reflect.Int32:   "number",
	reflect.Uint8:   "number",
	reflect.Uint16:  "number",
	reflect.Uint32:  "number",
	reflect.Int64:   "number",
	reflect.Uint64:  "number",
	reflect.Float32: "number",
	reflect.Float64: "number",
	reflect.Uint:    "number", // uintEquiv,
	reflect.Int:     "number", // intEquiv,
	reflect.String:  "string",
}

func firstLetterLower(in string) string {
	if len(in) < 2 {
		return strings.ToLower(in)
	}
	bts := []byte(in)
	lc := bytes.ToLower([]byte{bts[0]})
	rest := bts[1:]
	return string(bytes.Join([][]byte{lc, rest}, nil))
}

func generate(root reflect.Type, symbols map[string]reflect.Type) string {
	pending := []reflect.Type{root}
	symbols[root.Name()] = root
	tryAdd := func(t reflect.Type) {
		name := t.Name()
		tp, exist := symbols[name]
		if exist {
			tpPath, tPath := tp.PkgPath(), t.PkgPath()
			if tpPath != tPath {
				fmt.Printf("WARN: Same name %s different type, %s & %s\n", name, tPath, tpPath)
			}
			return
		}
		pending = append(pending, t)
		symbols[name] = t
	}
	str := "\n"
	for len(pending) > 0 {
		var t reflect.Type
		t, pending = pending[0], pending[1:]
		if t.Kind() == reflect.Interface {
			fmt.Printf("WARN: need to properly handle interface %s\n", t.Name())
			str += "interface " + t.Name() + " {}\n"
		} else {
			str += generate_struct(t, tryAdd)
		}
	}
	return str
}

func generate_struct(t reflect.Type, tryAdd func(reflect.Type)) string {
	_generate(t, tryAdd)
	structP := "interface " + t.Name() + " {\n"
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		structP += "  " + field.Name + ": " + _generate(field.Type, tryAdd) + "\n"
	}
	structP += "}\n\n"
	return structP
}

func _generate(inT reflect.Type, tryAdd func(reflect.Type)) string {
	kind := inT.Kind()
	switch kind {
	case reflect.Uint, reflect.Int, reflect.Int64, reflect.Uint64:
		if inT.Size() == 8 {
			println("WARN: not representable in nodejs ")
		}
		fallthrough
	case reflect.Bool,
		reflect.Int8, reflect.Int16, reflect.Int32,
		reflect.Uint8, reflect.Uint16, reflect.Uint32,
		reflect.Float32, reflect.Float64:
		fallthrough
	case reflect.String:
		// print("numeric ")
		return typeMap[kind]
	case reflect.Interface:
		tryAdd(inT)
		return inT.Name()
	case reflect.Struct:
		// print("struct ")
		tryAdd(inT)
		return inT.Name()
	case reflect.Map:
		// print("map ")
		return "record<" + _generate(inT.Key(), tryAdd) + ", " + _generate(inT.Elem(), tryAdd) + ">"
	case reflect.Array, reflect.Slice:
		// print("list ")
		elementT := inT.Elem()
		return _generate(elementT, tryAdd) + "[]"
	case reflect.Uintptr, reflect.Ptr:
		// print("ptr ")
		return _generate(inT.Elem(), tryAdd)
	case reflect.UnsafePointer:
		return "never"
	case reflect.Complex64, reflect.Complex128:
		panic("complex")
	case reflect.Func:
		// panic("func")
		return "Function"
	case reflect.Chan:
		panic("chan")
	case reflect.Invalid:
		panic("invalid")
	default:
		panic("never")
	}
}

func reflectFunc(struc reflect.Type) {
	for i := 0; i < struc.NumMethod(); i++ {
		method := struc.Method(i)
		t := method.Type
		for i := 0; i < t.NumIn(); i++ {
			arg := t.In(i)
			arg.Name() // expect simple
		}

		if t.NumOut() > 2 {
			panic("Too much return")
		}
		// 0 for no-return
		// 1 for a JSON-marshal-able struct as result
		// 2 for result and an err
		for i := 0; i < t.NumOut(); i++ {
			arg := t.Out(i)
			arg.Name() // expect simple
		}
	}
}
