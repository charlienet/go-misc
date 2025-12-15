package crypto

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/des"
	"crypto/rand"
	"errors"
	"fmt"
	"io"

	"github.com/charlienet/go-misc/bytesconv"
	"github.com/tjfoc/gmsm/sm4"
)

const (
	nonceSize = 12
)

// 对称加密算法
type Cipher interface {
	NewCTR(iv []byte) StreamCipher
	NewGCM(nonce []byte, opts ...optFunc) (CipherMode, error)
	NewGCMWithRandomNonce() (CipherMode, error)
	NewCBC(iv []byte, opts ...optFunc) (CipherMode, error)
	NewECB() CipherMode
}

type CipherMode interface {
	Encrypt(plainText []byte) bytesconv.BytesResult
	Decrypt(cipherText []byte) (bytesconv.BytesResult, error)
}

type StreamCipher interface {
	XORKeyStream(src []byte) []byte
	Stream(reader io.Reader) io.Reader
}

var supported = map[string]*creator{
	"SM4":     {sm4.NewCipher, sm4.BlockSize, sm4.BlockSize},
	"AES":     {aes.NewCipher, aes.BlockSize, aes.BlockSize},
	"AES-128": {aes.NewCipher, aes.BlockSize, aes.BlockSize},
	"AES-192": {aes.NewCipher, 24, aes.BlockSize},
	"AES-256": {aes.NewCipher, 32, aes.BlockSize},
	"DES":     {des.NewCipher, des.BlockSize, des.BlockSize},
	"3DES":    {des.NewTripleDESCipher, 24, des.BlockSize},
}

type creator struct {
	new       func(key []byte) (cipher.Block, error)
	blockSize int
	ivSize    int
}

type optFunc func(*symmetric)

func EmbedIV() optFunc {
	return func(a *symmetric) {
		a.embediv = true
	}
}

func EmbedNonce() optFunc {
	return func(a *symmetric) {
		a.embednonce = true
	}
}

func GenerateKey(algorithm string) (key, iv, nonce []byte, err error) {
	blockSize, ivSize := BlockSize(algorithm)

	random := make([]byte, blockSize+ivSize+nonceSize)
	if _, err := io.ReadFull(rand.Reader, random); err != nil {
		return nil, nil, nil, err
	}

	return random[:blockSize], random[blockSize : blockSize+ivSize], random[blockSize+ivSize:], nil
}

func BlockSize(algorithm string) (int, int) {
	creator, ok := supported[algorithm]
	if !ok {
		panic(fmt.Errorf("unsupported algorithm: %s", algorithm))
	}

	return creator.blockSize, creator.ivSize
}

func NewCipher(algorithm string, key []byte) (Cipher, error) {
	creator, ok := supported[algorithm]
	if !ok {
		return nil, fmt.Errorf("unsupported algorithm: %s", algorithm)
	}

	block, err := creator.new(key)
	if err != nil {
		return nil, err
	}

	return &symmetric{block: block}, nil
}

type symmetric struct {
	block       cipher.Block
	embediv     bool
	embednonce  bool
	randomNonce bool
}

func (a *symmetric) BlockSize() int {
	return a.block.BlockSize()
}

type stream struct {
	stream cipher.Stream
}

func (a *symmetric) NewCTR(iv []byte) StreamCipher {
	s := cipher.NewCTR(a.block, iv)
	return &stream{stream: s}
}

func (s *stream) XORKeyStream(src []byte) []byte {
	dst := make([]byte, len(src))
	s.stream.XORKeyStream(dst, src)
	return dst
}

func (s *stream) Stream(reader io.Reader) io.Reader {
	return cipher.StreamReader{S: s.stream, R: reader}
}

func (s *stream) Reset() {
}

func (a *symmetric) NewGCM(nonce []byte, opts ...optFunc) (CipherMode, error) {
	gcm, err := cipher.NewGCM(a.block)
	if err != nil {
		return nil, err
	}

	gcm.NonceSize()
	return &algo_gcm{symmetric: a, gcm: gcm, nonce: nonce}, nil
}

func (a *symmetric) NewGCMWithRandomNonce() (CipherMode, error) {
	gcm, err := cipher.NewGCM(a.block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	_, err = io.ReadFull(rand.Reader, nonce)
	if err != nil {
		return nil, err
	}

	a.embednonce = true
	a.randomNonce = true

	return &algo_gcm{symmetric: a, gcm: gcm, nonce: nonce}, nil
}

func (a *symmetric) NewCBC(iv []byte, opts ...optFunc) (CipherMode, error) {
	if len(iv) != a.BlockSize() {
		return nil, errors.New("iv length is not equal to block size")
	}

	for _, opt := range opts {
		opt(a)
	}

	return &algo_cbc{symmetric: a, iv: iv}, nil
}

func (a *symmetric) NewECB() CipherMode {
	return &algo_ecb{symmetric: a}
}

type algo_ecb struct {
	*symmetric
}

func (a *algo_ecb) Encrypt(plainText []byte) bytesconv.BytesResult {
	plainText = a.pkcs7Padding(plainText)
	dst := make([]byte, len(plainText))
	a.block.Encrypt(dst, plainText)

	return dst
}

func (a *algo_ecb) Decrypt(cipherText []byte) (bytesconv.BytesResult, error) {
	var dst = make([]byte, len(cipherText))
	a.block.Decrypt(dst, cipherText)

	return a.pkcs7UnPadding(dst)
}

type algo_cbc struct {
	*symmetric
	iv []byte
}

func (a *algo_cbc) Encrypt(plainText []byte) bytesconv.BytesResult {
	// The IV needs to be unique, but not secure. Therefore it's common to
	// include it at the beginning of the ciphertext.

	plainText = a.pkcs7Padding(plainText)
	stream := cipher.NewCBCEncrypter(a.block, a.iv)

	if a.embediv {
		cipherText := make([]byte, len(plainText)+a.BlockSize())
		copy(cipherText, a.iv)
		stream.CryptBlocks(cipherText[a.BlockSize():], plainText)

		return cipherText
	} else {
		cipherText := make([]byte, len(plainText))
		stream.CryptBlocks(cipherText, plainText)

		return cipherText
	}
}

func (a *algo_cbc) Decrypt(ciphertext []byte) (bytesconv.BytesResult, error) {
	// The IV needs to be unique, but not secure. Therefore it's common to
	// include it at the beginning of the ciphertext.
	if a.embediv {
		iv, cipherText := ciphertext[:a.BlockSize()], ciphertext[a.BlockSize():]
		stream := cipher.NewCBCDecrypter(a.block, iv)
		stream.CryptBlocks(cipherText, cipherText)

		return a.pkcs7UnPadding(cipherText)
	} else {
		stream := cipher.NewCBCDecrypter(a.block, a.iv)
		stream.CryptBlocks(ciphertext, ciphertext)

		return a.pkcs7UnPadding(ciphertext)
	}
}

type algo_gcm struct {
	*symmetric
	gcm   cipher.AEAD
	nonce []byte
}

func (a *algo_gcm) NonceSize() int {
	return a.gcm.NonceSize()
}

func (a *algo_gcm) Encrypt(plainText []byte) bytesconv.BytesResult {
	if a.randomNonce {
		a.nonce = make([]byte, a.gcm.NonceSize())
		// _, err := io.ReadFull(rand.Reader, a.nonce)
		io.ReadFull(rand.Reader, a.nonce)
	}

	if a.embednonce {
		return a.gcm.Seal(a.nonce, a.nonce, plainText, nil)
	} else {
		return a.gcm.Seal(nil, a.nonce, plainText, nil)
	}
}

func (a *algo_gcm) Decrypt(ciphertext []byte) (bytesconv.BytesResult, error) {
	if a.embednonce {
		nonceSize := a.gcm.NonceSize()
		nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
		return a.gcm.Open(nil, nonce, ciphertext, nil)
	} else {
		return a.gcm.Open(nil, a.nonce, ciphertext, nil)
	}
}

func (a *symmetric) pkcs7Padding(src []byte) []byte {
	padding := a.BlockSize() - len(src)%a.BlockSize()
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(src, padtext...)
}

func (a *symmetric) pkcs7UnPadding(src []byte) ([]byte, error) {
	length := len(src)
	unpadding := int(src[length-1])
	if unpadding > a.BlockSize() || unpadding == 0 {
		return nil, errors.New("invalid pkcs7 padding (unpadding > BlockSize || unpadding == 0)")
	}

	pad := src[len(src)-unpadding:]
	for i := 0; i < unpadding; i++ {
		if pad[i] != byte(unpadding) {
			return nil, errors.New("invalid pkcs7 padding (pad[i] != unpadding)")
		}
	}

	return src[:(length - unpadding)], nil
}
