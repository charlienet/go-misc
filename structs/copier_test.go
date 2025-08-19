package structs

import (
	"strings"
	"testing"

	"github.com/charlienet/go-misc/json"
)

func TestStructToStruct(t *testing.T) {
	var a Person
	var b = Person{
		Name: "test",
		Age:  10,
	}

	if err := Copy(&a, b); err != nil {
		t.Fatal(err)
	}
	t.Log(json.Struct2Json(a))
}

func TestStructToMap(t *testing.T) {
	a := Person{
		Name: "test",
		Age:  10,
	}

	dst := make(map[string]any)
	if err := Copy(&dst, a); err != nil {
		t.Fatal(err)
	}

	if dst["Name"] != "test" {
		t.Fatal("Name is not equal")
	}

	t.Log(json.Struct2Json(dst))
}

type Person struct {
	Name string
	Age  int
}

func TestMapToStruct(t *testing.T) {
	var dst Person

	m := map[string]any{
		"Name": "test",
		"Age":  10,
	}

	if err := Copy(&dst, m); err != nil {
		t.Fatal(err)
	}
	t.Log(json.Struct2Json(dst))
}

func TestMapToMap(t *testing.T) {
	m1 := map[string]any{}
	m2 := map[string]any{
		"Name": "test",
		"Age":  10,
		"c": map[string]any{
			"Name": "test11",
			"Age":  33,
		},
	}

	if err := Copy(&m1, m2); err != nil {
		t.Fatal(err)
	}

	t.Log(m1)
}

func TestValueConvert(t *testing.T) {
	var dst Person

	m := map[string]any{
		"Name":  "test",
		"Age11": 10,
	}

	if err := Copy(&dst, m,
		FieldNameMapping(map[string]string{"Age11": "Age"}),
		ValueConverter(func(s string, a any) any {
			if strings.EqualFold(s, "Age") {
				return a.(int) + 1
			}

			return a
		})); err != nil {
		t.Fatal(err)
	}

	t.Log(json.Struct2Json(dst))
}
