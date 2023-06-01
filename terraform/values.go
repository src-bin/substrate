package terraform

import (
	"fmt"
	"strings"
)

type Value interface {
	Empty() bool
	Raw() string
	Value() string
}

type ValueSlice []Value

func QSlice(ss []string) ValueSlice {
	vs := make(ValueSlice, len(ss))
	for i, s := range ss {
		vs[i] = Q(s)
	}
	return vs
}

func USlice(ss []string) ValueSlice {
	vs := make(ValueSlice, len(ss))
	for i, s := range ss {
		vs[i] = U(s)
	}
	return vs
}

func (vs ValueSlice) Empty() bool {
	return len(vs) == 0
}

func (vs ValueSlice) Raw() string {
	switch len(vs) {
	case 0:
		return "[]"
	case 1:
		return fmt.Sprintf("[%s]", vs[0].Raw())
	default:
		s := "[\n"
		for _, v := range vs {
			s += fmt.Sprintf("\t\t%s,\n", v.Raw())
		}
		s += "\t]"
		return s
	}
}

func (vs ValueSlice) Value() string {
	switch len(vs) {
	case 0:
		return "[]"
	case 1:
		return fmt.Sprintf("[%s]", vs[0].Value())
	default:
		s := "[\n"
		for _, v := range vs {
			s += fmt.Sprintf("\t\t%s,\n", v.Value())
		}
		s += "\t]"
		return s
	}
}

type quotedString string

func Q(args ...interface{}) Value {
	return quotedString(fmt.Sprint(rawValues(args...)...))
}

func Qf(format string, args ...interface{}) Value {
	return quotedString(fmt.Sprintf(format, rawValues(args...)...))
}

func (q quotedString) Empty() bool {
	return string(q) == ""
}

func (q quotedString) Raw() string {
	return string(q)
}

func (q quotedString) Value() string {
	if strings.Contains(string(q), "\n") {
		return fmt.Sprintf("<<EOF\n%s\nEOF", string(q)) // TODO handle q containing "EOF"
	}
	return fmt.Sprintf("%q", string(q))
}

type unquotedString string

func Bool(v bool) Value { return unquotedString(fmt.Sprintf("%t", v)) }
func False() Value      { return Bool(false) }
func True() Value       { return Bool(true) }

func U(args ...interface{}) Value {
	return unquotedString(fmt.Sprint(rawValues(args...)...))
}

func Uf(format string, args ...interface{}) Value {
	return unquotedString(fmt.Sprintf(format, rawValues(args...)...))
}

func (u unquotedString) Empty() bool {
	return string(u) == ""
}

func (u unquotedString) Raw() string {
	return string(u)
}

func (u unquotedString) Value() string {
	return string(u)
}

func rawValues(args ...interface{}) []interface{} {
	raws := make([]interface{}, len(args))
	for i, arg := range args {
		if v, ok := arg.(Value); ok {
			raws[i] = v.Raw()
		} else {
			raws[i] = arg
		}
	}
	return raws
}
