package bench

// 独立对比基准：本地 copier vs jinzhu/copier vs tiendc/go-deepcopy。
// 本文件位于独立模块（copier/bench）内，通过 replace 引用本地主模块，
// 保证主模块保持零依赖。
//
// 统一约定：
//   - 每个迭代内 var dst ... 新建目标，三库公平（复用 dst 时
//     jinzhu/tiendc 对 slice/map 字段可能 append 而非覆盖，会不公平）。
//   - 拷贝结果写入 package 级 sink，防止编译器优化掉拷贝调用。
//   - jinzhu 默认浅拷贝语义，另附 DeepCopy:true 变体（jinzhu-deep）。

import (
	"testing"
	"time"

	localcopier "github.com/charlienet/go-misc/copier"
	jinzhu "github.com/jinzhu/copier"
	deepcopy "github.com/tiendc/go-deepcopy"
)

// sink 承接拷贝结果，防止编译器优化。
var sink any

// benchMedium 与主模块 copier/benchmark_test.go 中的 benchMedium 保持一致
// （10 字段混合类型：string/int/float/bool/切片/map/时间）。
type benchMedium struct {
	Name    string
	Age     int
	Height  float64
	Active  bool
	Tags    []string
	Meta    map[string]string
	City    string
	Score   int64
	Level   uint8
	Created time.Time
}

// ---- 场景 1：同类型 struct → struct ----

func BenchmarkStructSameType(b *testing.B) {
	src := benchMedium{
		Name: "John", Age: 30, Height: 1.75, Active: true,
		Tags: []string{"a", "b"}, Meta: map[string]string{"k": "v"},
		City: "Beijing", Score: 100, Level: 3, Created: time.Now(),
	}

	b.Run("local", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			var dst benchMedium
			if err := localcopier.Copy(&dst, src); err != nil {
				b.Fatal(err)
			}
			sink = dst
		}
	})

	// jinzhu 默认浅拷贝（Tags/Meta 与 src 共享引用）
	b.Run("jinzhu", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			var dst benchMedium
			if err := jinzhu.Copy(&dst, &src); err != nil {
				b.Fatal(err)
			}
			sink = dst
		}
	})

	// jinzhu 深拷贝变体
	b.Run("jinzhu-deep", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			var dst benchMedium
			if err := jinzhu.CopyWithOption(&dst, &src, jinzhu.Option{DeepCopy: true}); err != nil {
				b.Fatal(err)
			}
			sink = dst
		}
	})

	b.Run("tiendc", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			var dst benchMedium
			if err := deepcopy.Copy(&dst, &src); err != nil {
				b.Fatal(err)
			}
			sink = dst
		}
	})
}

// ---- 场景 2：跨类型 struct → struct（同名字段 + 可转换类型） ----
// ID int→int64、Height float32→float64 走类型转换；Name/Age/Active/City 同类型直接拷贝。
// 刻意避开 int→string（Go 的 ConvertibleTo 是 rune 码点语义，三库结果语义不一致）。

type benchUser struct {
	ID     int
	Name   string
	Age    int
	Height float32
	Active bool
	City   string
}

type benchUserDTO struct {
	ID     int64
	Name   string
	Age    int
	Height float64
	Active bool
	City   string
}

func BenchmarkStructCrossType(b *testing.B) {
	src := benchUser{ID: 42, Name: "John", Age: 30, Height: 1.75, Active: true, City: "Beijing"}

	b.Run("local", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			var dst benchUserDTO
			if err := localcopier.Copy(&dst, src); err != nil {
				b.Fatal(err)
			}
			sink = dst
		}
	})

	b.Run("jinzhu", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			var dst benchUserDTO
			if err := jinzhu.Copy(&dst, &src); err != nil {
				b.Fatal(err)
			}
			sink = dst
		}
	})

	b.Run("jinzhu-deep", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			var dst benchUserDTO
			if err := jinzhu.CopyWithOption(&dst, &src, jinzhu.Option{DeepCopy: true}); err != nil {
				b.Fatal(err)
			}
			sink = dst
		}
	})

	b.Run("tiendc", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			var dst benchUserDTO
			if err := deepcopy.Copy(&dst, &src); err != nil {
				b.Fatal(err)
			}
			sink = dst
		}
	})
}

// ---- 场景 3：带嵌套 struct / 指针 / 切片 / map 的 struct（深拷贝成本） ----

type benchNested struct {
	Label string
	Value int
}

type benchComplex struct {
	Name   string
	Nested benchNested
	Ptr    *benchNested
	Items  []int
	Tags   []string
	Meta   map[string]string
	Data   map[string]any
}

func BenchmarkStructNested(b *testing.B) {
	src := benchComplex{
		Name:   "root",
		Nested: benchNested{Label: "n", Value: 1},
		Ptr:    &benchNested{Label: "p", Value: 2},
		Items:  []int{1, 2, 3},
		Tags:   []string{"a", "b", "c"},
		Meta:   map[string]string{"k": "v"},
		Data:   map[string]any{"x": 1, "y": []int{1, 2}},
	}

	b.Run("local", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			var dst benchComplex
			if err := localcopier.Copy(&dst, src); err != nil {
				b.Fatal(err)
			}
			sink = dst
		}
	})

	// jinzhu 默认浅拷贝：嵌套容器与 src 共享引用
	b.Run("jinzhu", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			var dst benchComplex
			if err := jinzhu.Copy(&dst, &src); err != nil {
				b.Fatal(err)
			}
			sink = dst
		}
	})

	b.Run("jinzhu-deep", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			var dst benchComplex
			if err := jinzhu.CopyWithOption(&dst, &src, jinzhu.Option{DeepCopy: true}); err != nil {
				b.Fatal(err)
			}
			sink = dst
		}
	})

	b.Run("tiendc", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			var dst benchComplex
			if err := deepcopy.Copy(&dst, &src); err != nil {
				b.Fatal(err)
			}
			sink = dst
		}
	})
}
