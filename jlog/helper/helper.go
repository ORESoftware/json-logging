package hlpr

import (
  "encoding/json"
  "fmt"
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
  . "github.com/logrusorgru/aurora/v4"
)

func addComma(i int, n int) string {
  if i < n-1 {
    return ", "
  }
  return ""
}

func handleMap(x interface{}, size int, brk bool, depth int, cache *map[*interface{}]string) string {

  var rv = reflect.ValueOf(x)

  if !rv.IsValid() {
    return fmt.Sprintf("<nil-00009> (Type: %T)", x)
  }

  keys := rv.MapKeys()
  n := len(keys)

  if n < 1 {
    return fmt.Sprintf(Black("(empty map - %T)").String(), x)
  }

  values := []string{}

  for i, k := range keys {

    val := rv.MapIndex(k)
    m := val.Interface()
    var ptr uintptr

    z := strings.Builder{}

    var count = 0
    for count++; i < 9; {

      if !(val.Kind() == reflect.Ptr || val.Kind() == reflect.Interface) {
        break
      }

      if val.IsNil() {
        z.WriteString(fmt.Sprintf("<nil (%T)>", m))
        break
      }

      if val.Kind() == reflect.Chan {
        break
      }

      val = val.Elem()

      if !val.IsValid() {
        z.WriteString(fmt.Sprintf("'%s'", Cyan(fmt.Sprintf("%s", k.Interface())).String()))
        z.WriteString(Bold(" —> ").String())
        z.WriteString(fmt.Sprintf("%v / %v", m, ptr))
        z.WriteString(addComma(i, n))
        break
      }

      m = val.Interface()
    }

    if !val.IsValid() {
      z.WriteString(fmt.Sprintf("'%s'", Cyan(fmt.Sprintf("%s", k.Interface())).String()))
      z.WriteString(Bold(" —> ").String())
      z.WriteString(fmt.Sprintf("%v / %v", m, ptr))
      z.WriteString(addComma(i, n))
      {
        str := z.String()
        size = size + len(str)
        values = append(values, str)
      }
      continue
    }

    if m == nil {
      z.WriteString(fmt.Sprintf("'%s'", Cyan(fmt.Sprintf("%s", k.Interface())).String()))
      z.WriteString(Bold(" —> ").String())
      z.WriteString(fmt.Sprintf("%v (%T)", m, m))
      z.WriteString(addComma(i, n))
      {
        str := z.String()
        size = size + len(str)
        values = append(values, str)
      }
      continue
    }

    if val.CanAddr() {
      // It's safe to use UnsafeAddr and NewAt since rf is addressable
      val = reflect.NewAt(val.Type(), unsafe.Pointer(val.UnsafeAddr())).Elem()
      m = val.Interface()
      ptr = val.Addr().Pointer()
    } else {
      // Handle the case where rf is not addressable
      // You might need to create a copy or take a different approach
      // Example: Creating a copy

      if val.Kind() != reflect.Chan {
        myCopy := reflect.New(val.Type()).Elem()
        myCopy.Set(val)
        val = myCopy
        m = myCopy.Interface()
        ptr = myCopy.Addr().Pointer()
      }
    }

    if !val.IsValid() {
      z.WriteString(fmt.Sprintf("'%s'", Cyan(fmt.Sprintf("%s", k.Interface())).String()))
      z.WriteString(Bold(" —> ").String())
      z.WriteString(fmt.Sprintf("%v / %v", m, ptr))
      z.WriteString(addComma(i, n))
      {
        str := z.String()
        size = size + len(str)
        values = append(values, str)
      }
      continue
    }

    if val.CanInterface() {
      z.WriteString(fmt.Sprintf("'%s'", Cyan(fmt.Sprintf("%s", k.Interface())).String()))
      z.WriteString(Bold(" —> ").String())
      z.WriteString(fmt.Sprintf("%T %+v %+v %v", ptr, val, m, val.String()))
      //z.WriteString(fmt.Sprintf(" 222 %v -- %v -- %v", rv, val.String(), val.Interface()))
      z.WriteString(addComma(i, n))
    } else {
      z.WriteString(fmt.Sprintf("'%s'", Cyan(fmt.Sprintf("%s", k.Interface())).String()))
      z.WriteString(Bold(" —> ").String())
      z.WriteString(fmt.Sprintf("%v / (%v)", m, m))
      z.WriteString(addComma(i, n))
    }

    // TODO: get colorzied version of the values in the map

    str := z.String()
    size = size + len(str)
    values = append(values, str)
  }

  if size > 100-depth {
    brk = true
    //size = 0
  }

  //keyType := reflect.ValueOf(keys).Type().Elem().String()
  //valType := rv.Type().Elem().String()
  //z := fmt.Sprintf("map<%s,%s>(", keyType, valType)
  //log.Println(z)

  var b strings.Builder

  b.WriteString(Bold(fmt.Sprintf("%T (", x)).String() + createNewline(brk, true))

  for i := 0; i < n; i++ {
    b.WriteString(createSpaces(depth, brk) + values[i] + createNewline(brk, true))
  }

  b.WriteString(createSpaces(depth-1, brk))
  b.WriteString(Bold(")").String() + createNewline(brk, false))
  return b.String()
}

func handleSliceAndArray(v interface{}, len int, brk bool, depth int, cache *map[*interface{}]string) string {

  rv := reflect.ValueOf(v)

  if !rv.IsValid() {
    return "<nil - 6667>"
  }

  n := rv.Len()
  t := rv.Type()

  if n < 1 {
    return Black("[").String() + "" + Black(fmt.Sprintf("] (empty %v)", t)).String()
  }

  if rv.Kind() == reflect.Chan {
    return Black("[").String() + "" + Black(fmt.Sprintf("] (empty %v)", t)).String()
  }

  elementType := t.Elem()

  if elementType.Kind() == reflect.Uint8 {
    return Bold("[]byte as str:").String() + fmt.Sprintf(" '%s'", v)
  }

  var b strings.Builder
  b.WriteString(createSpaces(depth, brk) + Bold("[").String())

  for i := 0; i < n; i++ {
    b.WriteString(createSpaces(depth, brk))
    x := rv.Index(i)
    if !x.IsValid() {
      b.WriteString("<nil>")
      continue
    }
    inf := x.Interface()

    //ptr := inf
    //if x.CanAddr() {
    //  ptr = x.Addr().Interface()
    //}
    //if x.IsValid() {
    //
    //}
    b.WriteString(getStringRepresentation(inf, len, brk, depth, cache))
    b.WriteString(addComma(i, n))
  }

  b.WriteString(createNewline(brk, true) + Bold("]").String())
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

func handleStruct(obj interface{}, size int, brk bool, depth int, cache *map[*interface{}]string) string {

  rv := reflect.ValueOf(obj)

  if !rv.IsValid() {
    return "<nil - 6669>"
  }

  n := rv.NumField()
  t := rv.Type()

  if n < 1 {
    return fmt.Sprintf(" (%s / %s) { }", t.Name, t.String())
  }

  //log.Println("ln:", ln)

  keys := []string{}
  values := []string{}

  for i := 0; i < n; i++ {

    var fv = t.Field(i)
    var k = fv.Name

    keys = append(keys, k+":")
    size = size + len(keys)

    rs := reflect.New(t).Elem()
    rs.Set(rv)
    rf := rs.Field(i)

    var v interface{}

    // TODO: we don't need to store in an array;
    // we can just flip a bool if the total size of string is more than (size + line) > 50
    // we could reset the bool once we start a new line
    //
    if rf.CanAddr() {
      // It's safe to use UnsafeAddr and NewAt since rf is addressable
      rf = reflect.NewAt(rf.Type(), unsafe.Pointer(rf.UnsafeAddr())).Elem()
      v = rf.Interface()
      //ptr = rf.Addr().Interface()
    } else {
      // Handle the case where rf is not addressable
      // You might need to create a copy or take a different approach
      // Example: Creating a copy
      myCopy := reflect.New(rf.Type()).Elem()
      myCopy.Set(rf)
      v = myCopy.Interface()
      //ptr = myCopy.Addr().Interface()
    }

    z := getStringRepresentation(v, size, brk, depth+1, cache)
    values = append(values, z)
    size = size + len(z)
  }

  if size > 100-depth {
    brk = true
  }

  var b strings.Builder
  b.WriteString(fmt.Sprintf("(%s)", t.String()))
  b.WriteString(" {" + createNewline(brk, n > 0))

  for i := 0; i < n; i++ {
    var k = Bold(Blue(keys[i])).String()
    b.WriteString(createSpaces(depth, brk) + k)
    b.WriteString(" ")
    b.WriteString(values[i])
    b.WriteString(addComma(i, n))
    b.WriteString(createNewline(brk, n > 0))
  }

  b.WriteString(createSpaces(depth-1, brk) + "}")
  b.WriteString(createNewline(brk, false))
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
    switch vv.Kind() {
    case reflect.Pointer, reflect.UnsafePointer:
      nm = vv.Elem().Name()
    case reflect.Func:
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
  return Black(Bold("<nil>")).String()
}

func getHighlightedString(val string) string {
  if len(val) < 1 {
    return Bold("''").String()
  }
  var trimmed = strings.TrimSpace(val)
  if len(trimmed) == len(val) {
    return Green(val).String()
  }
  return Bold("'").String() + Green(val).String() + Bold("'").String()
}

func getStringRepresentation(v interface{}, size int, brk bool, depth int, cache *map[*interface{}]string) (s string) {

  defer func() {
    if r := recover(); r != nil {
      fmt.Println("\n")
      fmt.Println(fmt.Sprintf("%v", r))
      debug.PrintStack()
      s = fmt.Sprintf("%v - (go: unknown type 2: '%v')", r, v)
    }
  }()

  pt := &v

  if &v != pt {
    panic("must be pointer to same object")
  }

  if v == nil {
    return "<nil (nil)>"
  }

  var rv = reflect.ValueOf(v)

  if !rv.IsValid() {
    return "<nil (invalid)>"
  }

  var kind = rv.Kind()

  for {

    if !(kind == reflect.Ptr || kind == reflect.Interface) {
      break
    }

    if rv.IsNil() {
      return "<nil>"
    }

    if kind == reflect.Chan {
      return fmt.Sprintf("(%v (%T))", v, v)
    }

    rv = rv.Elem()

    if !rv.IsValid() { // Check if the dereferenced value is valid
      return "<nil 114>"
    }

    v = rv.Interface()

    if v == nil {
      return "<nil 115>"
    }

    kind = rv.Kind()

  }

  if v == nil {
    return "<nil>"
  }

  if rv.Kind() == reflect.Ptr || rv.Kind() == reflect.Ptr {
    return fmt.Sprintf("pointer/interface -> (%v)", rv.Type().String())
  }

  if z, ok := v.(string); ok {
    return getHighlightedString(z)
  }

  if kind == reflect.Chan {
    return fmt.Sprintf("(chan (%s) %v)", rv.Type().Elem().String(), rv.Elem().Interface())
  }

  if kind == reflect.Map {
    return handleMap(v, size, brk, depth, cache)
  }

  if kind == reflect.Slice {
    if m, ok := v.(Stringer); ok {
      return fmt.Sprintf("%T - (As string: %s)", m, m)
    } else {
      return handleSliceAndArray(v, size, brk, depth, cache)
    }
  }

  if kind == reflect.Array {
    if m, ok := v.(Stringer); ok {
      return fmt.Sprintf("(%T (As string: '%s'))", m, m)
    } else {
      return handleSliceAndArray(v, size, brk, depth, cache)
    }
  }

  if kind == reflect.Func {
    return GetFuncSignature(v)
  }

  if kind == reflect.Struct {
    return handleStruct(v, size, brk, depth, cache)
  }

  if z, ok := v.(Stringer); ok && z != nil {
    return z.String()
  }

  if z, ok := v.(*Stringer); ok && z != nil {
    return (*z).String()
  }

  if kind == reflect.Bool {
    return BrightBlue(strconv.FormatBool(v.(bool))).String()
  }

  if _, ok := v.(bool); ok {
    return BrightBlue(strconv.FormatBool(v.(bool))).String()
  }

  if kind == reflect.Int {
    return Yellow(v).String()
  }

  if _, ok := v.(int); ok {
    return Yellow(strconv.Itoa(v.(int))).String()
  }

  if kind == reflect.Int8 {
    return Yellow(v).String()
  }

  if _, ok := v.(int8); ok {
    return Yellow(v.(int8)).String()
  }

  if kind == reflect.Int16 {
    return Yellow(v).String()
  }

  if _, ok := v.(int16); ok {
    return Yellow(v.(int16)).String()
  }

  if kind == reflect.Int32 {
    return Yellow(v).String()
  }

  if _, ok := v.(int32); ok {
    return Yellow(v.(int32)).String()
  }

  if kind == reflect.Int64 {
    return Yellow(v).String()
  }

  if _, ok := v.(int64); ok {
    return Yellow(strconv.FormatInt(v.(int64), 1)).String()
  }

  if kind == reflect.Uint {
    return Yellow(v).String()
  }

  if _, ok := v.(uint); ok {
    return Yellow(v.(uint)).String()
  }

  if kind == reflect.Uint8 {
    return Yellow(v).String()
  }

  if _, ok := v.(uint8); ok {
    return Yellow(v).String()
  }

  if kind == reflect.Uint16 {
    return Yellow(v).String()
  }

  if _, ok := v.(uint16); ok {
    return Yellow(v.(uint16)).String()
  }

  if kind == reflect.Uint32 {
    return Yellow(v).String()
  }

  if _, ok := v.(uint32); ok {
    return Yellow(v.(uint32)).String()
  }

  if kind == reflect.Uint64 {
    return Yellow(v).String()
  }

  if _, ok := v.(uint64); ok {
    return Yellow(v.(uint64)).String()
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
    // if false && originalV != v {
    //   return fmt.Sprintf("(go: unknown type 1a: '%+v/%+v/%v/%v', as JSON: '%s', kind: %s)", v, originalV, originalVal, z, kind.String())
    // } else {
    //   return fmt.Sprintf("(go: unknown type 2a: '%+v/%+v', as JSON: '%s', kind: %s)", v, &v, z, kind.String())
    // }

    return fmt.Sprintf("(go: unknown type 2a: '%+v/%+v', as JSON: '%s', kind: %s)", v, &v, z, kind.String())

  }

  // if false && originalV != v {
  //   return fmt.Sprintf("(go: unknown type 3a: '%+v / %+v / %v / %v')", v, &v, originalV, originalVal)
  // } else {
  //   return fmt.Sprintf("(go: unknown type 4a: '%+v / %+v')", v, &v)
  // }

  return fmt.Sprintf("(go: unknown type 4a: '%+v / %+v')", v, &v)

}

func DoCopyAndDerefStruct(s interface{}) interface{} {

  rv := reflect.ValueOf(s)

  if !rv.IsValid() {
    return nil
  }

  var kind = rv.Kind()

  if kind == reflect.Chan {
    return fmt.Sprintf("(%v (%T))", s, s)
  }

  val := rv.Elem()
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

  if !val.IsValid() {
    return nil
  }

  if val.Kind() == reflect.Chan {
    return fmt.Sprintf("(%v (%T))", s, s)
  }

  // Dereference pointer if s is a pointer
  if val.Kind() == reflect.Ptr || val.Kind() == reflect.Interface {
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
      rv := val.Index(i)
      if rv.IsValid() && rv.CanInterface() {
        slice[i] = CopyAndDereference(val.Index(i).Interface())
      } else {
        slice[i] = fmt.Sprintf("%v (%s)", rv, rv.String())
      }

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
  return getStringRepresentation(v, size, false, 2, &cache)
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

        if elem.IsValid() && elem.CanInterface() {

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

func hasField(obj interface{}, fieldName string) bool {
  // Get the type and value of the struct
  objType := reflect.TypeOf(obj)

  // Check if the struct has the specified field
  _, found := objType.FieldByName(fieldName)
  return found
}

func cleanStruct(v interface{}, cache Cache) interface{} {

  rv := reflect.ValueOf(v)

  if !rv.IsValid() {
    return nil
  }

  if hasField(v, "JLogMarker") {
    return v
  }

  // we turn struct into a map so we can display
  var ret = map[string]interface{}{}

  //if rv.Elem().Kind() != reflect.Struct {
  //  z := rv.Elem().Addr()
  //  if x, ok := (z.Interface()).(interface{}); ok {
  //    v = &x
  //  }
  //}
  //rv := rv.Elem() // Dereference the pointer to get the struct

  ret["JLogMarker"] = true

  for i := 0; i < rv.NumField(); i++ {

    fv := rv.Field(i)
    ft := rv.Type()      // Get the reflect.Type of the struct
    field := ft.Field(i) // Get the reflect.StructField

    if !fv.IsValid() {
      continue
    }

    if !fv.CanInterface() {
      ret[field.Name] = fmt.Sprintf("(%v) (%v)", ft.String(), fv.String())
      continue
    }

    ret[field.Name] = CleanUp(fv.Interface(), cache)
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

func cleanMap(v interface{}, cache Cache) (z interface{}) {

  // TODO: if keys to map are not strings, then create a slice/array of Key/Value Structs
  //type KeyValuePair struct {
  //	Key   int    `json:"key"`
  //	Value string `json:"value"`
  //}

  m := reflect.ValueOf(v)

  var ret = make(map[interface{}]interface{})
  keys := m.MapKeys()

  for _, k := range keys {
    val := m.MapIndex(k)
    inf := val.Interface()
    ret[k] = CleanUp(&inf, cache)
  }

  return ret
}

func cleanList(v interface{}, cache Cache) (z interface{}) {

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

func CleanUp(v interface{}, cache Cache) (z interface{}) {

  if v == nil {
    return fmt.Sprintf("<nil> (%T)", v)
  }

  rv := reflect.ValueOf(v)

  if !rv.IsValid() {
    return nil
  }

  if rv.Kind() == reflect.Pointer || rv.Kind() == reflect.Interface {

    if rv.IsNil() {
      return nil
    }
    if !rv.IsValid() {
      return nil
    }

    rv = rv.Elem()

    if !rv.IsValid() {
      return nil
    }

    return CleanUp(rv.Interface(), cache)
  }

  //if rv.Kind() == reflect.Ptr || kind == reflect.Interface {
  //  rv = rv.Elem()
  //  kind = rv.Kind()
  //
  //  if kind == reflect.Ptr || kind == reflect.Interface {
  //    // This block will not run for structInstance
  //    if rv.Elem().CanAddr() {
  //      ptrVal := rv.Elem().Addr()
  //      // Convert to interface and then to the specific pointer type (*int in this case)
  //      ptr, ok := ptrVal.Interface().(interface{})
  //      if ok {
  //        v = &ptr
  //      } else {
  //        return "(pointer thing 5)"
  //      }
  //    } else {
  //      return "(pointer thing 6)"
  //    }
  //  }
  //
  //}

  if v == nil {
    return fmt.Sprintf("<nil> (%T)", v)
  }

  //if rv.Kind() == reflect.Pointer || rv.Kind() == reflect.Interface {
  //  // Use Elem() to get the underlying type
  //  rv = rv.Elem()
  //  v = rv.Interface()
  //
  //  // Check again if the concrete value is also an interface
  //  if rv.Kind() == reflect.Interface {
  //    // Get type information about the interface
  //    typ := rv.Type()
  //
  //    // You can also check if the interface is nil
  //    if rv.IsNil() {
  //      return fmt.Sprintf("Nested interface type: %v, but it is nil", typ)
  //    } else {
  //      // Get more information about the non-nil interface
  //      concreteVal := rv.Elem()
  //      concreteType := concreteVal.Type()
  //      return fmt.Sprintf("Nested interface type: %v, contains value of type: %v", typ, concreteType)
  //    }
  //  }
  //}

  if rv.Kind() == reflect.Bool {
    return v
  }

  if isNonComplexNum(rv.Kind()) {
    return v
  }

  if rv.Kind() == reflect.Func {
    return "(go:func())"
  }

  if rv.Kind() == reflect.Complex64 {
    return fmt.Sprintf("(go:complex64:%+v)", v) // v.(complex64)
  }

  if rv.Kind() == reflect.Complex128 {
    return "(go:complex128)" //v.(complex128)
  }

  if rv.Kind() == reflect.Chan {
    return fmt.Sprintf("(go:chan:%+v)", v)
  }

  if rv.Kind() == reflect.UnsafePointer {
    return "(go:UnsafePointer)"
  }

  if rv.Kind() == reflect.Struct {
    return cleanStruct(v, cache)
  }

  if rv.Kind() == reflect.Map {
    // TODO: if keys to map are not strings, then create a slice/array of Key/Value Structs
    //type KeyValuePair struct {
    //	Key   int    `json:"key"`
    //	Value string `json:"value"`
    //}
    return cleanMap(v, cache)
  }

  if rv.Kind() == reflect.Slice {
    return cleanList(v, cache)
  }

  if rv.Kind() == reflect.Array {
    return cleanList(v, cache)
  }

  if z, ok := (v).(Stringer); ok {
    return z.String()
  }

  if z, ok := (v).(ToString); ok {
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

    // if strings.Contains(line, "oresoftware/json-logging") {
    //   continue
    // }

    if !strings.Contains(line, ".go:") {
      continue
    }

    var nl = fmt.Sprintf("%s", strings.TrimSpace(line))
    if len(nl) > 0 {
      filteredLines = append(filteredLines, nl)
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
