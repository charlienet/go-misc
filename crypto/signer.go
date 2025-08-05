package crypto

import "github.com/charlienet/go-misc/bytesconv"

// 对称加密
type ISymmetric interface {
	Encrypt(msg []byte) (bytesconv.BytesResult, error)
	Decrypt(cipherText []byte) (bytesconv.BytesResult, error)
}

// 非对称加密
type IAsymmetric interface {
	Encrypt(msg []byte) (bytesconv.BytesResult, error)
	Decrypt(ciphertext []byte) (bytesconv.BytesResult, error)
}

type Signer interface {
	Sign(msg []byte) (bytesconv.BytesResult, error)
	Verify(msg, sign []byte) bool
}
