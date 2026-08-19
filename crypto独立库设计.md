# crypto 回迁 go-misc 设计（github.com/charlienet/go-misc/crypto）

> 目标：将 filestore 的 `internal/pkg/crypto`（演进版分叉）合并回 go-misc 仓库的 `crypto` 包（同源主体），以 go-misc/crypto 为唯一归宿，统一提供完整加解密能力与高层 API 包装。本文档为签名级设计，可直接照做实现。

## 1. 背景与目标

### 1.1 现状（已调研确认）

- **filestore `internal/pkg/crypto`（B 库）**：纯加密原语库，零内部依赖、零业务耦合；SM4/AES/DES/3DES（CBC/GCM/CTR）+ SM2/RSA + 流式分块 AEAD（fsb1）；全仓库 8 处引用
- **go-misc `crypto`（A 库）**：与 B 库**同源已分叉**，是这套代码的原乡；已有 asymmetric.go、symmetric.go、rsa.go、sm2.go、hash.go、crypto_test.go
- 两边能力互补：A 有 hash.go（含国密 SM3）、NewECB、Cipher.BlockSize()/IVSize()、sync.Pool、Signer 接口拆分、bytesconv 通用组件；B 有 block_aead 流式加密、Cipher.Block()、initErr 延迟错误、RSA 弱密钥校验、EmbedIV/EmbedNonce、BytesResult.Open()、Name()、更全的测试
- go-misc go.mod 为 `go 1.26` 且**已含 `github.com/tjfoc/gmsm v1.4.1`**：合并零新增外部依赖
- B 库调用方（kms/store/migrate）**不使用 Asymmetric 的 Sign/Verify，也不调用 BytesResult.String()**：Signer 拆分与 String() 语义差异对主项目零影响

### 1.2 目标

- 以 go-misc/crypto 为基线，把 B 库的安全演进与流式 AEAD 能力并入，消灭分叉
- 保留 go-misc 全部现有能力（hash.go、NewECB、BlockSize/IVSize、Signer、bytesconv）
- 补齐统一 API：高层一行式 Encrypt/Decrypt、gcx1 自描述信封、GCM AAD 支持、算法常量
- 修正接口级缺陷：panic 清零（未知算法、随机数失败改为 error 返回）

### 1.3 非目标

- 不做 KMS（密钥生命周期、版本轮换、EncryptedKey 序列化）——二期再评估
- 不新建独立仓库（放 go-misc 的理由见决策记录）

### 1.4 决策记录

| 决策 | 结论 | 理由 |
|---|---|---|
| 归宿 | **回迁合并进 go-misc/crypto** | go-misc/crypto 已存在同源代码，单独开仓会造成两处 crypto 并存；内部使用场景下独立 semver 收益有限 |
| 合并基线 | go-misc/crypto（A 库）为主，并入 B 库能力 | A 是公开库主体；B 的安全演进（initErr、弱密钥校验、block_aead）全部并入 |
| 包名 | `crypto`（保持） | 使用方默认标识符不变 |
| BytesResult | 统一 `bytesconv.BytesResult`；`Open()` 方法补进 bytesconv；**String() 保持 Hex 语义不动** | 消除双类型；String() 改动会影响 go-misc 其它包，且当前无调用方依赖（B 语义为原文，列为已知陷阱警告） |
| Cipher.Block() | A 的 Cipher 接口加 `Block() cipher.Block`，symmetric 直接持有 block，**移除 sync.Pool** | block_aead 必需 Block()；块密码构造是一次性成本，池化收益可忽略 |
| 构造错误模式 | 采用 B 的 initErr 延迟错误（`new_sm2/new_rsa` 为未导出函数，无外部影响） | B 更优：构造不失败，错误延迟到操作时传播 |
| DES/3DES | **保留**（用户决策） | 独立库对兼容旧数据有用，文档强警告 |
| NewECB | **保留**（A 库已有） | 与 DES/3DES 同策略：保留 + 文档强警告"不安全"；不进高层 API |
| Signer | 保留 A 的独立 Signer 接口，Asymmetric 嵌入 Signer | 关注点分离更优；B 调用方不使用 Sign/Verify，零影响 |
| XORKeyStream | 统一返回 `[]byte`（B 语义） | 更符合标准库习惯 |
| Name() | 加入 Asymmetric 接口（rsa_algo/sm2_algo 已实现） | B 已有实现，接口补全 |
| Go 版本 | go-misc go.mod 已是 `go 1.26`，无需调整 | 覆盖 B 库 `range over int`（1.22）特性 |

## 2. 合并后 go-misc/crypto 目录结构

```
go-misc/crypto/
├── asymmetric.go     # A 基线 + initErr 模式（B）+ Name() 入接口（B）
├── symmetric.go      # A 基线 + Block() 方法（B）+ 恢复 EmbedIV/EmbedNonce（B）
├── rsa.go            # A 基线 + initErr + 弱密钥校验 + nil 防护（B）
├── sm2.go            # A 基线 + initErr + nil 防护（B）
├── hash.go           # A 保留（原样不动）
├── block_aead.go     # 【B 迁入】分块 GCM 流式 AEAD（fsb1），依赖 Cipher.Block()
├── envelope.go       # 【新增】gcx1 信封格式 + 高层 Encrypt/Decrypt/EncryptWithAAD/DecryptWithAAD
├── alg.go            # 【新增】算法常量 + NormalizeAlgorithm
├── doc.go            # 【新增】包文档：安全模型、算法推荐顺序、已知陷阱（String()/ECB/DES）
├── crypto_test.go    # 合并两边测试（A 的 testify 用例并入，B 的用例保留）
├── rsa_test.go       # 【B 迁入】含弱密钥拒绝、PSS 互通测试
├── sm2_test.go       # 【B 迁入】
├── block_aead_test.go# 【B 迁入】含篡改/重排/截断全套
└── envelope_test.go  # 【新增】
go-misc/bytesconv/
└── byte_result.go    # 追加 Open() io.Reader 方法（B 能力，纯新增无破坏）
```

依赖：`github.com/tjfoc/gmsm v1.4.1` 已在 go-misc go.mod（第 16 行），**零新增依赖**。

## 3. API 设计

### 3.1 合并后的对称接口（symmetric.go，以 A 为基线 + B 能力）

```go
type Cipher interface {
    Block() cipher.Block                              // 【B 并入】block_aead 依赖
    BlockSize() int                                   // A 保留
    IVSize() int                                      // A 保留
    NewCTR(iv []byte) StreamCipher                    // A 保留
    NewGCM(nonce []byte, opts ...optFunc) (CipherMode, error)  // A 保留，opts 扩展 WithAAD
    NewGCMWithRandomNonce() (CipherMode, error)       // 两边都有
    NewCBC(iv []byte, opts ...optFunc) (CipherMode, error)    // A 保留
    NewECB() (CipherMode, error)                      // A 保留（不安全，文档强警告）
}
type CipherMode interface {
    Encrypt(plainText []byte) (BytesResult, error)    // 【签名修正】原 A/B 均无 error，B 实现有 panic，统一为 error 返回
    Decrypt(cipherText []byte) (BytesResult, error)
}
type StreamCipher interface {
    XORKeyStream(src []byte) []byte                   // 【统一】B 语义 []byte
    Stream(reader io.Reader) io.Reader                // A 保留
}

func GenerateKey(algorithm string) (key, iv, nonce []byte, err error)  // 包级函数保留（B）
func NewCipher(algorithm string, key []byte) (Cipher, error)           // 保留
func BlockSize(algorithm string) (blockSize, ivSize int, err error)    // 【签名修正】原 panic 改 error

func EmbedIV() optFunc      // 【恢复】B 启用，A 注释了
func EmbedNonce() optFunc   // 【恢复】
func WithAAD(aad []byte) optFunc  // 【新增】仅 GCM 生效；CBC/CTR 使用返回错误
```

实现要点：

- `symmetric` 结构体直接持有 `block cipher.Block`（B 方式），**删除 A 的 sync.Pool**：Block() 暴露与池化语义冲突，且块密码构造为一次性成本，高并发收益可忽略；若未来实测有瓶颈，再加"取用/归还"专用接口，不破坏 Block()
- A 的 `supported` 注册表与 B 的合并：SM4/AES-128/AES-192/AES-256/DES/3DES 全保留，`"AES"` 重复条目删除，由 `NormalizeAlgorithm` 归一
- panic 清零：`algo_gcm.Encrypt` 随机 nonce 失败、未知算法一律 error 返回

### 3.2 合并后的非对称接口（asymmetric.go + rsa.go + sm2.go）

```go
type Signer interface {                               // A 保留
    Sign(msg []byte) (bytesconv.BytesResult, error)
    Verify(msg, sign []byte) bool
}
type Asymmetric interface {
    GenerateKey() (KeyPair, error)
    WithPrivateKey(privateKey string) error
    WithPublicKey(publicKey string) error
    ExportPublicKey() (string, error)
    Name() string                                     // 【B 并入】
    Encrypt(msg []byte) (bytesconv.BytesResult, error)
    Decrypt(ciphertext []byte) (bytesconv.BytesResult, error)
    Signer                                            // A 保留嵌入
}
type KeyPair struct { PrivateKey, PublicKey string }
func NewAsymmetric(algorithm string, keyFuncs ...keyFunc) (Asymmetric, error)
func WithPublicKey(publicKey string) keyFunc
func WithPrivateKey(privateKey string) keyFunc
```

实现要点（全部并入 B 的安全演进）：

- `rsa_algo`/`sm2_algo` 增加 `initErr error` 字段（B）：构造期解析失败延迟传播，所有操作方法先检查 `initErr` 与密钥 nil
- RSA `WithPrivateKey` 保留弱密钥校验：`BitLen < 2048` 拒绝（rsa.go:107-111，B）
- 工厂表 `supportedAsymmetricAlgorithms` 改为 B 风格：`func(opts ...keyFunc) Asymmetric`（未导出，无外部影响）

### 3.3 流式分块 AEAD（block_aead.go，B 原样迁入）

完整迁移（依赖 3.1 的 `Cipher.Block()`），API 与格式不变：

```go
const ChunkSize = 4096
const TagSize = 16
const FormatMagic = "fsb1"
var ErrCipherTooLong, ErrCipherTruncated, ErrSizeMismatch, ErrNonceOverflow, ErrInvalidBaseNonce error
func NewEncryptingReader(src io.Reader, c Cipher, baseNonce []byte, totalSize int64) (*EncryptingReader, error)
func NewDecryptingReader(src io.Reader, c Cipher, baseNonce []byte, totalSize int64) (*DecryptingReader, error)
// EncryptingReader.Length() int64 保留（不可 seek 流预知长度；注释中 S3 专属说明改为通用表述）
```

### 3.4 hash.go（A 保留，零修改）

A 的 `Hash` 枚举（MD4~SM3）与 `HashOptions.GetHash` 原样保留。注意：`HashOptions` 与 go-misc/crypto 现有使用方（如有）保持兼容，不做任何签名调整。

### 3.5 新增统一高层 API（envelope.go + alg.go）

```go
// 算法常量
const (
    AlgorithmSM4, AlgorithmAES128, AlgorithmAES192, AlgorithmAES256,
    AlgorithmDES, Algorithm3DES, AlgorithmSM2, AlgorithmRSA string
)
// 大小写不敏感归一："sm4"→"SM4"、"aes"/"AES-128"→"AES-128"；未知算法返回 error
func NormalizeAlgorithm(name string) (string, error)

// 高层一行式（固定 GCM 认证加密 + 随机 nonce，输出 gcx1 自描述信封）
func Encrypt(algorithm string, key []byte, plaintext []byte) ([]byte, error)
func Decrypt(key []byte, envelope []byte) ([]byte, error)
func EncryptWithAAD(algorithm string, key []byte, plaintext, aad []byte) ([]byte, error)
func DecryptWithAAD(key []byte, envelope, aad []byte) ([]byte, error)
```

## 4. 数据格式规范

### 4.1 fsb1 流式分块 AEAD（随迁冻结）

字节布局与语义不变（来源：B 库 block_aead.go，已在生产使用）：

```
块0: CT(4096B) || TAG(16B)    块1: ...    末块: CT(nB) || TAG(16B)
```

- 完整密文长度 = `totalSize + 16 * ceil(totalSize / 4096)`；空文件输出空流
- 块 nonce：baseNonce（12B）大端 96 位整数 + 块号，溢出返回 `ErrNonceOverflow`
- 每块 AAD（20B）：`"fsb1"(4B) || BE64(明文总长) || BE64(块号)`
- 错误：`ErrCipherTooLong` / `ErrCipherTruncated` / `ErrSizeMismatch`
- **冻结承诺**：格式随发布冻结，存量 S3 密文依赖它解密，禁止原地修改；V2 用新魔数并存

### 4.2 gcx1 高层信封（新增，随 v1.0.0 冻结）

```
offset 0:  magic(4B)      = 0x67 0x63 0x78 0x31 ("gcx1")
offset 4:  version(1B)    = 0x01
offset 5:  algID(1B)      = 0x01 SM4 | 0x02 AES-128 | 0x03 AES-192 | 0x04 AES-256 | 0x05 DES | 0x06 3DES
offset 6:  nonceLen(1B)   = 12（固定）
offset 7:  nonce(12B)
offset 19: ciphertext || TAG(16B)    // GCM Seal 输出
```

- 总开销 35B/次；未知 version/algID/nonceLen≠12 一律 error
- AAD 不入信封，由调用方持有，不一致即认证失败

### 4.3 EncryptedKey 字符串格式（不迁入）

`algorithm=,version=,key=` 属 KMS 语义，二期 kms 独立时随 kms 走。

## 5. 安全设计要点

1. **panic 清零**：合并后公开 API 一律不允许 panic（未知算法、随机数失败、池取用错误全部 error 返回）
2. **默认推荐 GCM**：高层 API 固定认证加密；CBC/CTR/ECB 仅低层按需
3. **CTR 固定 IV 禁加密**：`NewCTR` 文档保留「仅供解密旧格式，禁止加密」警告（沿用 B 库 data_key.go:49-53 注释精神，同步声明到 symmetric.go）
4. **弱密钥拒绝**：RSA < 2048 拒绝（B 能力，保留）
5. **不安全算法警告**：DES/3DES/ECB 保留但文档显著标注「不安全，仅兼容遗留数据，新代码禁用」；不进高层 API（algID 表含 DES/3DES 仅因低层存在，高层仍可用——README 明确建议只用 SM4/AES-GCM）
6. **已知陷阱声明（doc.go）**：`bytesconv.BytesResult.String()` 返回 **Hex** 而非原文（与 B 库旧语义不同，当前无调用方依赖，合并后 crypto 包内部不使用 String()）
7. **格式冻结**：fsb1、gcx1 随 v1.0.0 冻结，演进用版本号并存

## 6. 迁移指南（主仓库侧，合并完成后执行）

### 6.1 import 替换（8 处，机械替换）

`hexinpass.com/filestore/internal/pkg/crypto` → `github.com/charlienet/go-misc/crypto`：

| 文件 | 性质 |
|---|---|
| internal/kms/kms.go | 业务 |
| internal/kms/data_key.go | 业务 |
| internal/store/store.go | 业务 |
| internal/store/migrate.go | 业务 |
| internal/store/block_aead_integration_test.go | 测试 |
| internal/store/ensure_secrets_test.go | 测试 |
| internal/kms/kms_test.go | 测试 |
| internal/pkg/crypto 全部文件 | 随包迁走 |

### 6.2 调用点兼容性结论（已核验，零适配风险）

| 调用点 | 结论 |
|---|---|
| kms.go:37,228,238,243,248,274,279,292（Asymmetric） | 全兼容：Signer 嵌入不影响调用（调用方不用 Sign/Verify）；`.Bytes()` 两边都有；返回类型统一 bytesconv 后调用方只用 `[]byte` 比较/转换，隐式兼容 |
| store.go:1469（NewAsymmetric("SM2")） | 签名一致 |
| data_key.go:16,54,60,65（Cipher 引用） | 兼容：合并后 Cipher 有 Block()，DataKey 照常 |
| store.go:401,408,1337；migrate.go:188,198（block_aead/NewCTR） | 兼容：block_aead 已迁入 |
| kms.go:95,100,158（GenerateKey/NewCipher） | 兼容 |
| kms.go:243,248,279（BytesResult.Bytes()） | 兼容；**无任何调用方使用 .String()**（已核验） |

### 6.3 签名修正波及（与设计 v1 相同）

- `CipherMode.Encrypt` 带 error：store.go:1124,1153,1200,1320 + block_aead_integration_test.go:272,337,461,540,584 + kms_test.go:63,587 需补 error 分支
- 包级 `BlockSize` 带 error：kms.go:153 需补 error 分支

### 6.4 步骤

1. go-misc 仓库内完成合并（§8 路线图 M1-M3），`go build ./...` 零错误零警告、测试全绿（含 -race）
2. 主仓库 go.mod：`require github.com/charlienet/go-misc v0.0.0` + `replace github.com/charlienet/go-misc => ../go-misc`（本地灰度）
3. 替换 8 处 import + 修正 6.3 调用点 → `go build ./...` 零错误零警告
4. 删除 `internal/pkg/crypto` 目录
5. `go vet ./...`、`go test -race ./...` 全绿
6. 正式发布 go-misc tag（建议 v0.1.0），主仓库去掉 replace 转正式依赖
7. 不留 re-export 垫片：一次性切换 + 删除旧目录，避免新旧双源分叉

## 7. 测试计划

- **A 库现有测试**（crypto_test.go 316 行，testify）：AES-GCM/CBC/CTR、SM4-GCM、并发、错误路径、PKCS7 —— 保留，改为 B 的 testing 风格或保持 testify（以 go-misc 仓库惯例为准，保留 testify 即可）
- **B 库测试迁入**：crypto_test.go（SM4-CBC+EmbedIV）、rsa_test.go（PSS 互通、篡改检测、弱密钥拒绝）、sm2_test.go、block_aead_test.go（往返/篡改/重排/截断/溢出/Length/基准）
- **新增 envelope_test.go**：gcx1 往返、篡改（magic/version/algID/nonce/密文/tag）、AAD 不一致拒绝、空明文、未知 algID、-race 并发
- **新增**：NormalizeAlgorithm 大小写/别名/未知算法；Encrypt 签名修正后无 panic 测试；bytesconv.Open() 单测

## 8. 实施路线图（在 go-misc 仓库内执行）

| 阶段 | 内容 | 并行性 |
|---|---|---|
| M1 | 合并：A 基线 + B 能力并入（block_aead/initErr/弱密钥校验/EmbedIV/EmbedNonce/Name/Block()/Open() 补 bytesconv） | M1a 与 M2/M3 可并行 |
| M2 | 签名修正：Encrypt 带 error、包级 BlockSize 带 error、XORKeyStream 统一 []byte、panic 清零、AES 别名归一 | 与 M1/M3 并行 |
| M3 | 新增 envelope.go/alg.go 高层 API + WithAAD + doc.go + 测试 | 与 M1/M2 并行 |
| M4 | 主仓库切换：require+replace → 8 处 import 替换 → 6.3 调用点修正 → 删旧目录 → 全量 build/vet/test -race | 依赖 M2 完成 |
| M5 | 发布：go-misc 打 tag v0.1.0，README 增补 crypto 能力清单与格式冻结声明 | 依赖 M4 |
| M6 | （二期）kms 独立评估：依赖合并后 crypto，需验证 strings.SplitSeq（Go 1.26 无碍）与密钥持久化契约 | 串行 |

## 9. 版本与发布策略

- go-misc 无 tag 习惯：合并完成、主仓库切换成功后打 **v0.1.0**，之后 crypto 相关变更走 semver 递增
- v1.0.0 门槛：panic 清零、fsb1/gcx1 格式文档冻结、API 稳定
- 主项目灰度期用 `replace` 本地路径，正式后转伪版本或 tag 版本
- gmsm 替代评估（emmansun/gmsm）属独立演进项，不阻塞发布

## 10. 风险清单

| 风险 | 等级 | 缓解 |
|---|---|---|
| fsb1 与存量密文兼容 | 高 | 原样迁移 + block_aead_test 全套迁入 + 冻结声明 |
| String() 语义差异（Hex vs 原文） | 中 | 保持 bytesconv 原样（不动）；crypto 包内部禁用 String()；doc.go 声明陷阱；已核验主项目零调用 |
| Encrypt 签名变更编译破坏 | 中 | 一次性迁移全部调用点（§6.3 清单）+ 全量测试兜底 |
| sync.Pool 移除的性能影响 | 低 | 块密码构造一次性成本；如未来有瓶颈加专用接口，不破坏 Block() |
| ECB/DES/3DES 被误用 | 低 | 文档强警告；高层 API 明确推荐 SM4/AES-GCM |
| 合并时 A 库既有使用方（如有）被破坏 | 低 | 公开 API 仅增不改（Block()/Name() 为纯新增；Encrypt/XORKeyStream 签名修正波及面已列清单） |
| kms 二期依赖新路径 | 低 | 二期单独设计，与本期无耦合 |

## 11. 实施清单（逐文件操作）

> 前置：`git clone` go-misc 后创建合并分支。所有「参照 B」的行号均指 `/data/shangtong/filestore/internal/pkg/crypto` 或 `/tmp/opencode/go-misc/crypto` 中已核验的行号。每步完成后 `go build ./...` 零错误零警告。

### M1 合并（go-misc 仓库内）

**1. symmetric.go（核心重写，以 A 第 353 行版本为基底）**

| # | 操作 | 位置（A 库行号） | 说明 |
|---|---|---|---|
| 1.1 | `Cipher` 接口加 `Block() cipher.Block` | 23-31 | block_aead 依赖（B 能力） |
| 1.2 | `NewECB() CipherMode` → `NewECB() (CipherMode, error)` | 28 | panic 清零：原实现 pool 取 nil 会 panic |
| 1.3 | `CipherMode.Encrypt` 返回加 error | 34 | 签名修正（B 同款） |
| 1.4 | `StreamCipher.XORKeyStream` 返回 `bytesconv.BytesResult` → `[]byte` | 39 | 统一 B 语义 |
| 1.5 | 恢复 `EmbedIV()`/`EmbedNonce()` | 61-71 | 取消注释（B 已启用；algo_cbc.embediv、algo_gcm.embednonce 字段已存在） |
| 1.6 | `symmetric` 结构 `key/creator/pool` → `block cipher.Block`（+保留 creator 供 IVSize） | 112-116 | 删除 sync.Pool（B 方式） |
| 1.7 | **统一 `BlockSize()` 方法语义**：返回 `block.BlockSize()`（=16，块大小），不再返回 creator.blockSize（密钥长度） | 118-120 | A 现状：AES-192 返回 24、AES-256 返回 32，会导致 NewCBC 的 iv 长度校验（199 行）错误拒绝 16 字节 IV；B 语义（块大小 16）为正确行为，store 层生产已验证 |
| 1.8 | `IVSize()` 保持返回 creator.ivSize | 122-124 | 注册表语义 |
| 1.9 | `NewCipher`：构造时创建 block 直接持有，删 pool.New | 93-110 | |
| 1.10 | `NewCTR`：直接持有 `cipher.Stream`，删 pool | 131-142 | |
| 1.11 | 删除 `streamCipher.Reset()` | 163-164 | 死代码 |
| 1.12 | `NewGCM`/`NewGCMWithRandomNonce`：直接持有 `cipher.AEAD`，删 pool 及 New 函数内 `panic(err)` | 166-196 | panic 清零 |
| 1.13 | `algo_gcm`：加 `aad []byte` 字段 + `WithAAD(aad []byte) optFunc`；`Encrypt` 内 `io.ReadFull` 错误当前被忽略（307 行）改为返回 error；Seal/Open 第 4 参传 aad | 289-330 | B 的 panic（crypto.go:238）同源问题一并清除 |
| 1.14 | `algo_ecb`/`algo_cbc` 内 `a.pool.Get().(cipher.Block)` 改为直接用持有的 block | 219,230,246,270 | |
| 1.15 | 包级 `BlockSize`：panic → `(blockSize, ivSize int, err error)` | 84-91 | 签名修正（B 同款，kms.go:153 调用方同步改） |
| 1.16 | `supported` 注册表删除 `"AES"` 条目 | 45 | 与 AES-128 重复，由 NormalizeAlgorithm 归一（M3） |
| 1.17 | `NewGCM`/`NewCBC`/`NewECB` 的 opts 循环中应用 `WithAAD`（CBC/ECB 收到 WithAAD 返回错误） | 166-212 | 见设计 3.1 |

**2. asymmetric.go（A 基线）**

| # | 操作 | 位置 | 说明 |
|---|---|---|---|
| 2.1 | `Asymmetric` 接口加 `Name() string` | 接口声明 | B 能力（B sm2.go:45-47、rsa.go:54 有实现） |
| 2.2 | `supportedAsymmetricAlgorithms` 值类型 `func(opts ...keyFunc) (Asymmetric, error)` → `func(opts ...keyFunc) Asymmetric` | 注册表 | B 风格（initErr 延迟错误）；未导出，无外部影响；`NewAsymmetric` 导出签名不变 |

**3. rsa.go / sm2.go（A 基线 + B 安全演进）**

| # | 操作 | 参照（B 库行号） |
|---|---|---|
| 3.1 | `rsa_algo`/`sm2_algo` 加 `initErr error` 字段 | rsa.go:18、sm2.go:16 |
| 3.2 | `WithPrivateKey` 加 RSA 弱密钥校验（BitLen < 2048 拒绝） | rsa.go:107-111 |
| 3.3 | Encrypt/Decrypt/Sign/Verify/ExportPublicKey 加 initErr + nil 防护 | rsa.go:135-196、sm2.go:108-133 |
| 3.4 | `Name()` 实现 | sm2.go:45-47、rsa.go:54 |

**4. block_aead.go + block_aead_test.go（B 迁入）**

- 整体拷贝；`Length()` 注释中 S3 专属表述改为通用「不可 seek 流需预知长度」
- 依赖的 `Cipher.Block()` 已由 1.1 满足；包内无 import 变化

**5. bytesconv/byte_result.go**

- 追加：`func (r BytesResult) Open() io.Reader { return bytes.NewReader(r) }`（B 的 Open 能力；纯新增，无破坏）

**6. hash.go**：零修改。

**7. 测试合并**

| # | 操作 | 说明 |
|---|---|---|
| 7.1 | A crypto_test.go 适配签名变化 | Encrypt 带 error（34 处断言模式）、XORKeyStream 返回 []byte、BlockSize 带 error、NewECB 带 error |
| 7.2 | B crypto_test.go 的 SM4-CBC+EmbedIV 用例并入 A crypto_test.go | 补齐 EmbedIV 覆盖 |
| 7.3 | B 的 rsa_test.go / sm2_test.go / block_aead_test.go 迁入 | 同仓库同包，import 零改动；包内引用本地 `BytesResult` 处统一为 `bytesconv.BytesResult` |

### M2 签名修正验证（与 M1 同步完成，独立验证项）

- `grep -rn "panic(" crypto/`：仅允许测试中预期 panic 用例，生产代码清零
- `grep -rn "XORKeyStream" crypto/`：返回值均为 `[]byte`
- `gofmt -l .` 为空；`go vet ./...` 零警告；`go test -race ./...` 全绿

### M3 高层 API（新增文件）

| # | 文件 | 内容 |
|---|---|---|
| 3-1 | alg.go | 算法常量（AlgorithmSM4 ~ AlgorithmRSA）+ `NormalizeAlgorithm(name string) (string, error)`（大小写不敏感、AES→AES-128、未知报错） |
| 3-2 | envelope.go | gcx1 信封编解码（§4.2 字节布局）+ `Encrypt`/`Decrypt`/`EncryptWithAAD`/`DecryptWithAAD`（固定 GCM + 随机 nonce；未知 version/algID/nonceLen≠12 一律 error） |
| 3-3 | doc.go | 包文档：安全模型、算法推荐顺序（SM4/AES-GCM 优先）、陷阱声明（bytesconv.String()=Hex、ECB/DES/3DES 不安全）、fsb1/gcx1 冻结声明 |
| 3-4 | envelope_test.go | gcx1 往返、篡改（magic/version/algID/nonce/密文/tag 各字节位）、AAD 不一致拒绝、空明文、未知 algID 拒绝、-race 并发；NormalizeAlgorithm 大小写/别名/未知用例 |

### M4 主仓库切换（filestore）

1. go.mod：`require github.com/charlienet/go-misc v0.0.0` + `replace github.com/charlienet/go-misc => <本地路径>`（灰度）
2. 8 处 import 替换（§6.1 表）：`hexinpass.com/filestore/internal/pkg/crypto` → `github.com/charlienet/go-misc/crypto`
3. `Encrypt` 补 error 分支：store.go:1124,1153,1200,1320；block_aead_integration_test.go:272,337,461,540,584；kms_test.go:63,587
4. `BlockSize` 补 error 分支：kms.go:153（`keySize, ivSize := crypto.BlockSize(algorithm)` → 接收 error）
5. 删除 `internal/pkg/crypto` 目录（不留垫片）
6. 验证：`go build ./...` 零错误零警告 → `go vet ./...` → `go test -race ./...` 全绿

### M5 发布

1. go-misc：README 增补 crypto 能力清单（对称/非对称/流式 AEAD/高层 API）、fsb1+gcx1 格式冻结声明、ECB/DES/3DES 警告；打 tag `v0.1.0`
2. filestore：去掉 replace 转正式依赖（tag 或伪版本）
3. （二期）kms 独立评估：依赖合并后 crypto，验证 strings.SplitSeq（go 1.26 无碍）与密钥持久化契约（§上一版设计 3.4 注意事项）
