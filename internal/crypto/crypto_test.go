package crypto

import "testing"

func TestGenerateKeypair(t *testing.T) {
	kp, err := GenerateKeypair()
	if err != nil {
		t.Fatalf("GenerateKeypair: %v", err)
	}
	if kp.PublicKey == "" || kp.PrivateKey == "" {
		t.Fatal("empty key material")
	}
	if kp.PublicKey == kp.PrivateKey {
		t.Fatal("public and private keys must differ")
	}
}

func TestGenerateTokenUnique(t *testing.T) {
	a, err := GenerateToken()
	if err != nil {
		t.Fatalf("GenerateToken: %v", err)
	}
	b, _ := GenerateToken()
	if a == b {
		t.Fatal("tokens should be unique")
	}
	if len(a) < 40 {
		t.Fatalf("token too short: %d chars", len(a))
	}
}
