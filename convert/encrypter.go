package convert

import (
	"crypto/aes"
	"crypto/cipher"
)

type BlockEncrypter struct {
	encrypter cipher.BlockMode
	decrypter cipher.BlockMode
	padder    Padder
}

func (be BlockEncrypter) Encrypt(data []byte) []byte {
	if be.padder != nil {
		data = be.padder.Add(data)
	}

	encryptedData := make([]byte, len(data))
	be.encrypter.CryptBlocks(encryptedData, data)

	return encryptedData
}

func (be BlockEncrypter) Decrypt(encryptedData []byte) []byte {
	data := make([]byte, len(encryptedData))
	be.decrypter.CryptBlocks(data, encryptedData)

	if be.padder != nil {
		data = be.padder.Remove(data)
	}

	return data
}

func NewBlockEncrypter(key []byte, cipherType, encrypterType, paddingType string) *BlockEncrypter {
	var (
		block     cipher.Block
		encrypter cipher.BlockMode
		decrypter cipher.BlockMode
		err       error
		padder    Padder
	)

	switch cipherType {
	case "AES":
		block, err = aes.NewCipher(key)
		if err != nil {
			panic(err)
		}
	default:
		panic("Cipher type not recognized: " + cipherType)
	}

	switch encrypterType {
	case "ECB":
		encrypter = NewECBEncrypter(block)
		decrypter = NewECBDecrypter(block)
	default:
		panic("Encrypter type not recognized: " + encrypterType)
	}

	switch paddingType {
	case "PKCS5":
		padder = PKCS5{BlockSize: block.BlockSize()}
	default:
		panic("Padding type not recognized: " + paddingType)
	}

	return &BlockEncrypter{encrypter: encrypter, decrypter: decrypter, padder: padder}
}
