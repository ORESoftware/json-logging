package json_logging

import "reflect"

func cleanStruct(v interface{}) (z interface{}){

	return v;
}

func cleanMap(v interface{}) (z interface{}){

	return v;
}

func cleanUp(v interface{}) (z interface{}){

	val := reflect.ValueOf(v)
	kind := val.Kind()

	if  kind == reflect.Func {
		return "(go:func())"
	}

	if  kind == reflect.Complex64 {
		return "(go:complex64)" // v.(complex64)
	}

	if kind == reflect.Complex128 {
		return "(go:complex128)" //v.(complex128)
	}

	if  kind == reflect.Chan {
		return "(go:chan)"
	}

	if  kind == reflect.UnsafePointer {
		return "(go:unsafePointer)"
	}

	if kind == reflect.Struct {
		return cleanStruct(v)
	}

	return v;

}