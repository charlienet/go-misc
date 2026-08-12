package copier

// A+ 增强工作包覆盖率补充测试：覆盖 ToMap、WithTagName、错误路径与边界转换等缺口。

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

// ============ ToMap（copier.go 覆盖率缺口 0%） ============

func TestToMap(t *testing.T) {
	t.Run("struct to map equivalence with Copy", func(t *testing.T) {
		type addrT struct {
			City string
		}
		type srcT struct {
			Name    string
			Age     int
			Address addrT
		}

		src := srcT{Name: "John", Age: 30, Address: addrT{City: "BJ"}}

		m1, err := ToMap(src)
		assert.NoError(t, err)

		var m2 map[string]any
		assert.NoError(t, Copy(&m2, src))
		assert.Equal(t, m2, m1)
		assert.Equal(t, "John", m1["Name"])
		assert.Equal(t, map[string]any{"City": "BJ"}, m1["Address"])
	})

	t.Run("nested container deep copied", func(t *testing.T) {
		type srcT struct {
			Items []int
		}

		src := srcT{Items: []int{1, 2, 3}}
		m, err := ToMap(src)
		assert.NoError(t, err)

		// 深拷贝隔离：修改返回 map 的嵌套容器不影响 src
		m["Items"].([]int)[0] = 999
		assert.Equal(t, 1, src.Items[0])
	})

	t.Run("options applied", func(t *testing.T) {
		type srcT struct {
			Name string
			Sex  string
		}

		src := srcT{Name: "John", Sex: "Male"}
		m, err := ToMap(src, WithSkipFields("Sex"))
		assert.NoError(t, err)

		_, hasSex := m["Sex"]
		assert.False(t, hasSex)
		assert.Equal(t, "John", m["Name"])
	})
}

// ============ WithTagName（option.go 覆盖率缺口 0%） ============
// 同时覆盖 cpyStruct 的 tagIgnore 分支与 toName 的 toname 分支。

func TestWithTagName(t *testing.T) {
	type srcT struct {
		Name string `json:"toname=nick"`
		Age  int    `json:"-"`
	}
	type dstT struct {
		Nick string
		Age  int
	}

	src := srcT{Name: "John", Age: 30}

	t.Run("default copier tag ignores json tag", func(t *testing.T) {
		var dst dstT
		assert.NoError(t, Copy(&dst, src))
		assert.Equal(t, 30, dst.Age)   // 无 copier 标签，按字段名匹配
		assert.Equal(t, "", dst.Nick)  // json 标签不作为 copier 标签来源
	})

	t.Run("WithTagName json applies", func(t *testing.T) {
		var dst dstT
		assert.NoError(t, Copy(&dst, src, WithTagName("json")))
		assert.Equal(t, "John", dst.Nick) // toname=nick 生效
		assert.Equal(t, 0, dst.Age)       // json:"-" 忽略该字段
	})
}

// ============ 错误路径（deepCopyInner 分支缺口） ============

func TestAuditAplusErrorPaths(t *testing.T) {
	t.Run("struct dst with slice src returns ErrNotSupported", func(t *testing.T) {
		type dstT struct {
			A int
		}
		var dst dstT

		err := Copy(&dst, []int{1, 2})
		assert.ErrorIs(t, err, ErrNotSupported)
	})

	t.Run("struct dst with non-string-key map returns ErrMapKeyNotMatch", func(t *testing.T) {
		type dstT struct {
			A int
		}
		var dst dstT

		err := Copy(&dst, map[int]string{1: "a"})
		assert.ErrorIs(t, err, ErrMapKeyNotMatch)
	})
}

// ============ Pointer / Interface 边界 ============

func TestAuditAplusPointerEdges(t *testing.T) {
	t.Run("nil pointer field sets dst nil", func(t *testing.T) {
		type srcT struct {
			P *int
		}
		type dstT struct {
			P *int
		}

		src := srcT{P: nil}
		var dst dstT
		assert.NoError(t, Copy(&dst, src))
		assert.Nil(t, dst.P)
	})

	t.Run("nil interface to pointer field sets dst nil", func(t *testing.T) {
		type srcT struct {
			P any
		}
		type dstT struct {
			P *int
		}

		src := srcT{P: nil}
		var dst dstT
		assert.NoError(t, Copy(&dst, src))
		assert.Nil(t, dst.P)
	})

	t.Run("interface field with nil pointer", func(t *testing.T) {
		type srcT struct {
			Data *int
		}
		type dstT struct {
			Data any
		}

		src := srcT{Data: nil}
		var dst dstT
		assert.NoError(t, Copy(&dst, src))
		assert.Nil(t, dst.Data)
	})

	t.Run("interface field with non-nil pointer deep copied", func(t *testing.T) {
		type srcT struct {
			Data *int
		}
		type dstT struct {
			Data any
		}

		n := 42
		src := srcT{Data: &n}
		var dst dstT
		assert.NoError(t, Copy(&dst, src))

		got := dst.Data.(*int)
		assert.Equal(t, 42, *got)

		// 指针深拷贝隔离
		*got = 99
		assert.Equal(t, 42, n)
	})
}

// ============ Interface 容器赋值失败（窄接口） ============

func TestAuditAplusInterfaceNarrowError(t *testing.T) {
	type narrow interface {
		Foo()
	}

	t.Run("map value to narrow interface returns error", func(t *testing.T) {
		type srcT struct {
			Data map[string]int
		}
		type dstT struct {
			Data narrow
		}

		src := srcT{Data: map[string]int{"a": 1}}
		var dst dstT
		err := Copy(&dst, src)
		assert.Error(t, err)
	})

	t.Run("slice value to narrow interface returns error", func(t *testing.T) {
		type srcT struct {
			Data []int
		}
		type dstT struct {
			Data narrow
		}

		src := srcT{Data: []int{1}}
		var dst dstT
		err := Copy(&dst, src)
		assert.Error(t, err)
	})
}

// ============ default 分支类型转换补充 ============

func TestAuditAplusDefaultConversions(t *testing.T) {
	t.Run("[]byte to string", func(t *testing.T) {
		type srcT struct {
			B []byte
		}
		type dstT struct {
			B string
		}

		src := srcT{B: []byte("hi")}
		var dst dstT
		assert.NoError(t, Copy(&dst, src))
		assert.Equal(t, "hi", dst.B)
	})

	t.Run("int to string", func(t *testing.T) {
		type srcT struct {
			N int
		}
		type dstT struct {
			N string
		}

		src := srcT{N: 42}
		var dst dstT
		assert.NoError(t, Copy(&dst, src))
		assert.Equal(t, "42", dst.N)
	})

	t.Run("uint to string", func(t *testing.T) {
		type srcT struct {
			N uint
		}
		type dstT struct {
			N string
		}

		src := srcT{N: 7}
		var dst dstT
		assert.NoError(t, Copy(&dst, src))
		assert.Equal(t, "7", dst.N)
	})

	t.Run("float64 to string via fallback", func(t *testing.T) {
		type srcT struct {
			F float64
		}
		type dstT struct {
			F string
		}

		src := srcT{F: 3.14}
		var dst dstT
		assert.NoError(t, Copy(&dst, src))
		assert.Equal(t, "3.14", dst.F)
	})

	t.Run("string to float64", func(t *testing.T) {
		type srcT struct {
			F string
		}
		type dstT struct {
			F float64
		}

		src := srcT{F: "3.14"}
		var dst dstT
		assert.NoError(t, Copy(&dst, src))
		assert.Equal(t, 3.14, dst.F)
	})

	t.Run("string to uint", func(t *testing.T) {
		type srcT struct {
			N string
		}
		type dstT struct {
			N uint
		}

		src := srcT{N: "42"}
		var dst dstT
		assert.NoError(t, Copy(&dst, src))
		assert.Equal(t, uint(42), dst.N)
	})

	t.Run("bool false to uint", func(t *testing.T) {
		type srcT struct {
			B bool
		}
		type dstT struct {
			B uint
		}

		src := srcT{B: false}
		var dst dstT
		assert.NoError(t, Copy(&dst, src))
		assert.Equal(t, uint(0), dst.B)
	})

	t.Run("bool false to float64", func(t *testing.T) {
		type srcT struct {
			B bool
		}
		type dstT struct {
			B float64
		}

		src := srcT{B: false}
		var dst dstT
		assert.NoError(t, Copy(&dst, src))
		assert.Equal(t, float64(0), dst.B)
	})
}

// ============ map value 边界（deepCopyInner map->map 分支） ============

func TestAuditAplusMapValueEdges(t *testing.T) {
	t.Run("array value deep copied", func(t *testing.T) {
		from := map[string]any{"arr": [3]int{1, 2, 3}}
		to := map[string]any{}

		assert.NoError(t, Copy(&to, from))
		// 数组为值类型：to 中的数组与 from 相互独立（修改副本不影响 from）
		arr := to["arr"].([3]int)
		arr[0] = 999
		assert.Equal(t, 1, from["arr"].([3]int)[0])
		assert.Equal(t, [3]int{1, 2, 3}, to["arr"])
	})

	t.Run("nil pointer value does not panic", func(t *testing.T) {
		from := map[string]any{"ptr": (*auditPtrVal)(nil)}
		to := map[string]any{}

		assert.NotPanics(t, func() {
			assert.NoError(t, Copy(&to, from))
		})
		assert.Nil(t, to["ptr"])
	})

	t.Run("incompatible value type skipped silently", func(t *testing.T) {
		// string value 不可 ConvertibleTo int → 跳过该 key，不报错不 panic
		from := map[string]string{"a": "1"}
		to := map[string]int{}

		assert.NoError(t, Copy(&to, from))
		assert.Equal(t, 0, to["a"]) // to 保持空
	})
}

// ============ map key 类型边界（keyTypeCompatible） ============

func TestAuditAplusKeyType(t *testing.T) {
	t.Run("bool key to string key returns error", func(t *testing.T) {
		from := map[bool]int{true: 1}
		to := map[string]int{}

		err := Copy(&to, from)
		assert.ErrorIs(t, err, ErrInvalidCopyDestination)
	})

	t.Run("int key to int64 key works", func(t *testing.T) {
		from := map[int]string{1: "a"}
		to := map[int64]string{}

		assert.NoError(t, Copy(&to, from))
		assert.Equal(t, "a", to[1])
	})

	t.Run("named string key to string key works", func(t *testing.T) {
		type MyString string

		from := map[MyString]int{"a": 1}
		to := map[string]int{}

		assert.NoError(t, Copy(&to, from))
		assert.Equal(t, 1, to["a"])
	})
}

// ============ cpyStruct ignoreEmpty（struct->struct） ============

func TestAuditAplusIgnoreEmptyStruct(t *testing.T) {
	type srcT struct {
		Name string
	}
	type dstT struct {
		Name string
	}

	src := srcT{Name: ""}
	dst := dstT{Name: "preset"}

	assert.NoError(t, Copy(&dst, src, WithIgnoreEmpty()))
	assert.Equal(t, "preset", dst.Name) // 空值被跳过，保留预设
}

// ============ 深度超限错误传播矩阵（覆盖各递归 err 分支） ============

func TestAuditAplusMaxDepthPropagation(t *testing.T) {
	t.Run("struct to map nested struct", func(t *testing.T) {
		type child struct {
			Name string
		}
		type parent struct {
			Name  string
			Child child
		}

		src := parent{Name: "a", Child: child{Name: "b"}}
		dst := map[string]any{}

		err := Copy(&dst, src, WithMaxDepth(0))
		assert.ErrorIs(t, err, ErrMaxDepthExceeded)
	})

	t.Run("struct to map anonymous struct", func(t *testing.T) {
		type anonInner struct {
			X int
		}
		type parent struct {
			anonInner
		}

		src := parent{anonInner: anonInner{X: 1}}
		dst := map[string]any{}

		err := Copy(&dst, src, WithMaxDepth(0))
		assert.ErrorIs(t, err, ErrMaxDepthExceeded)
	})

	t.Run("map to struct nested", func(t *testing.T) {
		type person struct {
			Meta map[string]any
		}

		src := map[string]any{"Meta": map[string]any{"a": 1}}
		var dst person

		err := Copy(&dst, src, WithMaxDepth(0))
		assert.ErrorIs(t, err, ErrMaxDepthExceeded)
	})

	t.Run("map to map slice nested", func(t *testing.T) {
		src := map[string]any{"list": []any{map[string]any{"a": 1}}}
		dst := map[string]any{}

		err := Copy(&dst, src, WithMaxDepth(0))
		assert.ErrorIs(t, err, ErrMaxDepthExceeded)
	})

	t.Run("map to map pointer nested", func(t *testing.T) {
		src := map[string]any{"ptr": &auditPtrVal{Name: "x"}}
		dst := map[string]any{}

		err := Copy(&dst, src, WithMaxDepth(0))
		assert.ErrorIs(t, err, ErrMaxDepthExceeded)
	})

	t.Run("interface pointer nested", func(t *testing.T) {
		type srcT struct {
			Data *int
		}
		type dstT struct {
			Data any
		}

		n := 1
		src := srcT{Data: &n}
		var dst dstT

		// WithMaxDepth(1)：顶层 0 与字段递归 1 均未超限，进入 Interface case 后
		// 指针深拷贝递归（depth+1=2）超限，验证 Interface 内部错误传播
		err := Copy(&dst, src, WithMaxDepth(1))
		assert.ErrorIs(t, err, ErrMaxDepthExceeded)
	})
}

// ============ struct->map 边界（Map case struct 源分支） ============

func TestAuditAplusStructToMapEdges(t *testing.T) {
	t.Run("unexported non-anonymous field skipped", func(t *testing.T) {
		type srcT struct {
			Name   string
			hidden int
		}

		src := srcT{Name: "a", hidden: 1}
		dst := map[string]any{}

		assert.NoError(t, Copy(&dst, src))
		assert.Equal(t, "a", dst["Name"])
		_, hasHidden := dst["hidden"]
		assert.False(t, hasHidden)
	})

	t.Run("tagIgnore field skipped", func(t *testing.T) {
		type srcT struct {
			Name string `copier:"-"`
			Age  int
		}

		src := srcT{Name: "a", Age: 1}
		dst := map[string]any{}

		assert.NoError(t, Copy(&dst, src))
		assert.Equal(t, 1, dst["Age"])
		_, hasName := dst["Name"]
		assert.False(t, hasName)
	})

	t.Run("container fields deep copied", func(t *testing.T) {
		n := 1
		type srcT struct {
			Meta map[string]int
			Ptr  *int
			Arr  [2]int
			Data any
		}

		src := srcT{Meta: map[string]int{"a": 1}, Ptr: &n, Arr: [2]int{1, 2}, Data: []int{9}}
		dst := map[string]any{}

		assert.NoError(t, Copy(&dst, src))

		// 修改 dst 的嵌套容器不影响 src
		dst["Meta"].(map[string]int)["a"] = 999
		assert.Equal(t, 1, src.Meta["a"])

		*dst["Ptr"].(*int) = 99
		assert.Equal(t, 1, n)

		gotData := dst["Data"].([]int)
		gotData[0] = 888
		assert.Equal(t, 9, src.Data.([]int)[0])
	})

	t.Run("nil pointer field to map", func(t *testing.T) {
		type srcT struct {
			Ptr *int
		}

		src := srcT{Ptr: nil}
		dst := map[string]any{}

		assert.NoError(t, Copy(&dst, src))
		assert.Nil(t, dst["Ptr"])
	})

	t.Run("interface scalar value to map", func(t *testing.T) {
		type srcT struct {
			Data any
		}

		src := srcT{Data: "hello"}
		dst := map[string]any{}

		assert.NoError(t, Copy(&dst, src))
		assert.Equal(t, "hello", dst["Data"])
	})

	t.Run("container field depth exceeded", func(t *testing.T) {
		type srcT struct {
			Items []int
		}

		src := srcT{Items: []int{1}}
		dst := map[string]any{}

		err := Copy(&dst, src, WithMaxDepth(0))
		assert.ErrorIs(t, err, ErrMaxDepthExceeded)
	})
}

// ============ Interface case 容器成功/递归错误分支 ============

func TestAuditAplusInterfaceContainerEdges(t *testing.T) {
	t.Run("slice to any field deep copied", func(t *testing.T) {
		type srcT struct {
			Data []int
		}
		type dstT struct {
			Data any
		}

		src := srcT{Data: []int{1, 2}}
		var dst dstT
		assert.NoError(t, Copy(&dst, src))

		dst.Data.([]int)[0] = 999
		assert.Equal(t, 1, src.Data[0])
	})

	t.Run("map recursion depth exceeded", func(t *testing.T) {
		type srcT struct {
			Data map[string]int
		}
		type dstT struct {
			Data any
		}

		src := srcT{Data: map[string]int{"a": 1}}
		var dst dstT
		err := Copy(&dst, src, WithMaxDepth(1))
		assert.ErrorIs(t, err, ErrMaxDepthExceeded)
	})

	t.Run("slice recursion depth exceeded", func(t *testing.T) {
		type srcT struct {
			Data []int
		}
		type dstT struct {
			Data any
		}

		src := srcT{Data: []int{1}}
		var dst dstT
		err := Copy(&dst, src, WithMaxDepth(1))
		assert.ErrorIs(t, err, ErrMaxDepthExceeded)
	})
}

// ============ TypeConvert 边界（option.go） ============

func TestAuditAplusTypeConvertEdges(t *testing.T) {
	type srcT struct {
		F string
	}

	toInt := func(src any) (any, error) { return 1, nil }

	t.Run("nil Fn skipped", func(t *testing.T) {
		src := srcT{F: "x"}
		dst := map[string]any{}

		err := Copy(&dst, src, WithConverters(TypeConverter{FieldName: "F", SrcType: "", Fn: nil}))
		assert.NoError(t, err)
		assert.Equal(t, "x", dst["F"])
	})

	t.Run("field name mismatch skipped", func(t *testing.T) {
		src := srcT{F: "x"}
		dst := map[string]any{}

		err := Copy(&dst, src, WithConverters(TypeConverter{FieldName: "Other", SrcType: "", DstType: int(0), Fn: toInt}))
		assert.NoError(t, err)
		assert.Equal(t, "x", dst["F"])
	})

	t.Run("src type mismatch skipped", func(t *testing.T) {
		// SrcType int(0)，实际 value 是 string：rune 语义被排除，不匹配
		src := srcT{F: "x"}
		dst := map[string]any{}

		err := Copy(&dst, src, WithConverters(TypeConverter{FieldName: "F", SrcType: int(0), DstType: int(0), Fn: toInt}))
		assert.NoError(t, err)
		assert.Equal(t, "x", dst["F"])
	})

	t.Run("fn error skipped", func(t *testing.T) {
		src := srcT{F: "x"}
		dst := map[string]any{}

		err := Copy(&dst, src, WithConverters(TypeConverter{
			FieldName: "F", SrcType: "", DstType: int(0),
			Fn: func(src any) (any, error) { return nil, errors.New("boom") },
		}))
		assert.NoError(t, err)
		assert.Equal(t, "x", dst["F"])
	})

	t.Run("fn nil result skipped", func(t *testing.T) {
		src := srcT{F: "x"}
		dst := map[string]any{}

		err := Copy(&dst, src, WithConverters(TypeConverter{
			FieldName: "F", SrcType: "", DstType: int(0),
			Fn: func(src any) (any, error) { return nil, nil },
		}))
		assert.NoError(t, err)
		assert.Equal(t, "x", dst["F"])
	})

	t.Run("convertible src type matches via ConvertibleTo", func(t *testing.T) {
		type srcI struct {
			N int64
		}

		src := srcI{N: 5}
		dst := map[string]any{}

		err := Copy(&dst, src, WithConverters(TypeConverter{
			FieldName: "N",
			SrcType:   int(0),
			DstType:   int(0),
			Fn: func(src any) (any, error) {
				return int(src.(int64)) * 2, nil
			},
		}))
		assert.NoError(t, err)
		assert.Equal(t, 10, dst["N"])
	})
}

// ============ copyContainer 剩余分支 + ParseBool 失败 ============

func TestAuditAplusCopyContainerEdges(t *testing.T) {
	t.Run("interface nil value to map", func(t *testing.T) {
		type srcT struct {
			Data any
		}

		src := srcT{Data: nil}
		dst := map[string]any{}

		assert.NoError(t, Copy(&dst, src))
		assert.Nil(t, dst["Data"])
	})

	t.Run("map field depth exceeded", func(t *testing.T) {
		type srcT struct {
			Meta map[string]int
		}

		src := srcT{Meta: map[string]int{"a": 1}}
		dst := map[string]any{}

		err := Copy(&dst, src, WithMaxDepth(0))
		assert.ErrorIs(t, err, ErrMaxDepthExceeded)
	})

	t.Run("pointer field depth exceeded", func(t *testing.T) {
		n := 1
		type srcT struct {
			Ptr *int
		}

		src := srcT{Ptr: &n}
		dst := map[string]any{}

		err := Copy(&dst, src, WithMaxDepth(0))
		assert.ErrorIs(t, err, ErrMaxDepthExceeded)
	})
}

func TestAuditAplusParseFailures(t *testing.T) {
	t.Run("invalid string to bool not set", func(t *testing.T) {
		type srcT struct {
			B string
		}
		type dstT struct {
			B bool
		}

		src := srcT{B: "notabool"}
		dst := dstT{B: true}
		assert.NoError(t, Copy(&dst, src))
		assert.Equal(t, true, dst.B) // ParseBool 失败，保持原值
	})
}
