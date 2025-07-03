package utils

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"crypto/rand"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
)

func ComposeHash(input string, algo string, output string) (string, error) {
	// Choose hashing algorithm
	var hash []byte
	switch algo {
	case "md5":
		h := md5.Sum([]byte(input))
		hash = h[:]
	case "sha1":
		h := sha256.Sum224([]byte(input))
		hash = h[:]
	case "sha256":
		h := sha256.Sum256([]byte(input))
		hash = h[:]
	case "sha512":
		h := sha512.Sum512([]byte(input))
		hash = h[:]
	default:
		return "", fmt.Errorf("unsupported algorithm. Use one of: md5, sha1, sha256, sha512")
	}

	// Default to hex if not specified or invalid
	var result string
	switch output {
	case "base64":
		result = base64.StdEncoding.EncodeToString(hash)
	default:
		result = hex.EncodeToString(hash)
	}

	return result, nil
}

func Encrypt(text, key string) (string, error) {
	block, err := aes.NewCipher([]byte(key))
	if err != nil {
		return "", err
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, aesGCM.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	cipherText := aesGCM.Seal(nonce, nonce, []byte(text), nil)
	return base64.StdEncoding.EncodeToString(cipherText), nil
}

func Decrypt(encoded, key string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher([]byte(key))
	if err != nil {
		return "", err
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := aesGCM.NonceSize()
	if len(data) < nonceSize {
		return "", fmt.Errorf("cipherText too short")
	}

	nonce, cipherText := data[:nonceSize], data[nonceSize:]
	plaintext, err := aesGCM.Open(nil, nonce, cipherText, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}
