package utils

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"github.com/denisbrodbeck/machineid"
	log "github.com/sirupsen/logrus"
	"runtime"
)

var (
	aesKey = func() []byte {
		id, err := machineid.ProtectedID("map_rly")
		if err != nil {
			log.Fatal(" machine id unknown error")
		}
		var keyLen int
		if len(id) < 32 {
			keyLen = 16
		} else {
			keyLen = 32
		}
		ret := make([]byte, keyLen)
		for i := 0; i < keyLen; i++ {
			ret[i] = id[i]
		}
		return ret
	}()
	aesIv = []byte("zxqzxqzxqzxqzxqz")
)

func AesCbcEncrypt(plainText []byte) string {
	if len(aesKey) != 16 && len(aesKey) != 24 && len(aesKey) != 32 {
		return ""
	}
	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return ""
	}
	paddingText := pkCS5Padding(plainText, block.BlockSize())

	blockMode := cipher.NewCBCEncrypter(block, aesIv)
	cipherText := make([]byte, len(paddingText))
	blockMode.CryptBlocks(cipherText, paddingText)
	return base64.StdEncoding.EncodeToString(cipherText)
}

func AesCbcDecrypt(b64Str string) []byte {
	if len(aesKey) != 16 && len(aesKey) != 24 && len(aesKey) != 32 {
		return nil
	}
	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return nil
	}
	cipherText, err := base64.StdEncoding.DecodeString(b64Str)
	if err != nil {
		log.Fatal("AesCbcDecrypt error ", err)
	}
	defer func() {
		if err := recover(); err != nil {
			switch err.(type) {
			case runtime.Error:
				log.Println("runtime err:", err, "Check that the aesKey or text is correct")
			default:
				log.Println("error:", err)
			}
		}
	}()

	blockMode := cipher.NewCBCDecrypter(block, aesIv)
	paddingText := make([]byte, len(cipherText))
	blockMode.CryptBlocks(paddingText, cipherText)

	plainText := pkCS5UnPadding(paddingText)

	return plainText
}
func pkCS5Padding(plainText []byte, blockSize int) []byte {
	padding := blockSize - (len(plainText) % blockSize)
	padText := bytes.Repeat([]byte{byte(padding)}, padding)
	newText := append(plainText, padText...)
	return newText
}
func pkCS5UnPadding(plainText []byte) []byte {
	length := len(plainText)
	number := int(plainText[length-1])
	if number > length {
		return nil
	}
	return plainText[:length-number]
}
