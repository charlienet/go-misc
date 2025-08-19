package structs

import "reflect"

type getter interface {
	Iter() func(yield func(any, field) bool)
}

type structGetter struct {
	value  reflect.Value
	fields []field
}

func parseStructGetter(from reflect.Value, opt options) getter {
	typ := indirectType(from.Type())

	fields := make([]field, 0, typ.NumField())
	for i := 0; i < typ.NumField(); i++ {
		f := typ.Field(i)
		if opt.isSkipField(f.Name) {
			continue
		}

		fields = append(fields, parseStructField(f, opt))
	}

	return &structGetter{
		value:  from,
		fields: fields,
	}
}

func (g *structGetter) Iter() func(yield func(any, field) bool) {
	return func(yield func(any, field) bool) {
		for _, f := range g.fields {
			fv := g.value.FieldByName(f.name)
			s := fv.Interface()
			if !yield(s, f) {
				return
			}
		}
	}
}

type mapGetter struct {
	m      map[string]any
	fields []field
}

func parseMapGetter(m map[string]any, opt options) getter {
	fields := make([]field, 0, len(m))
	for k := range m {
		if opt.isSkipField(k) {
			continue
		}

		toName := opt.toName(k)
		fields = append(fields, field{
			name:   k,
			toName: toName,
		})
	}

	return &mapGetter{
		m:      m,
		fields: fields,
	}
}

func (g *mapGetter) Iter() func(yield func(any, field) bool) {
	return func(yield func(any, field) bool) {
		for _, f := range g.fields {

			if !yield(g.m[f.name], f) {
				return
			}
		}
	}
}
