package copier

import (
	"errors"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

// planSrc：含普通字段、tag- 忽略字段、toname 重命名字段、无对应 dst 的字段
type planSrc struct {
	Name    string
	Age     int
	Address string
	Skip    string `copier:"-"`
	Renamed string `copier:"toname=Target"`
}

type planDst struct {
	Name   string
	Age    int
	Target string
}

// ============ planEligible 判定 ============

func TestPlanEligible(t *testing.T) {
	// 默认配置 → eligible
	assert.True(t, planEligible(getOpt()))
	assert.True(t, planEligible(DefaultOptions))

	// 非匹配类选项不参与判定（运行时值/闭包处理，不影响字段匹配 plan）
	assert.True(t, planEligible(getOpt(WithIgnoreEmpty())))
	assert.True(t, planEligible(getOpt(WithMaxDepth(3))))
	assert.True(t, planEligible(getOpt(WithMethodMapping())))
	assert.True(t, planEligible(getOpt(WithConverters(TypeConverter{FieldName: "Age"}))))
	assert.True(t, planEligible(getOpt(WithValueConverter(func(string, any) any { return nil }))))

	// 标量选项（caseSensitive/tagName/must）已进键，仍可走缓存
	assert.True(t, planEligible(getOpt(WithCaseSensitive())))
	assert.True(t, planEligible(getOpt(WithTagName("json"))))
	assert.True(t, planEligible(getOpt(WithMust())))

	// 内容型选项（第三步才做）非空 → 不 eligible
	assert.False(t, planEligible(getOpt(WithNameFn(func(s string) string { return s }))))
	assert.False(t, planEligible(getOpt(WithNameMapping(map[string]string{"a": "b"}))))
	assert.False(t, planEligible(getOpt(WithSkipFields("Name"))))
}

// ============ 缓存命中：同 (srcType, dstType) 复用同一 plan ============

func TestGetStructPlanCacheHit(t *testing.T) {
	srcT := reflect.TypeOf(planSrc{})
	dstT := reflect.TypeOf(planDst{})

	p1 := getStructPlan(srcT, dstT, DefaultOptions)
	p2 := getStructPlan(srcT, dstT, DefaultOptions)
	assert.NotNil(t, p1)
	assert.Same(t, p1, p2) // 第二次命中缓存，同一实例
}

// ============ 非 eligible：不走缓存 ============

func TestGetStructPlanNotEligible(t *testing.T) {
	srcT := reflect.TypeOf(planSrc{})
	dstT := reflect.TypeOf(planDst{})

	assert.Nil(t, getStructPlan(srcT, dstT, getOpt(WithNameFn(func(s string) string { return s }))))
	assert.Nil(t, getStructPlan(srcT, dstT, getOpt(WithNameMapping(map[string]string{"a": "b"}))))
	assert.Nil(t, getStructPlan(srcT, dstT, getOpt(WithSkipFields("Name"))))
}

// ============ 标量选项进键：不同 planOpts 产生不同缓存键 ============

func TestPlanOptsDistinctKeys(t *testing.T) {
	srcT := reflect.TypeOf(planSrc{})
	dstT := reflect.TypeOf(planDst{})

	pDefault := getStructPlan(srcT, dstT, getOpt())
	pCase := getStructPlan(srcT, dstT, getOpt(WithCaseSensitive()))
	pTag := getStructPlan(srcT, dstT, getOpt(WithTagName("json")))
	pMust := getStructPlan(srcT, dstT, getOpt(WithMust()))

	assert.NotSame(t, pDefault, pCase)
	assert.NotSame(t, pDefault, pTag)
	assert.NotSame(t, pDefault, pMust)

	// 相同 opts 命中同一缓存实例
	assert.Same(t, pCase, getStructPlan(srcT, dstT, getOpt(WithCaseSensitive())))
	assert.Same(t, pTag, getStructPlan(srcT, dstT, getOpt(WithTagName("json"))))
	assert.Same(t, pMust, getStructPlan(srcT, dstT, getOpt(WithMust())))
}

// ============ buildStructPlan 内容 ============

func TestBuildStructPlanContent(t *testing.T) {
	plan := buildStructPlan(reflect.TypeOf(planSrc{}), reflect.TypeOf(planDst{}), DefaultOptions)
	// planSrc 字段：Name(0) Age(1) Address(2) Skip(3, tag-) Renamed(4, toname=Target)
	// tag- 字段直接不进 plan → 共 4 条
	assert.Len(t, plan.fields, 4)

	assert.Equal(t, 0, plan.fields[0].srcIdx)
	assert.Equal(t, "Name", plan.fields[0].name)
	assert.Equal(t, []int{0}, plan.fields[0].dstIdx)

	assert.Equal(t, 1, plan.fields[1].srcIdx)
	assert.Equal(t, []int{1}, plan.fields[1].dstIdx)

	// Address 在 dst 无匹配 → dstIdx 为空
	assert.Equal(t, 2, plan.fields[2].srcIdx)
	assert.Equal(t, "Address", plan.fields[2].name)
	assert.Nil(t, plan.fields[2].dstIdx)

	// toname=Target → name 为转换后名称，dstIdx 指向 Target(2)
	assert.Equal(t, 4, plan.fields[3].srcIdx)
	assert.Equal(t, "Target", plan.fields[3].name)
	assert.Equal(t, []int{2}, plan.fields[3].dstIdx)
}

// ============ plan 匹配与运行时 getFieldByName 完全一致 ============

func TestPlanMatchesRuntimeMatch(t *testing.T) {
	srcT := reflect.TypeOf(planSrc{})
	dstT := reflect.TypeOf(planDst{})
	plan := buildStructPlan(srcT, dstT, DefaultOptions)

	// 用可寻址伪值走与运行时相同的 getFieldByName，比对 plan 的 dstIdx 判定
	v := reflect.New(dstT).Elem()
	for _, m := range plan.fields {
		runtime := getFieldByName(v, m.name, DefaultOptions)
		if len(m.dstIdx) == 0 {
			assert.False(t, runtime.IsValid(), "field %s should have no match", m.name)
		} else {
			assert.True(t, runtime.IsValid(), "field %s should match", m.name)
		}
	}
}

// ============ plan 路径与运行时行为等价（含各非匹配类选项） ============

type planNestedInner struct{ N int }

type planNestedSrc struct{ Inner planNestedInner }

type planNestedDst struct{ Inner planNestedInner }

func TestPlanEquivalence(t *testing.T) {
	src := planSrc{Name: "n", Age: 7, Address: "a", Renamed: "r"}

	t.Run("default options", func(t *testing.T) {
		var dst planDst
		err := Copy(&dst, src)
		assert.NoError(t, err)
		assert.Equal(t, planDst{Name: "n", Age: 7, Target: "r"}, dst)
	})

	t.Run("with ignoreEmpty skips zero fields", func(t *testing.T) {
		var dst planDst
		err := Copy(&dst, planSrc{Name: "n"}, WithIgnoreEmpty())
		assert.NoError(t, err)
		assert.Equal(t, planDst{Name: "n"}, dst)
	})

	t.Run("with maxDepth respects nested limit", func(t *testing.T) {
		// plan 路径 deepCopyInner 同样受深度限制约束
		var dst planNestedDst
		err := Copy(&dst, planNestedSrc{Inner: planNestedInner{N: 5}}, WithMaxDepth(0))
		assert.Error(t, err)
		assert.True(t, errors.Is(err, ErrMaxDepthExceeded))
	})

	t.Run("with converters behaves as existing struct-to-struct path", func(t *testing.T) {
		// 既有行为：struct→struct 不调用 TypeConvert，结果与默认一致
		fnCalled := false
		var dst planDst
		err := Copy(&dst, src, WithConverters(TypeConverter{
			FieldName: "Age",
			SrcType:   int(0),
			DstType:   int64(0),
			Fn: func(src any) (any, error) {
				fnCalled = true
				return int64(src.(int) * 10), nil
			},
		}))
		assert.NoError(t, err)
		assert.False(t, fnCalled)
		assert.Equal(t, planDst{Name: "n", Age: 7, Target: "r"}, dst)
	})

	t.Run("with valueConverter receives src original field names", func(t *testing.T) {
		called := map[string]bool{}
		var dst planDst
		err := Copy(&dst, src, WithValueConverter(func(name string, v any) any {
			called[name] = true
			return v
		}))
		assert.NoError(t, err)
		// toname 字段 Renamed 的回调使用 src 原始名，而非转换后名
		assert.True(t, called["Name"])
		assert.True(t, called["Age"])
		assert.True(t, called["Renamed"])
		// 既有行为：Address 在 dst 无匹配字段（dstIdx 空），回调不触发
		assert.False(t, called["Address"])
		assert.Equal(t, planDst{Name: "n", Age: 7, Target: "r"}, dst)
	})
}

// ============ 关键回归：WithMethodMapping + 默认字段配置（plan 路径 setter/getter 仍生效） ============

type planMMSrc struct{ Name string }

type planMMDst struct {
	Stored string
}

func (d *planMMDst) Name(v string) { d.Stored = v }

type planGettersSrc struct{}

func (s *planGettersSrc) Title() string { return "t" }

type planGettersDst struct{ Title string }

func TestPlanWithMethodMapping(t *testing.T) {
	t.Run("setter works on plan path", func(t *testing.T) {
		// planEligible(WithMethodMapping())==true → 走 plan 路径；
		// src.Name 在 dst 无字段匹配（dstIdx 空）→ callSetter 降级
		var dst planMMDst
		err := Copy(&dst, planMMSrc{Name: "x"}, WithMethodMapping())
		assert.NoError(t, err)
		assert.Equal(t, "x", dst.Stored)
	})

	t.Run("getter works on plan path", func(t *testing.T) {
		var dst planGettersDst
		err := Copy(&dst, &planGettersSrc{}, WithMethodMapping())
		assert.NoError(t, err)
		assert.Equal(t, "t", dst.Title)
	})
}

// ============ 非 eligible 选项走原路径且行为正确 ============

func TestPlanNonEligiblePaths(t *testing.T) {
	src := planSrc{Name: "n", Age: 7, Renamed: "r"}

	t.Run("case sensitive", func(t *testing.T) {
		var dst planDst
		err := Copy(&dst, src, WithCaseSensitive())
		assert.NoError(t, err)
		assert.Equal(t, planDst{Name: "n", Age: 7, Target: "r"}, dst)
	})

	t.Run("skip fields", func(t *testing.T) {
		var dst planDst
		err := Copy(&dst, src, WithSkipFields("Name"))
		assert.NoError(t, err)
		assert.Equal(t, planDst{Age: 7, Target: "r"}, dst)
	})

	t.Run("must", func(t *testing.T) {
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
		assert.Equal(t, mustDst{Name: "m"}, dst) // 仅 must 字段被拷贝
	})
}
