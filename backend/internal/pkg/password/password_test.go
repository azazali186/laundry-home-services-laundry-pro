package password

import "testing"

func TestHashAndVerify(t *testing.T) {
	hash, err := Hash("S3cure@Pass", bcryptMinCost)
	if err != nil {
		t.Fatalf("Hash: %v", err)
	}
	if hash == "" || hash == "S3cure@Pass" {
		t.Fatal("Hash must not be empty or plaintext")
	}
	if !Verify(hash, "S3cure@Pass") {
		t.Fatal("Verify should accept the correct password")
	}
	if Verify(hash, "wrong-password") {
		t.Fatal("Verify must reject a wrong password")
	}
}

func TestHashCost(t *testing.T) {
	// Cost 4 is the minimum bcrypt allows; higher costs are slower, so use
	// the minimum for the correctness check and default elsewhere.
	if _, err := Hash("pw", bcryptMinCost); err != nil {
		t.Fatalf("Hash(min cost): %v", err)
	}
	if _, err := Hash("pw", 0); err != nil {
		t.Fatalf("Hash(default cost): %v", err)
	}
}

func TestVerifyInvalidHash(t *testing.T) {
	if Verify("not-a-bcrypt-hash", "pw") {
		t.Fatal("Verify must return false for a malformed hash")
	}
	if Verify("", "pw") {
		t.Fatal("Verify must return false for an empty hash")
	}
}

const bcryptMinCost = 4
