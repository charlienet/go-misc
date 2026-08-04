package copier

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWithSkipFields(t *testing.T) {
	type Person struct {
		Name string
		Age  int
		Sex  string
	}

	src := Person{Name: "John", Age: 30, Sex: "Male"}
	var dst Person

	err := Copy(&dst, src, WithSkipFields("Sex"))
	assert.NoError(t, err)
	assert.Equal(t, "John", dst.Name)
	assert.Equal(t, 30, dst.Age)
	assert.Equal(t, "", dst.Sex) // Sex 应该被跳过
}

func TestWithSkipFields_Multiple(t *testing.T) {
	type Person struct {
		Name    string
		Age     int
		Sex     string
		Address string
	}

	src := Person{Name: "John", Age: 30, Sex: "Male", Address: "Beijing"}
	var dst Person

	err := Copy(&dst, src, WithSkipFields("Sex", "Address"))
	assert.NoError(t, err)
	assert.Equal(t, "John", dst.Name)
	assert.Equal(t, 30, dst.Age)
	assert.Equal(t, "", dst.Sex)
	assert.Equal(t, "", dst.Address)
}

func TestWithValueConverter(t *testing.T) {
	type Person struct {
		Name string
		Age  int
	}

	src := Person{Name: "john", Age: 30}
	var dst Person

	// 将名字转换为大写
	err := Copy(&dst, src, WithValueConverter(func(fieldName string, value any) any {
		if fieldName == "Name" {
			if s, ok := value.(string); ok {
				return "Mr. " + s
			}
		}
		return value
	}))

	assert.NoError(t, err)
	assert.Equal(t, "Mr. john", dst.Name)
	assert.Equal(t, 30, dst.Age)
}

func TestWithValueConverter_StructToMap(t *testing.T) {
	type Person struct {
		Name string
		Age  int
	}

	src := Person{Name: "John", Age: 30}
	dst := make(map[string]any)

	err := Copy(&dst, src, WithValueConverter(func(fieldName string, value any) any {
		if fieldName == "Age" {
			if age, ok := value.(int); ok {
				return age + 1 // 年龄加1
			}
		}
		return value
	}))

	assert.NoError(t, err)
	assert.Equal(t, "John", dst["Name"])
	assert.Equal(t, 31, dst["Age"]) // 应该是31
}

func TestWithSkipFields_StructToMap(t *testing.T) {
	type Person struct {
		Name    string
		Age     int
		Sex     string
		Address string
	}

	src := Person{Name: "John", Age: 30, Sex: "Male", Address: "Beijing"}
	dst := make(map[string]any)

	err := Copy(&dst, src, WithSkipFields("Sex", "Address"))
	assert.NoError(t, err)
	assert.Equal(t, "John", dst["Name"])
	assert.Equal(t, 30, dst["Age"])
	_, hasSex := dst["Sex"]
	_, hasAddress := dst["Address"]
	assert.False(t, hasSex)
	assert.False(t, hasAddress)
}

func TestCombinedOptions(t *testing.T) {
	type Person struct {
		Name    string
		Age     int
		Sex     string
		Address string
	}

	src := Person{Name: "john", Age: 30, Sex: "Male", Address: "Beijing"}
	var dst Person

	err := Copy(&dst, src,
		WithSkipFields("Address"),
		WithValueConverter(func(fieldName string, value any) any {
			if fieldName == "Name" {
				if s, ok := value.(string); ok {
					return "Mr. " + s
				}
			}
			return value
		}),
	)

	assert.NoError(t, err)
	assert.Equal(t, "Mr. john", dst.Name)
	assert.Equal(t, 30, dst.Age)
	assert.Equal(t, "Male", dst.Sex)
	assert.Equal(t, "", dst.Address) // 被跳过
}
