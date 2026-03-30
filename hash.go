package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"

	"golang.org/x/crypto/argon2"
)

func main() {
	password := []byte("TempPass123!")

	salt := make([]byte, 16)
	_, _ = rand.Read(salt)

	hash := argon2.IDKey(password, salt, 1, 64*1024, 4, 32)

	b64Salt := base64.RawStdEncoding.EncodeToString(salt)
	b64Hash := base64.RawStdEncoding.EncodeToString(hash)

	fmt.Printf("$argon2id$v=19$m=65536,t=1,p=4$%s$%s\n", b64Salt, b64Hash)
}
