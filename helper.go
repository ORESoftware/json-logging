package json_logging

import (
	"github.com/logrusorgru/aurora"
	"reflect"
	"strconv"
)

func addComma(i int, n int) string {
	if i < n - 1 {
		return ","
	}
	return ""
}

func recurse(s string, v interface{}) string {

	val := reflect.ValueOf(v)
	t := val.Type()
	n := val.NumField()

	s += "{"

	for i := 0; i < n; i++ {

		k := t.Field(i).Name
		s += " " + k + ": "
		v := val.Field(i).Interface()

		if _, ok := v.(string); ok {
			s += "'" + aurora.Green(  v.(string)).String() + "'" + addComma(i, n)
			continue
		}

		if _, ok := v.(bool); ok {
			s += aurora.Yellow(strconv.FormatBool(v.(bool))).String() + addComma(i, n)
			continue
		}

		if _, ok := v.(int); ok {
			s += aurora.Blue(strconv.FormatInt(v.(int64), 1)).String() + addComma(i, n)
			continue
		}

		s += recurse(s, v)
	}

	s += " }"

	return s
}

func getPrettyString(v interface{}) string {
	return recurse("", v)
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
