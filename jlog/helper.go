package json_logging

import (
	"encoding/json"
	"fmt"
	"github.com/logrusorgru/aurora"
	"log"
	"reflect"
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

		z := fmt.Sprintf("%v", k.Interface()) + " —> " + fmt.Sprintf("%v", val.Interface()) + addComma(i, n)
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
		return fmt.Sprintf("[]byte as str: '%s'", *vv)
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

func handleStruct(val reflect.Value, size int, brk bool, depth int, cache *map[*interface{}]string) string {

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
		b.WriteString(createSpaces(depth, brk) + keys[i])
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

// TODO: use StringBuilder
//var sb strings.Builder
//sb.WriteString("Hello, ")
//sb.WriteString(" years old.")
//greeting := sb.String()
// write direct to stdout using: sb.WriteTo(os.Stdout)

func getFuncSignature(v interface{}) string {

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

func getStringRepresentation(v interface{}, vv *interface{}, size int, brk bool, depth int, cache *map[*interface{}]string) (s string) {

	defer func() {
		if r := recover(); r != nil {
			fmt.Println("\n")
			fmt.Println(fmt.Sprintf("%v", r))
			debug.PrintStack()
			s = fmt.Sprintf("%v - (go: unknown type 2: '%v')", r, v)
		}
	}()

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

	val := reflect.ValueOf(&v)

	if !val.IsValid() {
		return fmt.Sprintf("(%s <nil-11>)", reflect.TypeOf(v).Kind().String())
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
			return "<nil-pointer>"
		}

		val = val.Elem()
		if val.IsValid() { // Check if the dereferenced value is valid
			v = val.Interface()

			if v == nil {
				return "<nil-12>"
			}

			if &v == nil {
				return "<nil-18>"
			}

			val = reflect.ValueOf(v)
			kind = val.Kind()

		} else {
			return "<nil-111>"
		}
	}

	if v == reflect.Ptr {
		// Dereference the pointer

		elem := val.Elem()
		// Convert the dereferenced value to a string
		if elem.IsValid() {
			return fmt.Sprintf("%v", elem.Interface())
		} else {
			return "<nil-99>"
		}
	}

	if v == nil {
		return "<nil-13>"
	}

	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Ptr && rv.IsNil() {
		return fmt.Sprintf("<nil> (%v)", rv.Type())
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
		//return "(" + runtime.FuncForPC(val.Pointer()).Name() + "(func))"
		return getFuncSignature(v)
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
		(*cache)[vv] = handleStruct(val, size, brk, depth, cache)
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

func getPrettyString(v interface{}, size int) string {
	var cache = make(map[*interface{}]string)
	return getStringRepresentation(v, &v, size, false, 2, &cache)
}
