package saga

import (
	"crypto/rand"
	"encoding/hex"
)

// newSagaID generates a random 16-byte hex saga identifier.
func newSagaID() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}
