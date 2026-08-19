package hash

import (
	"bytes"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"errors"
	"hash"
	"hash/fnv"
	"strings"

	"github.com/cespare/xxhash/v2"
	"github.com/charlienet/go-misc/bytesconv"
	"github.com/charlienet/go-misc/crypto"
	"github.com/emmansun/gmsm/sm3"
	"github.com/spaolacci/murmur3"
)

var _ crypto.Signer = &hashComparer{}

type HashFunc func([]byte) bytesconv.BytesResult

var hashFuncs = map[string]HashFunc{
	"MD5":    Md5,
	"SHA1":   Sha1,
	"SHA224": Sha224,
	"SHA256": Sha256,
	"SHA384": Sha384,
	"SHA512": Sha512,
	"SM3":    Sm3,
}

type hashComparer struct {
	hashFunc HashFunc
}

func New(fname string) (*hashComparer, error) {
	f, err := ByName(fname)
	if err != nil {
		return nil, err
	}

	return &hashComparer{
		hashFunc: f,
	}, nil
}

func (c *hashComparer) Sign(msg []byte) (bytesconv.BytesResult, error) {
	ret := c.hashFunc(msg)
	return ret.Bytes(), nil
}

func (c *hashComparer) Verify(msg, target []byte) bool {
	ret := c.hashFunc(msg)
	ret.Bytes()

	return bytes.Equal(ret.Bytes(), target)
}

func ByName(name string) (HashFunc, error) {
	if f, ok := hashFuncs[strings.ToUpper(name)]; ok {
		return f, nil
	}

	return nil, errors.New("Unsupported hash functions")
}

func Md5(msg []byte) bytesconv.BytesResult { return sum(md5.New, msg) }

func Sha1(msg []byte) bytesconv.BytesResult { return sum(sha1.New, msg) }

func Sha224(msg []byte) bytesconv.BytesResult { return sum(sha256.New224, msg) }

func Sha256(msg []byte) bytesconv.BytesResult { return sum(sha256.New, msg) }

func Sha384(msg []byte) bytesconv.BytesResult { return sum(sha512.New384, msg) }

func Sha512(msg []byte) bytesconv.BytesResult { return sum(sha512.New, msg) }

func Sm3(msg []byte) bytesconv.BytesResult { return sum(sm3.New, msg) }

func Murmur3(msg []byte) uint64 {
	return murmur3.Sum64(msg)
}

func XXhash(msg []byte) []byte {
	d := xxhash.New()
	_, _ = d.Write(msg)
	return d.Sum(nil)
}

func XXHashUint64(msg []byte) uint64 {
	h := xxhash.New()
	_, _ = h.Write(msg)
	return h.Sum64()
}

func Funv32(msg []byte) uint32 {
	h := fnv.New32()
	_, _ = h.Write(msg)
	return h.Sum32()
}

func Funv64(msg []byte) uint64 {
	h := fnv.New64()
	_, _ = h.Write(msg)
	return h.Sum64()
}

func sum(f func() hash.Hash, msg []byte) bytesconv.BytesResult {
	h := f()

	_, _ = h.Write(msg)
	return h.Sum(nil)
}
