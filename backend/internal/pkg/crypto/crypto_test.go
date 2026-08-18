package crypto

import (
	"testing"
)

func TestRoundTrip(t *testing.T) {
	c, err := NewCipher("test-secret")
	if err != nil {
		t.Fatalf("NewCipher: %v", err)
	}
	cases := []string{
		"plain",
		"with spaces and Ünïcödé",
		"a fairly long value with 1234567890 symbols !@#$%^&*()",
	}
	for _, in := range cases {
		enc, err := c.Encrypt(in)
		if err != nil {
			t.Fatalf("Encrypt(%q): %v", in, err)
		}
		if enc == in {
			t.Fatalf("Encrypt returned plaintext for %q", in)
		}
		out, err := c.Decrypt(enc)
		if err != nil {
			t.Fatalf("Decrypt(%q): %v", enc, err)
		}
		if out != in {
			t.Fatalf("round trip mismatch: got %q want %q", out, in)
		}
	}
}

func TestEmptyInput(t *testing.T) {
	c, _ := NewCipher("test-secret")
	enc, err := c.Encrypt("")
	if err != nil || enc != "" {
		t.Fatalf("Encrypt(\"\") = %q, %v; want empty", enc, err)
	}
	out, err := c.Decrypt("")
	if err != nil || out != "" {
		t.Fatalf("Decrypt(\"\") = %q, %v; want empty", out, err)
	}
}

func TestNonceRandomization(t *testing.T) {
	c, _ := NewCipher("test-secret")
	a, _ := c.Encrypt("same input")
	b, _ := c.Encrypt("same input")
	if a == b {
		t.Fatal("two encryptions of the same input produced identical ciphertext (nonce not random)")
	}
}

func TestTamperDetection(t *testing.T) {
	c, _ := NewCipher("test-secret")
	enc, _ := c.Encrypt("secret payload")

	raw := []byte(enc)
	if raw[len(raw)-1] == 'A' {
		raw[len(raw)-1] = 'B'
	} else {
		raw[len(raw)-1] = 'A'
	}
	if _, err := c.Decrypt(string(raw)); err == nil {
		t.Fatal("tampered ciphertext was accepted")
	}
}

func TestWrongKey(t *testing.T) {
	c1, _ := NewCipher("key-one")
	c2, _ := NewCipher("key-two")
	enc, _ := c1.Encrypt("data")
	if _, err := c2.Decrypt(enc); err == nil {
		t.Fatal("decrypt with wrong key succeeded")
	}
}

func TestGarbageCiphertext(t *testing.T) {
	c, _ := NewCipher("test-secret")
	if _, err := c.Decrypt("!!!not-base64!!!"); err == nil {
		t.Fatal("garbage ciphertext accepted")
	}
	if _, err := c.Decrypt("short"); err == nil {
		t.Fatal("too-short ciphertext accepted")
	}
}
