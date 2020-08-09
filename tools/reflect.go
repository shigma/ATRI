package main

import (
	"bytes"
	"reflect"
	"strconv"
	"strings"

	"github.com/Mrs4s/MiraiGo/client"
)

func main() {
	LoginResponseT := reflect.TypeOf((*client.LoginResponse)(nil)).Elem()
	println(generate(LoginResponseT))
}

var test int
var intLength = reflect.TypeOf(test).Size()
var intEquiv string = map[bool]string{true: "Int64", false: "Int32"}[intLength == 8]
var uintEquiv string = map[bool]string{true: "UInt64", false: "UInt32"}[intLength == 8]

var typeMap = map[reflect.Kind]string{
	reflect.Bool:    "Bool",
	reflect.Int8:    "Int8",
	reflect.Int16:   "Int16",
	reflect.Int32:   "Int32",
	reflect.Uint8:   "UInt8",
	reflect.Uint16:  "UInt16",
	reflect.Uint32:  "UInt32",
	reflect.Int64:   "Int64",
	reflect.Uint64:  "UInt64",
	reflect.Float32: "Float32",
	reflect.Float64: "Float64",
	reflect.Uint:    uintEquiv,
	reflect.Int:     intEquiv,
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

func generate(inT reflect.Type) string {
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
		// print("numeric ")
		return typeMap[kind]
	case reflect.String:
		// print("string ")
		return "Text"
	case reflect.Struct:
		// print("struct ")
		structP := "struct " + inT.Name() + "{\n"
		for i := 0; i < inT.NumField(); i++ {
			field := inT.Field(i)
			structP += "  " + firstLetterLower(field.Name) + " @" + strconv.Itoa(i) + " : " + generate(field.Type) + ";\n"
		}
		structP += "}\n"
		return structP
	case reflect.Map:
		// TODO: inject this type
		// struct Map(Key, Value) {
		// 	entries @0 :List(Entry);
		// 	struct Entry {
		// 	  key @0 :Key;
		// 	  value @1 :Value;
		// 	}
		// }

		// print("map ")
		return "Map(" + generate(inT.Key()) + ", " + generate(inT.Elem()) + ")"

	case reflect.Array, reflect.Slice:
		// print("list ")
		elementT := inT.Elem()
		if elementT.Kind() == reflect.Int8 || elementT.Kind() == reflect.Uint8 {
			return "Data"
		} else {
			return "List(" + generate(elementT) + ")"
		}
	case reflect.Uintptr, reflect.Ptr, reflect.UnsafePointer:
		// print("ptr ")
		return generate(inT.Elem())
	case reflect.Complex64, reflect.Complex128:
		panic("complex")
	case reflect.Func:
		panic("func")
	case reflect.Chan:
		panic("chan")
	case reflect.Interface:
		panic("interface ")
	case reflect.Invalid:
		panic("invalid")
	default:
		panic("never ")
	}
}
