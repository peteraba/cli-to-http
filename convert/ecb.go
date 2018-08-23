package convert

import "crypto/cipher"

// ECB

type ecb struct {
	b         cipher.Block
	blockSize int
}

func newECB(b cipher.Block) *ecb {
	return &ecb{
		b:         b,
		blockSize: b.BlockSize(),
	}
}

type ecbEncrypter ecb

func NewECBEncrypter(b cipher.Block) cipher.BlockMode {
	return (*ecbEncrypter)(newECB(b))
}

func (e *ecbEncrypter) BlockSize() int {
	return e.blockSize
}

func (e *ecbEncrypter) CryptBlocks(dst, src []byte) {
	if len(src)%e.blockSize != 0 {
		panic("crypto/cipher: input not full blocks")
	}
	if len(dst) < len(src) {
		panic("crypto/cipher: output smaller than input")
	}
	for len(src) > 0 {
		e.b.Encrypt(dst, src[:e.blockSize])
		src = src[e.blockSize:]
		dst = dst[e.blockSize:]
	}
}

type ecbDecrypter ecb

func NewECBDecrypter(b cipher.Block) cipher.BlockMode {
	return (*ecbDecrypter)(newECB(b))
}

func (d *ecbDecrypter) BlockSize() int {
	return d.blockSize
}

func (d *ecbDecrypter) CryptBlocks(dst, src []byte) {
	if len(src)%d.blockSize != 0 {
		panic("crypto/cipher: input not full blocks")
	}
	if len(dst) < len(src) {
		panic("crypto/cipher: output smaller than input")
	}

	for len(src) > 0 {
		d.b.Decrypt(dst, src[:d.blockSize])
		src = src[d.blockSize:]
		dst = dst[d.blockSize:]
	}
}
