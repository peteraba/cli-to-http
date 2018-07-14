package convert

import (
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEncrypter(t *testing.T) {
	var encryptionFixtures = []struct {
		name                  string
		cipherType            string
		encrypterType         string
		paddingType           string
		key                   string
		text                  string
		verifiedEncryptedText string
	}{
		{
			"128bit",
			"AES",
			"ECB",
			"PKCS5",
			"abcdeABCDEabcdeA",
			"Nam a ultrices dolor. Morbi vulputate sapien aliquam, commodo enim non, tincidunt justo. Vivamus ornare vel nunc at sollicitudin.",
			"v3NTWHBgczJ/woUvbw19cdR7ZRBGBeJUkev1NoAzP7tRR9PUGI+BPupxridPC1GDUu7kbTWU9JeU3NeH0Nh1ha7lSVD4KNpezioxiXK5uoN0XCrlRUzNZvL38J/O4ldMqf9op8e9q1Cpdi2bSPYEr45yCAGpIl42GSElrnp7By7cCEx1pQcpcgha6gpmTtOU",
		},
	}
	for _, tt := range encryptionFixtures {
		t.Run(tt.name, func(t *testing.T) {
			be := NewBlockEncrypter([]byte(tt.key), tt.cipherType, tt.encrypterType, tt.paddingType)

			encryptedData := be.Encrypt([]byte(tt.text))

			encodedEncryptedData := base64.StdEncoding.EncodeToString(encryptedData)
			assert.Equal(t, tt.verifiedEncryptedText, encodedEncryptedData)

			encryptedDecryptedData := be.Decrypt(encryptedData)
			assert.Equal(t, tt.text, string(encryptedDecryptedData))
		})
	}
}
