package json_logging

import (
	"encoding/json"
	"fmt"
	"github.com/logrusorgru/aurora"
	"log"
	"reflect"
	"runtime"
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
		return fmt.Sprintf("(%T)", x)
	}

	//s := createSpaces(depth, brk) + aurora.Bold("map(").String() + createNewline(brk, true)

	values := []string{}

	for i, k := range keys {

		val := m.MapIndex(k)

		//z := getStringRepresentation(k.Interface(), nil, size, brk, depth+1, cache) +
		//	" => " +
		//	getStringRepresentation(val.Interface(), nil, size, brk, depth+1, cache) +
		//	addComma(i, n)

		z := fmt.Sprintf("%v", k.Interface()) + " => " + fmt.Sprintf("%v", val.Interface()) + addComma(i, n)
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

	b.WriteString(aurora.Black(fmt.Sprintf("%T (", x)).String() + createNewline(brk, true))

	for i := 0; i < n; i++ {
		b.WriteString(createSpaces(depth, brk) + values[i] + createNewline(brk, true))
	}

	b.WriteString(createSpaces(depth-1, brk))
	b.WriteString(aurora.Black(")").String() + createNewline(brk, false))
	return b.String()
}

func handleSliceAndArray(val reflect.Value, len int, brk bool, depth int, cache *map[*interface{}]string) string {

	n := val.Len()

	if n < 1 {
		return aurora.Bold("[").String() + " " + aurora.Bold("]").String()
	}

	var b strings.Builder
	b.WriteString(createSpaces(depth, brk) + aurora.Bold("[").String())

	for i := 0; i < n; i++ {
		b.WriteString(createSpaces(depth, brk))
		intrfce := val.Index(i).Interface()
		b.WriteString(getStringRepresentation(&intrfce, &intrfce, len, brk, depth, cache))
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

		//v := fv.Interface()
		z := getStringRepresentation(&v, &v, size, brk, depth+1, cache)

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

var mutex sync.Mutex

func getStringRepresentation(v interface{}, vv *interface{}, size int, brk bool, depth int, cache *map[*interface{}]string) (s string) {

	defer func() {
		if r := recover(); r != nil {
			s = fmt.Sprintf("%v - (go: unknown type 2: '%v')", r, v)
		}
	}()

	if &v == nil {
		return "<nil>"
	}

	if v == nil {
		return "<nil>"
	}

	val := reflect.ValueOf(v)

	if !val.IsValid() {
		return fmt.Sprintf("(%s <nil>)", reflect.TypeOf(v).Kind().String())
	}

	var kind = val.Kind()

	if kind == reflect.UnsafePointer {
		//v = val.Elem().Interface()
		//val = reflect.ValueOf(v)
		val = val.Elem()
		if val.IsValid() { // Check if the dereferenced value is valid
			v = val.Interface()

			if v == nil {
				return "<nil>"
			}

			val = reflect.ValueOf(v)
			kind = val.Kind()
		}
	}

	if kind == reflect.Uintptr {
		//v = val.Elem().Interface()
		//val = reflect.ValueOf(v)
		val = val.Elem()
		if val.IsValid() { // Check if the dereferenced value is valid
			v = val.Interface()

			if v == nil {
				return "<nil>"
			}

			val = reflect.ValueOf(v)
			kind = val.Kind()
		}
	}

	if kind == reflect.Pointer {
		//v = val.Elem().Interface()
		//val = reflect.ValueOf(v)
		val = val.Elem()
		if val.IsValid() { // Check if the dereferenced value is valid
			v = val.Interface()

			if v == nil {
				return "<nil>"
			}

			val = reflect.ValueOf(v)
			kind = val.Kind()
		}
	}

	if kind == reflect.Ptr {
		//v = val.Elem().Interface()
		//val = reflect.ValueOf(v)
		val = val.Elem()
		if val.IsValid() { // Check if the dereferenced value is valid
			v = val.Interface()

			if v == nil {
				return "<nil>"
			}

			val = reflect.ValueOf(v)
			kind = val.Kind()
		}
	}

	if v == reflect.Ptr {
		// Dereference the pointer
		elem := val.Elem()
		// Convert the dereferenced value to a string
		return fmt.Sprintf("%v", elem.Interface())
	}

	if v == nil {
		return "<nil>"
	}

	if &v == nil {
		return "<nil>"
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
		(*cache)[vv] = handleSliceAndArray(val, size, brk, depth, cache)
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
		(*cache)[vv] = handleSliceAndArray(val, size, brk, depth, cache)
		return (*cache)[vv]
	}

	if kind == reflect.Func {
		return "(" + runtime.FuncForPC(val.Pointer()).Name() + "(func))"
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

	//return "(boof)"

	if z, err := json.Marshal(v); err == nil {
		//fmt.Println("kind is:", kind.String())
		return fmt.Sprintf("(go: unknown type: '%+v/%+v', as JSON: '%s', kind: %s)", v, val, z, kind.String())
	}

	return fmt.Sprintf("(go: unknown type: '%+v / %+v')", v, val)

}

func getPrettyString(v interface{}, size int) string {
	var cache = make(map[*interface{}]string)
	return getStringRepresentation(v, &v, size, false, 2, (&cache))
}
