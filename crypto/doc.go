// Package crypto 提供对称加密、非对称加密、哈希及高层信封 API。
//
// # 安全模型
//
// 高层 API（Encrypt/Decrypt/EncryptWithAAD/DecryptWithAAD）固定使用 GCM
// 认证加密算法，每次加密生成随机 nonce，输出 gcx1 自描述信封格式。
// 所有公开 API 均不 panic，错误通过 error 返回。
//
// # 算法推荐顺序
//
// 推荐使用 SM4 或 AES-GCM（高层 API 的默认模式）：
//   - SM4/AES-GCM：认证加密，安全且高效，首选方案。
//   - CBC：仅用于兼容旧数据格式，新代码不应使用。
//   - CTR：仅用于解密旧格式数据，**禁止用于加密**（nonce 复用会导致明文泄露）。
//   - ECB/DES/3DES：不安全算法，仅兼容遗留数据，新代码**禁用**。
//
// # 已知陷阱
//
//   - bytesconv.BytesResult.String() 返回 Hex 编码字符串，而非原文。
//     如需原文，请使用 []byte 强制转换或 BytesResult.Open() 读取。
//   - DES/3DES 块大小为 8 字节，无法使用 GCM 认证加密。
//     高层 API 对 DES/3DES 返回明确错误。
//   - ECB 模式无随机性，相同明文块产生相同密文块，不推荐使用。
//
// # 格式冻结声明
//
//   - fsb1（流式分块 AEAD）和 gcx1（高层信封）格式随 v1.0.0 发布冻结。
//   - 格式演进通过版本号并存实现，禁止原地修改已冻结格式。
//
// # fsb1 baseNonce 唯一性职责
//
// 使用 fsb1 流式加密时，调用方**必须**保证 baseNonce 在密钥生命周期内唯一。
// nonce 复用将导致 GCM 认证失效，可能泄露明文。
package crypto
