package copier

import "fmt"

type exampleUser struct {
	Name string
	Age  int
}

func ExampleCopy() {
	src := exampleUser{Name: "John", Age: 30}
	var dst exampleUser
	if err := Copy(&dst, src); err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("%s is %d years old", dst.Name, dst.Age)
	// Output: John is 30 years old
}

func ExampleCopy_map() {
	src := map[string]any{"Name": "John", "Age": 30}
	type user struct {
		Name string
		Age  int
	}
	var dst user
	if err := Copy(&dst, src); err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("%s is %d years old", dst.Name, dst.Age)
	// Output: John is 30 years old
}

func ExampleCopy_options() {
	src := map[string]any{"Name": "John", "Age": 30, "Secret": "x"}
	type user struct {
		Name   string
		Age    int
		Secret string
	}
	var dst user
	if err := Copy(&dst, src, WithSkipFields("Secret")); err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("secret=%q", dst.Secret)
	// Output: secret=""
}

func ExampleToMap() {
	src := struct {
		Name string
		Age  int
	}{Name: "John", Age: 30}
	m, err := ToMap(src)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("age=%d", m["Age"].(int))
	// Output: age=30
}
