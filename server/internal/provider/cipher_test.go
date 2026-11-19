package provider

import (
	"errors"
	"testing"
)

// TestCipherRoundTrip 校验 AES-GCM 加解密与掩码规则。
func TestCipherRoundTrip(t *testing.T) {
	secret := "deeptalk-test-secret"
	plain := "sk-test-1234567890abcdef"
	enc, err := EncryptAPIKey(secret, plain)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	if enc == plain || enc == "" {
		t.Fatal("ciphertext should not equal plaintext")
	}
	dec, err := DecryptAPIKey(secret, enc)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}
	if dec != plain {
		t.Fatalf("decrypt = %q, want %q", dec, plain)
	}
	if _, err := DecryptAPIKey("another-secret", enc); err == nil {
		t.Fatal("decrypt with wrong secret should fail")
	}
	if masked := MaskAPIKey(plain); masked != "sk-••••••••cdef" {
		t.Fatalf("mask = %q", masked)
	}
	if masked := MaskAPIKey("short"); masked != "••••••••" {
		t.Fatalf("short mask = %q", masked)
	}
}

// TestValidateBaseURL 校验开发模式与生产模式的 Base URL 规则。
func TestValidateBaseURL(t *testing.T) {
	dev := &Service{devMode: true}
	if err := dev.ValidateBaseURL("http://127.0.0.1:8000/v1"); err != nil {
		t.Fatalf("dev local should pass: %v", err)
	}
	if err := dev.ValidateBaseURL("ftp://example.com"); !errors.Is(err, ErrInvalidBaseURL) {
		t.Fatalf("unsupported scheme err = %v, want ErrInvalidBaseURL", err)
	}
	if err := dev.ValidateBaseURL("https://user:pass@example.com/v1"); !errors.Is(err, ErrInvalidBaseURL) {
		t.Fatalf("userinfo err = %v, want ErrInvalidBaseURL", err)
	}
	prod := &Service{devMode: false}
	if err := prod.ValidateBaseURL("http://api.example.com/v1"); !errors.Is(err, ErrInvalidBaseURL) {
		t.Fatalf("prod http err = %v, want ErrInvalidBaseURL", err)
	}
	if err := prod.ValidateBaseURL("https://127.0.0.1/v1"); !errors.Is(err, ErrInvalidBaseURL) {
		t.Fatalf("prod loopback err = %v, want ErrInvalidBaseURL", err)
	}
}

// TestStatus 校验卡片状态推导。
func TestStatus(t *testing.T) {
	svc := &Service{}
	cases := []struct {
		p    Provider
		want string
	}{
		{Provider{Enabled: false, Verified: true}, StatusDisabled},
		{Provider{Enabled: true, Verified: false, LastError: "x"}, StatusFailed},
		{Provider{Enabled: true, Verified: true}, StatusActive},
		{Provider{Enabled: true, Verified: false}, StatusPending},
	}
	for _, item := range cases {
		if got := svc.Status(&item.p); got != item.want {
			t.Fatalf("status = %s, want %s", got, item.want)
		}
	}
}
