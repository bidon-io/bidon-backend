package admin

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"golang.org/x/crypto/argon2"
	"strings"
)

type PasswordService struct{}

func (s *PasswordService) Generate(password string) (string, error) {
	salt, err := s.generateSalt()
	if err != nil {
		return "", err
	}

	return s.hashPassword(password, salt), nil
}

func (s *PasswordService) Compare(storedPasswordHash, password string) bool {
	parts := strings.Split(storedPasswordHash, "$")
	if len(parts) != 2 {
		return false
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[0])
	if err != nil {
		return false
	}

	hash := s.hashPassword(password, salt)
	return hash == storedPasswordHash
}

func (s *PasswordService) generateSalt() ([]byte, error) {
	salt := make([]byte, 16)
	_, err := rand.Read(salt)
	return salt, err
}

func (s *PasswordService) hashPassword(password string, salt []byte) string {
	const (
		time    = 1
		memory  = 64 * 1024
		threads = 4
		keyLen  = 32
	)

	hash := argon2.IDKey([]byte(password), salt, time, memory, threads, keyLen)
	return fmt.Sprintf("%s$%s", base64.RawStdEncoding.EncodeToString(salt), base64.RawStdEncoding.EncodeToString(hash))
}
