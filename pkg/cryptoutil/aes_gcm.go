package cryptoutil

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
)

const aesKeySize = 32 // in bytes

func Encrypt(plaintext, secretKey []byte) ([]byte, error) {

	if len(secretKey) < aesKeySize {
		return nil, fmt.Errorf("secretKey must be atleast %d bytes", aesKeySize)
	}

	// Create a new AES cipher block from the secret key.
	block, err := aes.NewCipher(secretKey[:aesKeySize])
	if err != nil {
		return nil, err
	}

	// Wrap the cipher block in Galois Counter Mode.
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	// IMPORTANT: Never reuse a nonce with the same key. Always generate a new random nonce.
	// Create a unique nonce containing 12 random bytes.
	nonce := make([]byte, aesGCM.NonceSize())
	_, err = io.ReadFull(rand.Reader, nonce)
	if err != nil {
		return nil, err
	}

	// Encrypt the data using aesGCM.Seal(). By passing the nonce as the first
	// parameter, the encrypted data will be appended to the nonce — meaning
	// that the returned encryptedValue variable will be in the format
	// "{nonce}{encrypted plaintext data}".
	sealed := aesGCM.Seal(nonce, nonce, plaintext, nil)

	return sealed, nil
}

func Decrypt(sealed, secretKey []byte) ([]byte, error) {

	if len(secretKey) < aesKeySize {
		return nil, fmt.Errorf("secretKey must be atleast %d bytes", aesKeySize)
	}

	// Create a new AES cipher block from the secret key.
	block, err := aes.NewCipher(secretKey[:aesKeySize])
	if err != nil {
		return nil, err
	}

	// Wrap the cipher block in Galois Counter Mode.
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	// Get the nonce size.
	nonceSize := aesGCM.NonceSize()

	// To avoid a potential 'index out of range' panic in the next step, we
	// check that the length of the encrypted value is at least the nonce
	// size.
	if len(sealed) < nonceSize {
		return nil, errors.New("invalid sealed")
	}

	// Split apart the nonce from the actual encrypted data.
	nonce := sealed[:nonceSize]
	cipherText := sealed[nonceSize:]

	// Use aesGCM.Open() to decrypt and authenticate the data. If this fails,
	// return a ErrInvalidValue error.
	plaintext, err := aesGCM.Open(nil, nonce, cipherText, nil)
	if err != nil {
		return nil, err
	}

	return plaintext, nil
}
