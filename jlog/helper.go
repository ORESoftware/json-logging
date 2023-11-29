package json_logging

import (
	"encoding/json"
	"fmt"
	"github.com/logrusorgru/aurora"
	"reflect"
	"runtime"
	"strconv"
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

	//s := createSpaces(depth, brk) + aurora.Bold("map(").String() + createNewline(brk, true)

	values := []string{}

	for i, k := range keys {
		val := m.MapIndex(k)

		z := getStringRepresentation(k.Interface(), size, brk, depth+1, cache) +
			" => " +
			getStringRepresentation(val.Interface(), size, brk, depth+1, cache) +
			addComma(i, n)

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

	var h = ""
	if n < 1 {
		h = fmt.Sprintf("(%T", x)
	} else {
		h = aurora.Black("map(").String()
	}

	s := aurora.Black(h).String() + createNewline(brk, true)

	for i := 0; i < n; i++ {
		s += createSpaces(depth, brk) + values[i] + createNewline(brk, true)
	}

	var end = ")"
	if n > 0 {
		end = ")"
	}

	return s + aurora.Black(end).String() + createNewline(brk, true)
}

func handleSliceAndArray(val reflect.Value, len int, brk bool, depth int, cache *map[*interface{}]string) string {

	s := createSpaces(depth, brk) + aurora.Bold("[").String()

	n := val.Len()
	for i := 0; i < n; i++ {
		s += createSpaces(depth, brk) +
			getStringRepresentation(val.Index(i).Interface(), len, brk, depth, cache) +
			addComma(i, n)
	}

	return s + createNewline(brk, true) + aurora.Bold("]").String()
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

	v := ""
	for i := 0; i < n; i++ {
		v += " "
	}
	return v
}

func handleStruct(val reflect.Value, size int, brk bool, depth int, cache *map[*interface{}]string) string {

	n := val.NumField()
	t := val.Type()

	//log.Println("ln:", ln)

	keys := []string{}
	values := []string{}

	for i := 0; i < n; i++ {

		k := t.Field(i).Name
		// keys => s += createSpaces(depth, brk) + k + ":"

		keys = append(keys, k+":")
		size = size + len(keys)
		//if strings.ToLower(k[:1]) == k[:1] {
		//	s += "(unknown val)"
		//	continue
		//}

		//rs := reflect.ValueOf(val.Interface()).Elem()
		//rf := rs.Field(i)
		// note technique stolen from here: https://stackoverflow.com/a/43918797/12211419

		rs := reflect.New(t).Elem()
		rs.Set(val)
		rf := rs.Field(i)
		rf = reflect.NewAt(rf.Type(), unsafe.Pointer(rf.UnsafeAddr())).Elem()

		//if val.CanInterface() {
		//	s += "(unknown val)"
		//	continue
		//}

		//fv := val.FieldByName(k)
		//fmt.Println(fv.Interface()) // 2
		//v := val.Field(i).Interface()

		v := rf.Interface()

		if (*cache)[&v] != "" {
			return (*cache)[&v]
		}

		//v := fv.Interface()
		z := getStringRepresentation(v, size, brk, depth+1, cache)
		(*cache)[&v] = z

		values = append(values, z)
		size = size + len(z)

		//log.Println("m:", ln)
		//s += createSpaces(depth, brk) + z + addComma(i, n) + createNewline(brk, true)
	}

	//log.Println("size:", size, "n:", n)

	if size > 100-depth {
		brk = true
	}

	s := t.Name() + " {" + createNewline(brk, n > 3)

	for i := 0; i < n; i++ {
		s += createSpaces(depth, brk) + keys[i]
		s += " " + values[i] + addComma(i, n) + createNewline(brk, true)
	}

	s += createSpaces(depth-1, brk) + "}" + createNewline(brk, false)

	return s
}

type Stringer interface {
	String() string
}

type ToString interface {
	ToString() string
}

func getStringRepresentation(v interface{}, size int, brk bool, depth int, cache *map[*interface{}]string) (s string) {

	defer func() {
		if r := recover(); r != nil {
			s = fmt.Sprintf("(go: unknown type: '%+v')", v)
		}
	}()

	//if &v == nil {
	//	return "<nil>"
	//}
	//
	//if v == nil {
	//	return "<nil>"
	//}

	val := reflect.ValueOf(v)
	var kind = val.Kind()

	if kind == reflect.Ptr {
		//v = val.Elem().Interface()
		//val = reflect.ValueOf(v)
		val = val.Elem()
		if val.IsValid() { // Check if the dereferenced value is valid
			v = val.Interface()
		}
	}

	if v == nil {
		return "<nil>"
	}

	if kind == reflect.Chan {
		return fmt.Sprintf("(chan %s)", val.Type().Elem().String())
	}

	if kind == reflect.Map {
		return handleMap(v, val, size, brk, depth, cache)
	}

	if kind == reflect.Slice {
		return handleSliceAndArray(val, size, brk, depth, cache)
	}

	if kind == reflect.Array {
		return handleSliceAndArray(val, size, brk, depth, cache)
	}
	//
	//if kind == reflect.Ptr {
	//	return getStringRepFromPointer(&v, size, brk, depth)
	//}

	if kind == reflect.Func {
		return "(" + runtime.FuncForPC(val.Pointer()).Name() + "(func))"
	}

	if kind == reflect.Struct {
		if (*cache)[&v] != "" {
			return (*cache)[&v]
		}
		(*cache)[&v] = handleStruct(val, size, brk, depth, cache)
		return (*cache)[&v]
	}

	if kind == reflect.String {
		return "'" + aurora.Green(v).String() + "'"
	}

	if _, ok := v.(string); ok {
		return "'" + aurora.Green(v.(string)).String() + "'"
	}

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

	if z, ok := v.(Stringer); ok && z != nil && &z != nil {
		return z.String()
	}

	if z, ok := v.(ToString); ok && z != nil && &z != nil {
		return z.ToString()
	}

	if z, ok := v.(error); ok && z != nil && &z != nil {
		return z.Error()
	}

	if z, err := json.Marshal(v); err == nil {
		return fmt.Sprintf("(go: unknown type: '%+v', as JSON: '%s')", v, z)
	}

	return fmt.Sprintf("(go: unknown type: '%+v')", v)

}

func getPrettyString(v interface{}, size int) string {
	var cache = make(map[*interface{}]string)
	return getStringRepresentation(v, size, false, 2, (&cache))
}
