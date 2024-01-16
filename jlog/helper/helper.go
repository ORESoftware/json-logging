package hlpr

import (
	"encoding/json"
	"fmt"
	"github.com/logrusorgru/aurora"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"unsafe"
)

func addComma(i int, n int) string {
	if i < n-1 {
		return ", "
	}
	return ""
}

func handleMap(x interface{}, m reflect.Value, size int, brk bool, depth int, cache *map[*interface{}]string) string {

	keys := m.MapKeys()
	n := len(keys)

	if n < 1 {
		return fmt.Sprintf(aurora.Black("(empty %T)").String(), x)
	}

	//s := createSpaces(depth, brk) + aurora.Bold("map(").String() + createNewline(brk, true)

	values := []string{}

	for i, k := range keys {

		val := m.MapIndex(k)

		//z := getStringRepresentation(k.Interface(), nil, size, brk, depth+1, cache) +
		//	" => " +
		//	getStringRepresentation(val.Interface(), nil, size, brk, depth+1, cache) +
		//	addComma(i, n)

		// TODO: get colorzied version of the values in the map
		z := fmt.Sprintf("'%v'", k.Interface()) + aurora.Bold(" —> ").String() + fmt.Sprintf("%v", val.Interface()) + addComma(i, n)
		size = size + len(z)
		values = append(values, z)
	}

	if size > 100-depth {
		brk = true
		//size = 0
	}

	//keyType := reflect.ValueOf(keys).Type().Elem().String()
	//valType := m.Type().Elem().String()
	//z := fmt.Sprintf("map<%s,%s>(", keyType, valType)
	//log.Println(z)

	var b strings.Builder

	b.WriteString(aurora.Bold(fmt.Sprintf("%T (", x)).String() + createNewline(brk, true))

	for i := 0; i < n; i++ {
		b.WriteString(createSpaces(depth, brk) + values[i] + createNewline(brk, true))
	}

	b.WriteString(createSpaces(depth-1, brk))
	b.WriteString(aurora.Bold(")").String() + createNewline(brk, false))
	return b.String()
}

func handleSliceAndArray(vv *interface{}, val reflect.Value, len int, brk bool, depth int, cache *map[*interface{}]string) string {

	n := val.Len()
	t := val.Type()

	if n < 1 {
		return aurora.Black("[").String() + "" + aurora.Black(fmt.Sprintf("] (empty %v)", t)).String()
	}

	// sliceType := reflect.TypeOf(vv)

	//if val.Type() != sliceType {
	//	panic(fmt.Sprintf("mismatched types: %v %v", val.Type(), sliceType))
	//}

	// Get the type of the elements in the slice
	elementType := t.Elem()

	if elementType.Kind() == reflect.Uint8 {
		return aurora.Bold("[]byte as str:").String() + fmt.Sprintf(" '%s'", *vv)
	}

	var b strings.Builder
	b.WriteString(createSpaces(depth, brk) + aurora.Bold("[").String())

	for i := 0; i < n; i++ {
		b.WriteString(createSpaces(depth, brk))
		x := val.Index(i)
		val := x.Interface()
		ptr := val
		if x.CanAddr() {
			ptr = x.Addr().Interface()
		}
		//if x.IsValid() {
		//
		//}
		b.WriteString(getStringRepresentation(val, &ptr, len, brk, depth, cache))
		b.WriteString(addComma(i, n))
	}

	b.WriteString(createNewline(brk, true) + aurora.Bold("]").String())
	return b.String()
}

func createNewline(brk bool, also bool) string {
	if brk && also {
		return "\n"
	}
	return ""
}

func createSpaces(n int, brk bool) string {

	if !brk {
		return ""
	}

	var b strings.Builder
	for i := 0; i < n; i++ {
		b.WriteString(" ")
	}
	return b.String()
}

func HandleStruct(val reflect.Value, size int, brk bool, depth int, cache *map[*interface{}]string) string {

	n := val.NumField()
	t := val.Type()

	if n < 1 {
		return fmt.Sprintf(" %s { }", t.Name())
	}

	//log.Println("ln:", ln)

	keys := []string{}
	values := []string{}

	for i := 0; i < n; i++ {

		k := t.Field(i).Name

		keys = append(keys, k+":")
		size = size + len(keys)

		rs := reflect.New(t).Elem()
		rs.Set(val)
		rf := rs.Field(i)

		var v interface{}
		var ptr interface{}

		if rf.CanAddr() {
			// It's safe to use UnsafeAddr and NewAt since rf is addressable
			rf = reflect.NewAt(rf.Type(), unsafe.Pointer(rf.UnsafeAddr())).Elem()
			v = rf.Interface()
			ptr = rf.Addr().Interface()
		} else {
			// Handle the case where rf is not addressable
			// You might need to create a copy or take a different approach
			// Example: Creating a copy
			myCopy := reflect.New(rf.Type()).Elem()
			myCopy.Set(rf)
			v = myCopy.Interface()
			ptr = myCopy.Addr().Interface()
		}

		//v := fv.Interface()
		z := getStringRepresentation(&v, &ptr, size, brk, depth+1, cache)

		values = append(values, z)
		size = size + len(z)

		//log.Println("m:", ln)
		//s += createSpaces(depth, brk) + z + addComma(i, n) + createNewline(brk, true)
	}

	//log.Println("size:", size, "n:", n)

	if size > 100-depth {
		brk = true
	}

	var b strings.Builder
	b.WriteString(t.Name() + " {" + createNewline(brk, n > 0))

	for i := 0; i < n; i++ {
		var k = aurora.Bold(aurora.Blue(keys[i])).String()
		b.WriteString(createSpaces(depth, brk) + k)
		b.WriteString(" " + values[i] + addComma(i, n) + createNewline(brk, n > 0))
	}

	b.WriteString(createSpaces(depth-1, brk) + "}" + createNewline(brk, false))
	return b.String()
}

type Stringer interface {
	String() string
}

type ToString interface {
	ToString() string
}

func GetFuncSignature(v interface{}) string {

	funcValue := reflect.ValueOf(v)
	funcType := funcValue.Type()
	name := funcType.Name()

	// Function signature
	var params []string
	for i := 0; i < funcType.NumIn(); i++ {
		vv := funcType.In(i)
		nm := vv.Name()
		if vv.Kind() == reflect.Ptr {
			nm = vv.Elem().Name()
		}
		if vv.Kind() == reflect.Pointer {
			nm = vv.Elem().Name()
		}
		if vv.Kind() == reflect.UnsafePointer {
			nm = vv.Elem().Name()
		}
		if vv.Kind() == reflect.Func {
			nm = vv.Elem().Name()
			if strings.TrimSpace(nm) == "" {
				nm = "func"
			}
		}
		if strings.TrimSpace(nm) == "" {
			nm = vv.String()
		}
		if strings.TrimSpace(nm) == "" {
			nm = "<unk>"
		}
		params = append(params, nm)
	}

	var returns []string
	for i := 0; i < funcType.NumOut(); i++ {
		vv := funcType.Out(i)
		nm := vv.Name()
		kind := vv.Kind()
		if kind == reflect.Ptr {
			nm = vv.Elem().Name()
		}
		if kind == reflect.Pointer {
			nm = vv.Elem().Name()
		}

		if kind == reflect.UnsafePointer {
			nm = vv.Elem().Name()
		}
		if kind == reflect.Func {
			nm = vv.Elem().Name()
			if strings.TrimSpace(nm) == "" {
				nm = "func"
			}
		}
		if strings.TrimSpace(nm) == "" {
			nm = vv.String()
		}
		if strings.TrimSpace(nm) == "" {
			nm = "<unk>"
		}
		returns = append(returns, nm)
	}

	paramsStr := strings.Join(params, ", ")
	returnsStr := strings.Join(returns, ", ")

	if len(returns) < 1 {
		if name != "" {
			return fmt.Sprintf("func %s(%s)", name, paramsStr)
		}

		return fmt.Sprintf("(func(%s))", paramsStr)
	}

	if len(returns) < 2 {
		if name != "" {
			return fmt.Sprintf("(func %s(%s) => %s)", name, paramsStr, returnsStr)
		}

		return fmt.Sprintf("(func(%s) => %s)", paramsStr, returnsStr)
	}

	if name != "" {
		return fmt.Sprintf("(func %s(%s) => (%s))", name, paramsStr, returnsStr)
	}

	return fmt.Sprintf("(func(%s) => (%s))", paramsStr, returnsStr)
}

var mutex sync.Mutex

func getFormattedNilStr(str string) string {
	return aurora.Black(aurora.Bold("<nil>")).String()
}

func getStringRepresentation(v interface{}, vv *interface{}, size int, brk bool, depth int, cache *map[*interface{}]string) (s string) {

	defer func() {
		if r := recover(); r != nil {
			fmt.Println("\n")
			fmt.Println(fmt.Sprintf("%v", r))
			debug.PrintStack()
			s = fmt.Sprintf("%v - (go: unknown type 2: '%v')", r, v)
		}
	}()

	if z, ok := v.(aurora.Value); ok && z != nil {
		return z.String()
	}

	if z, ok := v.(*aurora.Value); ok && z != nil {
		return (*z).String()
	}

	if z, ok := (*vv).(aurora.Value); ok && z != nil {
		return (z).String()
	}

	if z, ok := (*vv).(*aurora.Value); ok && z != nil {
		return (*z).String()
	}

	if z, ok := v.(Stringer); ok && z != nil {
		return z.String()
	}

	if z, ok := v.(*Stringer); ok && z != nil {
		return (*z).String()
	}

	if z, ok := (*vv).(Stringer); ok && z != nil {
		return z.String()
	}

	if z, ok := (*vv).(*Stringer); ok && z != nil {
		return (*z).String()
	}

	mutex.Lock()

	if v, ok := (*cache)[vv]; ok {
		// TODO: verify the caching logic
		log.Println(aurora.Red("map cached used."))
		mutex.Unlock()
		return v
	}

	mutex.Unlock()

	if v == nil {
		return "<nil-2>"
	}

	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Ptr && rv.IsNil() {
		return fmt.Sprintf("<nil> (%v)", rv.Type())
	}

	val := reflect.ValueOf(&v)

	if !val.IsValid() {
		return fmt.Sprintf("<nil-11> (%s)", reflect.TypeOf(v).Kind().String())
	}

	if val.Kind() == reflect.Ptr && val.IsNil() {
		return fmt.Sprintf("<nil-111> (%v)", val.Type())
	}

	originalV := v
	originalVal := val
	var kind = val.Kind()

	if kind == reflect.Uintptr || kind == reflect.UnsafePointer || kind == reflect.Ptr {
		//v = val.Elem().Interface()
		//val = reflect.ValueOf(v)

		if val.IsNil() {
			return "<nil-pointer>"
		}

		val = val.Elem()

		if val.IsValid() { // Check if the dereferenced value is valid
			v = val.Interface()

			if v == nil {
				return "<nil-8>"
			}

			val = reflect.ValueOf(v)
			kind = val.Kind()

		} else {
			return "<nil-112>"
		}
	}

	if kind == reflect.Ptr {
		//v = val.Elem().Interface()
		//val = reflect.ValueOf(v)

		if val.IsNil() {
			return "<nil-44>"
		}

		val = val.Elem()

		if !val.IsValid() { // Check if the dereferenced value is valid
			return "<nil-111>"
		}

		v = val.Interface()

		if v == nil {
			return "<nil-12>"
		}

		val = reflect.ValueOf(v)
		kind = val.Kind()
	}

	if kind == reflect.Ptr {
		// Dereference the pointer
		val = reflect.ValueOf(v)
		kind = val.Kind()
		val = val.Elem()

		if !val.IsValid() {
			// Handle zero Value if necessary
			return "<nil>"
		}

		v = val.Interface()
		// Convert the dereferenced value to a string
		//if !elem.IsValid() {
		//	return fmt.Sprintf("%v", elem.Interface())
		//} else {
		//	return "<nil-99>"
		//}
		//if !val.IsValid() {
		//	return "<nil-99>"
		//}

		//if !val.IsValid() {
		//	return fmt.Sprintf("%v", val.Interface())
		//} else {
		//	return "<nil-94>"
		//}
	}

	if v == nil {
		return "<nil-13>"
	}

	rfx := reflect.ValueOf(v)
	if rfx.Kind() == reflect.Ptr && rfx.IsNil() {
		return fmt.Sprintf("<nil> (%v)", reflect.TypeOf(v).String())
	}

	if kind == reflect.Interface {
		myVal := reflect.ValueOf(v)
		myElem := myVal.Elem()

		if !myElem.IsValid() {
			// Handle zero Value if necessary
			return "<nil-8888>"
		}

		myInf := myElem.Interface()
		return getStringRepresentation(myInf, &myInf, size, brk, depth, cache)
	}

	if kind == reflect.Chan {
		return fmt.Sprintf("(chan %s)", val.Type().Elem().String())
	}

	if kind == reflect.Map {
		mutex.Lock()
		if v, ok := (*cache)[vv]; ok {
			// TODO: verify the caching logic
			log.Println(aurora.Red("map cached used."))
			mutex.Unlock()
			return v
		}
		(*cache)[vv] = "(circular)"
		//fmt.Printf("Map cache: '%v'", len(*cache))
		mutex.Unlock()
		(*cache)[vv] = handleMap(v, val, size, brk, depth, cache)
		return (*cache)[vv]
	}

	if kind == reflect.Slice {

		mutex.Lock()
		if v, ok := (*cache)[vv]; ok {
			// TODO: verify the caching logic
			log.Println(aurora.Red("slice cached used."))
			mutex.Unlock()
			return v
		}
		(*cache)[vv] = "(circular)"
		//fmt.Printf("Slice cache: '%v'", len(*cache))
		mutex.Unlock()

		var x = ""
		if m, ok := v.(Stringer); ok {
			x = m.String()
			(*cache)[vv] = fmt.Sprintf("%T - (As string: %s)", m, x)
		} else {
			(*cache)[vv] = handleSliceAndArray(vv, val, size, brk, depth, cache)
		}

		//safeStdout.Write([]byte((*cache)[&v]))
		return (*cache)[vv]
	}

	if kind == reflect.Array {
		mutex.Lock()
		if v, ok := (*cache)[vv]; ok {
			// TODO: verify the caching logic
			mutex.Unlock()
			return v
		}
		(*cache)[vv] = "(circular)"
		//fmt.Printf("Array cache: '%v'", len(*cache))
		mutex.Unlock()

		var x = ""
		if m, ok := v.(Stringer); ok {
			x = m.String()
			(*cache)[vv] = fmt.Sprintf("(%T (As string: '%s'))", m, x)
		} else {
			(*cache)[vv] = handleSliceAndArray(vv, val, size, brk, depth, cache)

		}

		return (*cache)[vv]
	}

	if kind == reflect.Func {
		return GetFuncSignature(v)
	}

	if kind == reflect.Struct {
		mutex.Lock()
		if v, ok := (*cache)[vv]; ok {
			// TODO: verify the caching logic
			mutex.Unlock()
			return v
		}
		(*cache)[vv] = "(circular)"
		//fmt.Printf("struct cache: '%v'", len(*cache))
		mutex.Unlock()
		(*cache)[vv] = HandleStruct(val, size, brk, depth, cache)
		return (*cache)[vv]
	}

	if z, ok := v.(string); ok {
		if len(z) < 1 {
			return aurora.Bold("''").String()
		}
		var trimmed = strings.TrimSpace(z)
		if len(trimmed) == len(z) {
			return aurora.Green(z).String()
		}
		return aurora.Bold("'").String() + aurora.Green(z).String() + aurora.Bold("'").String()
	}

	//if kind == reflect.String {
	//	//return "'" + aurora.Green(v).String() + "'"
	//	if len(val) < 1 {
	//		return aurora.Bold("''").String()
	//	}
	//	var trimmed = strings.TrimSpace(z)
	//	if len(trimmed) == len(z) {
	//		return aurora.Green(z).String()
	//	}
	//	return aurora.Bold("'").String() + aurora.Green(z).String() + aurora.Bold("'").String()
	//}

	if kind == reflect.Bool {
		return aurora.BrightBlue(strconv.FormatBool(v.(bool))).String()
	}

	if _, ok := v.(bool); ok {
		return aurora.BrightBlue(strconv.FormatBool(v.(bool))).String()
	}

	if kind == reflect.Int {
		return aurora.Yellow(v).String()
	}

	if _, ok := v.(int); ok {
		return aurora.Yellow(strconv.Itoa(v.(int))).String()
	}

	if kind == reflect.Int8 {
		return aurora.Yellow(v).String()
	}

	if _, ok := v.(int8); ok {
		return aurora.Yellow(v.(int8)).String()
	}

	if kind == reflect.Int16 {
		return aurora.Yellow(v).String()
	}

	if _, ok := v.(int16); ok {
		return aurora.Yellow(v.(int16)).String()
	}

	if kind == reflect.Int32 {
		return aurora.Yellow(v).String()
	}

	if _, ok := v.(int32); ok {
		return aurora.Yellow(v.(int32)).String()
	}

	if kind == reflect.Int64 {
		return aurora.Yellow(v).String()
	}

	if _, ok := v.(int64); ok {
		return aurora.Yellow(strconv.FormatInt(v.(int64), 1)).String()
	}

	if kind == reflect.Uint {
		return aurora.Yellow(v).String()
	}

	if _, ok := v.(uint); ok {
		return aurora.Yellow(v.(uint)).String()
	}

	if kind == reflect.Uint8 {
		return aurora.Yellow(v).String()
	}

	if _, ok := v.(uint8); ok {
		return aurora.Yellow(v).String()
	}

	if kind == reflect.Uint16 {
		return aurora.Yellow(v).String()
	}

	if _, ok := v.(uint16); ok {
		return aurora.Yellow(v.(uint16)).String()
	}

	if kind == reflect.Uint32 {
		return aurora.Yellow(v).String()
	}

	if _, ok := v.(uint32); ok {
		return aurora.Yellow(v.(uint32)).String()
	}

	if kind == reflect.Uint64 {
		return aurora.Yellow(v).String()
	}

	if _, ok := v.(uint64); ok {
		return aurora.Yellow(v.(uint64)).String()
	}

	if z, ok := v.(Stringer); ok && z != nil && &z != nil && v != nil {
		return z.String()
	}

	if z, ok := v.(ToString); ok && z != nil && &z != nil {
		return z.ToString()
	}

	if z, ok := v.(error); ok && z != nil && &z != nil {
		return z.Error()
	}

	//return "(boof)"

	if z, err := json.Marshal(v); err == nil {
		//fmt.Println("kind is:", kind.String())
		if originalV != v {
			return fmt.Sprintf("(go: unknown type 1a: '%+v/%+v/%v/%v', as JSON: '%s', kind: %s)", v, val, originalV, originalVal, z, kind.String())
		} else {
			return fmt.Sprintf("(go: unknown type 2a: '%+v/%+v', as JSON: '%s', kind: %s)", v, val, z, kind.String())
		}

	}

	if originalV != v {
		return fmt.Sprintf("(go: unknown type 3a: '%+v / %+v / %v / %v')", v, val, originalV, originalVal)
	} else {
		return fmt.Sprintf("(go: unknown type 4a: '%+v / %+v')", v, val)
	}

}

func DoCopyAndDerefStruct(s interface{}) interface{} {
	val := reflect.ValueOf(s).Elem()
	newStruct := reflect.New(val.Type()).Elem()

	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		newField := newStruct.Field(i)
		if field.Kind() == reflect.Ptr && !field.IsNil() {
			newField.Set(reflect.Indirect(field))
		} else {
			newField.Set(field)
		}
	}

	return newStruct.Interface()
}

func CopyAndDereference(s interface{}) interface{} {
	// // get reflect value
	val := reflect.ValueOf(s)

	// Dereference pointer if s is a pointer
	if val.Kind() == reflect.Ptr {
		if val.IsNil() {
			return nil
		}
		derefVal := val.Elem()
		if !derefVal.IsValid() {
			// Handle zero Value if necessary
			return nil
		}
		return CopyAndDereference(derefVal.Interface())
	}

	// Checking the type of myArray or mySlice
	if val.Kind() == reflect.Slice || val.Kind() == reflect.Array {
		n := val.Len()
		slice := make([]interface{}, n)
		for i := 0; i < n; i++ {
			// Recursively copy and dereference each element in the slice or array
			slice[i] = CopyAndDereference(val.Index(i).Interface())
		}
		return slice
	}

	// Checking the type of myStruct
	if val.Kind() == reflect.Struct {
		return DoCopyAndDerefStruct(s)
	}

	// Return the original value for types that are not pointer, slice, array, or struct
	return s
}

func GetPrettyString(v interface{}, size int) string {
	var cache = make(map[*interface{}]string)
	return getStringRepresentation(v, &v, size, false, 2, &cache)
}

type Cache = *map[*interface{}]*interface{}

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

func cleanStructWorks(val reflect.Value, cache Cache) interface{} {

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
				elem := fieldVal.Elem()

				if elem.CanInterface() {

					xyz := elem.Interface()
					copiedVal := reflect.ValueOf(CleanUp(&xyz, cache))

					// Set the copied value to the new pointer
					newPtr.Elem().Set(copiedVal)

					// Set the new pointer to the field
					newStruct.Field(i).Set(newPtr)
				}

			}
		} else if fieldVal.CanSet() {
			// For non-pointer fields, just copy the value
			newStruct.Field(i).Set(fieldVal)
		}
	}

	return newStruct.Interface()
}

func cleanStruct(v *interface{}, cache Cache) interface{} {

	val := reflect.ValueOf(*v)
	// we turn struct into a map so we can display
	var ret = map[string]interface{}{}

	if val.Elem().Kind() != reflect.Struct {
		z := val.Elem().Addr()
		if x, ok := (z.Interface()).(interface{}); ok {
			v = &x
		}
	}
	//val := val.Elem() // Dereference the pointer to get the struct

	for i := 0; i < val.NumField(); i++ {
		fieldVal := val.Field(i)
		fieldType := val.Type()     // Get the reflect.Type of the struct
		field := fieldType.Field(i) // Get the reflect.StructField
		itff := fieldVal.Interface()

		if fieldVal.Kind() == reflect.Ptr || fieldVal.Kind() == reflect.Interface {

			if !fieldVal.IsNil() {
				//ret[field.Name] = "(pointer)"
				//continue
				// Create a new instance of the type that the pointer points to
				newPtr := reflect.New(fieldVal.Elem().Type())

				// Recursively copy the value and get a reflect.Value
				copiedVal := reflect.ValueOf(CleanUp(&itff, cache))

				// Set the copied value to the new pointer
				newPtr.Elem().Set(copiedVal)
				intf := copiedVal.Interface()

				//// Set the new pointer to the field
				//newStruct.Field(i).Set(newPtr)
				ret[field.Name] = CleanUp(&intf, cache)
			} else {
				ret[field.Name] = "(nil pointer)"
			}

		} else {
			ret[field.Name] = CleanUp(&itff, cache)
		}

	}

	// Iterate over each field and copy

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

func cleanMap(v *interface{}, cache Cache) (z interface{}) {

	// TODO: if keys to map are not strings, then create a slice/array of Key/Value Structs
	//type KeyValuePair struct {
	//	Key   int    `json:"key"`
	//	Value string `json:"value"`
	//}

	m := reflect.ValueOf(*v)

	var ret = make(map[interface{}]interface{})
	keys := m.MapKeys()

	for _, k := range keys {
		val := m.MapIndex(k)
		inf := val.Interface()
		ret[k] = CleanUp(&inf, cache)
	}

	return ret
}

func cleanList(v *interface{}, cache Cache) (z interface{}) {

	val := reflect.ValueOf(v)

	var ret = []interface{}{}

	for i := 0; i < val.Len(); i++ {
		element := val.Index(i)
		inf := element.Interface()
		ret = append(ret, CleanUp(&inf, cache))
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

func CleanUp(v *interface{}, cache Cache) (z interface{}) {

	// TODO: this is not really working

	val := reflect.ValueOf(v)
	originalV := v

	//if (*cache)[v] != nil {
	//	return fmt.Sprintf("pointer 1: %+v", v)
	//}
	//
	//(*cache)[v] = new(interface{})

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

	kind := val.Kind()

	//if kind == reflect.Pointer {
	//	(*cache)[&v] = new(interface{})
	//}

	//originalV := v

	if kind == reflect.Pointer || kind == reflect.Interface {
		val = val.Elem()
		kind = val.Kind()
	}

	if kind == reflect.Pointer || kind == reflect.Interface {
		val = val.Elem()
		kind = val.Kind()
	}

	if kind == reflect.Ptr || kind == reflect.Interface {
		val = val.Elem()
		kind = val.Kind()

		if kind == reflect.Ptr || kind == reflect.Interface {
			// This block will not run for structInstance
			if val.Elem().CanAddr() {
				ptrVal := val.Elem().Addr()
				// Convert to interface and then to the specific pointer type (*int in this case)
				ptr, ok := ptrVal.Interface().(interface{})
				if ok {
					v = &ptr
				} else {
					return "(pointer thing 5)"
				}
			} else {
				return "(pointer thing 6)"
			}
		}

	}

	if v == nil {
		return fmt.Sprintf("<nil> (%T)", v)
	}

	if kind == reflect.Pointer || val.Kind() == reflect.Interface {
		// Use Elem() to get the underlying type

		val = val.Elem()
		kind = val.Kind()
		//v = val.Interface()

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

	if originalV != v && &originalV != &v {
		if (*cache)[v] != nil {
			return fmt.Sprintf("pointer 2: %+v", v)
		}

		(*cache)[v] = new(interface{})
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
		return cleanStruct(v, cache)
	}

	if kind == reflect.Map {
		// TODO: if keys to map are not strings, then create a slice/array of Key/Value Structs
		//type KeyValuePair struct {
		//	Key   int    `json:"key"`
		//	Value string `json:"value"`
		//}
		return cleanMap(v, cache)
	}

	if kind == reflect.Slice {
		return cleanList(v, cache)
	}

	if kind == reflect.Array {
		return cleanList(v, cache)
	}

	if z, ok := (*v).(Stringer); ok {
		return z.String()
	}

	if z, ok := (*v).(ToString); ok {
		return z.ToString()
	}

	return fmt.Sprintf("unknown type: %v", v)
}

func GetFilteredStacktrace() *[]string {
	// Capture the stack trace
	buf := make([]byte, 1024)
	n := runtime.Stack(buf, false)
	stackTrace := string(buf[:n])

	// Filter the stack trace
	lines := strings.Split(stackTrace, "\n")
	var filteredLines = []string{}
	for _, line := range lines {
		if !strings.Contains(line, "oresoftware/json-logging") {
			filteredLines = append(filteredLines, fmt.Sprintf("%s", strings.TrimSpace(line)))
		}
	}

	return &filteredLines
}

func OpenFile(fp string) (*os.File, error) {

	// Get the current working directory
	wd, err := os.Getwd()

	if err != nil {
		return nil, err
	}

	if !filepath.IsAbs(fp) {
		fp = filepath.Clean(filepath.Join(wd, "/", fp))
	}

	actualPath, err := filepath.EvalSymlinks(fp)

	if err != nil {
		return nil, err
	}

	if !filepath.IsAbs(actualPath) {
		fp = filepath.Clean(filepath.Join(wd, "/", actualPath))
	}

	// Open the file with O_APPEND flag
	return os.OpenFile(actualPath, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0644)

}
