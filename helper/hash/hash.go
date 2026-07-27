package helperhash

import "golang.org/x/crypto/bcrypt"

func HashPassword(naked string) string {
	hash, err := bcrypt.GenerateFromPassword([]byte(naked), bcrypt.DefaultCost)
	if err != nil {
		return ""
	}
	return string(hash)
}

func ComparePass(hash, naked string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(naked))
	return err == nil
}
