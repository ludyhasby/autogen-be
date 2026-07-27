package helpergenerator

import (
	"strings"
)

func UsernameGenerator(email string) (username string) {
	parts := strings.Split(email, "@")
	return parts[0]
}
