package main

import (
	"os"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	hash, err := bcrypt.GenerateFromPassword([]byte("password123"), 10)
	if err != nil {
		panic(err)
	}
	os.WriteFile("/tmp/real_hash.txt", hash, 0644)
}
