package copier

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// 本文件覆盖 deepCopyInner 入口统一解包 interface 的修复：
// map[string]any 等场景的容器值以 interface 承载，复制到 struct 容器字段时
// 必须解包后才能按具体类型深拷贝（修复前 Slice/Struct 分支对 interface Value
// 调用 Len/NumField 导致 panic）。

type ifcInner struct{ N int }

type ifcContainerDst struct {
	Tags []string
	Meta map[string]string
	Ptr  *ifcInner
	Deep map[string]any
}

// ============ 1. map→struct：容器值深拷贝成功且隔离 ============

func TestIfaceContainerMapToStruct(t *testing.T) {
	src := map[string]any{
		"Tags": []string{"a", "b"},
		"Meta": map[string]string{"k": "v"},
		"Ptr":  &ifcInner{N: 1},
		"Deep": map[string]any{"x": 1, "y": []int{1, 2}},
	}

	var dst ifcContainerDst
	err := Copy(&dst, src)
	assert.NoError(t, err)
	assert.Equal(t, []string{"a", "b"}, dst.Tags)
	assert.Equal(t, map[string]string{"k": "v"}, dst.Meta)
	assert.Equal(t, 1, dst.Ptr.N)
	assert.Equal(t, map[string]any{"x": 1, "y": []int{1, 2}}, dst.Deep)

	// 深拷贝隔离：修改 dst 副本不污染 src
	dst.Tags[0] = "zzz"
	dst.Meta["k"] = "zzz"
	dst.Ptr.N = 99
	dst.Deep["x"] = 99
	assert.Equal(t, "a", src["Tags"].([]string)[0])
	assert.Equal(t, "v", src["Meta"].(map[string]string)["k"])
	assert.Equal(t, 1, src["Ptr"].(*ifcInner).N)
	assert.Equal(t, 1, src["Deep"].(map[string]any)["x"])
}

// ============ 2. map→struct：nil interface 值 → 字段置零，不 panic 不报错 ============

type ifcNilDst struct {
	Inner ifcInner
	Name  string
}

func TestIfaceNilValueZeroed(t *testing.T) {
	dst := ifcNilDst{Inner: ifcInner{N: 5}, Name: "keep"}
	src := map[string]any{"Inner": nil, "Name": nil}

	err := Copy(&dst, src)
	assert.NoError(t, err)
	// nil interface 值：dst 可写时置零（修复前嵌套 struct 字段返回 ErrNotSupported）
	assert.Equal(t, ifcInner{}, dst.Inner)
	assert.Equal(t, "", dst.Name)
}

// ============ 3. struct→struct：src interface 字段承载 []string → dst 具体 slice 字段 ============

type ifcStructSrc struct{ Tags any }

type ifcStructDst struct{ Tags []string }

func TestIfaceStructToStructContainer(t *testing.T) {
	src := ifcStructSrc{Tags: []string{"a", "b"}}

	var dst ifcStructDst
	err := Copy(&dst, src)
	assert.NoError(t, err)
	assert.Equal(t, []string{"a", "b"}, dst.Tags)

	// 深拷贝隔离
	dst.Tags[0] = "zzz"
	assert.Equal(t, "a", src.Tags.([]string)[0])
}

// ============ 4. 嵌套 interface（interface 内套 interface 承载容器） ============

func TestIfaceNestedInterface(t *testing.T) {
	var v any = []string{"x", "y"}
	var outer any = v // 动态类型为 interface{}（含 []string）

	src := map[string]any{"Tags": outer}
	var dst ifcContainerDst
	err := Copy(&dst, src)
	assert.NoError(t, err)
	assert.Equal(t, []string{"x", "y"}, dst.Tags)
}

// ============ 5. map→map 回归：容器值深拷贝仍正常（内部有独立解包，确认无双重影响） ============

func TestIfaceMapToMapRegression(t *testing.T) {
	src := map[string]any{
		"a": map[string]any{"b": []int{1, 2}},
		"c": []string{"x"},
	}

	var dst map[string]any
	err := Copy(&dst, src)
	assert.NoError(t, err)
	assert.Equal(t, map[string]any{"b": []int{1, 2}}, dst["a"])
	assert.Equal(t, []string{"x"}, dst["c"])

	// 深拷贝隔离
	dst["a"].(map[string]any)["b"].([]int)[0] = 99
	dst["c"].([]string)[0] = "zzz"
	assert.Equal(t, 1, src["a"].(map[string]any)["b"].([]int)[0])
	assert.Equal(t, "x", src["c"].([]string)[0])
}

// ============ 6. 标量不变：map→struct 标量值仍正常复制（含 strconv 转换） ============

type ifcScalarDst struct {
	Age  int
	Num  string
	Flag bool
}

func TestIfaceScalarConversions(t *testing.T) {
	src := map[string]any{
		"Age":  "42",   // string → int（strconv）
		"Num":  100,    // int → string
		"Flag": "true", // string → bool
	}

	var dst ifcScalarDst
	err := Copy(&dst, src)
	assert.NoError(t, err)
	assert.Equal(t, 42, dst.Age)
	assert.Equal(t, "100", dst.Num)
	assert.Equal(t, true, dst.Flag)
}
