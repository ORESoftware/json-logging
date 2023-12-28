package json_logging

import (
	"fmt"
	"log"
	"reflect"
	"unsafe"
)

type Cache = *map[*interface{}]string

func copyStruct_Old(original interface{}, cache Cache) interface{} {
	originalVal := reflect.ValueOf(original)
	originalValElem := originalVal.Elem()
	originalValIntf := originalValElem.Interface()

	if originalVal.Kind() == reflect.Ptr {
		if k, ok := (*cache)[&originalValIntf]; ok {
			return k
		}
	}

	if originalValElem.Kind() == reflect.Ptr {
		if k, ok := (*cache)[&originalValIntf]; ok {
			return k
		}
	}

	//if originalVal.Kind() != reflect.Ptr || originalVal.Elem().Kind() != reflect.Struct {
	//	return original
	//}
	copyVal := reflect.New(originalVal.Type()).Elem()

	for i := 0; i < originalVal.NumField(); i++ {
		if originalVal.Field(i).CanInterface() { //only copy uppercase/expore
			copyVal.Field(i).Set(originalVal.Field(i))
		}
	}

	return copyVal.Addr().Interface()
}

func cleanStruct(val reflect.Value, cache Cache) interface{} {

	//if val.Kind() != reflect.Struct {
	//	panic("cleanStruct only accepts structs")
	//}

	// Create a new struct of the same type
	newStruct := reflect.New(val.Type()).Elem()

	// Iterate over each field and copy
	for i := 0; i < val.NumField(); i++ {
		fieldVal := val.Field(i)

		// Check if field is a pointer
		if fieldVal.Kind() == reflect.Ptr {
			if !fieldVal.IsNil() {
				// Create a new instance of the type that the pointer points to
				newPtr := reflect.New(fieldVal.Elem().Type())

				// Recursively copy the value and get a reflect.Value
				copiedVal := reflect.ValueOf(cleanStruct(fieldVal.Elem(), cache))

				// Set the copied value to the new pointer
				newPtr.Elem().Set(copiedVal)

				// Set the new pointer to the field
				newStruct.Field(i).Set(newPtr)
			}
		} else if fieldVal.CanSet() {
			// For non-pointer fields, just copy the value
			newStruct.Field(i).Set(fieldVal)
		}
	}

	return newStruct.Interface()
}

func cleanStructNew(val reflect.Value, cache Cache) interface{} {

	// we turn struct into a map so we can display
	var ret = map[interface{}]interface{}{}

	// Iterate over each field and copy
	for i := 0; i < val.NumField(); i++ {
		fieldVal := val.Field(i)
		fieldType := val.Type()     // Get the reflect.Type of the struct
		field := fieldType.Field(i) // Get the reflect.StructField

		if fieldVal.Kind() == reflect.Ptr {
			if !fieldVal.IsNil() {
				// Create a new instance of the type that the pointer points to
				newPtr := reflect.New(fieldVal.Elem().Type())

				// Recursively copy the value and get a reflect.Value
				copiedVal := reflect.ValueOf(cleanStruct(fieldVal.Elem(), cache))

				// Set the copied value to the new pointer
				newPtr.Elem().Set(copiedVal)

				//// Set the new pointer to the field
				//newStruct.Field(i).Set(newPtr)
				ret[field.Name] = cleanUp(newPtr, cache)
			}
		} else {
			ret[field.Name] = cleanUp(fieldVal, cache)
		}

	}

	return ret
}

func cleanStructOld(val reflect.Value) (z interface{}) {

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

func cleanMap(m reflect.Value, cache Cache) (z interface{}) {

	// TODO: if keys to map are not strings, then create a slice/array of Key/Value Structs
	//type KeyValuePair struct {
	//	Key   int    `json:"key"`
	//	Value string `json:"value"`
	//}

	var ret = make(map[interface{}]interface{})
	keys := m.MapKeys()

	for _, k := range keys {
		val := m.MapIndex(k)
		ret[k] = cleanUp(val, cache)
	}

	return ret
}

func cleanList(m reflect.Value, cache Cache) (z interface{}) {

	var ret = []interface{}{}

	for i := 0; i < m.Len(); i++ {
		// Get the element at index i
		element := m.Index(i)
		ret = append(ret, cleanUp(element, cache))
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

func cleanUp(v interface{}, cache *map[*interface{}]string) (z interface{}) {

	// TODO: this is not really working

	// https://chat.openai.com/share/2113eb47-c685-48f1-81d1-96c4956f4ea5

	/*

		In Go, json.Marshal returns an error in a few specific scenarios where the data structure provided to it cannot be serialized into JSON. These scenarios include:

		Unsupported Types: Go's json package does not support the serialization of certain types. If you try to marshal channels, functions, or complex numbers, json.Marshal will return an error.

		Cyclic References: If the data structure contains cyclic references (i.e., a struct that directly or indirectly references itself), json.Marshal will return an error. JSON cannot represent cyclic data structures.

		Invalid UTF-8 Strings: If a string or a slice of bytes contains invalid UTF-8 sequences and is set to be marshaled into a JSON string, json.Marshal may return an error since JSON strings must be valid UTF-8.

		Marshaler Errors: If a type implements the json.Marshaler interface and its MarshalJSON method returns an error, json.Marshal will propagate that error.

		Pointer to Uninitialized Struct: If you pass a pointer to an uninitialized struct (a nil pointer), json.Marshal will return an error.

		Large Floating-Point Values: Extremely large floating-point values (like math.Inf or math.NaN) can cause json.Marshal to return an error, as they do not have a direct representation in JSON.

		Unsupported Map Key Types: In Go, a map can have keys of nearly any type, but JSON only supports string keys in objects. If you try to marshal a map with non-string keys (like map[int]string), json.Marshal will return an error.

		It's important to note that json.Marshal does not return an error for marshaling private (unexported) struct fields. Instead, it silently ignores them. To include private fields in the JSON output, you either need to export these fields (make their first letter uppercase) or provide a custom marshaling method.

		Understanding these conditions can help in ensuring that the data structures used with json.Marshal are compatible with JSON's serialization requirements.


	*/

	val := reflect.ValueOf(&v)
	kind := val.Kind()

	if kind == reflect.Pointer || kind == reflect.Interface {
		val = val.Elem()
		kind = val.Kind()
		v = val.Interface()
	}

	if kind == reflect.Pointer || kind == reflect.Interface {
		val = val.Elem()
		kind = val.Kind()
		v = val.Interface()
	}

	if kind == reflect.Pointer || kind == reflect.Interface {
		val = val.Elem()
		kind = val.Kind()
		v = val.Interface()
	}

	if v == nil {
		return fmt.Sprintf("<nil> (%T)", v)
	}

	if val.Kind() == reflect.Interface {
		// Use Elem() to get the underlying type

		val = val.Elem()
		kind = val.Kind()
		v = val.Interface()

		// Check again if the concrete value is also an interface
		if val.Kind() == reflect.Interface {
			// Get type information about the interface
			typ := val.Type()

			// You can also check if the interface is nil
			if val.IsNil() {
				return fmt.Sprintf("Nested interface type: %v, but it is nil", typ)
			} else {
				// Get more information about the non-nil interface
				concreteVal := val.Elem()
				concreteType := concreteVal.Type()
				return fmt.Sprintf("Nested interface type: %v, contains value of type: %v", typ, concreteType)
			}
		}
	}

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
		return fmt.Sprintf("(go:complex64:%+v)", v) // v.(complex64)
	}

	if kind == reflect.Complex128 {
		return "(go:complex128)" //v.(complex128)
	}

	if kind == reflect.Chan {
		return fmt.Sprintf("(go:chan:%+v)", v)
	}

	if kind == reflect.UnsafePointer {
		return "(go:UnsafePointer)"
	}

	if kind == reflect.Interface {
		//return copyStruct(v, cache)
		//actualValue := val.Elem()
		//t := actualValue.Type(
		return "inf Interface type"
	}

	if kind == reflect.Struct {
		//panic("here")
		//return copyStruct(v, cache)
		//actualValue := val.Elem()
		//t := actualValue.Type()
		//if t.Kind() != reflect.Interface {
		//	intf := actualValue.Interface()
		//	return cleanUp(intf, cache)
		//}
		//fmt.Println(val)
		return cleanStruct(val, cache)
	}

	if kind == reflect.Map {
		// TODO: if keys to map are not strings, then create a slice/array of Key/Value Structs
		//type KeyValuePair struct {
		//	Key   int    `json:"key"`
		//	Value string `json:"value"`
		//}
		return cleanMap(val, cache)
	}

	if kind == reflect.Slice {
		return cleanList(val, cache)
	}

	if kind == reflect.Array {
		return cleanList(val, cache)
	}

	fmt.Println("here 3")
	fmt.Println("kind:", kind.String())

	if z, ok := v.(Stringer); ok {
		return z.String()
	}

	if z, ok := v.(ToString); ok {
		return z.ToString()
	}

	return fmt.Sprintf("unknown type: %v", v)
}
