package copier

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// ============ 第二步：struct→struct 标量选项（进键后走 plan 路径） ============

type pSSCaseDst struct {
	name string // 未导出，声明在前
	Name string
}

func TestPlanStructCaseSensitive(t *testing.T) {
	type src struct{ Name string }

	t.Run("case-insensitive matches unexported first then skips", func(t *testing.T) {
		var dst pSSCaseDst
		err := Copy(&dst, src{Name: "x"})
		assert.NoError(t, err)
		// EqualFold 匹配第一个字段 name（未导出）→ CanSet false → 跳过
		assert.Equal(t, "", dst.Name)
	})

	t.Run("case-sensitive matches exported Name", func(t *testing.T) {
		var dst pSSCaseDst
		err := Copy(&dst, src{Name: "x"}, WithCaseSensitive())
		assert.NoError(t, err)
		assert.Equal(t, "x", dst.Name)
	})
}

func TestPlanStructMust(t *testing.T) {
	type mustSrc struct {
		Name string `copier:"must"`
		Age  int
	}
	type mustDst struct {
		Name string
		Age  int
	}
	var dst mustDst
	err := Copy(&dst, mustSrc{Name: "m", Age: 1}, WithMust())
	assert.NoError(t, err)
	assert.Equal(t, mustDst{Name: "m"}, dst)
}

func TestPlanStructTagName(t *testing.T) {
	type jsonSrc struct {
		Name string `json:"toname=json_name"`
	}
	type jsonDst struct {
		Json_Name string
	}
	var dst jsonDst
	err := Copy(&dst, jsonSrc{Name: "n"}, WithTagName("json"))
	assert.NoError(t, err)
	assert.Equal(t, "n", dst.Json_Name)
}

// ============ 第四步：struct→map ============

type pStoMInner struct{ N int }

type pStoMEmbed struct{ Emb string }

type pStoMSrc struct {
	Name    string
	Renamed string `copier:"toname=target"`
	Nested  pStoMInner
	pStoMEmbed
	Skip string `copier:"-"`
	Must string `copier:"must"`
}

type pStoMJsonSrc struct {
	Name string `json:"toname=json_name"`
	Age  int    `json:"toname=json_age"`
}

func TestBuildStructToMapPlanContent(t *testing.T) {
	plan := buildStructToMapPlan(reflect.TypeOf(pStoMSrc{}), getOpt())
	// 字段序：Name(0) Renamed(1) Nested(2) pStoMEmbed(3, 匿名 struct→expand) Skip(4, tag-) Must(5)
	// tag- 的 Skip 不进 plan
	assert.Len(t, plan.entries, 5)

	assert.False(t, plan.entries[0].expand)
	assert.Equal(t, "Name", plan.entries[0].name)

	assert.Equal(t, "target", plan.entries[1].name) // toname 转换

	assert.False(t, plan.entries[2].expand)

	assert.True(t, plan.entries[3].expand) // 匿名 struct 字段

	assert.Equal(t, "Must", plan.entries[4].name)
}

func TestGetStructToMapPlanCacheHit(t *testing.T) {
	srcT := reflect.TypeOf(pStoMSrc{})
	assert.Same(t, getStructToMapPlan(srcT, getOpt()), getStructToMapPlan(srcT, getOpt()))
	// 标量选项进键：不同 opts 不同 plan
	assert.NotSame(t, getStructToMapPlan(srcT, getOpt()), getStructToMapPlan(srcT, getOpt(WithMust())))
}

func TestPlanStructToMap(t *testing.T) {
	src := pStoMSrc{
		Name:       "n",
		Renamed:    "r",
		Nested:     pStoMInner{N: 5},
		pStoMEmbed: pStoMEmbed{Emb: "e"},
		Skip:       "s",
		Must:       "m",
	}

	t.Run("default", func(t *testing.T) {
		var dst map[string]any
		err := Copy(&dst, src)
		assert.NoError(t, err)
		assert.Equal(t, "n", dst["Name"])
		assert.Equal(t, "r", dst["target"])                    // toname
		assert.Equal(t, map[string]any{"N": 5}, dst["Nested"]) // 嵌套 struct 展开
		assert.Equal(t, "e", dst["Emb"])                       // 匿名 struct expand
		_, hasSkip := dst["Skip"]
		assert.False(t, hasSkip)          // tag- 过滤
		assert.Equal(t, "m", dst["Must"]) // 默认不过滤 must 字段
	})

	t.Run("with must only must fields", func(t *testing.T) {
		var dst map[string]any
		err := Copy(&dst, src, WithMust())
		assert.NoError(t, err)
		_, hasName := dst["Name"]
		assert.False(t, hasName)
		assert.Equal(t, "m", dst["Must"])
	})

	t.Run("with tagName json", func(t *testing.T) {
		var dst map[string]any
		err := Copy(&dst, pStoMJsonSrc{Name: "n", Age: 1}, WithTagName("json"))
		assert.NoError(t, err)
		assert.Equal(t, "n", dst["json_name"])
		assert.Equal(t, 1, dst["json_age"])
	})

	t.Run("with ignoreEmpty skips zero fields", func(t *testing.T) {
		var dst map[string]any
		err := Copy(&dst, pStoMSrc{Name: "n"}, WithIgnoreEmpty())
		assert.NoError(t, err)
		_, hasRenamed := dst["target"]
		assert.False(t, hasRenamed) // 零值字段跳过
		assert.Equal(t, "n", dst["Name"])
	})

	t.Run("with TypeConvert applied", func(t *testing.T) {
		var dst map[string]any
		err := Copy(&dst, pStoMSrc{Name: "abc"}, WithConverters(TypeConverter{
			FieldName: "Name",
			SrcType:   string(""),
			DstType:   int(0),
			Fn: func(src any) (any, error) {
				return len(src.(string)), nil
			},
		}))
		assert.NoError(t, err)
		assert.Equal(t, 3, dst["Name"]) // 转换生效（converted=true 不再展开 struct）
	})

	t.Run("with valueConverter receives src original names", func(t *testing.T) {
		called := map[string]bool{}
		var dst map[string]any
		err := Copy(&dst, pStoMSrc{Name: "n", Renamed: "r"}, WithValueConverter(func(name string, v any) any {
			called[name] = true
			return v
		}))
		assert.NoError(t, err)
		// 回调使用 src 原始字段名，而非 toname 转换后名
		assert.True(t, called["Renamed"])
		assert.True(t, called["Name"])
	})
}

// ============ 第四步：map→struct ============

type pMtoSBase struct{ BaseField string }

type pMtoSDst struct {
	Name string
	Age  int
	pMtoSBase
}

type pMtoSCaseDst struct {
	name string // 未导出，声明在前
	Name string
}

func TestBuildMapToStructPlanContent(t *testing.T) {
	plan := buildMapToStructPlan(reflect.TypeOf(pMtoSDst{}), getOpt())
	assert.Equal(t, []int{0}, plan.lookup["name"])
	assert.Equal(t, []int{1}, plan.lookup["age"])
	assert.Equal(t, []int{2, 0}, plan.lookup["basefield"]) // 嵌入提升字段链

	// 大小写敏感时键为原名
	casePlan := buildMapToStructPlan(reflect.TypeOf(pMtoSCaseDst{}), getOpt(WithCaseSensitive()))
	assert.Equal(t, []int{0}, casePlan.lookup["name"])
	assert.Equal(t, []int{1}, casePlan.lookup["Name"])
}

func TestGetMapToStructPlanCacheHit(t *testing.T) {
	dstT := reflect.TypeOf(pMtoSDst{})
	assert.Same(t, getMapToStructPlan(dstT, getOpt()), getMapToStructPlan(dstT, getOpt()))
	assert.NotSame(t, getMapToStructPlan(dstT, getOpt()), getMapToStructPlan(dstT, getOpt(WithCaseSensitive())))
}

func TestPlanMapToStruct(t *testing.T) {
	t.Run("default match with embedded promotion", func(t *testing.T) {
		src := map[string]any{"Name": "n", "Age": 7, "basefield": "b"}
		var dst pMtoSDst
		err := Copy(&dst, src)
		assert.NoError(t, err)
		assert.Equal(t, "n", dst.Name)
		assert.Equal(t, 7, dst.Age)
		assert.Equal(t, "b", dst.BaseField) // 嵌入提升字段
	})

	t.Run("case-insensitive keys", func(t *testing.T) {
		src := map[string]any{"NAME": "n", "AGE": 7}
		var dst pMtoSDst
		err := Copy(&dst, src)
		assert.NoError(t, err)
		assert.Equal(t, "n", dst.Name)
		assert.Equal(t, 7, dst.Age)
	})

	t.Run("non-string key returns ErrMapKeyNotMatch", func(t *testing.T) {
		src := map[int]any{1: "x"}
		var dst pMtoSDst
		err := Copy(&dst, src)
		assert.Error(t, err)
		assert.True(t, errors.Is(err, ErrMapKeyNotMatch))
	})

	t.Run("with valueConverter receives converted name", func(t *testing.T) {
		called := map[string]bool{}
		var dst pMtoSDst
		err := Copy(&dst, map[string]any{"Name": "x", "Age": 7}, WithValueConverter(func(name string, v any) any {
			called[name] = true
			return v
		}))
		assert.NoError(t, err)
		assert.True(t, called["Name"])
		assert.True(t, called["Age"])
	})

	t.Run("with ignoreEmpty keeps initial value", func(t *testing.T) {
		dst := pMtoSDst{pMtoSBase: pMtoSBase{BaseField: "keep"}}
		err := Copy(&dst, map[string]any{"basefield": ""}, WithIgnoreEmpty())
		assert.NoError(t, err)
		assert.Equal(t, "keep", dst.BaseField) // 零值跳过，保留初始值
	})
}

func TestPlanMapToStructCaseSensitive(t *testing.T) {
	t.Run("case-sensitive matches exported Name", func(t *testing.T) {
		var dst pMtoSCaseDst
		err := Copy(&dst, map[string]any{"Name": "x"}, WithCaseSensitive())
		assert.NoError(t, err)
		assert.Equal(t, "x", dst.Name)
	})

	t.Run("case-insensitive matches unexported first then skips", func(t *testing.T) {
		var dst pMtoSCaseDst
		err := Copy(&dst, map[string]any{"Name": "x"})
		assert.NoError(t, err)
		assert.Equal(t, "", dst.Name)
	})
}

func TestPlanMapToStructNotEligible(t *testing.T) {
	dstT := reflect.TypeOf(pMtoSDst{})

	assert.Nil(t, getMapToStructPlan(dstT, getOpt(WithNameFn(func(s string) string { return s }))))
	assert.Nil(t, getMapToStructPlan(dstT, getOpt(WithNameMapping(map[string]string{"a": "b"}))))
	assert.Nil(t, getMapToStructPlan(dstT, getOpt(WithSkipFields("Name"))))

	// 非 eligible 时走原路径，行为不变（NameFn 作用于 map key）
	var dst pMtoSDst
	err := Copy(&dst, map[string]any{"name": "x"}, WithNameFn(strings.ToUpper))
	assert.NoError(t, err)
	assert.Equal(t, "x", dst.Name)
}
