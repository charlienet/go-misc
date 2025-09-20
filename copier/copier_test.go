package copier

import (
	"reflect"
	"testing"
	"time"

	"github.com/charlienet/go-misc/json"
	"github.com/stretchr/testify/assert"
)

type Person struct {
	Age     int     ``
	Name    string  ``
	Address Address ``
}

type Address struct {
	City   string
	Street string
}

type testCasse struct {
	actual any
	got    any
	err    error
}

func TestNameConverter(t *testing.T) {
	m1 := map[string]any{
		"Name": "Json",
	}

	m2 := map[string]any{}

	if err := Copy(&m2, m1, WithNameFn(func(s string) string {
		if s == "Name" {
			return "ssssssssssssssss"
		}

		return s
	})); err != nil {
		t.Fatal(err)
	}

	t.Log(json.Struct2JsonIndent(m2))
}

func TestCopy(t *testing.T) {
	src := Person{
		Name: "Json",
		Age:  10,
	}

	var dst Person
	assert.NoError(t, Copy(&dst, src))
	assert.Equal(t, src, dst)
}

func TestStructToStruct(t *testing.T) {
	for _, tc := range []testCasse{
		func() testCasse {
			type dst struct {
				id   int
				name string
			}
			type src dst

			d := dst{}
			s := src{id: 3, name: "name"}
			err := Copy(&d, s)
			return testCasse{err: err}
		}(),
		// func() testCasse {
		// 	type core struct {
		// 		ID   int
		// 		Name string
		// 	}

		// 	type dst struct {
		// 		id   int
		// 		name string
		// 		core
		// 	}

		// 	type src dst

		// 	var d dst
		// 	s := src{core: core{ID: 3, Name: "name"}}
		// 	err := Copy(&d, &s)
		// 	return testCasse{got: d, actual: s, err: err}
		// }(),
	} {
		assert.Nil(t, tc.err)
		assert.Equal(t, tc.got, tc.actual)
	}

	var dst Person
	var src = Person{
		Name: "test",
		Age:  10,
	}

	if err := Copy(&dst, src); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, src, dst)
}

func TestSrcDoublePtr(t *testing.T) {
	m1 := Person{
		Name: "Json",
	}

	m1Ptr := &m1
	var m2 Person

	if err := Copy(&m2, &m1Ptr); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, m1, m2)
}

func TestConvertibleTo(t *testing.T) {
	type p1 struct {
		Age int
	}

	type p2 struct {
		Age int64
	}

	src := p1{Age: 10}
	dst := p2{}

	if err := Copy(&dst, src); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, int64(10), dst.Age)
}

func TestSlice(t *testing.T) {
	for _, tc := range []testCasse{
		func() testCasse {
			src := []int{1, 2, 3}
			var dst []int

			err := Copy(&dst, src)
			return testCasse{got: dst, actual: src, err: err}
		}(),
		func() testCasse {
			type dst struct {
				A [3]int
			}

			type src struct {
				A []int
			}

			var d dst
			s := src{A: []int{1, 2, 3, 4, 5}}

			var need dst
			need.A = [3]int{1, 2, 3}

			err := Copy(&d, s)
			return testCasse{got: d, actual: need, err: err}
		}(),
	} {
		assert.NoError(t, tc.err)
		assert.Equal(t, tc.got, tc.actual)

		t.Log(json.Struct2JsonIndent(tc.got))
	}
}

func TestTime(t *testing.T) {
	type testCaseWithTime1 struct {
		T1 time.Time
		I  int
	}

	type testCaseWithTime2 struct {
		T1 time.Time
		I  int
	}

	var t1 testCaseWithTime1
	t2 := testCaseWithTime2{T1: time.Now(), I: 3}

	err := Copy(&t1, &t2)
	assert.NoError(t, err)
	assert.Equal(t, t1.T1, t2.T1)
	assert.Equal(t, t1.I, t2.I)
}

func TestAnonymnousFields(t *testing.T) {
	t.Run("Should work with unexported ptr fields", func(t *testing.T) {
		type nested struct {
			A string
		}
		type parentA struct {
			*nested
		}
		type parentB struct {
			*nested
		}

		from := parentA{nested: &nested{A: "a"}}
		to := parentB{}

		err := Copy(&to, &from)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
			return
		}

		from.nested.A = "b"

		if to.nested != nil {
			t.Errorf("should be nil")
		}
	})
}

func TestConvert(t *testing.T) {
	str := "Json"
	var ptr = &str

	src := reflect.ValueOf(ptr).Elem()

	var dst string

	cv := src.Convert(reflect.ValueOf(dst).Type())
	println(cv.Kind(), cv.Interface().(string))

	dstType := reflect.ValueOf(&dst)
	if dstType.CanSet() {
		dstType.Set(cv)
	}

	t.Log(dst)
}
