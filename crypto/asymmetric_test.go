package crypto

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestECDSA_SignAndVerify(t *testing.T) {
	algo, err := NewAsymmetric("ECDSA")
	require.NoError(t, err)
	
	kp, err := algo.GenerateKey()
	require.NoError(t, err)
	
	signer, err := NewAsymmetric("ECDSA", WithPrivateKeyObject(kp.PrivateKey))
	require.NoError(t, err)
	
	verifier, err := NewAsymmetric("ECDSA", WithPublicKeyObject(kp.PublicKey))
	require.NoError(t, err)
	
	data := []byte("hello world")
	sig, err := signer.Sign(data)
	require.NoError(t, err)
	
	assert.True(t, verifier.Verify(data, sig))
	assert.False(t, verifier.Verify([]byte("tampered"), sig))
}

func TestEd25519_SignAndVerify(t *testing.T) {
	algo, err := NewAsymmetric("Ed25519")
	require.NoError(t, err)
	
	kp, err := algo.GenerateKey()
	require.NoError(t, err)
	
	signer, err := NewAsymmetric("Ed25519", WithPrivateKeyObject(kp.PrivateKey))
	require.NoError(t, err)
	
	verifier, err := NewAsymmetric("Ed25519", WithPublicKeyObject(kp.PublicKey))
	require.NoError(t, err)
	
	data := []byte("hello world")
	sig, err := signer.Sign(data)
	require.NoError(t, err)
	
	assert.True(t, verifier.Verify(data, sig))
	assert.False(t, verifier.Verify([]byte("tampered"), sig))
}

func TestECDSA_EncryptNotSupported(t *testing.T) {
	algo, _ := NewAsymmetric("ECDSA")
	_, err := algo.Encrypt([]byte("test"))
	assert.Error(t, err)
}

func TestEd25519_EncryptNotSupported(t *testing.T) {
	algo, _ := NewAsymmetric("Ed25519")
	_, err := algo.Encrypt([]byte("test"))
	assert.Error(t, err)
}

func TestECDSA_DecryptNotSupported(t *testing.T) {
	algo, _ := NewAsymmetric("ECDSA")
	_, err := algo.Decrypt([]byte("test"))
	assert.Error(t, err)
}

func TestEd25519_DecryptNotSupported(t *testing.T) {
	algo, _ := NewAsymmetric("Ed25519")
	_, err := algo.Decrypt([]byte("test"))
	assert.Error(t, err)
}

func TestECDSA_Name(t *testing.T) {
	algo, _ := NewAsymmetric("ECDSA")
	assert.Equal(t, "ECDSA", algo.Name())
}

func TestEd25519_Name(t *testing.T) {
	algo, _ := NewAsymmetric("Ed25519")
	assert.Equal(t, "Ed25519", algo.Name())
}

func TestECDSA_ExportPublicKey(t *testing.T) {
	algo, err := NewAsymmetric("ECDSA")
	require.NoError(t, err)
	
	kp, err := algo.GenerateKey()
	require.NoError(t, err)
	
	signer, err := NewAsymmetric("ECDSA", WithPrivateKeyObject(kp.PrivateKey))
	require.NoError(t, err)
	
	publicKeyStr, err := signer.ExportPublicKey()
	require.NoError(t, err)
	assert.NotEmpty(t, publicKeyStr)
}

func TestEd25519_ExportPublicKey(t *testing.T) {
	algo, err := NewAsymmetric("Ed25519")
	require.NoError(t, err)
	
	kp, err := algo.GenerateKey()
	require.NoError(t, err)
	
	signer, err := NewAsymmetric("Ed25519", WithPrivateKeyObject(kp.PrivateKey))
	require.NoError(t, err)
	
	publicKeyStr, err := signer.ExportPublicKey()
	require.NoError(t, err)
	assert.NotEmpty(t, publicKeyStr)
}