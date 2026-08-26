package symmetric

import "github.com/charlienet/go-misc/crypto"

// New 创建可复用对称加密对象（crypto.Encryptor 的薄包装，一行转发）。
// 构造一次绑定算法+模式+密钥+选项，之后可反复调用 Encrypt/Decrypt
// （方法级并发安全，无锁）。构造流程与错误语义详见 crypto.NewEncryptor 文档。
//
// 示例：
//
//	e, err := symmetric.New(crypto.AES128, crypto.GCM, symmetric.WithKey(key))
//	if err != nil {
//		return err
//	}
//	ct, err := e.Encrypt(plaintext)
//	pt, err := e.Decrypt(ct)
func New(alg crypto.Algorithm, mode crypto.Mode, opts ...crypto.Option) (*crypto.Encryptor, error) {
	return crypto.NewEncryptor(alg, mode, opts...)
}
