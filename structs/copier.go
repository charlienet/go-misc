package structs

import (
	"maps"
	"reflect"
)

// struct -> map
// struct -> struct
// map	 -> struct
// map	 -> map
func Copy(dst, src any, opts ...option) error {
	opt := acquireOptions(opts)
	return copier(dst, src, opt)
}

func copier(dst, src any, opt options) error {
	if dst != nil && reflect.ValueOf(dst).Kind() != reflect.Ptr {
		return ErrNonPointerArgument
	}

	var (
		from = indirect(reflect.ValueOf(src))
		to   = indirect(reflect.ValueOf(dst))
		err  error
	)

	if !to.CanAddr() {
		return ErrInvalidCopyDestination
	}

	if !from.IsValid() {
		return ErrNotSupported
	}

	if from.Kind() == reflect.Map && to.Kind() == reflect.Map {
		fromMap := from.Interface().(map[string]any)
		toMap := to.Interface().(map[string]any)

		if opt.valueConverter != nil {
			for k, v := range fromMap {
				toMap[k] = opt.valueConverter(k, v)
			}
		}

		maps.Copy(toMap, fromMap)
	}

	var get getter
	switch from.Kind() {
	case reflect.Struct:
		get = parseStructGetter(from, opt)
	case reflect.Map:
		get = parseMapGetter(from.Interface().(map[string]any), opt)
	default:
		return ErrNotSupported
	}

	println(get)
	switch to.Kind() {
	case reflect.Struct:
		for v, f := range get.Iter() {
			tv := to.FieldByName(f.toName)
			if tv.IsValid() && tv.CanSet() {
				if opt.valueConverter != nil {
					v = opt.valueConverter(f.name, v)
				}
				tv.Set(reflect.ValueOf(v))
			}
		}

	case reflect.Map:
		dstMap := to.Interface().(map[string]any)
		for v, f := range get.Iter() {
			if opt.valueConverter != nil {
				v = opt.valueConverter(f.name, v)
			}

			dstMap[f.toName] = v
		}
	}

	return err
}
