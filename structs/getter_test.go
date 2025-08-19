package structs

import (
	"testing"
)

func TestStructGetter(t *testing.T) {

}

func TestMapGettter(t *testing.T) {
	opt := defaultOptions()

	m := map[string]any{
		"test": "test",
	}
	getter := parseMapGetter(m, opt)
	for v, f := range getter.Iter() {
		t.Logf("%v: %v", v, f)
	}
}
