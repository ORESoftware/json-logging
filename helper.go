package json_logging

import (
	"github.com/logrusorgru/aurora"
	"reflect"
	"strconv"
	"strings"
)

func addComma(i int, n int) string {
	if i < n-1 {
		return ", "
	}
	return ""
}

func handleMap(m reflect.Value, depth int) string {

	keys := m.MapKeys()

	n := len(keys)
	s := aurora.Bold(" map(").String()

	for i, k := range keys {
		val := m.MapIndex(k)
		s += getStringRepresentation(k.Interface(), depth) + " => " +
			getStringRepresentation(val.Interface(), depth) + addComma(i, n)
	}

	return s + aurora.Bold(")").String()
}

func handleSliceAndArray(val reflect.Value, depth int) string {

	s := aurora.Bold("[").String()

	n := val.Len()
	for i := 0; i < n; i++ {
		s += getStringRepresentation(val.Index(i).Interface(), depth) + addComma(i, n)
	}

	return s + aurora.Bold("]").String()
}

func handleStruct(val reflect.Value, depth int) string {

	n := val.NumField()
	t := val.Type()

	s := "{"

	for i := 0; i < n; i++ {

		k := t.Field(i).Name
		s += " " + k + ": "

		if strings.ToLower(k[:1]) == k[:1] {
			s += "(unknown val)"
			continue
		}

		v := val.Field(i).Interface()

		s += getStringRepresentation(v, depth+1) + addComma(i, n)
	}

	s += " }"

	return s
}

func getStringRepresentation(v interface{}, depth int) string {

	val := reflect.ValueOf(v)

	if val.Kind() == reflect.Map {
		return handleMap(val, depth)
	}

	if val.Kind() == reflect.Slice {
		return handleSliceAndArray(val, depth)
	}

	if val.Kind() == reflect.Array {
		return handleSliceAndArray(val, depth)
	}

	if val.Kind() == reflect.Func {
		return " (func)"
	}

	if val.Kind() == reflect.Struct {
		return handleStruct(val, depth)
	}

	if _, ok := v.(string); ok {
		return "'" + aurora.Green(v.(string)).String() + "'"
	}

	if _, ok := v.(bool); ok {
		return aurora.BrightBlue(strconv.FormatBool(v.(bool))).String()
	}

	if _, ok := v.(int64); ok {
		return aurora.Yellow(strconv.FormatInt(v.(int64), 1)).String()
	}

	if _, ok := v.(int); ok {
		return aurora.Yellow(strconv.Itoa(v.(int))).String()
	}

	return " (unknown type)"

}

func getPrettyString(v interface{}) string {
	return getStringRepresentation(v, 0)
}
