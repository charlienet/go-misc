package copier

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

// ============ getter：无 error 单返回 ============

type mmGetterSrc struct{}

func (s *mmGetterSrc) Name() string { return "hello" }

type mmGetterDst struct{ Name string }

func TestMethodMappingGetterBasic(t *testing.T) {
	var dst mmGetterDst
	err := Copy(&dst, &mmGetterSrc{}, WithMethodMapping())
	assert.NoError(t, err)
	assert.Equal(t, "hello", dst.Name)
}

// ============ getter：返回 (值, error) ============

type mmGetterErrSrc struct{}

func (s *mmGetterErrSrc) Data() (string, error) { return "ok", nil }

type mmGetterErrDst struct{ Data string }

type mmGetterFailSrc struct{}

func (s *mmGetterFailSrc) Data() (string, error) { return "", errors.New("boom") }

func TestMethodMappingGetterWithError(t *testing.T) {
	t.Run("nil error writes value", func(t *testing.T) {
		var dst mmGetterErrDst
		err := Copy(&dst, &mmGetterErrSrc{}, WithMethodMapping())
		assert.NoError(t, err)
		assert.Equal(t, "ok", dst.Data)
	})

	t.Run("non-nil error returns ErrMethodReturnError", func(t *testing.T) {
		var dst mmGetterErrDst
		err := Copy(&dst, &mmGetterFailSrc{}, WithMethodMapping())
		assert.Error(t, err)
		assert.True(t, errors.Is(err, ErrMethodReturnError))
	})
}

// ============ getter：返回值经 deepCopyInner 继承类型转换 ============

type mmConvSrc struct{}

func (s *mmConvSrc) Count() int { return 42 }

type mmConvDst struct{ Count string }

func TestMethodMappingGetterTypeConversion(t *testing.T) {
	var dst mmConvDst
	err := Copy(&dst, &mmConvSrc{}, WithMethodMapping())
	assert.NoError(t, err)
	assert.Equal(t, "42", dst.Count)
}

// ============ setter：无返回 / 返回 error 两种签名 ============

type mmSetterSrc struct{ Name string }

type mmSetterDst struct {
	Stored string
}

// setter 方法名 == 目标字段名（dst 无 Name 字段，与方法名互斥规则不冲突）
func (d *mmSetterDst) Name(v string) { d.Stored = v }

type mmSetterErrDst struct {
	Stored string
}

func (d *mmSetterErrDst) Name(v string) error {
	d.Stored = v
	return errors.New("boom")
}

func TestMethodMappingSetter(t *testing.T) {
	t.Run("setter without return", func(t *testing.T) {
		var dst mmSetterDst
		err := Copy(&dst, mmSetterSrc{Name: "x"}, WithMethodMapping())
		assert.NoError(t, err)
		assert.Equal(t, "x", dst.Stored)
	})

	t.Run("setter returning error", func(t *testing.T) {
		var dst mmSetterErrDst
		err := Copy(&dst, mmSetterSrc{Name: "x"}, WithMethodMapping())
		assert.Error(t, err)
		assert.True(t, errors.Is(err, ErrMethodReturnError))
	})
}

// ============ setter：参数类型不匹配静默跳过 ============

type mmSetterMismatchSrc struct{ Count int }

type mmSetterMismatchDst struct {
	Stored int
}

func (d *mmSetterMismatchDst) Count(v string) { d.Stored = len(v) }

func TestMethodMappingSetterMismatchSilentlySkipped(t *testing.T) {
	var dst mmSetterMismatchDst
	// src.Count 为 int，SetCount 接受 string：签名不符，静默跳过，不报错不误用
	err := Copy(&dst, mmSetterMismatchSrc{Count: 5}, WithMethodMapping())
	assert.NoError(t, err)
	assert.Equal(t, 0, dst.Stored)
}

// ============ 默认（不传 WithMethodMapping）方法绝不被调用 ============

type mmDefaultSrc struct {
	Called *bool
}

func (s *mmDefaultSrc) Name() string {
	*s.Called = true
	return "x"
}

type mmDefaultDst struct{ Name string }

func TestMethodMappingOffByDefault(t *testing.T) {
	t.Run("getter never called without option", func(t *testing.T) {
		called := false
		src := &mmDefaultSrc{Called: &called}
		var dst mmDefaultDst
		err := Copy(&dst, src)
		assert.NoError(t, err)
		assert.False(t, called)
		assert.Equal(t, "", dst.Name)
	})

	t.Run("getter called with option", func(t *testing.T) {
		called := false
		src := &mmDefaultSrc{Called: &called}
		var dst mmDefaultDst
		err := Copy(&dst, src, WithMethodMapping())
		assert.NoError(t, err)
		assert.True(t, called)
		assert.Equal(t, "x", dst.Name)
	})
}

// ============ 与 toname tag 组合（setter 用转换后名称匹配） ============

type mmTonameSrc struct {
	Name string `copier:"toname=SetName"`
}

type mmTonameDst struct {
	Stored string
}

func (d *mmTonameDst) SetName(v string) { d.Stored = v }

func TestMethodMappingWithToname(t *testing.T) {
	var dst mmTonameDst
	err := Copy(&dst, mmTonameSrc{Name: "toname"}, WithMethodMapping())
	assert.NoError(t, err)
	assert.Equal(t, "toname", dst.Stored)
}

// ============ 与 WithIgnoreEmpty 组合 ============

type mmEmptySrc struct{ Name string }

type mmEmptyDst struct {
	Stored string
}

func (d *mmEmptyDst) Name(v string) { d.Stored = v }

type mmEmptyGetterSrc struct{}

func (s *mmEmptyGetterSrc) Name() string { return "" }

type mmEmptyGetterDst struct{ Name string }

func TestMethodMappingWithIgnoreEmpty(t *testing.T) {
	t.Run("setter not called when src field zero", func(t *testing.T) {
		// src.Name 为零值：调用前判空，setter 不触发，保留初始值
		dst := mmEmptyDst{Stored: "keep"}
		err := Copy(&dst, mmEmptySrc{}, WithMethodMapping(), WithIgnoreEmpty())
		assert.NoError(t, err)
		assert.Equal(t, "keep", dst.Stored)
	})

	t.Run("setter called when src field non-zero", func(t *testing.T) {
		dst := mmEmptyDst{Stored: "keep"}
		err := Copy(&dst, mmEmptySrc{Name: "v"}, WithMethodMapping(), WithIgnoreEmpty())
		assert.NoError(t, err)
		assert.Equal(t, "v", dst.Stored)
	})

	t.Run("getter zero return skipped", func(t *testing.T) {
		// getter 返回零值：调用后判空，跳过写入，保留初始值
		dst := mmEmptyGetterDst{Name: "keep"}
		err := Copy(&dst, &mmEmptyGetterSrc{}, WithMethodMapping(), WithIgnoreEmpty())
		assert.NoError(t, err)
		assert.Equal(t, "keep", dst.Name)
	})

	t.Run("getter zero return without IgnoreEmpty overwrites", func(t *testing.T) {
		dst := mmEmptyGetterDst{Name: "keep"}
		err := Copy(&dst, &mmEmptyGetterSrc{}, WithMethodMapping())
		assert.NoError(t, err)
		assert.Equal(t, "", dst.Name)
	})
}

// ============ 嵌套 struct：方法映射与深度限制不冲突 ============

type mmNestedInner struct{ N int }

type mmNestedSrc struct{}

func (s *mmNestedSrc) Inner() mmNestedInner { return mmNestedInner{N: 42} }

type mmNestedDst struct{ Inner mmNestedInner }

func TestMethodMappingNestedWithMaxDepth(t *testing.T) {
	t.Run("nested getter result copied", func(t *testing.T) {
		var dst mmNestedDst
		err := Copy(&dst, &mmNestedSrc{}, WithMethodMapping())
		assert.NoError(t, err)
		assert.Equal(t, 42, dst.Inner.N)
	})

	t.Run("nested getter result respects max depth", func(t *testing.T) {
		// getter 返回值经 deepCopyInner(depth+1) 写入，同样受 WithMaxDepth 约束
		var dst mmNestedDst
		err := Copy(&dst, &mmNestedSrc{}, WithMethodMapping(), WithMaxDepth(0))
		assert.Error(t, err)
		assert.True(t, errors.Is(err, ErrMaxDepthExceeded))
	})
}

// ============ 优先级：字段直接匹配优先于 setter 方法 ============

type mmPrioritySrc struct{ Name string }

type mmPriorityDst struct {
	Name   string
	Stored string
}

func (d *mmPriorityDst) SetName(v string) { d.Stored = "via-setter" }

func TestMethodMappingFieldBeatsSetter(t *testing.T) {
	// dst 有同名字段 Name：字段直接匹配，setter 不触发
	var dst mmPriorityDst
	err := Copy(&dst, mmPrioritySrc{Name: "direct"}, WithMethodMapping())
	assert.NoError(t, err)
	assert.Equal(t, "direct", dst.Name)
	assert.Equal(t, "", dst.Stored)
}
