package copier

// 本文件为"审计复现测试"：逐一复现审计报告中的缺陷。
// 每个用例断言"期望的正确行为"，当前生产代码满足不了 → 测试 FAIL 即复现成功。
// FAIL 是预期结果，不要试图修改测试使其通过，也不要修改生产代码。

import (
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// ============ C-1 Slice/Array 深拷贝缺失（copier.go deepCopy 无 Slice case） ============

func TestAuditC1SliceDeepCopy(t *testing.T) {
	t.Run("same type slice deep isolation", func(t *testing.T) {
		// 期望：深拷贝，修改 dst 不影响 src（当前 default 分支 dst.Set(src.Convert(...))
		// 浅拷贝共享底层数组 → FAIL）
		type srcS struct{ Items []int }
		type dstS struct{ Items []int }

		src := srcS{Items: []int{1, 2, 3}}
		var dst dstS

		err := Copy(&dst, src)
		assert.NoError(t, err)

		dst.Items[0] = 999
		assert.Equal(t, 1, src.Items[0])
	})

	t.Run("slice element type conversion", func(t *testing.T) {
		// 期望：[]int → []int64 应转换后拷贝（当前 slice 类型不可 ConvertibleTo，
		// 走 default 的 else 分支且无 Slice case → 静默丢数据 → FAIL）
		type srcS struct{ Items []int }
		type dstS struct{ Items []int64 }

		src := srcS{Items: []int{1, 2, 3}}
		var dst dstS

		err := Copy(&dst, src)
		assert.NoError(t, err)
		assert.Equal(t, []int64{1, 2, 3}, dst.Items)
	})
}

// ============ C-3 Interface 字段静默丢失（copier.go:209-211 错误的 Kind 比较） ============
// H-3 nil interface 处理合并到本组 t.Run 内（单独子用例）

func TestAuditC3InterfaceField(t *testing.T) {
	type srcI struct{ Data any }
	type dstI struct{ Data any }

	t.Run("string value in interface field", func(t *testing.T) {
		// 任务原始用例：src/dst 字段均为 any 类型。实测 src.Kind()==Interface==dst.Kind()，
		// 走 Elem→dst.Set 正常拷贝 → PASS（未复现，观察记录：本场景不触发 L209-211 缺陷）
		src := srcI{Data: "hello"}
		var dst dstI

		err := Copy(&dst, src)
		assert.NoError(t, err)
		assert.Equal(t, "hello", dst.Data)
	})

	t.Run("concrete type to interface field", func(t *testing.T) {
		// 真实触发场景：src 具体类型字段（string）→ dst interface 字段。
		// deepCopy case Interface 中 src.Kind()(String) != dst.Kind()(Interface) → 直接 return nil
		// → dst.Data 保持 nil → FAIL（复现审计 C-3）
		type srcC struct{ Data string }
		type dstC struct{ Data any }

		src := srcC{Data: "hello"}
		var dst dstC

		err := Copy(&dst, src)
		assert.NoError(t, err)
		assert.Equal(t, "hello", dst.Data)
	})

	t.Run("nil interface should not panic", func(t *testing.T) {
		// H-3：Data 为 nil 时不应 panic（当前 src.Elem() 无效 → return nil，不 panic → PASS）
		src := srcI{Data: nil}
		var dst dstI

		assert.NotPanics(t, func() {
			err := Copy(&dst, src)
			assert.NoError(t, err)
		})
	})

	t.Run("nested struct in interface", func(t *testing.T) {
		// 期望：嵌套 struct 正常拷贝（当前 src.Kind()==Interface 与 dst 相等 → Elem() → Struct
		// → 非 Ptr → dst.Set(src) 直接设置 → PASS）
		src := srcI{Data: Address{City: "Beijing"}}
		var dst dstI

		err := Copy(&dst, src)
		assert.NoError(t, err)
		assert.Equal(t, Address{City: "Beijing"}, dst.Data)
	})
}

// ============ C-4 循环引用栈溢出（deepCopy 无 visited 检测） ============

// C-4 测试辅助类型：函数内类型声明不允许前向引用，循环引用类型需定义在包级
type auditNodeC4 struct {
	Name   string
	Parent *auditNodeC4
}

type auditAC4 struct {
	Name string
	B    *auditBC4
}

type auditBC4 struct {
	Name string
	A    *auditAC4
}

func TestAuditC4CycleReference(t *testing.T) {
	// 修复前：deepCopy 无 visited 检测，自引用结构体会导致进程级栈溢出（fatal error，无法 recover）。
	// 修复后：deepCopy 按递归深度记录已访问指针地址，自引用/互引用安全终止（不 panic、拷贝正确）。

	t.Run("self reference", func(t *testing.T) {
		n := &auditNodeC4{Name: "root"}
		n.Parent = n // 自引用

		var dst auditNodeC4
		assert.NotPanics(t, func() {
			err := Copy(&dst, n)
			assert.NoError(t, err)
		})

		// 顶层 Name 拷贝正确
		assert.Equal(t, "root", dst.Name)
		// 循环引用在递归第二层被终止：Parent 若被拷贝，其内容与顶层一致，
		// 且不允许再出现指向自身的更深引用（否则说明循环未终止）
		if dst.Parent != nil {
			assert.Equal(t, "root", dst.Parent.Name)
			assert.Nil(t, dst.Parent.Parent)
		}
	})

	t.Run("mutual reference", func(t *testing.T) {
		a := &auditAC4{Name: "a"}
		b := &auditBC4{Name: "b"}
		a.B = b
		b.A = a // 互引用

		var dst auditAC4
		assert.NotPanics(t, func() {
			err := Copy(&dst, a)
			assert.NoError(t, err)
		})

		assert.Equal(t, "a", dst.Name)
		if dst.B != nil {
			assert.Equal(t, "b", dst.B.Name)
			if dst.B.A != nil {
				assert.Equal(t, "a", dst.B.A.Name)
				assert.Nil(t, dst.B.A.B)
			}
		}
	})
}

// ============ C-5 深度限制失效（ExceedMaxDepth 只在 cpyStruct，struct→map 路径无限制） ============

func TestAuditC5MaxDepth(t *testing.T) {
	// Go 不允许 struct 值类型直接递归（类型大小无限），因此用不同类型链构造 5 层值嵌套。
	// 注意：函数内类型声明的作用域从声明处开始（不允许前向引用），故需倒序定义。
	// struct→map 路径（deepCopy case reflect.Map 的 L160 检测 value.Kind()==Struct）
	// 会逐层递归展开嵌套 struct。
	type L5 struct {
		Name string
	}
	type L4 struct {
		Name  string
		Child L5
	}
	type L3 struct {
		Name  string
		Child L4
	}
	type L2 struct {
		Name  string
		Child L3
	}
	type L1 struct {
		Name  string
		Child L2
	}

	src := L1{Name: "n1", Child: L2{Name: "n2", Child: L3{Name: "n3", Child: L4{Name: "n4", Child: L5{Name: "n5"}}}}}

	dst := map[string]any{}
	// 期望：深度超过 WithMaxDepth(3) 应返回错误（当前 struct→map 路径在
	// deepCopy case reflect.Map 中无 ExceedMaxDepth 检查，逐层拷贝到底且无错 → FAIL）
	err := Copy(&dst, src, WithMaxDepth(3))
	assert.Error(t, err)
}

// ============ H-1 map→struct 选项失效（copier.go:75-90 无 ignoreEmpty/skipFields 检查） ============

func TestAuditH1MapToStructOptions(t *testing.T) {
	type PersonH struct {
		Name string
		Age  int
		Sex  string
	}

	m := map[string]any{
		"Name": "",
		"Age":  0,
		"Sex":  "skip-me",
	}
	dst := PersonH{Name: "preset", Age: 42, Sex: "original"}

	// 期望：WithIgnoreEmpty 使空值不覆盖目标非空字段；WithSkipFields("Sex") 使 Sex 不被拷贝。
	// 当前 map→struct 分支（copier.go:75-90）完全忽略这两个选项 → 全部被覆盖 → FAIL
	err := Copy(&dst, m, WithIgnoreEmpty(), WithSkipFields("Sex"))
	assert.NoError(t, err)
	assert.Equal(t, "preset", dst.Name)
	assert.Equal(t, 42, dst.Age)
	assert.Equal(t, "original", dst.Sex)
}

// ============ H-2 CanSet panic 防护（多处 dst.Set 无 CanSet 守卫） ============

func TestAuditH2CanSetPanic(t *testing.T) {
	t.Run("unexported field match should not panic", func(t *testing.T) {
		// 场景：getFieldByName 用 FieldByNameFunc(strings.EqualFold) 大小写不敏感匹配，
		// 源字段 Name（导出）会匹配到目标字段 name（未导出，CanSet=false）。
		// deepCopy default 分支无条件 dst.Set(...) → reflect panic。
		// 期望：不 panic（当前 panic → FAIL）
		type srcE struct{ Name string }
		type dstE struct{ name string }

		src := srcE{Name: "hello"}
		var dst dstE

		defer func() {
			if r := recover(); r != nil {
				t.Errorf("unexpected panic: %v", r)
			}
		}()

		_ = Copy(&dst, src)

		// 匹配到未导出字段时应跳过：字段保持零值，未被写入
		assert.Equal(t, "", dst.name)
	})
}

// ============ M-3 TypeConvert 存根不生效（option.go:59-61 恒返回 false） ============

func TestAuditM3TypeConvertStub(t *testing.T) {
	type Order struct {
		CreatedAt string
	}

	src := Order{CreatedAt: "2024-01-02 15:04:05"}
	dst := map[string]any{}

	// 注册 string → time.Time 转换器（struct→map 路径）
	err := Copy(&dst, src, WithConverters(TypeConverter{
		FieldName: "CreatedAt",
		SrcType:   "",
		DstType:   time.Time{},
		Fn: func(src any) (any, error) {
			return time.Parse("2006-01-02 15:04:05", src.(string))
		},
	}))
	assert.NoError(t, err)

	// 期望：转换器生效，CreatedAt 转为 time.Time
	// 当前：TypeConvert 恒返回 (value, false)，converters 从不被调用 → 仍是 string → FAIL
	assert.IsType(t, time.Time{}, dst["CreatedAt"])
	assert.Equal(t, time.Date(2024, 1, 2, 15, 4, 5, 0, time.UTC), dst["CreatedAt"])
}

// ============ M-4 类型转换缺失（copier.go:235-266 仅 String/Int 常见类型） ============

func TestAuditM4TypeConvertMissing(t *testing.T) {
	t.Run("float64 to int", func(t *testing.T) {
		// 观察：本用例 PASS，未复现缺陷——reflect 的 ConvertibleTo 对数值类型返回 true
		// （Go 数值转换规则），default 分支经 dst.Set(src.Convert(int)) 截断转换成功
		// （3.7 → 3）。审计所述"静默不转换"不适用于 float→int。
		type fsrc struct{ N float64 }
		type fdst struct{ N int }

		src := fsrc{N: 3.7}
		var dst fdst

		err := Copy(&dst, src)
		assert.NoError(t, err)
		assert.Equal(t, 3, dst.N)
	})

	t.Run("string to int", func(t *testing.T) {
		// 真实触发场景：string→int 不可 ConvertibleTo，default 分支 Int case 无 string
		// 分支 → 静默不转换，dst.N 保持 0 → FAIL（复现审计 M-4）
		type ssrc struct{ N string }
		type sdst struct{ N int }

		src := ssrc{N: "42"}
		var dst sdst

		err := Copy(&dst, src)
		assert.NoError(t, err)
		assert.Equal(t, 42, dst.N)
	})

	t.Run("bool to string", func(t *testing.T) {
		// 期望：bool → string 转换正确（src/dst 字段同名 B）
		// 观察：当前经 default 分支 String case 的 fallback fmt.Sprintf("%v", v) 实际可转
		// （"true"）→ 本用例 PASS，未复现缺陷
		type bsrc struct{ B bool }
		type bdst struct{ B string }

		src := bsrc{B: true}
		var dst bdst

		err := Copy(&dst, src)
		assert.NoError(t, err)
		assert.Equal(t, "true", dst.B)
	})
}

// ============ 错误路径覆盖（现有测试缺少） ============

func TestAuditErrPaths(t *testing.T) {
	t.Run("nil src", func(t *testing.T) {
		var dst Person
		err := Copy(&dst, nil)
		assert.Error(t, err)
	})

	t.Run("nil src pointer", func(t *testing.T) {
		var dst Person
		err := Copy(&dst, (*Person)(nil))
		assert.Error(t, err)
	})

	t.Run("nil dst", func(t *testing.T) {
		src := Person{Name: "x"}
		err := Copy(nil, src)
		assert.Error(t, err)
	})

	t.Run("non-addressable dst (value not pointer)", func(t *testing.T) {
		src := Person{Name: "x"}
		var dst Person
		err := Copy(dst, src)
		assert.Error(t, err)
	})

	t.Run("nil map dst auto init (map to map)", func(t *testing.T) {
		src := map[string]any{"a": 1}
		var dst map[string]any
		err := Copy(&dst, src)
		assert.NoError(t, err)
		assert.Equal(t, src, dst)
	})
}

// ============ L-2 indirectType 额外解引用 Slice（copier.go:360） ============

func TestAuditL2IndirectSlice(t *testing.T) {
	data := []byte("hello")
	src := &data
	var dst []byte

	err := Copy(&dst, src)
	assert.NoError(t, err)
	assert.Equal(t, []byte("hello"), dst)

	// 观察记录：copier() 的 indirect 已先解掉指针，*[]byte→[]byte 内容正确 → PASS。
	// indirectType 对 Slice 的额外解引用在本用例不触发；该缺陷主要影响 map→map 分支
	// 的 key 类型检查（fromType 被错误解引用到元素类型），需 src 为 slice of map 场景
	// 才会暴露，本用例未复现。
}

// ============ 正向补充（M-5：现有测试只覆盖 happy path，顺带补强） ============

func TestAuditPositiveSliceNested(t *testing.T) {
	// 期望：slice 嵌套 struct 元素深拷贝，修改 dst 元素不得污染 src
	// 当前：与 C-1 同根因（slice 浅拷贝共享底层数组）→ 修改污染 src → FAIL
	type Inner struct{ Name string }
	type OSrc struct{ Items []Inner }
	type ODst struct{ Items []Inner }

	src := OSrc{Items: []Inner{{Name: "a"}, {Name: "b"}}}
	var dst ODst

	err := Copy(&dst, src)
	assert.NoError(t, err)

	dst.Items[0].Name = "changed"
	assert.Equal(t, "a", src.Items[0].Name)
}

func TestAuditPositiveNestedMap(t *testing.T) {
	// 期望：map[string]any 嵌套 map/slice 内容拷贝正确，且深拷贝隔离
	from := map[string]any{
		"name": "x",
		"meta": map[string]any{"a": 1},
		"list": []int{1, 2, 3},
	}
	to := map[string]any{}

	err := Copy(&to, from)
	assert.NoError(t, err)
	assert.Equal(t, from, to)

	// 深拷贝隔离：修改 to 的嵌套容器不得污染 from
	to["meta"].(map[string]any)["a"] = 999
	to["list"].([]int)[0] = 999
	assert.Equal(t, 1, from["meta"].(map[string]any)["a"])
	assert.Equal(t, 1, from["list"].([]int)[0])
}

// ============ N-1 Interface 分支静默丢数据（目标为具名窄接口时） ============

// N-1 测试辅助类型：Go 不允许在函数内为局部类型定义方法，需定义在包级
type auditNamer interface {
	AuditName() string
}

type auditNamed struct {
	V string
}

func (a auditNamed) AuditName() string { return a.V }

func TestAuditN1InterfaceAssignError(t *testing.T) {
	t.Run("concrete value to narrow interface not satisfied", func(t *testing.T) {
		// 源具体类型（string）不实现 auditNamer 接口 → 应返回非 nil 错误而非静默丢弃
		type srcS struct{ Data string }
		type dstS struct{ Data auditNamer }

		src := srcS{Data: "hello"}
		var dst dstS

		err := Copy(&dst, src)
		assert.Error(t, err)
		assert.Nil(t, dst.Data)
	})

	t.Run("pointer deep copy to narrow interface not satisfied", func(t *testing.T) {
		// 指针深拷贝后 *string 不实现 auditNamer → 应返回错误而非静默丢弃
		type srcP struct{ Data *string }
		type dstP struct{ Data auditNamer }

		s := "hello"
		src := srcP{Data: &s}
		var dst dstP

		err := Copy(&dst, src)
		assert.Error(t, err)
		assert.Nil(t, dst.Data)
	})

	t.Run("concrete value satisfying narrow interface copied", func(t *testing.T) {
		// 源 auditNamed 实现 auditNamer → 正常拷贝
		type srcOK struct{ Data auditNamed }
		type dstOK struct{ Data auditNamer }

		src := srcOK{Data: auditNamed{V: "world"}}
		var dst dstOK

		err := Copy(&dst, src)
		assert.NoError(t, err)
		assert.Equal(t, "world", dst.Data.AuditName())
	})

	t.Run("any to any still works", func(t *testing.T) {
		// any->any 正常路径不得回归
		type srcI struct{ Data any }
		type dstI struct{ Data any }

		src := srcI{Data: "hello"}
		var dst dstI

		err := Copy(&dst, src)
		assert.NoError(t, err)
		assert.Equal(t, "hello", dst.Data)
	})
}

// ============ N-2 ignoreEmpty 与 valueConverter 顺序（先转换后判空） ============

func TestAuditN2ValueConverterOrder(t *testing.T) {
	naConverter := func(fieldName string, value any) any {
		if fieldName == "Name" {
			if s, ok := value.(string); ok && s == "" {
				return "N/A"
			}
		}
		return value
	}

	t.Run("map to struct", func(t *testing.T) {
		type personN2 struct {
			Name string
			Age  int
		}

		m := map[string]any{"Name": "", "Age": 30}
		dst := personN2{Name: "preset", Age: 42}

		err := Copy(&dst, m, WithIgnoreEmpty(), WithValueConverter(naConverter))
		assert.NoError(t, err)
		assert.Equal(t, "N/A", dst.Name) // 空值经转换器转为有意义值，不应被 ignoreEmpty 跳过
		assert.Equal(t, 30, dst.Age)
	})

	t.Run("struct to struct", func(t *testing.T) {
		type srcN2 struct {
			Name string
			Age  int
		}
		type dstN2 struct {
			Name string
			Age  int
		}

		src := srcN2{Name: "", Age: 30}
		var dst dstN2

		err := Copy(&dst, src, WithIgnoreEmpty(), WithValueConverter(naConverter))
		assert.NoError(t, err)
		assert.Equal(t, "N/A", dst.Name)
		assert.Equal(t, 30, dst.Age)
	})

	t.Run("struct to map", func(t *testing.T) {
		type srcN2 struct {
			Name string
		}

		src := srcN2{Name: ""}
		dst := map[string]any{}

		err := Copy(&dst, src, WithIgnoreEmpty(), WithValueConverter(naConverter))
		assert.NoError(t, err)
		assert.Equal(t, "N/A", dst["Name"])
	})

	t.Run("pure empty without converter still ignored", func(t *testing.T) {
		// 无转换器时纯空值仍被忽略（H-1 行为不得回归）
		type personN2 struct {
			Name string
			Age  int
		}

		m := map[string]any{"Name": "", "Age": 30}
		dst := personN2{Name: "preset", Age: 42}

		err := Copy(&dst, m, WithIgnoreEmpty())
		assert.NoError(t, err)
		assert.Equal(t, "preset", dst.Name) // 空值被忽略，保留预设值
		assert.Equal(t, 30, dst.Age)        // 非空值正常覆盖
	})
}

// ============ oracle 建议：TypeConvert 与 valueConverter 组合执行 ============

func TestAuditN2TypeConvertValueConverterCombine(t *testing.T) {
	type Order struct {
		Count string
	}

	src := Order{Count: "5"}
	dst := map[string]any{}

	// struct->map 路径：TypeConvert 先转 string->int，valueConverter 再翻倍
	err := Copy(&dst, src,
		WithConverters(TypeConverter{
			FieldName: "Count",
			SrcType:   "",
			DstType:   int(0),
			Fn: func(src any) (any, error) {
				return strconv.Atoi(src.(string))
			},
		}),
		WithValueConverter(func(fieldName string, value any) any {
			if fieldName == "Count" {
				if n, ok := value.(int); ok {
					return n * 2
				}
			}
			return value
		}),
	)

	assert.NoError(t, err)
	assert.Equal(t, 10, dst["Count"])
}

// ============ oracle 建议：WithCaseSensitive + map->struct 组合 ============

func TestAuditN2CaseSensitiveMapToStruct(t *testing.T) {
	type personCS struct {
		Name string
	}

	t.Run("exact match copied", func(t *testing.T) {
		m := map[string]any{"Name": "John"}
		var dst personCS

		err := Copy(&dst, m, WithCaseSensitive())
		assert.NoError(t, err)
		assert.Equal(t, "John", dst.Name)
	})

	t.Run("case mismatch not copied", func(t *testing.T) {
		m := map[string]any{"name": "John"}
		dst := personCS{Name: "preset"}

		err := Copy(&dst, m, WithCaseSensitive())
		assert.NoError(t, err)
		assert.Equal(t, "preset", dst.Name) // 大小写不敏感匹配被关闭，name 不匹配 Name
	})

	t.Run("case insensitive default still works", func(t *testing.T) {
		m := map[string]any{"name": "John"}
		var dst personCS

		err := Copy(&dst, m)
		assert.NoError(t, err)
		assert.Equal(t, "John", dst.Name)
	})
}

// ============ N-3 bool 跨类型转换缺失 ============

func TestAuditN3BoolCrossType(t *testing.T) {
	t.Run("string to bool", func(t *testing.T) {
		type ssrc struct{ Flag string }
		type sdst struct{ Flag bool }

		src := ssrc{Flag: "true"}
		var dst sdst

		err := Copy(&dst, src)
		assert.NoError(t, err)
		assert.Equal(t, true, dst.Flag)
	})

	t.Run("string to bool false", func(t *testing.T) {
		type ssrc struct{ Flag string }
		type sdst struct{ Flag bool }

		src := ssrc{Flag: "false"}
		var dst sdst

		err := Copy(&dst, src)
		assert.NoError(t, err)
		assert.Equal(t, false, dst.Flag)
	})

	t.Run("bool to int", func(t *testing.T) {
		type bsrc struct{ A bool }
		type bdst struct{ A int }

		src := bsrc{A: true}
		var dst bdst

		err := Copy(&dst, src)
		assert.NoError(t, err)
		assert.Equal(t, 1, dst.A)
	})

	t.Run("bool to int64", func(t *testing.T) {
		type bsrc struct{ A bool }
		type bdst struct{ A int64 }

		src := bsrc{A: false}
		var dst bdst

		err := Copy(&dst, src)
		assert.NoError(t, err)
		assert.Equal(t, int64(0), dst.A)
	})

	t.Run("bool to uint", func(t *testing.T) {
		type bsrc struct{ A bool }
		type bdst struct{ A uint }

		src := bsrc{A: true}
		var dst bdst

		err := Copy(&dst, src)
		assert.NoError(t, err)
		assert.Equal(t, uint(1), dst.A)
	})

	t.Run("bool to float64", func(t *testing.T) {
		type bsrc struct{ A bool }
		type bdst struct{ A float64 }

		src := bsrc{A: true}
		var dst bdst

		err := Copy(&dst, src)
		assert.NoError(t, err)
		assert.Equal(t, float64(1), dst.A)
	})
}

// ============ oracle 建议：TypeConvert DstType 精确匹配边界 ============

func TestAuditN3TypeConvertDstTypeBoundary(t *testing.T) {
	type Order struct {
		Count string
	}

	t.Run("dst type mismatch falls back to original", func(t *testing.T) {
		src := Order{Count: "5"}
		dst := map[string]any{}

		// 声明 DstType 为 int，但 Fn 实际返回 string → 转换不生效，保留原值
		err := Copy(&dst, src, WithConverters(TypeConverter{
			FieldName: "Count",
			SrcType:   "",
			DstType:   int(0),
			Fn: func(src any) (any, error) {
				return src.(string) + "!", nil
			},
		}))

		assert.NoError(t, err)
		assert.Equal(t, "5", dst["Count"])
	})

	t.Run("dst type match applies", func(t *testing.T) {
		src := Order{Count: "5"}
		dst := map[string]any{}

		err := Copy(&dst, src, WithConverters(TypeConverter{
			FieldName: "Count",
			SrcType:   "",
			DstType:   int(0),
			Fn: func(src any) (any, error) {
				return strconv.Atoi(src.(string))
			},
		}))

		assert.NoError(t, err)
		assert.Equal(t, 5, dst["Count"])
	})
}

// ============ N-4 WithMust() 声明未实现 ============

func TestAuditN4WithMust(t *testing.T) {
	type srcM struct {
		Name string `copier:"must"`
		Age  int
		Sex  string `copier:"must,toname=Gender"`
	}
	type dstM struct {
		Name   string
		Age    int
		Gender string
	}

	src := srcM{Name: "John", Age: 30, Sex: "Male"}

	t.Run("only must fields copied", func(t *testing.T) {
		var dst dstM

		err := Copy(&dst, src, WithMust())
		assert.NoError(t, err)
		assert.Equal(t, "John", dst.Name)    // must 字段被拷贝
		assert.Equal(t, 0, dst.Age)          // 非 must 字段不被拷贝
		assert.Equal(t, "Male", dst.Gender)  // must+toname 组合生效
	})

	t.Run("must tag no effect without WithMust", func(t *testing.T) {
		var dst dstM

		err := Copy(&dst, src)
		assert.NoError(t, err)
		assert.Equal(t, "John", dst.Name)   // toname 仍生效
		assert.Equal(t, 30, dst.Age)
		assert.Equal(t, "Male", dst.Gender)
	})

	t.Run("struct to map with must", func(t *testing.T) {
		dst := map[string]any{}

		err := Copy(&dst, src, WithMust())
		assert.NoError(t, err)
		assert.Equal(t, "John", dst["Name"])
		assert.Equal(t, "Male", dst["Gender"])
		_, hasAge := dst["Age"]
		assert.False(t, hasAge)
	})
}

// ============ LR-1 map->map 深拷贝隔离 ============

// LR-1 测试辅助类型（oracle 规格：包级定义）
type auditPtrVal struct {
	Name string
}

func TestAuditLR1MapDeepCopyIsolation(t *testing.T) {
	t.Run("nested map isolation", func(t *testing.T) {
		from := map[string]any{"meta": map[string]any{"a": 1}}
		to := map[string]any{}

		assert.NoError(t, Copy(&to, from))

		to["meta"].(map[string]any)["a"] = 999
		assert.Equal(t, 1, from["meta"].(map[string]any)["a"])
	})

	t.Run("slice value isolation", func(t *testing.T) {
		from := map[string]any{"list": []int{1, 2, 3}}
		to := map[string]any{}

		assert.NoError(t, Copy(&to, from))

		to["list"].([]int)[0] = 999
		assert.Equal(t, 1, from["list"].([]int)[0])
	})

	t.Run("pointer value isolation", func(t *testing.T) {
		from := map[string]any{"ptr": &auditPtrVal{Name: "a"}}
		to := map[string]any{}

		assert.NoError(t, Copy(&to, from))

		to["ptr"].(*auditPtrVal).Name = "changed"
		assert.Equal(t, "a", from["ptr"].(*auditPtrVal).Name)
	})

	t.Run("interface-wrapped slice isolation", func(t *testing.T) {
		// []any 内嵌 map：修改 to 的嵌套 map，src 不变
		from := map[string]any{"list": []any{map[string]any{"a": 1}}}
		to := map[string]any{}

		assert.NoError(t, Copy(&to, from))

		to["list"].([]any)[0].(map[string]any)["a"] = 999
		assert.Equal(t, 1, from["list"].([]any)[0].(map[string]any)["a"])
	})

	t.Run("deep nested modification leaves src intact", func(t *testing.T) {
		from := map[string]any{
			"a": map[string]any{
				"b": map[string]any{
					"c": []int{1, 2},
				},
			},
		}
		to := map[string]any{}

		assert.NoError(t, Copy(&to, from))

		// 全层级修改
		to["a"].(map[string]any)["x"] = "new"
		to["a"].(map[string]any)["b"].(map[string]any)["c"].([]int)[0] = 999

		// src 逐层不变
		_, hasX := from["a"].(map[string]any)["x"]
		assert.False(t, hasX)
		assert.Equal(t, 1, from["a"].(map[string]any)["b"].(map[string]any)["c"].([]int)[0])
	})

	t.Run("nil value does not panic", func(t *testing.T) {
		from := map[string]any{"nil": nil, "ok": 1}
		to := map[string]any{}

		assert.NotPanics(t, func() {
			err := Copy(&to, from)
			assert.NoError(t, err)
		})

		assert.Nil(t, to["nil"])
		assert.Equal(t, 1, to["ok"])
	})

	t.Run("empty map value is independent", func(t *testing.T) {
		from := map[string]any{"m": map[string]any{}}
		to := map[string]any{}

		assert.NoError(t, Copy(&to, from))

		to["m"].(map[string]any)["new"] = 1
		_, hasNew := from["m"].(map[string]any)["new"]
		assert.False(t, hasNew)
	})

	t.Run("nil map dst auto init", func(t *testing.T) {
		var to map[string]any
		from := map[string]any{"a": 1}

		assert.NoError(t, Copy(&to, from))
		assert.NotNil(t, to)
		assert.Equal(t, from, to)
	})

	t.Run("with MaxDepth option", func(t *testing.T) {
		// 顶层之下 3 层 map 嵌套，WithMaxDepth(2) 时第 3 层（depth=3）超限
		from := map[string]any{"a": map[string]any{"b": map[string]any{"c": map[string]any{"d": 1}}}}
		to := map[string]any{}

		err := Copy(&to, from, WithMaxDepth(2))
		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrMaxDepthExceeded)
	})

	t.Run("mixed value types map", func(t *testing.T) {
		from := map[string]any{
			"name": "x",
			"meta": map[string]any{"a": 1},
			"list": []int{1, 2, 3},
			"ptr":  &auditPtrVal{Name: "p"},
		}
		to := map[string]any{}

		assert.NoError(t, Copy(&to, from))
		assert.Equal(t, from, to)

		// 修改后全部隔离
		to["meta"].(map[string]any)["a"] = 999
		to["list"].([]int)[0] = 999
		to["ptr"].(*auditPtrVal).Name = "changed"
		assert.Equal(t, 1, from["meta"].(map[string]any)["a"])
		assert.Equal(t, 1, from["list"].([]int)[0])
		assert.Equal(t, "p", from["ptr"].(*auditPtrVal).Name)
	})

	t.Run("[]byte value deep copied", func(t *testing.T) {
		from := map[string]any{"data": []byte("hello")}
		to := map[string]any{}

		assert.NoError(t, Copy(&to, from))

		to["data"].([]byte)[0] = 'X'
		assert.Equal(t, byte('h'), from["data"].([]byte)[0])
	})
}

// ============ P2 map key 类型不兼容时整体报错 ============

func TestAuditP2KeyTypeIncompatible(t *testing.T) {
	t.Run("int key to string key map returns error", func(t *testing.T) {
		from := map[int]string{1: "a"}
		to := map[string]string{}

		err := Copy(&to, from)
		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrInvalidCopyDestination)
	})

	t.Run("int key to string key map dst unchanged", func(t *testing.T) {
		from := map[int]string{1: "a"}
		to := map[string]string{"preset": "keep"}

		err := Copy(&to, from)
		assert.Error(t, err)
		// 报错时 dst 不得被静默清空
		assert.Equal(t, "keep", to["preset"])
	})

	t.Run("same key type map still works", func(t *testing.T) {
		from := map[string]any{"a": 1}
		to := map[string]any{}

		err := Copy(&to, from)
		assert.NoError(t, err)
		assert.Equal(t, from, to)
	})

	t.Run("compatible key type works", func(t *testing.T) {
		from := map[string]int{"a": 1}
		to := map[string]any{}

		err := Copy(&to, from)
		assert.NoError(t, err)
		assert.Equal(t, 1, to["a"])
	})
}

// ============ P3 struct 值内部指针共享语义锁定 ============

// P3 测试辅助类型（包级定义，与审计测试风格一致）
type auditP3Inner struct {
	Name string
}

type auditP3Outer struct {
	Ptr *auditP3Inner
}

func TestAuditP3StructValuePointerSharing(t *testing.T) {
	t.Run("map with struct value shares internal pointer", func(t *testing.T) {
		// 预期行为，非缺陷：map value 为 struct 值类型时遵循 Go 值拷贝语义，
		// struct 内部指针字段与 src 共享引用（深拷贝仅发生在容器/指针值类型层面）。
		from := map[string]auditP3Outer{
			"a": {Ptr: &auditP3Inner{Name: "original"}},
		}
		to := map[string]auditP3Outer{}

		assert.NoError(t, Copy(&to, from))

		to["a"].Ptr.Name = "changed"
		assert.Equal(t, "changed", from["a"].Ptr.Name) // 同步变化 = 共享是预期行为
	})

	t.Run("map with pointer value deep copies internal pointer", func(t *testing.T) {
		// 指针值类型走 Pointer 分支递归深拷贝，内部指针同样隔离
		from := map[string]*auditP3Outer{
			"a": {Ptr: &auditP3Inner{Name: "original"}},
		}
		to := map[string]*auditP3Outer{}

		assert.NoError(t, Copy(&to, from))

		to["a"].Ptr.Name = "changed"
		assert.Equal(t, "original", from["a"].Ptr.Name) // 隔离，src 不变
	})
}
