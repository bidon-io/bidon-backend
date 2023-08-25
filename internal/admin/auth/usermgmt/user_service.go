package usermgmt

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"

	"github.com/bidon-io/bidon-backend/internal/db"
)

type UserService struct {
	db *db.DB
}

func NewUserService(db *db.DB) *UserService {
	return &UserService{db: db}
}

func (s *UserService) CreateUser(email, password string) (*db.User, error) {
	user := &db.User{
		Email: email,
	}

	err := s.SetPassword(user, password)
	if err != nil {
		return nil, err
	}

	err = s.db.Create(user).Error
	if err != nil {
		return nil, err
	}

	return user, nil
}

func (s *UserService) GetUserByEmail(email string) (*db.User, error) {
	var user db.User
	query := s.db.Where("email = ?", email)
	if err := query.First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (s *UserService) SetPassword(user *db.User, password string) error {
	salt, err := s.generateSalt()
	if err != nil {
		return err
	}

	user.Password = s.hashPassword(password, salt)
	return nil
}

func (s *UserService) ComparePassword(storedPasswordHash, password string) bool {
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

func (s *UserService) generateSalt() ([]byte, error) {
	salt := make([]byte, 16)
	_, err := rand.Read(salt)
	return salt, err
}

func (s *UserService) hashPassword(password string, salt []byte) string {
	const (
		time    = 1
		memory  = 64 * 1024
		threads = 4
		keyLen  = 32
	)

	hash := argon2.IDKey([]byte(password), salt, time, memory, threads, keyLen)
	return fmt.Sprintf("%s$%s", base64.RawStdEncoding.EncodeToString(salt), base64.RawStdEncoding.EncodeToString(hash))
}
