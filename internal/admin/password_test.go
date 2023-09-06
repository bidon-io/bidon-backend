package admin

import "testing"

func TestPasswordService_Generate(t *testing.T) {
	service := &PasswordService{}
	password := "testPassword"
	hashedPassword, err := service.Generate(password)

	if err != nil {
		t.Fatalf("Expected no error, but got: %v", err)
	}

	if hashedPassword == "" {
		t.Fatalf("Expected hashed password, but got empty string")
	}
}

func TestPasswordService_Compare(t *testing.T) {
	service := &PasswordService{}
	password := "testPassword"
	hashedPassword, _ := service.Generate(password)

	if !service.Compare(hashedPassword, password) {
		t.Fatalf("Expected passwords to match, but they didn't")
	}

	if service.Compare(hashedPassword, "wrongPassword") {
		t.Fatalf("Expected passwords not to match, but they did")
	}
}

func TestPasswordService_GenerateSalt(t *testing.T) {
	service := &PasswordService{}
	salt, err := service.generateSalt()

	if err != nil {
		t.Fatalf("Expected no error, but got: %v", err)
	}

	if len(salt) != 16 {
		t.Fatalf("Expected salt length to be 16, but got: %d", len(salt))
	}
}
