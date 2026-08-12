package copier

import (
	"fmt"
	"reflect"
	"sync"
)

// errorInterface 是 error 接口的 reflect.Type，用于校验方法返回类型。
var errorInterface = reflect.TypeOf((*error)(nil)).Elem()

// methodCache 缓存某类型的 getter/setter 方法索引。
// 方法列表是类型属性、不依赖任何 option，可全局缓存。
type methodCache struct {
	getters map[string]reflect.Method
	setters map[string]reflect.Method
}

// methodCacheMap 全局方法缓存，键为 reflect.Type，值为 *methodCache。
// 并发安全：sync.Map 支持并发读写，且类型的反射方法列表在运行期不可变。
var methodCacheMap sync.Map

// resolveMethods 解析类型 t 的全部导出 getter/setter 方法。
// 在 reflect.PointerTo(t) 上遍历，同时覆盖值接收者与指针接收者方法。
//
// getter 规则：NumIn()==1（仅 receiver）且 NumOut()==1 或 2；
// NumOut()==2 时 Out(1) 必须实现 error 接口。
// setter 规则：NumIn()==2（receiver + 1 参数）且 NumOut()==0 或 1；
// NumOut()==1 时 Out(0) 必须实现 error 接口。
func resolveMethods(t reflect.Type) *methodCache {
	if v, ok := methodCacheMap.Load(t); ok {
		return v.(*methodCache)
	}

	pt := reflect.PointerTo(t)
	c := &methodCache{
		getters: make(map[string]reflect.Method),
		setters: make(map[string]reflect.Method),
	}

	for i := 0; i < pt.NumMethod(); i++ {
		m := pt.Method(i)
		if !m.IsExported() {
			continue
		}

		mt := m.Type
		switch {
		case mt.NumIn() == 1 && (mt.NumOut() == 1 || mt.NumOut() == 2):
			if mt.NumOut() == 2 && !mt.Out(1).Implements(errorInterface) {
				continue
			}
			c.getters[m.Name] = m
		case mt.NumIn() == 2 && (mt.NumOut() == 0 || mt.NumOut() == 1):
			if mt.NumOut() == 1 && !mt.Out(0).Implements(errorInterface) {
				continue
			}
			c.setters[m.Name] = m
		}
	}

	actual, _ := methodCacheMap.LoadOrStore(t, c)
	return actual.(*methodCache)
}

// callSetter 方向 A：dst 无同名字段（或字段不可设置）时，
// 尝试调用 dst 上的同名 setter 方法，将 src 字段值作为参数传入。
// 方法不存在、签名不符（参数类型不可赋值）时静默跳过，不报错不误用。
func callSetter(dst reflect.Value, name string, srcField reflect.Value, opt *options) error {
	if !dst.CanAddr() {
		return nil
	}

	cache := resolveMethods(dst.Type())
	setter, ok := cache.setters[name]
	if !ok {
		return nil
	}

	// 参数类型不匹配时静默跳过（方法存在但签名不符，不能误用）
	if !srcField.Type().AssignableTo(setter.Type.In(1)) {
		return nil
	}

	// ignoreEmpty：调用前判断 src 字段是否为零值（避免无意义的 setter 调用）
	if opt.ignoreEmpty && srcField.IsZero() {
		return nil
	}

	out := dst.Addr().MethodByName(name).Call([]reflect.Value{srcField})
	if len(out) == 1 && !out[0].IsNil() {
		return fmt.Errorf("%w: %v", ErrMethodReturnError, out[0].Interface())
	}

	return nil
}

// copyGetters 方向 B：将 src 上的同名 getter 方法返回值写入 dst 字段。
// dst 字段原名（不经过 toname/NameConvert 转换）直接匹配 src 方法名；
// 因 Go 中类型的方法名与字段名互斥，若 src 有同名字段则该字段已由主循环处理，
// 字段优先，此处跳过。
// 返回值经 deepCopyInner 写入，自动继承深度限制与类型转换语义。
func copyGetters(dst, src reflect.Value, depth int, opt *options, visited map[uintptr]bool) error {
	cache := resolveMethods(src.Type())
	if len(cache.getters) == 0 {
		return nil
	}

	// src 不可寻址时在副本上调用，避免 getter 副作用修改源值
	recv := src
	if !recv.CanAddr() {
		tmp := reflect.New(src.Type())
		tmp.Elem().Set(src)
		recv = tmp.Elem()
	}
	recvPtr := recv.Addr()

	for i, n := 0, dst.NumField(); i < n; i++ {
		sf := dst.Type().Field(i)
		if sf.PkgPath != "" && !sf.Anonymous {
			continue
		}

		dstField := dst.Field(i)
		if !dstField.CanSet() {
			continue
		}

		name := sf.Name
		// 字段优先：src 存在同名字段时跳过 getter
		if _, ok := src.Type().FieldByName(name); ok {
			continue
		}

		if _, ok := cache.getters[name]; !ok {
			continue
		}

		out := recvPtr.MethodByName(name).Call(nil)

		val := out[0]
		if len(out) == 2 && !out[1].IsNil() {
			return fmt.Errorf("%w: %v", ErrMethodReturnError, out[1].Interface())
		}

		// ignoreEmpty：调用后返回值零值时跳过写入。
		// 与方向 A setter 的"调用前判空"不同：getter 的副作用必须先执行，
		// 再根据返回值决定是否写入。
		if opt.ignoreEmpty && val.IsZero() {
			continue
		}

		if err := deepCopyInner(dstField, val, depth+1, opt, visited); err != nil {
			return err
		}
	}

	return nil
}
