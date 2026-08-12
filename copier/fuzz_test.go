package copier

// Fuzz 测试：fuzz 数据驱动 Copy / parseTag / TypeConvert，发现 panic 即崩溃（预期价值）。

import (
	"reflect"
	"testing"
)

// FuzzCopy 用 fuzz 数据构造 map[string]any / struct 变体后驱动 Copy，
// 覆盖 map→map、map→struct、struct→map（ToMap）三条主要路径。
func FuzzCopy(f *testing.F) {
	f.Add("hello")
	f.Add("name=John,age=30")
	f.Add("[]int{1,2,3}")
	f.Add("")
	f.Add("map,nested,slice,ptr")

	f.Fuzz(func(t *testing.T, data string) {
		src := map[string]any{
			"name": data,
			"age":  len(data),
			"meta": map[string]any{"k": data},
			"list": []any{data, len(data)},
		}

		// map → map
		var to1 map[string]any
		if err := Copy(&to1, src); err != nil {
			t.Fatal(err)
		}

		// map → struct
		var to2 struct {
			Name string
			Age  int
		}
		if err := Copy(&to2, src); err != nil {
			t.Fatal(err)
		}

		// struct → map
		if _, err := ToMap(to2); err != nil {
			t.Fatal(err)
		}
	})
}

// FuzzParseTag fuzz 任意字符串 → parseTag 不 panic。
func FuzzParseTag(f *testing.F) {
	f.Add("must,toname=xxx")
	f.Add("-")
	f.Add("ignore")
	f.Add("")
	f.Add("must,ignore,toname=")
	f.Add("toname=xxx,must")

	f.Fuzz(func(t *testing.T, tag string) {
		opt := parseTag(tag)
		_ = opt.Contains(tagRequired)
		_ = opt.Contains(tagSkip)
		_ = opt.ToName()
	})
}

// FuzzTypeConvert fuzz 任意字符串值经 opt.TypeConvert 调用不 panic，
// 覆盖字段名匹配/不匹配、DstType 精确匹配等边界。
func FuzzTypeConvert(f *testing.F) {
	f.Add("hello")
	f.Add("42")
	f.Add("")

	f.Fuzz(func(t *testing.T, value string) {
		opt := getOpt(WithConverters(TypeConverter{
			FieldName: "F",
			SrcType:   "",
			DstType:   "",
			Fn: func(src any) (any, error) {
				return src, nil
			},
		}))

		v := reflect.ValueOf(value)
		_, _ = opt.TypeConvert("F", v)
		_, _ = opt.TypeConvert("Other", v)
	})
}
