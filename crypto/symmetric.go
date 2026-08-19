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
	"github.com/emmansun/gmsm/sm4"
)

const (
	nonceSize = 12
)

// Cipher 对称加密算法接口。
type Cipher interface {
	Block() cipher.Block
	BlockSize() int // 块大小（=16）
	IVSize() int    // iv 长度
	NewCTR(iv []byte) StreamCipher
	NewGCM(nonce []byte, opts ...optFunc) (CipherMode, error)
	NewGCMWithRandomNonce() (CipherMode, error)
	NewCBC(iv []byte, opts ...optFunc) (CipherMode, error)
	NewECB() (CipherMode, error)
}

type CipherMode interface {
	Encrypt(plainText []byte) (bytesconv.BytesResult, error)
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
	new     func(key []byte) (cipher.Block, error)
	keySize int // 密钥长度（AES-128:16, AES-192:24, AES-256:32, SM4:16, DES:8, 3DES:24）
	ivSize  int
}

type optFunc func(*symmetric)

// modeConfig 仅在构造期间使用，用于将选项传播到 mode 对象。
type modeConfig struct {
	embediv     bool
	embednonce  bool
	randomNonce bool
	aad         []byte
}

func EmbedIV() optFunc {
	return func(a *symmetric) {
		if a.cfg != nil {
			a.cfg.embediv = true
		}
	}
}

func EmbedNonce() optFunc {
	return func(a *symmetric) {
		if a.cfg != nil {
			a.cfg.embednonce = true
		}
	}
}

func WithAAD(aad []byte) optFunc {
	return func(a *symmetric) {
		if a.cfg != nil {
			a.cfg.aad = aad
		}
	}
}

func GenerateKey(algorithm string) (key, iv, nonce []byte, err error) {
	keySize, ivSize, err := BlockSize(algorithm)
	if err != nil {
		return nil, nil, nil, err
	}

	random := make([]byte, keySize+ivSize+nonceSize)
	if _, err := io.ReadFull(rand.Reader, random); err != nil {
		return nil, nil, nil, err
	}

	return random[:keySize], random[keySize : keySize+ivSize], random[keySize+ivSize:], nil
}

// BlockSize 返回算法的密钥长度和 IV 长度。未知算法返回 error。
func BlockSize(algorithm string) (blockSize, ivSize int, err error) {
	c, ok := supported[algorithm]
	if !ok {
		return 0, 0, fmt.Errorf("unsupported algorithm: %s", algorithm)
	}

	return c.keySize, c.ivSize, nil
}

func NewCipher(algorithm string, key []byte) (Cipher, error) {
	c, ok := supported[algorithm]
	if !ok {
		return nil, fmt.Errorf("unsupported algorithm: %s", algorithm)
	}

	block, err := c.new(key)
	if err != nil {
		return nil, err
	}

	return &symmetric{block: block, creator: c}, nil
}

type symmetric struct {
	block   cipher.Block
	creator *creator
	cfg     *modeConfig // 仅构造期间使用，构造后置 nil
}

func (a *symmetric) Block() cipher.Block {
	return a.block
}

// BlockSize 返回块大小（=16），不是密钥长度。
func (a *symmetric) BlockSize() int {
	return a.block.BlockSize()
}

func (a *symmetric) IVSize() int {
	return a.creator.ivSize
}

// applyOpts 在构造期间应用选项，返回收集到的配置。
func (a *symmetric) applyOpts(opts []optFunc) *modeConfig {
	cfg := &modeConfig{}
	a.cfg = cfg
	for _, opt := range opts {
		opt(a)
	}
	a.cfg = nil
	return cfg
}

// --- StreamCipher ---

type streamCipher struct {
	stream cipher.Stream
}

func (a *symmetric) NewCTR(iv []byte) StreamCipher {
	return &streamCipher{stream: cipher.NewCTR(a.block, iv)}
}

func (s *streamCipher) XORKeyStream(src []byte) []byte {
	dst := make([]byte, len(src))
	s.stream.XORKeyStream(dst, src)
	return dst
}

func (s *streamCipher) Stream(reader io.Reader) io.Reader {
	return cipher.StreamReader{S: s.stream, R: reader}
}

// --- GCM ---

func (a *symmetric) NewGCM(nonce []byte, opts ...optFunc) (CipherMode, error) {
	gcm, err := cipher.NewGCM(a.block)
	if err != nil {
		return nil, err
	}

	cfg := a.applyOpts(opts)

	return &algo_gcm{
		block:       a.block,
		gcm:         gcm,
		nonce:       nonce,
		embednonce:  cfg.embednonce,
		randomNonce: cfg.randomNonce,
		aad:         cfg.aad,
	}, nil
}

func (a *symmetric) NewGCMWithRandomNonce() (CipherMode, error) {
	gcm, err := cipher.NewGCM(a.block)
	if err != nil {
		return nil, err
	}

	return &algo_gcm{
		block:       a.block,
		gcm:         gcm,
		embednonce:  true,
		randomNonce: true,
	}, nil
}

type algo_gcm struct {
	block       cipher.Block
	gcm         cipher.AEAD
	nonce       []byte
	embednonce  bool
	randomNonce bool
	aad         []byte
}

func (a *algo_gcm) NonceSize() int {
	return a.gcm.NonceSize()
}

func (a *algo_gcm) Encrypt(plainText []byte) (bytesconv.BytesResult, error) {
	nonce := make([]byte, a.gcm.NonceSize())
	if a.randomNonce || len(a.nonce) == 0 {
		if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
			return nil, err
		}
	} else {
		copy(nonce, a.nonce)
	}

	if a.embednonce {
		return a.gcm.Seal(nonce, nonce, plainText, a.aad), nil
	}
	return a.gcm.Seal(nil, nonce, plainText, a.aad), nil
}

func (a *algo_gcm) Decrypt(ciphertext []byte) (bytesconv.BytesResult, error) {
	if a.embednonce {
		ns := a.gcm.NonceSize()
		if len(ciphertext) < ns {
			return nil, errors.New("ciphertext too short for embedded nonce")
		}
		nonce, cipherText := ciphertext[:ns], ciphertext[ns:]
		return a.gcm.Open(nil, nonce, cipherText, a.aad)
	}
	return a.gcm.Open(nil, a.nonce, ciphertext, a.aad)
}

// --- CBC ---

func (a *symmetric) NewCBC(iv []byte, opts ...optFunc) (CipherMode, error) {
	if len(iv) != a.BlockSize() {
		return nil, errors.New("iv length is not equal to block size")
	}

	cfg := a.applyOpts(opts)

	if cfg.aad != nil {
		return nil, errors.New("WithAAD 仅支持 GCM")
	}

	return &algo_cbc{
		block:   a.block,
		creator: a.creator,
		iv:      iv,
		embediv: cfg.embediv,
	}, nil
}

type algo_cbc struct {
	block   cipher.Block
	creator *creator
	iv      []byte
	embediv bool
	aad     []byte
}

func (a *algo_cbc) Encrypt(plainText []byte) (bytesconv.BytesResult, error) {
	plainText = pkcs7Padding(a.block, plainText)
	stream := cipher.NewCBCEncrypter(a.block, a.iv)

	if a.embediv {
		bs := a.block.BlockSize()
		cipherText := make([]byte, len(plainText)+bs)
		copy(cipherText, a.iv)
		stream.CryptBlocks(cipherText[bs:], plainText)
		return cipherText, nil
	}

	cipherText := make([]byte, len(plainText))
	stream.CryptBlocks(cipherText, plainText)
	return cipherText, nil
}

func (a *algo_cbc) Decrypt(ciphertext []byte) (bytesconv.BytesResult, error) {
	if a.embediv {
		bs := a.block.BlockSize()
		if len(ciphertext) < bs {
			return nil, errors.New("ciphertext too short for embedded IV")
		}
		iv, cipherText := ciphertext[:bs], ciphertext[bs:]
		stream := cipher.NewCBCDecrypter(a.block, iv)
		stream.CryptBlocks(cipherText, cipherText)
		return pkcs7UnPadding(a.block, cipherText)
	}

	stream := cipher.NewCBCDecrypter(a.block, a.iv)
	stream.CryptBlocks(ciphertext, ciphertext)
	return pkcs7UnPadding(a.block, ciphertext)
}

// --- ECB ---

func (a *symmetric) NewECB() (CipherMode, error) {
	return &algo_ecb{
		block:   a.block,
		creator: a.creator,
	}, nil
}

type algo_ecb struct {
	block   cipher.Block
	creator *creator
}

func (a *algo_ecb) Encrypt(plainText []byte) (bytesconv.BytesResult, error) {
	plainText = pkcs7Padding(a.block, plainText)
	dst := make([]byte, len(plainText))
	a.block.Encrypt(dst, plainText)
	return dst, nil
}

func (a *algo_ecb) Decrypt(cipherText []byte) (bytesconv.BytesResult, error) {
	dst := make([]byte, len(cipherText))
	a.block.Decrypt(dst, cipherText)
	return pkcs7UnPadding(a.block, dst)
}

// --- PKCS7 padding ---

func pkcs7Padding(block cipher.Block, src []byte) []byte {
	bs := block.BlockSize()
	padding := bs - len(src)%bs
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(src, padtext...)
}

func pkcs7UnPadding(block cipher.Block, src []byte) ([]byte, error) {
	bs := block.BlockSize()
	length := len(src)
	unpadding := int(src[length-1])
	if unpadding > bs || unpadding == 0 {
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
