// Package password wraps bcrypt hashing for user credentials.
package password

import "golang.org/x/crypto/bcrypt"

// Hash hashes a plaintext password with the given cost.
func Hash(plain string, cost int) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(plain), cost)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// Verify checks a plaintext password against a bcrypt hash.
func Verify(hash, plain string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain)) == nil
}
