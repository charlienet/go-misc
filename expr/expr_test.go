package expr

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTernary(t *testing.T) {
	assert.Equal(t, "yes", Ternary(true, "yes", "no"))
	assert.Equal(t, "no", Ternary(false, "yes", "no"))
	assert.Equal(t, 1, Ternary(true, 1, 2))
	assert.Equal(t, 2, Ternary(false, 1, 2))
}

func TestTernaryF(t *testing.T) {
	called := false
	result := TernaryF(true, func() string {
		return "yes"
	}, func() string {
		called = true
		return "no"
	})
	assert.Equal(t, "yes", result)
	assert.False(t, called, "else func should not be called when condition is true")

	result = TernaryF(false, func() string {
		return "yes"
	}, func() string {
		return "no"
	})
	assert.Equal(t, "no", result)
}

func TestIfElse(t *testing.T) {
	// 测试 If -> Else
	result := If(true, "first").Else("second")
	assert.Equal(t, "first", result)

	result = If(false, "first").Else("second")
	assert.Equal(t, "second", result)
}

func TestIfElseIf(t *testing.T) {
	result := If(false, "first").
		ElseIf(true, "second").
		Else("third")
	assert.Equal(t, "second", result)

	result = If(false, "first").
		ElseIf(false, "second").
		Else("third")
	assert.Equal(t, "third", result)

	result = If(true, "first").
		ElseIf(true, "second").
		Else("third")
	assert.Equal(t, "first", result)
}

func TestIfF(t *testing.T) {
	called := false
	result := IfF(true, func() string {
		return "yes"
	}).Else("no")
	assert.Equal(t, "yes", result)

	result = IfF(false, func() string {
		called = true
		return "yes"
	}).Else("no")
	assert.Equal(t, "no", result)
	assert.False(t, called)
}

func TestElseIfF(t *testing.T) {
	called := false
	result := If(false, "first").
		ElseIfF(true, func() string {
			return "second"
		}).
		Else("third")
	assert.Equal(t, "second", result)

	result = If(false, "first").
		ElseIfF(false, func() string {
			called = true
			return "second"
		}).
		Else("third")
	assert.Equal(t, "third", result)
	assert.False(t, called)
}

func TestElseF(t *testing.T) {
	result := If(false, "first").ElseF(func() string {
		return "computed"
	})
	assert.Equal(t, "computed", result)
}

func TestSwitch(t *testing.T) {
	result := Switch[int, string](1).
		Case(1, "one").
		Case(2, "two").
		Default("other")
	assert.Equal(t, "one", result)

	result = Switch[int, string](2).
		Case(1, "one").
		Case(2, "two").
		Default("other")
	assert.Equal(t, "two", result)

	result = Switch[int, string](3).
		Case(1, "one").
		Case(2, "two").
		Default("other")
	assert.Equal(t, "other", result)
}

func TestSwitchF(t *testing.T) {
	called := false
	result := SwitchF[int, string](func() int { return 1 }).
		CaseF(1, func() string {
			return "one"
		}).
		CaseF(2, func() string {
			called = true
			return "two"
		}).
		Default("other")
	assert.Equal(t, "one", result)
	assert.False(t, called)
}

func TestSwitch_DefaultF(t *testing.T) {
	result := Switch[int, string](3).
		Case(1, "one").
		DefaultF(func() string {
			return "computed"
		})
	assert.Equal(t, "computed", result)
}

func TestSwitch_StringPredicate(t *testing.T) {
	result := Switch[string, int]("hello").
		Case("hello", 1).
		Case("world", 2).
		Default(0)
	assert.Equal(t, 1, result)
}
