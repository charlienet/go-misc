package json

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStruct2Json(t *testing.T) {
	type Person struct {
		Name string
		Age  int
	}

	p := Person{Name: "John", Age: 30}
	result := Struct2Json(p)
	assert.Contains(t, result, "John")
	assert.Contains(t, result, "30")
}

func TestStruct2JsonIndent(t *testing.T) {
	type Person struct {
		Name string
		Age  int
	}

	p := Person{Name: "John", Age: 30}
	result := Struct2JsonIndent(p)
	assert.Contains(t, result, "\n")
	assert.Contains(t, result, "  ")
}

func TestMustStruct2Json(t *testing.T) {
	type Person struct {
		Name string
	}

	p := Person{Name: "John"}
	result := MustStruct2Json(p)
	assert.Contains(t, result, "John")
}

func TestMustStruct2JsonIndent(t *testing.T) {
	type Person struct {
		Name string
	}

	p := Person{Name: "John"}
	result := MustStruct2JsonIndent(p)
	assert.Contains(t, result, "\n")
}

func TestMustStruct2Json_Panic(t *testing.T) {
	// channel 不可序列化，应该 panic
	assert.Panics(t, func() {
		MustStruct2Json(make(chan int))
	})
}

func TestStruct2Json_Error(t *testing.T) {
	// channel 不可序列化，应该返回空字符串
	result := Struct2Json(make(chan int))
	assert.Equal(t, "", result)
}

func TestPascal2Camel_JSON(t *testing.T) {
	v := struct {
		UserName string
	}{UserName: "john"}

	result := Struct2Json(Pascal2Camel{v})
	assert.Contains(t, result, "userName")
}

func TestPascal2Snake_JSON(t *testing.T) {
	v := struct {
		UserName string
	}{UserName: "john"}

	result := Struct2Json(Pascal2Snake{v})
	// Pascal2Snake 将 PascalCase 转为 snake_case，但首字母大写保留
	assert.Contains(t, result, "User_Name")
}

func TestPascal2UpperSnake_JSON(t *testing.T) {
	v := struct {
		UserName string
	}{UserName: "john"}

	result := Struct2Json(Pascal2UpperSnake{v})
	assert.Contains(t, result, "USER_NAME")
}

func TestSnake2Camel_JSON(t *testing.T) {
	s := map[string]any{
		"user_name": "john",
	}

	result := Struct2Json(Snake2Camel{s})
	assert.Contains(t, result, "userName")
}

func TestSnake2Pascal_JSON(t *testing.T) {
	s := map[string]any{
		"user_name": "john",
	}

	result := Struct2Json(Snake2Pascal{s})
	assert.Contains(t, result, "UserName")
}

func TestCamel2Pascal_JSON(t *testing.T) {
	v := struct {
		UserName string
	}{UserName: "john"}

	result := Struct2Json(Camel2Pascal{v})
	assert.Contains(t, result, "UserName")
}

func TestNestedStruct_JSON(t *testing.T) {
	type Address struct {
		City string
	}
	type Person struct {
		Name    string
		Address Address
	}

	p := Person{Name: "John", Address: Address{City: "Beijing"}}
	result := Struct2Json(Pascal2Camel{p})
	assert.Contains(t, result, "name")
	assert.Contains(t, result, "address")
	assert.Contains(t, result, "city")
}

func TestSlice_JSON(t *testing.T) {
	v := struct {
		Items []string
	}{Items: []string{"a", "b", "c"}}

	result := Struct2Json(Pascal2Camel{v})
	assert.Contains(t, result, "items")
	assert.Contains(t, result, "\"a\"")
}

func TestPointer_JSON(t *testing.T) {
	type Person struct {
		Name *string
	}

	name := "John"
	p := Person{Name: &name}
	result := Struct2Json(Pascal2Camel{p})
	assert.Contains(t, result, "John")

	// nil 指针
	p2 := Person{Name: nil}
	result2 := Struct2Json(Pascal2Camel{p2})
	assert.Contains(t, result2, "null")
}

func TestMap_JSON(t *testing.T) {
	m := map[string]any{
		"user_name": "john",
		"age":       30,
	}

	result := Struct2Json(Snake2Camel{m})
	assert.Contains(t, result, "userName")
}

func TestRegisterFuzzyDecoders(t *testing.T) {
	// 不应该 panic
	assert.NotPanics(t, func() {
		RegisterFuzzyDecoders()
	})
}

func TestMarshalUnmarshal(t *testing.T) {
	type Person struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}

	p := Person{Name: "John", Age: 30}
	data, err := Marshal(p)
	assert.NoError(t, err)

	var p2 Person
	err = Unmarshal(data, &p2)
	assert.NoError(t, err)
	assert.Equal(t, p, p2)
}

func TestJsonTag_Override(t *testing.T) {
	type Person struct {
		Name string `json:"custom_name"`
	}

	p := Person{Name: "John"}
	result := Struct2Json(Pascal2Camel{p})
	// 有 json tag 时应该使用 tag 名
	assert.Contains(t, result, "custom_name")
}

func TestJsonTag_Ignore(t *testing.T) {
	type Person struct {
		Name   string
		Secret string `json:"-"`
	}

	p := Person{Name: "John", Secret: "hidden"}
	result := Struct2Json(p)
	assert.Contains(t, result, "John")
	assert.NotContains(t, result, "hidden")
}
