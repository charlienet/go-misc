package copier

// Benchmark：copy 核心路径的性能与内存分配基准。

import (
	"testing"
	"time"
)

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

// BenchmarkCopyStruct 中型 struct（10 字段混合类型）struct→struct。
func BenchmarkCopyStruct(b *testing.B) {
	src := benchMedium{
		Name: "John", Age: 30, Height: 1.75, Active: true,
		Tags: []string{"a", "b"}, Meta: map[string]string{"k": "v"},
		City: "Beijing", Score: 100, Level: 3, Created: time.Now(),
	}
	var dst benchMedium

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if err := Copy(&dst, src); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkCopyMap map[string]any（标量为主）→ map[string]any。
func BenchmarkCopyMap(b *testing.B) {
	src := map[string]any{
		"name": "John", "age": 30, "height": 1.75, "active": true,
		"city": "Beijing", "score": int64(100),
	}

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		var dst map[string]any
		if err := Copy(&dst, src); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkCopyNestedMap 含嵌套 map/slice/pointer 的深拷贝（体现深拷贝成本）。
func BenchmarkCopyNestedMap(b *testing.B) {
	src := map[string]any{
		"meta": map[string]any{"a": 1, "b": 2},
		"list": []int{1, 2, 3},
		"ptr":  &benchMedium{Name: "x"},
	}

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		var dst map[string]any
		if err := Copy(&dst, src); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkCopyWithOptions 带 WithSkipFields + WithConverters 的组合选项。
func BenchmarkCopyWithOptions(b *testing.B) {
	src := map[string]any{
		"Name": "John", "Age": 30, "Score": int64(100), "Level": uint8(3),
	}

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		var dst benchMedium
		if err := Copy(&dst, src,
			WithSkipFields("Score"),
			WithConverters(TypeConverter{
				FieldName: "Name",
				SrcType:   "",
				DstType:   "",
				Fn: func(src any) (any, error) {
					return src, nil
				},
			}),
		); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkCopyStructToMap 中型 struct → map[string]any（默认选项，走 plan 缓存路径）。
func BenchmarkCopyStructToMap(b *testing.B) {
	src := benchMedium{
		Name: "John", Age: 30, Height: 1.75, Active: true,
		Tags: []string{"a", "b"}, Meta: map[string]string{"k": "v"},
		City: "Beijing", Score: 100, Level: 3, Created: time.Now(),
	}

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		var dst map[string]any
		if err := Copy(&dst, src); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkCopyMapToStruct map[string]any → 中型 struct（默认选项，走 plan 缓存路径）。
// 混入一个不匹配 key（"Unknown"），与真实场景一致（静默跳过）。
// 容器值（Tags/Meta）经 deepCopyInner 入口 interface 解包后深拷贝。
func BenchmarkCopyMapToStruct(b *testing.B) {
	src := map[string]any{
		"Name": "John", "Age": 30, "Height": 1.75, "Active": true,
		"Tags": []string{"a", "b"}, "Meta": map[string]string{"k": "v"},
		"City": "Beijing", "Score": int64(100), "Level": uint8(3),
		"Unknown": "x",
	}

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		var dst benchMedium
		if err := Copy(&dst, src); err != nil {
			b.Fatal(err)
		}
	}
}
