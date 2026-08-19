package hmac

import (
	"bytes"
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"errors"
	"hash"
	"strings"

	"github.com/charlienet/go-misc/bytesconv"
	"github.com/emmansun/gmsm/sm3"
)

type HMacFunc func(key, msg []byte) bytesconv.BytesResult

var hmacFuncs = map[string]HMacFunc{
	"HMACMD5":    Md5,
	"HMACSHA1":   Sha1,
	"HMACSHA224": Sha224,
	"HMACSHA256": Sha256,
	"HMACSHA384": Sha384,
	"HMACSHA512": Sha512,
	"HMACSM3":    Sm3,
}

type hashComparer struct {
	key      []byte
	hashFunc HMacFunc
}

func New(fname string, key []byte) (*hashComparer, error) {
	f, err := ByName(fname)
	if err != nil {
		return nil, err
	}

	return &hashComparer{
		key:      key,
		hashFunc: f,
	}, nil
}

func (c *hashComparer) Sign(msg []byte) (bytesconv.BytesResult, error) {
	ret := c.hashFunc(c.key, msg)
	return ret, nil
}

func (c *hashComparer) Verify(msg, target []byte) bool {
	ret := c.hashFunc(c.key, msg)

	return bytes.Equal(ret.Bytes(), target)
}

func ByName(name string) (HMacFunc, error) {
	if f, ok := hmacFuncs[strings.ToUpper(name)]; ok {
		return f, nil
	}

	return nil, errors.New("Unsupported hash function:" + name)
}

func Md5(key, msg []byte) bytesconv.BytesResult { return sum(md5.New, key, msg) }

func Sha1(key, msg []byte) bytesconv.BytesResult { return sum(sha1.New, key, msg) }

func Sha224(key, msg []byte) bytesconv.BytesResult { return sum(sha256.New224, key, msg) }

func Sha256(key, msg []byte) bytesconv.BytesResult { return sum(sha256.New, key, msg) }

func Sha384(key, msg []byte) bytesconv.BytesResult { return sum(sha512.New384, key, msg) }

func Sha512(key, msg []byte) bytesconv.BytesResult { return sum(sha512.New, key, msg) }

func Sm3(key, msg []byte) bytesconv.BytesResult { return sum(sm3.New, key, msg) }

func sum(f func() hash.Hash, key, msg []byte) bytesconv.BytesResult {
	h := hmac.New(f, key)

	h.Write(msg)
	return h.Sum(nil)
}
