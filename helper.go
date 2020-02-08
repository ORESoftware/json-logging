package json_logging

import (
	"fmt"
	"github.com/logrusorgru/aurora"
	"reflect"
	"strconv"
)

func addComma(i int, n int) string {
	if i < n-1 {
		return ","
	}
	return ""
}

func handleMap(m reflect.Value, depth int) string {

	keys := m.MapKeys()

	s := "map ("

	for i, k := range keys {
		fmt.Println("i:", i, "k:", k)
		val := m.MapIndex(k)
		fmt.Println("value:", val)
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
		v := val.Field(i).Interface()

		if _, ok := v.(string); ok {
			s += "'" + aurora.Green(v.(string)).String() + "'" + addComma(i, n)
			continue
		}

		if _, ok := v.(bool); ok {
			s += aurora.Yellow(strconv.FormatBool(v.(bool))).String() + addComma(i, n)
			continue
		}

		if _, ok := v.(int64); ok {
			s += aurora.Blue(strconv.FormatInt(v.(int64), 1)).String() + addComma(i, n)
			continue
		}

		if _, ok := v.(int); ok {
			s += aurora.Blue(strconv.Itoa(v.(int))).String() + addComma(i, n)
			continue
		}

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
		return " (func)"
	}

	return "(unknonwn type)"

}

func getPrettyString(v interface{}) string {
	return getStringRepresentation(v, 0)
}

func getPrettyString2(v interface{}) string {

	s := ""

	recurse1 := func(z interface{}) {

		val := reflect.ValueOf(z)
		t := val.Type()

		for i := 0; i < val.NumField(); i++ {

			k := t.Field(i).Name
			s += k
			v := val.Field(i).Interface()

			if _, ok := v.(string); ok {
				s += " '" + v.(string) + "',"
				continue
			}

			if _, ok := v.(bool); ok {
				s += " '" + strconv.FormatBool(v.(bool)) + "',"
				continue
			}

			if _, ok := v.(int); ok {
				s += " '" + strconv.FormatInt(v.(int64), 1) + "',"
				continue
			}

			//recurse1(nil)

		}

	}

	recurse1(v)

	return s
}
