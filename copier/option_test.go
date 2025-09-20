package copier

import (
	"testing"

	"github.com/charlienet/go-misc/stringx"
	"github.com/stretchr/testify/assert"
)

func TestNameConvertFn(t *testing.T) {
	opt := getOpt(WithNameFn(stringx.Camel2Pascal))
	r := opt.NameConvert("test")

	assert.Equal(t, "Test", r)
}

func TestNameMapping(t *testing.T) {
	opt := getOpt(WithNameMapping(map[string]string{
		"test": "Test1111111",
	}))

	r := opt.NameConvert("test")
	assert.Equal(t, "Test1111111", r)
}
