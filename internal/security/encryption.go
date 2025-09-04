package security

import (
	"crypto/aes"
	"crypto/cipher"
	crand "crypto/rand"
	"encoding/hex"
	"log"
	"math/rand"
	"os"
	"time"
)

var charset = "qwertyuiopasdfghjklzxcvbnmQWERTYUIOPASDFGHJKLZXCVBNM1234567890-_|!/"
var seededRand *rand.Rand = rand.New(
	rand.NewSource(time.Now().UnixNano()))

func stringWithCharset(length int64, charset string) string {
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(b)
}

type Encrypter interface {
	EncryptAES(string) string
	DecryptAES(string) ([]byte, error)
}

type AESEncrypter struct {
	Key []byte
}

func NewAESEncrypter() *AESEncrypter {
	return &AESEncrypter{Key: []byte(os.Getenv("SIMPLECI_HASH_KEY"))}
}

func (e *AESEncrypter) EncryptAES(text string) string {
	c, err := aes.NewCipher(e.Key)
	if err != nil {
		log.Fatal(err)
	}

	gcm, err := cipher.NewGCM(c)
	if err != nil {
		log.Fatal(err)
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := crand.Read(nonce); err != nil {
		log.Fatal(err)
	}

	out := gcm.Seal(nonce, nonce, []byte(text), nil)
	return hex.EncodeToString(out)
}

func (e *AESEncrypter) DecryptAES(encrypted string) ([]byte, error) {
	cipherText, err := hex.DecodeString(encrypted)
	if err != nil {
		log.Println("err decoding hex")
		return nil, err
	}

	c, err := aes.NewCipher(e.Key)
	if err != nil {
		log.Println("err new cipher")
		return nil, err
	}

	gcm, err := cipher.NewGCM(c)
	if err != nil {
		log.Println("err new gcm")
		return nil, err
	}
	if len(cipherText) < gcm.NonceSize() {
		log.Println("len cipherText < gcm.NonceSize")
	}
	nonceSize := gcm.NonceSize()
	nonce, cipherText := cipherText[:nonceSize], cipherText[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, cipherText, nil)
	if err != nil {
		log.Println("err opening gcm")
		return nil, err
	}
	return plaintext, nil
}

func NewKeys() ([]byte, []byte) {
	var hashKey []byte
	var blockKey []byte

	// check if keys are stored in env variables
	hk, hkOk := os.LookupEnv("SIMPLECI_HASH_KEY")
	bk, bkOk := os.LookupEnv("SIMPLECI_BLOCK_KEY")

	if hkOk {
		// use key from env
		hashKey = []byte(hk)
	} else {
		// generate key and store in .env
		hashKey = []byte(GenerateRandomKey(32))
		writeToDotenv("SIMPLECI_HASH_KEY", string(hashKey))
	}
	if bkOk {
		// use key from env
		blockKey = []byte(bk)
	} else {
		// generate key and store in .env
		blockKey = []byte(GenerateRandomKey(24))
		writeToDotenv("SIMPLECI_BLOCK_KEY", string(blockKey))
	}
	return hashKey, blockKey
}

func writeToDotenv(name, value string) {
	f, err := os.OpenFile(".env", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	if _, err := f.Write([]byte(name + "=" + value + "\n")); err != nil {
		log.Fatal(err)
	}
}

func GenerateRandomKey(length int64) string {
	return stringWithCharset(length, charset)
}
