package util

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"golang.org/x/crypto/scrypt"
	"os"
)

type KeyInfo struct {
	Id     string `json:"id"`
	Crypto struct {
		Cipher       string `json:"cipher"`
		CipherText   string `json:"ciphertext"`
		CipherParams struct {
			IV string `json:"iv"`
		} `json:"cipherparams"`
		KDF       string `json:"kdf"`
		KDFParams struct {
			DKLen int    `json:"dklen"`
			N     int    `json:"n"`
			P     int    `json:"p"`
			R     int    `json:"r"`
			Salt  string `json:"salt"`
		} `json:"kdfparams"`
		MAC string `json:"mac"`
	} `json:"crypto"`
	Version int `json:"version"`
}

func DecryptKeystoreText(password, filename string) (string, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return "", fmt.Errorf("read file failed: %w", err)
	}

	var ks KeyInfo
	if err := json.Unmarshal(data, &ks); err != nil {
		return "", err
	}

	salt, _ := hex.DecodeString(ks.Crypto.KDFParams.Salt)
	iv, _ := hex.DecodeString(ks.Crypto.CipherParams.IV)
	ciphertext, _ := hex.DecodeString(ks.Crypto.CipherText)
	macExpected, _ := hex.DecodeString(ks.Crypto.MAC)

	dk, err := scrypt.Key([]byte(password), salt, ks.Crypto.KDFParams.N, ks.Crypto.KDFParams.R, ks.Crypto.KDFParams.P, ks.Crypto.KDFParams.DKLen)
	if err != nil {
		return "", err
	}
	encKey := dk[:16]
	macKey := dk[16:]

	// verify MAC
	mac := hmac.New(sha256.New, macKey)
	mac.Write(ciphertext)
	if !hmac.Equal(mac.Sum(nil), macExpected) {
		return "", fmt.Errorf("invalid password or tampered file")
	}

	block, err := aes.NewCipher(encKey)
	if err != nil {
		return "", err
	}
	stream := cipher.NewCTR(block, iv)
	plaintext := make([]byte, len(ciphertext))
	stream.XORKeyStream(plaintext, ciphertext)

	return string(plaintext), nil
}
