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

	// 统一解包 src 的 interface 层：map[string]any 等场景的值以 interface 承载
	// （可能嵌套多层 interface），在分支分发前解包，避免 Slice/Struct 等分支
	// 对 interface Value 调用 Len/NumField 等导致 panic（如 map 容器值 → struct
	// 容器字段）；nil/无效值在 dst 可写时置零后返回，与 Pointer/Interface 分支
	// 的 nil 处理语义一致。
	// 注：Pointer/Interface 分支内部原有的解包循环此后恒为 false（死代码），
	// 保留不动以最小化 diff。
	for src.IsValid() && src.Kind() == reflect.Interface {
		src = src.Elem()
	}
	if !src.IsValid() {
		if dst.CanSet() {
			dst.Set(reflect.Zero(dst.Type()))
		}
		return nil
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

		// struct -> map（plan 缓存快路径，行为与原循环完全等价）
		if plan := getStructToMapPlan(src.Type(), opt); plan != nil {
			return cpyStructToMap(dst, src, plan, depth, opt, visited)
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
			// map -> struct（plan 缓存快路径，行为与原循环完全等价）
			if plan := getMapToStructPlan(dst.Type(), opt); plan != nil {
				return cpyMapToStruct(dst, src, plan, depth, opt, visited)
			}

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

	// plan 缓存：默认字段匹配配置下走预计算路径，行为与下方逐字段逻辑完全等价
	if plan := getStructPlan(src.Type(), dst.Type(), opt); plan != nil {
		return cpyStructByPlan(dst, src, plan, depth, opt, visited)
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
		// 未匹配到字段或字段不可设置（如大小写不敏感匹配到未导出字段）时，
		// 降级尝试 dst 的 setter 方法（仅启用方法映射时，默认关闭保持向后兼容）
		if !dstValue.IsValid() || !dstValue.CanSet() {
			if opt.methodMapping {
				if err := callSetter(dst, name, src.Field(i), opt); err != nil {
					return err
				}
			}
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

	// 方向 B：src getter 方法 → dst 字段（仅启用方法映射时，默认关闭）
	if opt.methodMapping {
		if err := copyGetters(dst, src, depth, opt, visited); err != nil {
			return err
		}
	}

	return nil
}

// cpyStructByPlan 按预计算 plan.fields 迭代，行为与 cpyStruct 逐字段逻辑完全等价：
// 对每条 fieldMapping：
//   - dstIdx 非空：dst.FieldByIndex 取值，不可设置时降级 setter（仅启用方法映射时）；
//     可设置则走与现有 cpyStruct 相同的后续逻辑（valueConverter → ignoreEmpty → deepCopyInner）
//   - dstIdx 为空：无匹配字段，降级尝试 callSetter（仅启用方法映射时），否则静默跳过
//
// 全部字段处理完后，与现有一致：启用方法映射时调用 copyGetters。
// plan 运行时零分配：name/srcName 预存在 plan 中，字段索引直接定位。
func cpyStructByPlan(dst, src reflect.Value, plan *structPlan, depth int, opt *options, visited map[uintptr]bool) error {
	for _, m := range plan.fields {
		srcField := src.Field(m.srcIdx)

		if len(m.dstIdx) == 0 {
			// 无匹配字段：降级尝试 dst 的 setter 方法（仅启用方法映射时）
			if opt.methodMapping {
				if err := callSetter(dst, m.name, srcField, opt); err != nil {
					return err
				}
			}
			continue
		}

		// 常见场景（直接字段）用 Field 直接索引，避免 FieldByIndex 的索引链遍历
		var dstValue reflect.Value
		if len(m.dstIdx) == 1 {
			dstValue = dst.Field(m.dstIdx[0])
		} else {
			dstValue = dst.FieldByIndex(m.dstIdx)
		}
		// 未导出字段等不可设置时，降级尝试 dst 的 setter 方法
		if !dstValue.CanSet() {
			if opt.methodMapping {
				if err := callSetter(dst, m.name, srcField, opt); err != nil {
					return err
				}
			}
			continue
		}

		sField := srcField

		// 应用值转换函数（先转换，后判空，避免空值被提前跳过）
		if opt.valueConverter != nil {
			c := opt.valueConverter(m.srcName, sField.Interface())
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

	// 方向 B：src getter 方法 → dst 字段（仅启用方法映射时，默认关闭）
	if opt.methodMapping {
		if err := copyGetters(dst, src, depth, opt, visited); err != nil {
			return err
		}
	}

	return nil
}

// cpyStructToMap 按预计算 plan.entries 迭代，行为与 deepCopyInner Map 分支
// struct→map 循环完全等价：
//   - expand=true：匿名 struct 字段递归展开
//   - 否则：TypeConvert → valueConverter → ignoreEmpty →
//     未转换 struct 展开嵌套 map / 容器 copyContainer / 直接 SetMapIndex
func cpyStructToMap(dst, src reflect.Value, plan *structToMapPlan, depth int, opt *options, visited map[uintptr]bool) error {
	for _, e := range plan.entries {
		if e.expand {
			// 匿名 struct 字段：递归展开
			if err := deepCopyInner(dst, src.Field(e.srcIdx), depth+1, opt, visited); err != nil {
				return err
			}
			continue
		}

		srcField := src.Field(e.srcIdx)
		value, converted := opt.TypeConvert(e.srcName, srcField)

		// 应用值转换函数（先转换，后判空，避免空值被提前跳过）
		if opt.valueConverter != nil {
			c := opt.valueConverter(e.srcName, value.Interface())
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
			newDst := reflect.ValueOf(make(map[string]any, srcField.NumField()))
			if err := deepCopyInner(newDst, value, depth+1, opt, visited); err != nil {
				return err
			}

			dst.SetMapIndex(reflect.ValueOf(e.name), newDst)
		} else if !converted && isContainerKind(value.Kind()) {
			copied, err := copyContainer(value, depth, opt, visited)
			if err != nil {
				return err
			}

			dst.SetMapIndex(reflect.ValueOf(e.name), copied)
		} else {
			dst.SetMapIndex(reflect.ValueOf(e.name), value)
		}
	}

	return nil
}

// cpyMapToStruct 按预计算 plan.lookup 迭代，行为与 deepCopyInner Struct 分支
// map→struct 循环完全等价：
// key 转 string（非 string → ErrMapKeyNotMatch）→ NameConvert → 查表定位 dst 字段
// （与 getFieldByName 等价，FieldByIndex 索引链含嵌入提升）→ CanSet 检查 →
// isSkipField → valueConverter → ignoreEmpty → deepCopyInner 递归。
func cpyMapToStruct(dst, src reflect.Value, plan *mapToStructPlan, depth int, opt *options, visited map[uintptr]bool) error {
	for _, key := range src.MapKeys() {
		name, ok := key.Interface().(string)
		if !ok {
			return ErrMapKeyNotMatch
		}

		v := src.MapIndex(key)
		name = opt.NameConvert(name)

		// 查表定位 dst 字段（与 getFieldByName 等价）
		lookupKey := name
		if !opt.caseSensitive {
			lookupKey = strings.ToLower(name)
		}
		index, ok := plan.lookup[lookupKey]
		if !ok {
			continue
		}

		tv := dst.FieldByIndex(index)
		// 未匹配到字段或字段不可设置（未导出等）时跳过
		if !tv.CanSet() {
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
