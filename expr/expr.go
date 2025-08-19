package expr

// 如为真返回参数一，否则返回参数二
func Ternary[T any](e bool, v1, v2 T) T {
	if e {
		return v1
	}
	return v2
}

func TernaryF[T any](condition bool, ifFunc func() T, elseFunc func() T) T {
	if condition {
		return ifFunc()
	}

	return elseFunc()
}

type ifElse[T any] struct {
	result T
	done   bool
}

func If[T any](condition bool, result T) *ifElse[T] {
	if condition {
		return &ifElse[T]{result, true}
	}

	var t T
	return &ifElse[T]{t, false}
}

func IfF[T any](condition bool, resultF func() T) *ifElse[T] {
	if condition {
		return &ifElse[T]{resultF(), true}
	}

	var t T
	return &ifElse[T]{t, false}
}

func (i *ifElse[T]) ElseIf(condition bool, result T) *ifElse[T] {
	if !i.done && condition {
		i.result = result
		i.done = true
	}

	return i
}

func (i *ifElse[T]) ElseIfF(condition bool, resultF func() T) *ifElse[T] {
	if !i.done && condition {
		i.result = resultF()
		i.done = true
	}

	return i
}

func (i *ifElse[T]) Else(result T) T {
	if i.done {
		return i.result
	}

	return result
}

func (i *ifElse[T]) ElseF(resultF func() T) T {
	if i.done {
		return i.result
	}

	return resultF()
}

type switchCase[T comparable, R any] struct {
	predicate T
	result    R
	done      bool
}

func Switch[T comparable, R any](predicate T) *switchCase[T, R] {
	var result R

	return &switchCase[T, R]{
		predicate,
		result,
		false,
	}
}

func SwitchF[T comparable, R any](predicate func() T) *switchCase[T, R] {
	return Switch[T, R](predicate())
}

func (s *switchCase[T, R]) Case(val T, result R) *switchCase[T, R] {
	if !s.done && s.predicate == val {
		s.result = result
		s.done = true
	}

	return s
}

func (s *switchCase[T, R]) CaseF(val T, cb func() R) *switchCase[T, R] {
	if !s.done && s.predicate == val {
		s.result = cb()
		s.done = true
	}

	return s
}

func (s *switchCase[T, R]) Default(result R) R {
	if !s.done {
		s.result = result
	}

	return s.result
}

func (s *switchCase[T, R]) DefaultF(cb func() R) R {
	if !s.done {
		s.result = cb()
	}

	return s.result
}
