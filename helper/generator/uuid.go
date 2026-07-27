package helpergenerator

import (
	"crypto/rand"
	"math/big"
	"time"
)

const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func randomCode(length int) string {
	bytes := make([]byte, length)
	for i := 0; i < length; i++ {
		num, _ := rand.Int(rand.Reader, big.NewInt(int64(len(letters))))
		bytes[i] = letters[num.Int64()]
	}
	return string(bytes)
}

func UUIDGen(sufix string) string {

	UUID := randomCode(5)

	date := time.Now().Format("020106")

	return sufix + "-" + date + "-" + UUID
}
