package crypto

import (
	"errors"
	"fmt"
)

// gcx1 信封格式常量。
const (
	gcx1Magic     = "gcx1"                       // 4 字节魔数
	gcx1Version   = 0x01                         // 格式版本
	gcx1NonceLen  = 12                           // 固定 nonce 长度
	gcx1HeaderLen = 7                            // magic(4) + version(1) + algID(1) + nonceLen(1)
	gcx1MinLen    = gcx1HeaderLen + gcx1NonceLen // = 19，最小合法长度
)

var (
	errGcx1TooShort         = errors.New("gcx1: 密文过短，不足 19 字节")
	errGcx1MagicMismatch    = errors.New("gcx1: 魔数不匹配，非 gcx1 格式")
	errGcx1VersionMismatch  = errors.New("gcx1: 版本号不匹配")
	errGcx1NonceLenMismatch = errors.New("gcx1: nonce 长度不为 12")
	errGcx1UnknownAlgID     = errors.New("gcx1: 未知算法标识")
	errGcx1DESNotSupported  = errors.New(
		"DES/3DES 不支持 GCM 认证加密（块大小 8 字节），" +
			"如需兼容遗留数据请使用低层 API 的 CBC/CTR 模式",
	)
)

// Encrypt 使用指定算法和密钥加密明文，输出 gcx1 自描述信封格式。
// 固定使用 GCM 认证加密 + 随机 nonce。
// 总开销 35 字节（头 19 字节 + GCM tag 16 字节）。
//
// 支持的算法：SM4、AES-128、AES-192、AES-256。
// DES/3DES 因块大小为 8 字节无法使用 GCM，返回明确错误。
func Encrypt(algorithm string, key []byte, plaintext []byte) ([]byte, error) {
	alg, err := NormalizeAlgorithm(algorithm)
	if err != nil {
		return nil, err
	}
	if alg == AlgorithmDES || alg == Algorithm3DES {
		return nil, errGcx1DESNotSupported
	}
	algID, ok := algorithmIDMap[alg]
	if !ok {
		return nil, fmt.Errorf("gcx1: 算法 %s 无对应 algID", alg)
	}

	c, err := NewCipher(alg, key)
	if err != nil {
		return nil, err
	}
	gcm, err := c.NewGCMWithRandomNonce()
	if err != nil {
		return nil, err
	}

	sealed, err := gcm.Encrypt(plaintext)
	if err != nil {
		return nil, err
	}
	return buildGcx1(algID, sealed), nil
}

// EncryptWithAAD 使用指定算法和密钥加密明文，同时附加 AAD（额外认证数据）。
// AAD 不入信封，由调用方在解密时提供相同值。
func EncryptWithAAD(algorithm string, key []byte, plaintext, aad []byte) ([]byte, error) {
	alg, err := NormalizeAlgorithm(algorithm)
	if err != nil {
		return nil, err
	}
	if alg == AlgorithmDES || alg == Algorithm3DES {
		return nil, errGcx1DESNotSupported
	}
	algID, ok := algorithmIDMap[alg]
	if !ok {
		return nil, fmt.Errorf("gcx1: 算法 %s 无对应 algID", alg)
	}

	c, err := NewCipher(alg, key)
	if err != nil {
		return nil, err
	}
	// 使用 EmbedNonce + nil nonce：nonce 为 nil 时 len(nonce)==0 触发随机生成，
	// embednonce=true 确保 Seal 输出 nonce(12B)||ciphertext||tag(16B)，
	// 与 NewGCMWithRandomNonce 行为一致，仅额外携带 AAD。
	gcm, err := c.NewGCM(nil, WithAAD(aad), EmbedNonce())
	if err != nil {
		return nil, err
	}

	sealed, err := gcm.Encrypt(plaintext)
	if err != nil {
		return nil, err
	}
	return buildGcx1(algID, sealed), nil
}

// buildGcx1 将 algID + sealed 数据组装为完整的 gcx1 信封。
// sealed 格式（由 NewGCMWithRandomNonce / embednonce=true 产生）：
//
//	nonce(12B) || ciphertext || tag(16B)
func buildGcx1(algID byte, sealed []byte) []byte {
	nonce := sealed[:gcx1NonceLen]
	payload := sealed[gcx1NonceLen:]

	envelope := make([]byte, 0, gcx1HeaderLen+len(sealed))
	envelope = append(envelope, gcx1Magic...)
	envelope = append(envelope, gcx1Version)
	envelope = append(envelope, algID)
	envelope = append(envelope, gcx1NonceLen)
	envelope = append(envelope, nonce...)
	envelope = append(envelope, payload...)
	return envelope
}

// Decrypt 解密 gcx1 信封格式的密文。
// 校验顺序：长度 → 魔数 → 版本 → nonceLen → algID → GCM 认证解密。
func Decrypt(key []byte, envelope []byte) ([]byte, error) {
	return decryptInternal(key, envelope, nil)
}

// DecryptWithAAD 解密 gcx1 信封格式的密文，同时验证 AAD。
// 调用方必须提供加密时使用的相同 AAD，否则 GCM 认证失败。
func DecryptWithAAD(key []byte, envelope, aad []byte) ([]byte, error) {
	return decryptInternal(key, envelope, aad)
}

func decryptInternal(key, envelope, aad []byte) ([]byte, error) {
	if len(envelope) < gcx1MinLen {
		return nil, errGcx1TooShort
	}
	if string(envelope[:4]) != gcx1Magic {
		return nil, errGcx1MagicMismatch
	}
	if envelope[4] != gcx1Version {
		return nil, errGcx1VersionMismatch
	}
	if envelope[6] != gcx1NonceLen {
		return nil, errGcx1NonceLenMismatch
	}

	algID := envelope[5]
	alg, ok := idAlgorithmMap[algID]
	if !ok {
		return nil, errGcx1UnknownAlgID
	}
	if alg == AlgorithmDES || alg == Algorithm3DES {
		return nil, errGcx1DESNotSupported
	}

	nonce := envelope[gcx1HeaderLen : gcx1HeaderLen+gcx1NonceLen]
	ciphertext := envelope[gcx1HeaderLen+gcx1NonceLen:]

	c, err := NewCipher(alg, key)
	if err != nil {
		return nil, err
	}

	// 构造选项：仅传 AAD（若提供），nonce 由信封提取
	var opts []optFunc
	if len(aad) > 0 {
		opts = append(opts, WithAAD(aad))
	}
	gcm, err := c.NewGCM(nonce, opts...)
	if err != nil {
		return nil, err
	}

	plaintext, err := gcm.Decrypt(ciphertext)
	if err != nil {
		return nil, fmt.Errorf("gcx1: 解密失败: %w", err)
	}
	return plaintext, nil
}
