package json

import "github.com/charlienet/misc/bytesconv"

// MustStruct2JsonIndent 结构转换为带格式字符串
func MustStruct2JsonIndent(obj any) string {
	b, err := MarshalIndent(obj, "", "  ")
	if err != nil {
		panic(err)
	}

	return bytesconv.BytesToString(b)
}

// MustStruct2Json 结构转换为json字符串
func MustStruct2Json(obj any) string {
	b, err := Marshal(obj)
	if err != nil {
		panic(err)
	}
	return bytesconv.BytesToString(b)
}

// Struct2JsonIndent 结构转换为带格式字符串
func Struct2JsonIndent(obj any) string {
	b, _ := MarshalIndent(obj, "", "  ")
	return bytesconv.BytesToString(b)
}

// Struct2Json 结构转换为json字符串
func Struct2Json(obj any) string {
	b, _ := Marshal(obj)
	return bytesconv.BytesToString(b)
}
