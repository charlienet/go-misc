// Package copier 提供 struct / map 之间的深拷贝能力，支持类型转换、字段映射、标签控制等。
//
// # 拷贝语义
//
// Copy 对以下容器类型执行深拷贝（修改 dst 的嵌套容器不会污染 src）：
//   - Slice/Array：逐元素递归拷贝
//   - Map（含 map[string]any 嵌套）：递归创建独立副本
//   - Pointer：创建新对象并递归拷贝
//
// 以下场景遵循 Go 值拷贝语义（浅拷贝）：
//   - struct 值类型字段：struct 内部指针字段仍与 src 共享引用。
//     若需深拷贝 struct 内部指针，应将 map value / struct field 声明为指针类型
//     （如 *SomeStruct 而非 SomeStruct）。
//   - map→map 的顶层：map 本身是新对象，但 key 不做深拷贝（Go map key 必须是可比较标量）。
//
// # 类型转换
//
// Copy 在字段级别支持自动类型转换（string↔int/uint/float/bool 等），由 reflect
// 内置规则和 strconv 协同处理。如需自定义类型转换（如 string→time.Time），
// 请使用 WithConverters 注册 TypeConverter。map 值的转换同样通过 TypeConvert 完成，
// 而非 valueConverter（后者专用于 struct 字段名级别转换）。
//
// # 循环引用
//
// Copy 在递归拷贝过程中会检测指针循环引用并安全终止，不会导致栈溢出。
// 使用 WithMaxDepth 可限制最大递归深度，超限返回 ErrMaxDepthExceeded。
//
// # 标签系统
//
// 结构体字段可通过 copier 标签控制拷贝行为：
//   copier:"-"          忽略该字段
//   copier:"must"       仅当启用 WithMust 时拷贝该字段
//   copier:"toname=XXX" 拷贝到目标字段 XXX（可与 WithMust 组合使用）
//
// # 并发安全
//
// Copy 每次调用独立创建内部状态，无全局可变缓存，多 goroutine 并发调用安全。
//
// # 已知限制
//
//   - map→map 拷贝要求源与目标的 key 类型兼容（如 string→string），
//     不兼容（map[int]→map[string]）返回 ErrInvalidCopyDestination。
//   - map→map 的嵌套容器通过深拷贝隔离，但 map→map 路径不调用 valueConverter；
//     如需 map 值级转换请使用 TypeConvert。
//   - 无输入 map 大小的硬限制；极大 map 的拷贝由调用方自行评估内存风险。
//     WithMaxDepth 可限制递归深度。
package copier

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"
)

// ToMap 将任意类型 src 转换为 map[string]any。
// 等价于 Copy(&result, src, opts...)，语义与 Copy 一致。
// 嵌套容器被深拷贝，修改返回的 map 不会影响 src。
func ToMap(src any, opts ...option) (map[string]any, error) {
	var result map[string]any
	err := Copy(&result, src, opts...)
	return result, err
}

// Copy 将 src 的值深拷贝到 dst，支持 struct↔map↔slice 之间的互转。
//
// dst 必须是非 nil 的可寻址指针（如 &target），src 可以为值或指针。
//
// 拷贝遵循以下规则：
//   - 同名字段/同 key 默认按名称匹配（大小写不敏感，可通过 WithCaseSensitive 开启敏感）
//   - 可转换的基础类型自动转换（int↔string 等）
//   - 嵌套容器（slice/map/pointer）递归深拷贝
//   - 未匹配的字段/未导出的字段/不可设置的字段静默跳过
//
// 选项函数（With*）可控制：最大深度、忽略空值、大小写敏感、字段跳过、
// 字段/值转换、标签名、must 模式等。详见包文档。
//
// 返回 error 的场景：
//   - dst 不可寻址（非指针或 nil）
//   - src 无效（nil 值）
//   - map key 类型不兼容
//   - 超出最大递归深度（WithMaxDepth）
//   - 接口类型不满足时（窄接口赋值失败）
func Copy(dst, src any, opts ...option) error {
	return copier(dst, src, getOpt(opts...))
}

func copier(dst, src any, opt *options) error {
	var (
		to   = indirect(reflect.ValueOf(dst))
		from = indirect(reflect.ValueOf(src))
	)

	if !from.IsValid() {
		return ErrInvalidCopyFrom
	}

	if !to.CanAddr() {
		return ErrInvalidCopyDestination
	}

	// map -> map / map -> struct / struct -> struct / struct -> map / slice 等统一交给 deepCopy
	return deepCopy(to, from, 0, opt)
}

// cpySliceArray 深拷贝 Slice/Array：逐元素 deepCopy，修改 dst 元素不得污染 src。
// Slice 为空时按 src 长度重新分配；Array 不能 MakeSlice 覆盖，按较短长度逐元素拷贝。
func cpySliceArray(dst, src reflect.Value, depth int, opt *options, visited map[uintptr]bool) error {
	l := src.Len()
	if dst.Len() > 0 && dst.Len() < src.Len() {
		l = dst.Len()
	}

	if dst.Kind() == reflect.Slice {
		if dst.Len() == 0 && src.Len() > 0 {
			dst.Set(reflect.MakeSlice(dst.Type(), l, l))
		}
	} else {
		// Array：逐元素拷贝，勿用 MakeSlice 覆盖
		if dst.Len() < l {
			l = dst.Len()
		}
	}

	for i := 0; i < l; i++ {
		if err := deepCopyInner(dst.Index(i), src.Index(i), depth+1, opt, visited); err != nil {
			return err
		}
	}

	return nil
}

// deepCopy 是 deepCopyInner 的入口包装，负责创建本次拷贝的循环引用记录
func deepCopy(dst, src reflect.Value, depth int, opt *options) error {
	visited := make(map[uintptr]bool)
	return deepCopyInner(dst, src, depth, opt, visited)
}

func deepCopyInner(dst, src reflect.Value, depth int, opt *options, visited map[uintptr]bool) error {
	// 深度限制统一在递归入口执行，struct->map 路径同样受 WithMaxDepth 约束
	if opt.ExceedMaxDepth(depth) {
		return ErrMaxDepthExceeded
	}

	switch dst.Kind() {
	case reflect.Map:
		if dst.IsNil() && dst.CanSet() {
			dst.Set(reflect.MakeMap(dst.Type()))
		}

		// 嵌套 map -> map：深拷贝，修改 dst 的嵌套容器不得污染 src
		if src.Kind() == reflect.Map {
			// 整体 key 类型不兼容时直接报错，避免 dst 被静默清空。
			// 注意：不能直接用 ConvertibleTo 判断——Go 中整数可转换为 string（rune 语义），
			// map[int]→map[string] 会被误判为兼容，须排除 string↔数值 的跨类别转换。
			if !keyTypeCompatible(src.Type().Key(), dst.Type().Key()) {
				return ErrInvalidCopyDestination
			}

			for _, key := range src.MapKeys() {
				// 1) key 类型兼容：不可转换则跳过该键（防御，正常情况已被整体检查拦截）
				if !key.Type().ConvertibleTo(dst.Type().Key()) {
					continue
				}
				k := key.Convert(dst.Type().Key())

				// 2) TypeConvert（仅 string key 传入字段名，契约：Fn 收到的 src 可能是引用值）
				fieldName := ""
				if k.Kind() == reflect.String {
					fieldName = k.String()
				}
				v, _ := opt.TypeConvert(fieldName, src.MapIndex(key))

				// 3) NameConvert（仅 string key）
				if k.Kind() == reflect.String {
					k = reflect.ValueOf(opt.NameConvert(k.String()))
				}

				// 4) value 类型兼容：不可转换则跳过
				if !v.Type().ConvertibleTo(dst.Type().Elem()) {
					continue
				}
				v = v.Convert(dst.Type().Elem())

				// 5) 解包 interface，判断具体内容是否为容器/指针
				actual := v
				if actual.Kind() == reflect.Interface && actual.IsValid() {
					actual = actual.Elem()
				}

				// 6) 分发：nil 值直接写零值；容器/指针递归深拷贝；其余直接写入
				if !actual.IsValid() {
					dst.SetMapIndex(k, reflect.Zero(v.Type()))
					continue
				}

				switch actual.Kind() {
				case reflect.Map:
					newMap := reflect.MakeMap(actual.Type())
					if err := deepCopyInner(newMap, actual, depth+1, opt, visited); err != nil {
						return err
					}
					dst.SetMapIndex(k, wrapIfInterface(v, newMap))
				case reflect.Slice, reflect.Array:
					var newVal reflect.Value
					if actual.Kind() == reflect.Array {
						newVal = reflect.New(actual.Type()).Elem()
					} else {
						newVal = reflect.MakeSlice(actual.Type(), actual.Len(), actual.Len())
					}
					if err := cpySliceArray(newVal, actual, depth+1, opt, visited); err != nil {
						return err
					}
					dst.SetMapIndex(k, wrapIfInterface(v, newVal))
				case reflect.Pointer:
					if actual.IsNil() {
						dst.SetMapIndex(k, reflect.Zero(v.Type()))
						continue
					}
					newPtr := reflect.New(actual.Type().Elem())
					if err := deepCopyInner(newPtr, actual, depth+1, opt, visited); err != nil {
						return err
					}
					dst.SetMapIndex(k, wrapIfInterface(v, newPtr))
				default:
					// 标量/struct 值/非容器：值拷贝直接写入
					dst.SetMapIndex(k, v)
				}
			}
			return nil
		}

		// struct -> map
		for i, n := 0, src.NumField(); i < n; i++ {
			sf := src.Type().Field(i)
			if sf.PkgPath != "" && !sf.Anonymous {
				continue
			}

			// 检查是否在跳过列表中
			if opt.isSkipField(sf.Name) {
				continue
			}

			tag := parseTag(sf.Tag.Get(opt.tagName))
			if tag.Contains(tagIgnore) {
				continue
			}

			// must 模式：只拷贝带 must 标签的字段
			if opt.must && !tag.Contains(tagMust) {
				continue
			}

			if sf.Anonymous && sf.Type.Kind() == reflect.Struct {
				if err := deepCopyInner(dst, src.Field(i), depth+1, opt, visited); err != nil {
					return err
				}

				continue
			}

			name := toName(sf.Name, tag, opt)
			value, converted := opt.TypeConvert(sf.Name, src.Field(i))

			// 应用值转换函数（先转换，后判空，避免空值被提前跳过）
			if opt.valueConverter != nil {
				c := opt.valueConverter(sf.Name, value.Interface())
				if c != nil {
					value = reflect.ValueOf(c)
				}
			}

			// 判空基于转换后的结果
			if opt.ignoreEmpty && value.IsZero() {
				continue
			}

			// 未经过类型转换的 struct 展开为嵌套 map；容器/指针字段深拷贝隔离；其余直接写入
			if !converted && value.Kind() == reflect.Struct {
				newDst := reflect.ValueOf(make(map[string]any, src.Field(i).NumField()))
				if err := deepCopyInner(newDst, value, depth+1, opt, visited); err != nil {
					return err
				}

				dst.SetMapIndex(reflect.ValueOf(name), newDst)
			} else if !converted && isContainerKind(value.Kind()) {
				copied, err := copyContainer(value, depth, opt, visited)
				if err != nil {
					return err
				}

				dst.SetMapIndex(reflect.ValueOf(name), copied)
			} else {
				dst.SetMapIndex(reflect.ValueOf(name), value)
			}
		}

		return nil
	case reflect.Struct:
		if src.Kind() == reflect.Struct {
			return cpyStruct(dst, src, depth, opt, visited)
		} else if src.Kind() == reflect.Map {
			// map -> struct
			for _, key := range src.MapKeys() {
				name, ok := key.Interface().(string)
				if !ok {
					return ErrMapKeyNotMatch
				}

				v := src.MapIndex(key)
				name = opt.NameConvert(name)
				tv := getFieldByName(dst, name, opt)

				// 未匹配到字段或字段不可设置（未导出等）时跳过
				if !tv.IsValid() || !tv.CanSet() {
					continue
				}

				if opt.isSkipField(name) {
					continue
				}

				// 应用值转换函数（先转换，后判空，避免空值被提前跳过）
				if opt.valueConverter != nil {
					c := opt.valueConverter(name, v.Interface())
					if c != nil {
						v = reflect.ValueOf(c)
					}
				}

				if opt.ignoreEmpty {
					// map[string]any 的值以 interface 承载，解包后再判断具体值是否为零值
					vv := v
					if vv.Kind() == reflect.Interface {
						vv = vv.Elem()
					}
					if !vv.IsValid() || vv.IsZero() {
						continue
					}
				}

				if err := deepCopyInner(tv, v, depth+1, opt, visited); err != nil {
					return err
				}
			}
			return nil
		}
		return ErrNotSupported
	case reflect.Slice, reflect.Array:
		return cpySliceArray(dst, src, depth, opt, visited)
	case reflect.Pointer:
		// 解包 interface（如 interface 中嵌套 interface）
		for src.Kind() == reflect.Interface {
			src = src.Elem()
			if !src.IsValid() {
				if dst.CanSet() {
					dst.Set(reflect.Zero(dst.Type()))
				}
				return nil
			}
		}

		// 循环引用检测：记录已访问的源指针地址，自引用/互引用安全终止
		if src.Kind() == reflect.Pointer {
			if src.IsNil() {
				if dst.CanSet() {
					dst.Set(reflect.Zero(dst.Type()))
				}
				return nil
			}

			addr := src.Pointer()
			if visited[addr] {
				// 已访问过该指针，终止递归避免栈溢出
				return nil
			}
			visited[addr] = true
			defer delete(visited, addr)

			src = src.Elem()
		}

		if dst.IsNil() {
			if !dst.CanSet() {
				return nil
			}
			dst.Set(reflect.New(dst.Type().Elem()))
		}

		dst = dst.Elem()
		return deepCopyInner(dst, src, depth, opt, visited)
	case reflect.Interface:
		if src.Kind() == reflect.Interface {
			src = src.Elem()
			if !src.IsValid() {
				if dst.CanSet() {
					dst.Set(reflect.Zero(dst.Type()))
				}
				return nil
			}
		}

		if !dst.CanSet() {
			return nil
		}

		// 指针/容器类型深拷贝，避免共享同一引用（interface 内嵌 map/slice/指针时同样隔离）
		switch src.Kind() {
		case reflect.Pointer:
			if src.IsNil() {
				dst.Set(reflect.Zero(dst.Type()))
				return nil
			}

			newDst := reflect.New(src.Type().Elem())
			if err := deepCopyInner(newDst, src, depth+1, opt, visited); err != nil {
				return err
			}

			if newDst.Type().AssignableTo(dst.Type()) {
				dst.Set(newDst)
				return nil
			}

			return fmt.Errorf("copier: cannot assign %v to %v", src.Type(), dst.Type())
		case reflect.Map:
			newMap := reflect.MakeMap(src.Type())
			if err := deepCopyInner(newMap, src, depth+1, opt, visited); err != nil {
				return err
			}

			if newMap.Type().AssignableTo(dst.Type()) {
				dst.Set(newMap)
				return nil
			}

			return fmt.Errorf("copier: cannot assign %v to %v", src.Type(), dst.Type())
		case reflect.Slice:
			newSlice := reflect.MakeSlice(src.Type(), src.Len(), src.Len())
			if err := cpySliceArray(newSlice, src, depth+1, opt, visited); err != nil {
				return err
			}

			if newSlice.Type().AssignableTo(dst.Type()) {
				dst.Set(newSlice)
				return nil
			}

			return fmt.Errorf("copier: cannot assign %v to %v", src.Type(), dst.Type())
		}

		if src.Type().AssignableTo(dst.Type()) {
			dst.Set(src)
			return nil
		}

		if src.Type().ConvertibleTo(dst.Type()) {
			dst.Set(src.Convert(dst.Type()))
			return nil
		}

		return fmt.Errorf("copier: cannot assign %v to %v", src.Type(), dst.Type())
	default:
		if !dst.CanSet() {
			return nil
		}

		// 数值↔string 跨类别转换不走 reflect ConvertibleTo（Go 中整数→string 是 rune 码点语义），
		// 交由下方 String/Int/Uint/Float 分支用 strconv 做十进制转换
		if src.Type().AssignableTo(dst.Type()) ||
			(!isNumericStringCross(src.Type(), dst.Type()) && src.Type().ConvertibleTo(dst.Type())) {
			dst.Set(src.Convert(dst.Type()))
			return nil
		}

		switch dst.Kind() {
		case reflect.String:
			switch v := src.Interface().(type) {
			case string:
				dst.SetString(v)
			case []byte:
				dst.SetString(string(v))
			case bool:
				dst.SetString(strconv.FormatBool(v))
			case int, int8, int16, int32, int64:
				dst.SetString(fmt.Sprintf("%d", v))
			case uint, uint8, uint16, uint32, uint64:
				dst.SetString(fmt.Sprintf("%d", v))
			default:
				dst.SetString(fmt.Sprintf("%v", v))
			}
		case reflect.Bool:
			switch v := src.Interface().(type) {
			case string:
				if b, err := strconv.ParseBool(v); err == nil {
					dst.SetBool(b)
				}
			case bool:
				dst.SetBool(v)
			}
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			switch v := src.Interface().(type) {
			case string:
				if n, err := strconv.ParseInt(v, 10, 64); err == nil {
					dst.SetInt(n)
				}
			case bool:
				if v {
					dst.SetInt(1)
				} else {
					dst.SetInt(0)
				}
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
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
			switch v := src.Interface().(type) {
			case string:
				if n, err := strconv.ParseUint(v, 10, 64); err == nil {
					dst.SetUint(n)
				}
			case bool:
				if v {
					dst.SetUint(1)
				} else {
					dst.SetUint(0)
				}
			case uint:
				dst.SetUint(uint64(v))
			case uint8:
				dst.SetUint(uint64(v))
			case uint16:
				dst.SetUint(uint64(v))
			case uint32:
				dst.SetUint(uint64(v))
			case uint64:
				dst.SetUint(uint64(v))
			case uintptr:
				dst.SetUint(uint64(v))
			}
		case reflect.Float32, reflect.Float64:
			switch v := src.Interface().(type) {
			case string:
				if f, err := strconv.ParseFloat(v, 64); err == nil {
					dst.SetFloat(f)
				}
			case bool:
				if v {
					dst.SetFloat(1)
				} else {
					dst.SetFloat(0)
				}
			case float64:
				dst.SetFloat(v)
			case float32:
				dst.SetFloat(float64(v))
			}
		}

		return nil
	}
}

func cpyStruct(dst, src reflect.Value, depth int, opt *options, visited map[uintptr]bool) error {
	if dst.CanSet() {
		if _, ok := src.Interface().(time.Time); ok {
			dst.Set(src)
			return nil
		}
	}

	typ := src.Type()
	for i, n := 0, src.NumField(); i < n; i++ {
		sf := typ.Field(i)
		if sf.PkgPath != "" && !sf.Anonymous {
			continue
		}

		// 检查是否在跳过列表中
		if opt.isSkipField(sf.Name) {
			continue
		}

		tag := parseTag(sf.Tag.Get(opt.tagName))
		if tag.Contains(tagIgnore) {
			continue
		}

		// must 模式：只拷贝带 must 标签的字段
		if opt.must && !tag.Contains(tagMust) {
			continue
		}

		name := toName(sf.Name, tag, opt)

		dstValue := getFieldByName(dst, name, opt)
		// 未匹配到字段或字段不可设置（如大小写不敏感匹配到未导出字段）时跳过
		if !dstValue.IsValid() || !dstValue.CanSet() {
			continue
		}

		sField := src.Field(i)

		// 应用值转换函数（先转换，后判空，避免空值被提前跳过）
		if opt.valueConverter != nil {
			c := opt.valueConverter(sf.Name, sField.Interface())
			if c != nil {
				sField = reflect.ValueOf(c)
			}
		}

		// 判空基于转换后的结果
		if opt.ignoreEmpty && sField.IsZero() {
			continue
		}

		if err := deepCopyInner(dstValue, sField, depth+1, opt, visited); err != nil {
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

// wrapIfInterface 将深拷贝结果重新包装为原值的类型形态：
// 原值（original）是 interface 类型（如 map[string]any 的元素）时返回 interface 包装值，
// 否则原样返回 copied（如 map[string]map[string]int 等具体元素类型场景）。
func wrapIfInterface(original, copied reflect.Value) reflect.Value {
	if original.Kind() == reflect.Interface {
		return reflect.ValueOf(copied.Interface())
	}

	return copied
}

func isContainerKind(k reflect.Kind) bool {
	switch k {
	case reflect.Map, reflect.Slice, reflect.Array, reflect.Pointer, reflect.Interface:
		return true
	}

	return false
}

// copyContainer 深拷贝容器/指针值并返回独立副本（与 struct->map 字段级别的
// 深拷贝语义一致），保证修改 dst 的嵌套容器不污染 src。
func copyContainer(src reflect.Value, depth int, opt *options, visited map[uintptr]bool) (reflect.Value, error) {
	actual := src
	if actual.Kind() == reflect.Interface && actual.IsValid() {
		actual = actual.Elem()
	}

	if !actual.IsValid() {
		return src, nil
	}

	switch actual.Kind() {
	case reflect.Map:
		newMap := reflect.MakeMap(actual.Type())
		if err := deepCopyInner(newMap, actual, depth+1, opt, visited); err != nil {
			return reflect.Value{}, err
		}
		return wrapIfInterface(src, newMap), nil
	case reflect.Slice, reflect.Array:
		var newVal reflect.Value
		if actual.Kind() == reflect.Array {
			newVal = reflect.New(actual.Type()).Elem()
		} else {
			newVal = reflect.MakeSlice(actual.Type(), actual.Len(), actual.Len())
		}
		if err := cpySliceArray(newVal, actual, depth+1, opt, visited); err != nil {
			return reflect.Value{}, err
		}
		return wrapIfInterface(src, newVal), nil
	case reflect.Pointer:
		if actual.IsNil() {
			return reflect.Zero(src.Type()), nil
		}
		newPtr := reflect.New(actual.Type().Elem())
		if err := deepCopyInner(newPtr, actual, depth+1, opt, visited); err != nil {
			return reflect.Value{}, err
		}
		return wrapIfInterface(src, newPtr), nil
	default:
		return src, nil
	}
}

// keyTypeCompatible 判断 map key 类型是否兼容：
// 可赋值（AssignableTo，含同类型、named type→底层、具体类型→接口）视为兼容；
// 可转换时仅允许同类基础类型（数值→数值、string→string），
// 排除 string↔数值 的跨类别转换（Go 中整数→string 是 rune 语义，不符合 map key 拷贝预期）。
func keyTypeCompatible(src, dst reflect.Type) bool {
	if src.AssignableTo(dst) {
		return true
	}

	if !src.ConvertibleTo(dst) {
		return false
	}

	srcKind, dstKind := src.Kind(), dst.Kind()

	// 数值类之间允许转换（int→int64、MyInt→int 等）
	if isNumericKind(srcKind) && isNumericKind(dstKind) {
		return true
	}

	// string 仅允许 string↔string（named string 类型）
	return srcKind == reflect.String && dstKind == reflect.String
}

func isNumericKind(k reflect.Kind) bool {
	switch k {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr,
		reflect.Float32, reflect.Float64:
		return true
	}

	return false
}

// isNumericStringCross 判断 src→dst 是否为 数值↔string 跨类别转换。
// Go 中整数可转换为 string（rune 码点语义），这类转换不符合字段拷贝的十进制期望，
// 需排除在 reflect ConvertibleTo 短路之外。
func isNumericStringCross(src, dst reflect.Type) bool {
	return (isNumericKind(src.Kind()) && dst.Kind() == reflect.String) ||
		(src.Kind() == reflect.String && isNumericKind(dst.Kind()))
}

func indirect(reflectValue reflect.Value) reflect.Value {
	for reflectValue.Kind() == reflect.Pointer {
		reflectValue = reflectValue.Elem()
	}

	return reflectValue
}
