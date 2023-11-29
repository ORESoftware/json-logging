package json_logging

import (
	"log"
	"reflect"
	"unsafe"
)

func cleanStruct(val reflect.Value) (z interface{}) {

	n := val.NumField()
	t := val.Type()

	var ret = struct {
	}{}

	for i := 0; i < n; i++ {

		k := t.Field(i).Name

		rs := reflect.New(t).Elem()
		rs.Set(val)
		rf := rs.Field(i)
		rf = reflect.NewAt(rf.Type(), unsafe.Pointer(rf.UnsafeAddr())).Elem()

		v := rf.Interface()

		log.Println(k, v)
	}

	//log.Println("size:", size, "n:", n)

	return ret

}

func cleanMap(m reflect.Value) (z interface{}) {

	var ret = make(map[interface{}]interface{})
	keys := m.MapKeys()

	for _, k := range keys {
		val := m.MapIndex(k)
		ret[k] = cleanUp(val)
	}

	return ret
}

func isNonComplexNum(kind reflect.Kind) bool {
	return kind == reflect.Int ||
		kind == reflect.Int8 ||
		kind == reflect.Int16 ||
		kind == reflect.Int32 ||
		kind == reflect.Int64 ||
		kind == reflect.Uint8 ||
		kind == reflect.Uint16 ||
		kind == reflect.Uint32 ||
		kind == reflect.Uint64
}

func cleanUp(v interface{}) (z interface{}) {

	val := reflect.ValueOf(v)
	kind := val.Kind()

	if kind == reflect.Bool {
		return v
	}

	if kind == reflect.String {
		return v
	}

	if isNonComplexNum(kind) {
		return v
	}

	if kind == reflect.Func {
		return "(go:func())"
	}

	if kind == reflect.Complex64 {
		return "(go:complex64)" // v.(complex64)
	}

	if kind == reflect.Complex128 {
		return "(go:complex128)" //v.(complex128)
	}

	if kind == reflect.Chan {
		return "(go:chan)"
	}

	if kind == reflect.UnsafePointer {
		return "(go:unsafePointer)"
	}

	if kind == reflect.Struct {
		return cleanStruct(val)
	}

	if kind == reflect.Struct {
		return cleanMap(val)
	}

	return v
}
