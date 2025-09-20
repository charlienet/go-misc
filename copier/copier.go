package copier

import (
	"fmt"
	"reflect"
	"strings"
	"time"
)

// 对象复制器，支持strucat->map[string]any, map[string]any->struct, struct->struct, map[string]any->map[string]any

// Support Map
// Support Struct
// Support Slice

// ToMap 是一个将任意类型转换为 map[string]any 的函数
// 它接收一个任意类型的参数 src，并返回一个 map[string]any 类型的结果和一个可能的错误
// 目前该函数的实现是返回 nil 值和 nil 错误，实际实现需要根据具体需求来完成
func ToMap(src any, opts ...option) (map[string]any, error) {
	var result map[string]any
	err := Copy(&result, src, opts...)
	return result, err
}

// Copy 函数用于将 src 的值复制到 dst 中
// dst 是目标值的任意类型，src 是源值的任意类型
// 该函数返回一个 error 类型的值，表示复制过程中可能出现的错误
func Copy(dst, src any, opts ...option) error {
	return copier(dst, src, getOpt(opts...))
}

func copier(dst, src any, opt *options) error {
	var (
		to   = indirect(reflect.ValueOf(dst))
		from = indirect(reflect.ValueOf(src))
	)

	// if from.IsNil() {
	// 	return ErrInvalidCopyFrom
	// }

	if !from.IsValid() {
		return ErrNotSupported
	}

	if !to.CanAddr() {
		return ErrInvalidCopyDestination
	}

	fromType, _ := indirectType(from.Type())
	toType, _ := indirectType(to.Type())

	if fromType.Kind() == reflect.Map && toType.Kind() == reflect.Map {
		if !fromType.Key().ConvertibleTo(toType.Key()) {
			return ErrInvalidCopyDestination
		}

		if to.IsNil() {
			to.Set(reflect.MakeMapWithSize(toType, from.Len()))
		}

		for _, key := range from.MapKeys() {
			value, _ := opt.TypeConvert(from.MapIndex(key))
			if name, ok := key.Interface().(string); ok {
				name = opt.NameConvert(name)
				key = reflect.ValueOf(name)
			}

			to.SetMapIndex(key, value)
		}

		return nil
	}

	if from.Kind() == reflect.Map && to.Kind() == reflect.Struct {
		for _, key := range from.MapKeys() {
			if key.Kind() == reflect.String {
				keyString := key.String()
				value := from.MapIndex(key)
				field := getFieldByName(to, keyString, opt)

				if field.CanSet() {
					if err := deepCopy(field, value, 0, opt); err != nil {
						return err
					}
				}
			}

			// if name, ok := key.Interface().(string); ok {
			// 	value := from.MapIndex(key)

			// 	tv := getFieldByName(to, name, opt)

			// 	if err := deepCopy(tv, value, 0, opt); err != nil {
			// 		return err
			// 	}
			// 	// if tv.IsValid() && tv.CanSet() {
			// 	// 	// if opt.valueConverter != nil {
			// 	// 	// 	v = opt.valueConverter(f.name, v)
			// 	// 	// }

			// 	// 	// fromCopy := reflect.New(tv.Type())
			// 	// 	// fromCopy.Set(reflect.ValueOf(v))
			// 	// 	println("::::::", tv.Type().String())
			// 	// 	ccc := value.Convert(tv.Type())
			// 	// 	cccc := ccc.Interface()
			// 	// 	_ = cccc

			// 	// 	to.Set(value.Convert(tv.Type()))
			// 	// }
			// }

		}
		return nil
	}

	return deepCopy(to, from, 0, opt)
}

func cpySliceArray(dst, src reflect.Value, fieldName string, depth int, opt *options) error {
	l := src.Len()
	if dst.Len() > 0 && dst.Len() < src.Len() {
		l = dst.Len()
	}

	if dst.Kind() == reflect.Slice && dst.Len() == 0 && src.Len() > 0 {
		dstType := dst.Type().Elem()
		newDst := reflect.MakeSlice(reflect.SliceOf(dstType), l, l)
		dst.Set(newDst)
	}

	for i := 0; i < l; i++ {
		if err := deepCopy(dst.Index(i), src.Index(i), depth, opt); err != nil {
			return err
		}
	}

	return nil
}

func deepCopy(dst, src reflect.Value, depth int, opt *options) error {
	switch dst.Kind() {
	case reflect.Map:
		// struct -> map
		// map->struct
		for i, n := 0, src.NumField(); i < n; i++ {
			sf := src.Type().Field(i)
			if sf.PkgPath != "" && !sf.Anonymous {
				continue
			}

			if sf.Anonymous && sf.Type.Kind() == reflect.Struct {
				if err := deepCopy(dst, src.Field(i), depth+1, opt); err != nil {
					return err
				}

				continue
			}

			tag := parseTag(sf.Tag.Get(opt.tagName))
			if tag.Contains(tagIgnore) {
				continue
			}

			name := toName(sf.Name, tag, opt)
			value, _ := opt.TypeConvert(src.Field(i))

			if opt.ignoreEmpty && value.IsZero() {
				continue
			}

			if value.Kind() == reflect.Struct {
				newDst := reflect.ValueOf(make(map[string]any, src.Field(i).NumField()))
				if err := deepCopy(newDst, value, depth+1, opt); err != nil {
					return err
				}

				dst.SetMapIndex(reflect.ValueOf(name), newDst)
			} else {
				dst.SetMapIndex(reflect.ValueOf(name), value)
			}
		}

		return nil
	case reflect.Struct:
		if src.Kind() == reflect.Struct {
			return cpyStruct(dst, src, depth, opt)
		} else if src.Kind() == reflect.Map {
			// map -> struct
			for _, key := range src.MapKeys() {
				if name, ok := key.Interface().(string); ok {
					v := src.MapIndex(key)

					name = opt.NameConvert(name)
					tv := getFieldByName(dst, name, opt)

					if err := deepCopy(tv, v, depth+1, opt); err != nil {
						return err
					}

					// if tv.IsValid() && tv.CanSet() {
					// 	fromCopy := reflect.New(tv.Type())
					// 	println("fromCopy:", fromCopy.Kind().String())
					// 	fromCopy.Set(v)
					// 	// dst.Set(fromCopy.Convert(dst.Type()))
					// }

					// value, _ := opt.TypeConvert(src.MapIndex(key))
					// name = opt.NameConvert(name)

					// if !dstValue.IsValid() {
					// 	continue
					// }

					// if opt.ignoreEmpty && value.IsZero() {
					// 	continue
					// }

					// println("dst value:", dstValue.Kind().String(), "name:", name, "value:", value.Kind().String())
					// if err := deepCopy(dstValue, value, depth+1, opt); err != nil {
					// 	return err
					// }

				} else {
					return ErrMapKeyNotMatch
				}

			}
			return nil
		} else {
			return ErrNotSupported
		}
	case reflect.Pointer:
		if dst.IsNil() {
			if !dst.CanSet() {
				return nil
			}
			p := reflect.New(dst.Type().Elem())
			dst.Set(p)
		}

		src = src.Elem()
		dst = dst.Elem()
		return deepCopy(dst, src, depth, opt)
	case reflect.Interface:
		if src.Kind() != dst.Kind() {
			return nil
		}

		src = src.Elem()
		newDst := reflect.New(src.Type().Elem())

		if err := deepCopy(newDst, src, depth, opt); err != nil {
			return err
		}

		dst.Set(newDst)
		return nil
	default:
		if src.Type().AssignableTo(dst.Type()) || src.Type().ConvertibleTo(dst.Type()) {
			dst.Set(src.Convert(dst.Type()))
		} else {
			switch dst.Kind() {
			case reflect.String:
				switch v := src.Interface().(type) {
				case string:
					dst.SetString(v)
				case []byte:
					dst.SetString(string(v))
				case int, int8, int16, int32, int64:
					dst.SetString(fmt.Sprintf("%d", v))
				case uint, uint8, uint16, uint32, uint64:
					dst.SetString(fmt.Sprintf("%d", v))
				default:
					dst.SetString(fmt.Sprintf("%v", v))
				}
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				switch v := src.Interface().(type) {
				case int:
					dst.SetInt(int64(v))
				case int8:
					dst.SetInt(int64(v))
				case int16:
					dst.SetInt(int64(v))
				case int32:
					dst.SetInt(int64(v))
				case int64:
					dst.SetInt(int64(v))
				}
			}
		}

		return nil
	}
}

func cpyStruct(dst, src reflect.Value, depth int, opt *options) error {
	// if to.Type().Kind() == reflect.Struct {
	// 	if to.CanSet() {
	// 		p := reflect.New(to.Type().Elem())
	// 		to.Set(p)
	// 		return cpyStruct(to.Elem(), from, fieldName, opt)
	// 	}
	// }

	// if to.CanSet() {
	// 	if _, ok := from.Interface().(time.Time); ok {
	// 		to.Set(from)
	// 		return nil
	// 	}
	// }

	if dst.CanSet() {
		if _, ok := src.Interface().(time.Time); ok {
			dst.Set(src)
			return nil
		}
	}

	if opt.ExceedMaxDepth(depth) {
		return nil
	}

	typ := src.Type()
	for i, n := 0, src.NumField(); i < n; i++ {
		sf := typ.Field(i)
		if sf.PkgPath != "" && !sf.Anonymous {
			continue
		}

		tag := parseTag(sf.Tag.Get(opt.tagName))
		if tag.Contains(tagIgnore) {
			continue
		}

		name := toName(sf.Name, tag, opt)

		dstValue := getFieldByName(dst, name, opt)
		if !dstValue.IsValid() {
			continue
		}

		sField := src.Field(i)

		if opt.ignoreEmpty && sField.IsZero() {
			continue
		}

		if err := deepCopy(dstValue, sField, depth+1, opt); err != nil {
			return err
		}
	}

	return nil
}

func getFieldByName(v reflect.Value, name string, opt *options) reflect.Value {
	if opt.caseSensitive {
		return v.FieldByName(name)
	} else {
		return v.FieldByNameFunc(func(s string) bool {
			eq := strings.EqualFold(s, name)
			return eq
		})
	}
}

func toName(name string, tag *tagOption, opt *options) string {
	toname := tag.ToName()
	if toname != "" {
		return toname
	}

	return opt.NameConvert(name)
}

func indirect(reflectValue reflect.Value) reflect.Value {
	for reflectValue.Kind() == reflect.Pointer {
		reflectValue = reflectValue.Elem()
	}

	return reflectValue
}

func indirectType(reflectType reflect.Type) (_ reflect.Type, isPtr bool) {
	for reflectType.Kind() == reflect.Ptr || reflectType.Kind() == reflect.Slice {
		reflectType = reflectType.Elem()
		isPtr = true
	}

	return reflectType, isPtr
}
