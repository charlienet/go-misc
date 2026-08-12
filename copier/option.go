package copier

import "slices"

import "reflect"

const (
	noDepthLimited = -1
)

type options struct {
	tagName          string                // 标签名
	maxDepth         int                   // 最大复制深度
	ignoreEmpty      bool                  // 复制时忽略空字段
	caseSensitive    bool                  // 复制时大小写敏感
	must             bool                  // 只复制具有must标识的字段
	converters       []TypeConverter       // 类型转换器
	fieldNameMapping map[string]string     // 字段名转映射
	nameConverter    func(string) string   // 字段名转换器
	skipFields       []string              // 跳过的字段列表
	valueConverter   func(string, any) any // 值转换函数
}

type option func(*options)

type TypeConverter struct {
	FieldName string
	SrcType   any
	DstType   any
	Fn        func(src any) (dst any, err error)
}

func getOpt(opts ...option) *options {
	// 复制 DefaultOptions 避免测试间状态污染
	opt := *DefaultOptions

	for _, o := range opts {
		o(&opt)
	}

	return &opt
}

func (opt *options) NameConvert(name string) string {
	if opt.nameConverter != nil {
		name = opt.nameConverter(name)
	}

	if toname, ok := opt.fieldNameMapping[name]; ok {
		name = toname
	}

	return name
}

// TypeConvert 遍历 opt.converters，按 TypeConverter 声明的字段名/源类型/目标类型匹配并执行转换。
// 匹配成功返回 (转换结果, true)，否则返回 (原值, false)。
func (opt options) TypeConvert(fieldName string, value reflect.Value) (reflect.Value, bool) {
	if len(opt.converters) == 0 {
		return value, false
	}

	for _, tc := range opt.converters {
		if tc.Fn == nil {
			continue
		}

		if tc.FieldName != "" && tc.FieldName != fieldName {
			continue
		}

		if tc.SrcType != nil && !typeMatch(tc.SrcType, value.Type()) {
			continue
		}

		converted, err := tc.Fn(value.Interface())
		if err != nil || converted == nil {
			continue
		}

		if tc.DstType != nil && reflect.TypeOf(converted) != reflect.TypeOf(tc.DstType) {
			continue
		}

		return reflect.ValueOf(converted), true
	}

	return value, false
}

func typeMatch(src any, t reflect.Type) bool {
	st := reflect.TypeOf(src)
	if st == nil {
		return false
	}

	if st == t {
		return true
	}

	// 排除数值↔string 跨类别（Go 中整数→string 是 rune 语义，不符合 SrcType 匹配预期）
	if isNumericStringCross(st, t) {
		return false
	}

	return st.ConvertibleTo(t) || t.ConvertibleTo(st)
}

func (opt *options) ExceedMaxDepth(depth int) bool {
	return opt.maxDepth != noDepthLimited && depth > opt.maxDepth
}

func (opt *options) isSkipField(fieldName string) bool {
	return slices.Contains(opt.skipFields, fieldName)
}

func WithMaxDepth(depth int) option {
	return func(o *options) {
		o.maxDepth = depth
	}
}

func WithIgnoreEmpty() option {
	return func(o *options) {
		o.ignoreEmpty = true
	}
}

func WithCaseSensitive() option {
	return func(o *options) {
		o.caseSensitive = true
	}
}

func WithMust() option {
	return func(o *options) {
		o.must = true
	}
}

func WithConverters(converters ...TypeConverter) option {
	return func(o *options) {
		o.converters = converters
	}
}

func WithNameMapping(mappings map[string]string) option {
	return func(o *options) {
		o.fieldNameMapping = mappings
	}
}

func WithNameFn(fn func(string) string) option {
	return func(o *options) {
		o.nameConverter = fn
	}
}

func WithTagName(tagName string) option {
	return func(o *options) {
		o.tagName = tagName
	}
}

func WithSkipFields(fields ...string) option {
	return func(o *options) {
		o.skipFields = fields
	}
}

func WithValueConverter(fn func(string, any) any) option {
	return func(o *options) {
		o.valueConverter = fn
	}
}

var DefaultOptions = &options{
	maxDepth: noDepthLimited,
	tagName:  defaultTag,
}
