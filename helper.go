package json_logging

import (
	"github.com/logrusorgru/aurora"
	"reflect"
	"strconv"
	"strings"
)

func addComma(i int, n int) string {
	if i < n-1 {
		return ","
	}
	return ""
}

func handleMap(m reflect.Value, depth int) string {

	keys := m.MapKeys()

	n := len(keys)
	s := "map ("

	for i, k := range keys {
		val := m.MapIndex(k)
		s += getStringRepresentation(k, depth) + " => " + getStringRepresentation(val, depth) + addComma(i, n)
	}

	return s + ")"
}

func handleSliceAndArray(m []interface{}, depth int) string {
	return ""
}

func handleStruct(val reflect.Value, depth int) string {

	n := val.NumField()
	t := val.Type()

	s := "{"

	for i := 0; i < n; i++ {

		k := t.Field(i).Name
		s += " " + k + ": "

		if strings.ToLower(k[1:]) == k[1:] {
			s += "(unknown val)"
			continue
		}

		v := val.Field(i).Interface()

		s += getStringRepresentation(v, depth+1)
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
		return handleSliceAndArray(v.([]interface{}), depth)
	}

	if val.Kind() == reflect.Array {
		return handleSliceAndArray(v.([]interface{}), depth)
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
		return aurora.Yellow(strconv.FormatBool(v.(bool))).String()
	}

	if _, ok := v.(int64); ok {
		return aurora.Blue(strconv.FormatInt(v.(int64), 1)).String()
	}

	if _, ok := v.(int); ok {
		return aurora.Blue(strconv.Itoa(v.(int))).String()
	}

	return " (unknown type)"

}

func getPrettyString(v interface{}) string {
	return getStringRepresentation(v, 0)
}

