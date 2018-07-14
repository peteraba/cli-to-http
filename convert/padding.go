package convert

import "bytes"

type Padder interface {
	Add(origContent []byte) []byte
	Remove(paddedContent []byte) []byte
}

type PKCS5 struct {
	BlockSize int
}

func (p PKCS5) Add(origContent []byte) []byte {
	origLength := len(origContent)
	paddingLength := p.BlockSize - (origLength % p.BlockSize)
	padding := bytes.Repeat([]byte{byte(paddingLength)}, paddingLength)
	return append(origContent, padding...)
}

func (p PKCS5) Remove(paddedContent []byte) []byte {
	paddedLength := len(paddedContent)
	paddingLength := int(paddedContent[paddedLength-1])
	return paddedContent[:(paddedLength - paddingLength)]
}
