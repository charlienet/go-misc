package copier

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

type FieldNameConverter struct {
	SrcFieldName string
	DstFieldName string
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

func (opt options) TypeConvert(value reflect.Value) (reflect.Value, bool) {
	return value, false
}

func (opt *options) ExceedMaxDepth(depth int) bool {
	return opt.maxDepth != noDepthLimited && depth > opt.maxDepth
}

func (opt *options) isSkipField(fieldName string) bool {
	for _, skip := range opt.skipFields {
		if skip == fieldName {
			return true
		}
	}
	return false
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
