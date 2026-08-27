/*
Package expr provides utilities for conditional expressions and chainable control structures,
including ternary operators and fluent If/Else/Switch constructs with generic support.

This package enables more expressive conditional logic in Go through functional-style APIs
that support method chaining, making complex conditional statements more readable and maintainable.

Exported Functions:
  - Ternary[T any](bool, T, T) T: Evaluates condition and returns first value if true, second if false
  - TernaryF[T any](bool, func() T, func() T) T: Lazy evaluation version of Ternary with function parameters
  - If[T any](bool, T) *ifElse[T]: Starts a fluent If/Else chain
  - IfF[T any](bool, func() T) *ifElse[T]: Lazy evaluation version of If with function parameter
  - Switch[T comparable, R any](T) *switchCase[T, R]: Starts a fluent Switch/Case chain
  - SwitchF[T comparable, R any](func() T) *switchCase[T, R]: Lazy evaluation version of Switch with function parameter

Fluent API Methods:
  - (*ifElse[T]).ElseIf(bool, T) *ifElse[T]: Adds an additional condition to If/Else chain
  - (*ifElse[T]).ElseIfF(bool, func() T) *ifElse[T]: Lazy evaluation version of ElseIf
  - (*ifElse[T]).Else(T) T: Ends If/Else chain with final else clause
  - (*ifElse[T]).ElseF(func() T) T: Lazy evaluation version of Else
  - (*switchCase[T, R]).Case(T, R) *switchCase[T, R]: Adds a case to Switch/Case chain
  - (*switchCase[T, R]).CaseF(T, func() R) *switchCase[T, R]: Lazy evaluation version of Case with function
  - (*switchCase[T, R]).Default(R) R: Ends Switch/Case chain with default clause
  - (*switchCase[T, R]).DefaultF(func() R) R: Lazy evaluation version of Default with function

Examples:
	// Simple ternary expression:
	// result := expr.Ternary(true, "yes", "no") // returns "yes"

	// Lazy evaluation ternary (only executes relevant function):
	// expensiveCalc := func() int { return 42 }
	// simpleValue := 0
	// result := expr.TernaryF(false, expensiveCalc, func() int { return simpleValue })

	// Fluent If/Else chain:
	// status := expr.If(score >= 90, "A").
	//     ElseIf(score >= 80, "B").
	//     ElseIf(score >= 70, "C").
	//     Else("F")

	// Fluent Switch/Case chain:
	// category := expr.Switch(status).
	//     Case("A", "Excellent").
	//     Case("B", "Good").
	//     Case("C", "Average").
	//     Default("Needs Improvement")

	// Using function callbacks for lazy evaluation:
	// result := expr.If(condition, func() string {
	//     return expensiveCalculation()
	// }).Else(func() string {
	//     return defaultValue()
	// }).Else(defaultValue2)

Generic Support:
  - All functions support Go generics, allowing type-safe operations on any type
  - Switch/Case supports comparable types as predicates and any type as results
  - Method chaining maintains type safety throughout the chain
*/
package expr