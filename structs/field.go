package structs

import (
	"reflect"
	"strings"

	"github.com/charlienet/go-misc/expr"
)

type field struct {
	name        string
	tagName     string
	toName      string
	ignoreEmpty bool
	ignore      bool
}

func parseStructField(fi reflect.StructField, opt options) field {
	name, opts := parseTag(fi.Tag.Get(opt.TagName))

	return field{
		name:        fi.Name,
		toName:      opt.toName(fi.Name),
		tagName:     expr.Ternary(isValidTag(name), name, expr.Ternary(opt.nameConverter != nil, opt.nameConverter(fi.Name), fi.Name)),
		ignoreEmpty: opt.IgnoreEmpty || (opts.Contains("omitempty") && opt.omitempty),
		ignore:      (name == "-" && opt.Ignore) || isSkipField(fi.Name, opt.SkipFields),
	}
}

func (f field) shouldIgnore(s reflect.Value) bool {
	return f.ignore || (s.IsZero() && f.ignoreEmpty)
}

func isSkipField(name string, skips []string) bool {
	for _, v := range skips {
		if strings.EqualFold(v, name) {
			return true
		}
	}

	return false
}
