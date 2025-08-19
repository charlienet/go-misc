package structs

import (
	"errors"
	"reflect"
)

var (
	ErrInvalidCopyDestination      = errors.New("copy destination is invalid")
	ErrNonPointerArgument          = errors.New("non-pointer argument")
	ErrNilArguments                = errors.New("nil arguments")
	ErrNotSupported                = errors.New("only structs, maps, and slices are supported")
	ErrExpectedMapAsDestination    = errors.New("dst was expected to be a map")
	ErrExpectedStructAsDestination = errors.New("dst was expected to be a struct")
)

func indirect(v reflect.Value) reflect.Value {
	if v.Kind() == reflect.Pointer {
		v = v.Elem()
	}

	return v
}

func indirectType(t reflect.Type) reflect.Type {
	if t.Kind() == reflect.Pointer || t.Kind() == reflect.Slice {
		return t.Elem()
	}

	return t
}
