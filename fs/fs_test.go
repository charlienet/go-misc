package fs

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExist(t *testing.T) {
	// 测试存在的文件
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.txt")
	err := os.WriteFile(tmpFile, []byte("test"), 0644)
	assert.NoError(t, err)

	assert.True(t, Exist(tmpFile))

	// 测试存在的目录
	assert.True(t, Exist(tmpDir))

	// 测试不存在的文件
	assert.False(t, Exist(filepath.Join(tmpDir, "nonexistent.txt")))

	// 测试空路径
	assert.False(t, Exist(""))
}
