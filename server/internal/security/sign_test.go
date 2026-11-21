package security

import (
	"errors"
	"strconv"
	"testing"
	"time"
)

func signedRequest(t *testing.T, v *Verifier, keyID, method, path, query, body string, ts int64) (string, string, string) {
	t.Helper()
	nonce := GenerateNonce()
	sign := v.Sign(keyID, method, path, query, body, ts, nonce)
	return nonce, sign, ""
}

func TestVerifyOK(t *testing.T) {
	v := NewVerifier("secret-key")
	keyID := NewKeyID()
	ts := time.Now().Unix()
	nonce := GenerateNonce()
	sign := v.Sign(keyID, "POST", "/api/v1/c/conversations", "", `{"content":"hi"}`, ts, nonce)
	if err := v.Verify(keyID, "POST", "/api/v1/c/conversations", "", []byte(`{"content":"hi"}`), itoa(ts), nonce, sign); err != nil {
		t.Fatalf("Verify: %v", err)
	}
}

func TestVerifyReplay(t *testing.T) {
	v := NewVerifier("secret-key")
	keyID := NewKeyID()
	ts := time.Now().Unix()
	nonce := GenerateNonce()
	sign := v.Sign(keyID, "POST", "/x", "", "body", ts, nonce)
	if err := v.Verify(keyID, "POST", "/x", "", []byte("body"), itoa(ts), nonce, sign); err != nil {
		t.Fatalf("first: %v", err)
	}
	if err := v.Verify(keyID, "POST", "/x", "", []byte("body"), itoa(ts), nonce, sign); !errors.Is(err, ErrReplay) {
		t.Fatalf("second err = %v, want ErrReplay", err)
	}
}

func TestVerifyExpired(t *testing.T) {
	v := NewVerifier("secret-key")
	keyID := NewKeyID()
	ts := time.Now().Add(-10 * time.Minute).Unix()
	nonce := GenerateNonce()
	sign := v.Sign(keyID, "POST", "/x", "", "", ts, nonce)
	if err := v.Verify(keyID, "POST", "/x", "", nil, itoa(ts), nonce, sign); !errors.Is(err, ErrExpired) {
		t.Fatalf("err = %v, want ErrExpired", err)
	}
}

func TestVerifyTamperedBody(t *testing.T) {
	v := NewVerifier("secret-key")
	keyID := NewKeyID()
	ts := time.Now().Unix()
	nonce := GenerateNonce()
	sign := v.Sign(keyID, "POST", "/x", "", "original", ts, nonce)
	if err := v.Verify(keyID, "POST", "/x", "", []byte("tampered"), itoa(ts), nonce, sign); !errors.Is(err, ErrMismatch) {
		t.Fatalf("err = %v, want ErrMismatch", err)
	}
}

func TestVerifyWrongKey(t *testing.T) {
	v := NewVerifier("secret-key")
	keyID := NewKeyID()
	other := NewKeyID()
	ts := time.Now().Unix()
	nonce := GenerateNonce()
	sign := v.Sign(other, "POST", "/x", "", "", ts, nonce)
	if err := v.Verify(keyID, "POST", "/x", "", nil, itoa(ts), nonce, sign); !errors.Is(err, ErrMismatch) {
		t.Fatalf("err = %v, want ErrMismatch", err)
	}
}

func TestQueryOrderIsStable(t *testing.T) {
	v := NewVerifier("secret-key")
	keyID := NewKeyID()
	ts := time.Now().Unix()
	nonce := GenerateNonce()
	sign := v.Sign(keyID, "GET", "/x", "b=2&a=1", "", ts, nonce)
	if err := v.Verify(keyID, "GET", "/x", "a=1&b=2", nil, itoa(ts), nonce, sign); err != nil {
		t.Fatalf("Verify: %v", err)
	}
}

func TestDisabledVerifierAllows(t *testing.T) {
	v := NewVerifier("")
	if !v.Disabled() {
		t.Fatal("should be disabled")
	}
	if err := v.Verify("k", "POST", "/x", "", nil, "", "", ""); err != nil {
		t.Fatalf("disabled verifier should pass: %v", err)
	}
}

func itoa(value int64) string {
	return strconv.FormatInt(value, 10)
}
