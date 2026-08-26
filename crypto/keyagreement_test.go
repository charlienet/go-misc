package crypto

import (
    "testing"
    
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestECDH_KeyAgreement(t *testing.T) {
    ka, err := NewKeyAgreement("ECDH")
    require.NoError(t, err)
    
    // Alice 生成密钥
    aliceKP, err := ka.GenerateKey()
    require.NoError(t, err)
    
    // Bob 生成密钥
    ka2, _ := NewKeyAgreement("ECDH")
    bobKP, err := ka2.GenerateKey()
    require.NoError(t, err)
    
    // Alice 计算共享密钥
    secret1, err := ka.DeriveSharedSecret(bobKP.PublicKey)
    require.NoError(t, err)
    
    // Bob 计算共享密钥
    secret2, err := ka2.DeriveSharedSecret(aliceKP.PublicKey)
    require.NoError(t, err)
    
    // 双方应该得到相同的共享密钥
    assert.Equal(t, secret1, secret2)
}

func TestX25519_KeyAgreement(t *testing.T) {
    ka, err := NewKeyAgreement("X25519")
    require.NoError(t, err)
    
    aliceKP, err := ka.GenerateKey()
    require.NoError(t, err)
    
    ka2, _ := NewKeyAgreement("X25519")
    bobKP, err := ka2.GenerateKey()
    require.NoError(t, err)
    
    secret1, err := ka.DeriveSharedSecret(bobKP.PublicKey)
    require.NoError(t, err)
    
    secret2, err := ka2.DeriveSharedSecret(aliceKP.PublicKey)
    require.NoError(t, err)
    
    assert.Equal(t, secret1, secret2)
}

func TestSM2_KeyAgreement(t *testing.T) {
    ka, err := NewKeyAgreement("SM2")
    require.NoError(t, err)
    
    aliceKP, err := ka.GenerateKey()
    require.NoError(t, err)
    
    ka2, _ := NewKeyAgreement("SM2")
    bobKP, err := ka2.GenerateKey()
    require.NoError(t, err)
    
    secret1, err := ka.DeriveSharedSecret(bobKP.PublicKey)
    require.NoError(t, err)
    
    secret2, err := ka2.DeriveSharedSecret(aliceKP.PublicKey)
    require.NoError(t, err)
    
    assert.Equal(t, secret1, secret2)
}

func TestKeyAgreement_Concurrent(t *testing.T) {
    done := make(chan bool, 10)
    
    for i := 0; i < 10; i++ {
        go func() {
            ka, _ := NewKeyAgreement("X25519")
            kp, err := ka.GenerateKey()
            assert.NoError(t, err)
            assert.NotNil(t, kp)
            done <- true
        }()
    }
    
    for i := 0; i < 10; i++ {
        <-done
    }
}