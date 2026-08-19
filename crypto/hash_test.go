package crypto

import (
	stdcrypto "crypto"
	"testing"

	"github.com/stretchr/testify/assert"
)

// ==================== HashOptions.GetHash ====================

func TestHashOptions_GetHash(t *testing.T) {
	opts := &HashOptions{H: stdcrypto.SHA256}
	msg := []byte("hello")
	hash := opts.GetHash(msg)
	assert.Len(t, hash, 32) // SHA-256 输出 32 字节
}
