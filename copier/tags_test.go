package copier

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseTag(t *testing.T) {
	tag := "must,toname=xxx"
	opt := parseTag(tag)

	exist := opt.Contains(tagMust)
	assert.True(t, exist)
	exist = opt.Contains(32)
	assert.False(t, exist)

	toname := opt.ToName()
	// assert.True(t, ok)
	assert.Equal(t, "xxx", toname)

	tag = "-"
	opt = parseTag(tag)
	assert.True(t, opt.Contains(tagIgnore))

}

func BenchmarkTagParse(b *testing.B) {
	for b.Loop() {
		parseTag("must")
	}
}
