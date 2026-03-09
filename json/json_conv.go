package json

import (
	"bytes"
	"encoding/json"
	"reflect"
	"sort"
	"strings"
	"sync"

	"github.com/charlienet/go-misc/stringx"
)

var (
	fieldCache sync.Map // map[reflect.Type][]field
)

// json.Marshaler 接口的反射类型
var jsonMarshalerType = reflect.TypeOf((*json.Marshaler)(nil)).Elem()

// field 表示结构体的一个导出字段（经冲突解决后的最终字段）
type field struct {
	name      string // 输出的键名（若来自 tag 则为 tag 名，否则待转换）
	origName  string // 原始字段名（用于无 tag 时转换）
	index     []int  // 字段索引路径
	typ       reflect.Type
	omitempty bool
	fromTag   bool // true 表示 name 直接来自 tag，不再转换
}

// ====================== 命名风格转换器 ======================

type Snake2Camel struct{ Value any }

func (c Snake2Camel) MarshalJSON() ([]byte, error) {
	return marshalJSON(c.Value, stringx.Snake2Camel)
}

type Snake2Pascal struct{ Value any }

func (c Snake2Pascal) MarshalJSON() ([]byte, error) {
	return marshalJSON(c.Value, stringx.Snake2Pascal)
}

type Pascal2Camel struct{ Value any }

func (c Pascal2Camel) MarshalJSON() ([]byte, error) {
	return marshalJSON(c.Value, stringx.Pascal2Camel)
}

type Camel2Pascal struct{ Value any }

func (c Camel2Pascal) MarshalJSON() ([]byte, error) {
	return marshalJSON(c.Value, stringx.Camel2Pascal)
}

type Pascal2Snake struct{ Value any }

func (c Pascal2Snake) MarshalJSON() ([]byte, error) {
	return marshalJSON(c.Value, stringx.Pascal2Snake)
}

type Pascal2UpperSnake struct{ Value any }

func (c Pascal2UpperSnake) MarshalJSON() ([]byte, error) {
	return marshalJSON(c.Value, stringx.Pascal2UpperSnake)
}

// ====================== 核心递归编码函数 ======================

// marshalJSON 是公共入口，调用 encodeValue 递归处理值
func marshalJSON(v any, convert func(string) string) ([]byte, error) {
	return encodeValue(reflect.ValueOf(v), convert)
}

// encodeValue 递归处理任意值，返回其 JSON 片段
func encodeValue(val reflect.Value, convert func(string) string) ([]byte, error) {
	if !val.IsValid() {
		return []byte("null"), nil
	}

	// 解引用指针和接口
	for val.Kind() == reflect.Ptr || val.Kind() == reflect.Interface {
		if val.IsNil() {
			return []byte("null"), nil
		}
		val = val.Elem()
	}

	// 检查是否实现了 json.Marshaler
	if val.Type().Implements(jsonMarshalerType) {
		return val.Interface().(json.Marshaler).MarshalJSON()
	}

	switch val.Kind() {
	case reflect.Struct:
		return encodeStruct(val, convert)
	case reflect.Map:
		return encodeMap(val, convert)
	case reflect.Slice, reflect.Array:
		return encodeSlice(val, convert)
	default:
		// 基本类型直接使用标准库编码
		return json.Marshal(val.Interface())
	}
}

// encodeStruct 编码结构体，键名优先使用 tag，否则由 convert 转换
func encodeStruct(val reflect.Value, convert func(string) string) ([]byte, error) {
	typ := val.Type()
	fields := cachedTypeFields(typ)

	buf := &bytes.Buffer{}
	buf.WriteByte('{')
	first := true

	for _, f := range fields {
		// 获取字段值
		fieldVal := val.FieldByIndex(f.index)
		if !fieldVal.IsValid() {
			continue
		}

		// 决定最终的键名
		var key string
		if f.fromTag {
			key = f.name // 直接使用 tag 中的名称
		} else {
			key = convert(f.origName) // 应用命名风格转换
		}

		// 编码字段值（递归）
		elemBytes, err := encodeValue(fieldVal, convert)
		if err != nil {
			return nil, err
		}

		// 写入键值对
		if !first {
			buf.WriteByte(',')
		}
		first = false

		// 键名需 JSON 转义
		keyBytes, err := json.Marshal(key)
		if err != nil {
			return nil, err
		}
		buf.Write(keyBytes) // 已包含引号
		buf.WriteByte(':')
		buf.Write(elemBytes)
	}

	buf.WriteByte('}')
	return buf.Bytes(), nil
}

// encodeMap 编码 map，键名经过 convert 转换（不处理 tag），值递归处理
func encodeMap(val reflect.Value, convert func(string) string) ([]byte, error) {
	// 非字符串键的 map 在 JSON 中不支持
	if val.Type().Key().Kind() != reflect.String {
		return nil, &json.UnsupportedTypeError{Type: val.Type()}
	}

	buf := &bytes.Buffer{}
	buf.WriteByte('{')
	first := true
	iter := val.MapRange()

	for iter.Next() {
		k := iter.Key().String()
		v := iter.Value()

		// 键名经过转换函数处理
		convertedKey := convert(k)

		// 编码值
		valueBytes, err := encodeValue(v, convert)
		if err != nil {
			return nil, err
		}

		if !first {
			buf.WriteByte(',')
		}
		first = false

		// 将转换后的键名 JSON 转义后写入
		keyBytes, err := json.Marshal(convertedKey)
		if err != nil {
			return nil, err
		}
		buf.Write(keyBytes) // 已包含引号
		buf.WriteByte(':')
		buf.Write(valueBytes)
	}

	buf.WriteByte('}')
	return buf.Bytes(), nil
}

// encodeSlice 编码切片或数组
func encodeSlice(val reflect.Value, convert func(string) string) ([]byte, error) {
	buf := &bytes.Buffer{}
	buf.WriteByte('[')
	for i := 0; i < val.Len(); i++ {
		elem := val.Index(i)
		elemBytes, err := encodeValue(elem, convert)
		if err != nil {
			return nil, err
		}
		if i > 0 {
			buf.WriteByte(',')
		}
		buf.Write(elemBytes)
	}
	buf.WriteByte(']')
	return buf.Bytes(), nil
}

// ====================== 结构体字段缓存与解析 ======================

// cachedTypeFields 获取类型的字段列表（带缓存）
func cachedTypeFields(typ reflect.Type) []field {
	if cached, ok := fieldCache.Load(typ); ok {
		return cached.([]field)
	}
	fields := typeFields(typ)
	fieldCache.Store(typ, fields)
	return fields
}

// typeFields 解析结构体类型的所有导出字段，严格遵循标准库冲突处理规则
func typeFields(typ reflect.Type) []field {
	var fields []field

	// 递归收集所有字段（包括匿名结构体提升）
	var visit func(t reflect.Type, baseIndex []int, visited map[reflect.Type]bool)
	visit = func(t reflect.Type, baseIndex []int, visited map[reflect.Type]bool) {
		if visited[t] {
			return
		}
		visited[t] = true

		for i := 0; i < t.NumField(); i++ {
			f := t.Field(i)

			// 忽略非导出字段
			if f.PkgPath != "" && !f.Anonymous {
				continue
			}

			// 当前字段的索引路径
			index := make([]int, len(baseIndex)+1)
			copy(index, baseIndex)
			index[len(baseIndex)] = i

			// 处理匿名字段
			if f.Anonymous {
				// 检查匿名字段自身的 json tag
				tag := f.Tag.Get("json")
				if tag != "" && tag != "-" {
					// 有 tag，将其作为普通字段处理（不展开）
					name, omitempty := parseTag(tag)
					if name == "" {
						name = f.Name // 实际上标准库中，若 tag 存在但为空，会使用字段名，但这里 name 已由 parseTag 处理
					}
					fields = append(fields, field{
						name:      name,
						origName:  f.Name,
						index:     index,
						typ:       f.Type,
						omitempty: omitempty,
						fromTag:   true, // 因为 tag 非空
					})
					continue
				}
				// 无 tag，展开匿名字段
				ft := f.Type
				if ft.Kind() == reflect.Ptr {
					ft = ft.Elem()
				}
				if ft.Kind() == reflect.Struct {
					visit(ft, index, visited)
				}
				continue
			}

			// 非匿名字段，解析 json tag
			tag := f.Tag.Get("json")
			if tag == "-" {
				continue
			}
			name, omitempty := parseTag(tag)
			fromTag := false
			if name == "" {
				name = f.Name
			} else {
				fromTag = true
			}
			fields = append(fields, field{
				name:      name,
				origName:  f.Name,
				index:     index,
				typ:       f.Type,
				omitempty: omitempty,
				fromTag:   fromTag,
			})
		}
	}

	visit(typ, nil, make(map[reflect.Type]bool))

	// 按名称分组，解决冲突
	byName := make(map[string][]field)
	for _, f := range fields {
		byName[f.name] = append(byName[f.name], f)
	}

	// 为每个名称选择胜出的字段
	result := make([]field, 0, len(byName))
	for _, cands := range byName {
		if len(cands) == 1 {
			result = append(result, cands[0])
			continue
		}
		// 选择深度最小的（索引长度最小）
		best := cands[0]
		for _, c := range cands[1:] {
			if len(c.index) < len(best.index) {
				best = c
				continue
			}
			if len(c.index) == len(best.index) {
				// 深度相同，按索引字典序比较，选更小的（声明顺序靠前）
				if lessIndex(c.index, best.index) {
					best = c
				}
			}
		}
		result = append(result, best)
	}

	// 按深度和索引排序，使输出顺序与标准库一致
	sort.Slice(result, func(i, j int) bool {
		if len(result[i].index) != len(result[j].index) {
			return len(result[i].index) < len(result[j].index)
		}
		return lessIndex(result[i].index, result[j].index)
	})

	return result
}

// parseTag 解析 json tag，返回名称和 omitempty 标志
func parseTag(tag string) (name string, omitempty bool) {
	if tag == "" {
		return "", false
	}
	parts := strings.Split(tag, ",")
	name = parts[0]
	for _, opt := range parts[1:] {
		if opt == "omitempty" {
			omitempty = true
		}
	}
	return
}

// lessIndex 比较两个索引切片，按字典序
func lessIndex(a, b []int) bool {
	for i := 0; i < len(a) && i < len(b); i++ {
		if a[i] != b[i] {
			return a[i] < b[i]
		}
	}
	return len(a) < len(b)
}
