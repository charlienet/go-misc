package structs

import (
	"slices"

	"github.com/charlienet/go-misc/stringx"
)

const defaultTagName = "json"

type option func(*options)

type options struct {
	SkipFields       []string
	TagName          string
	DeepCopy         bool
	omitempty        bool
	IgnoreEmpty      bool
	Ignore           bool
	overwrite        bool
	fieldNameMapping map[string]string     // 名称映射
	nameConverter    func(string) string   // 名称转换
	valueConverter   func(string, any) any // 值转换
}

func TagName(tagName string) option {
	return func(o *options) {
		o.TagName = tagName
	}
}

func SkipFields(fields ...string) option {
	return func(o *options) {
		o.SkipFields = fields
	}
}

func DeepCopy() option {
	return func(o *options) {
		o.DeepCopy = true
	}
}

func Omitempty() option {
	return func(o *options) {
		o.omitempty = true
	}
}

func IgnoreEmpty() option {
	return func(o *options) {
		o.IgnoreEmpty = true
	}
}

func Ignore() option {
	return func(o *options) {
		o.Ignore = true
	}
}

func Overwrite() option {
	return func(o *options) {
		o.overwrite = true
	}
}

func FieldNameMapping(mapping map[string]string) option {
	return func(o *options) {
		o.fieldNameMapping = mapping
	}
}

func ValueConverter(f func(string, any) any) option {
	return func(o *options) {
		o.valueConverter = f
	}
}

func Lcfirst() option {
	return func(o *options) {
		o.nameConverter = stringx.Pascal2Camel
	}
}

func Camel2Case() option {
	return func(o *options) {
		o.nameConverter = stringx.Camel2Pascal
	}
}

func defaultOptions() options {
	return options{
		TagName:       defaultTagName,
		Ignore:        true,
		nameConverter: func(s string) string { return s },
	}
}

func acquireOptions(opts []option) options {
	o := defaultOptions()
	for _, f := range opts {
		f(&o)
	}

	return o
}

func (o options) isSkipField(fieldName string) bool {
	return slices.Contains(o.SkipFields, fieldName)
}

func (o options) toName(name string) string {
	if v, ok := o.fieldNameMapping[name]; ok {
		return v
	}

	return name
}
