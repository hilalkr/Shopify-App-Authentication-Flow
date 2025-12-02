package shopify

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/url"
	"sort"
	"strings"
	"testing"
)

func canonicalQuery(v url.Values) string {
	keys := make([]string, 0, len(v))
	for k := range v {
		if k == "hmac" || k == "signature" {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var parts []string
	for _, k := range keys {
		parts = append(parts, url.QueryEscape(k)+"="+url.QueryEscape(v.Get(k)))
	}
	return strings.Join(parts, "&")
}

func signForTest(v url.Values, secret string) string {
	msg := canonicalQuery(v)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(msg))
	return hex.EncodeToString(mac.Sum(nil))
}

func TestValidateHMAC_OK(t *testing.T) {
	secret := "test_secret"

	v := url.Values{}
	v.Set("shop", "test-store.myshopify.com")
	v.Set("code", "abc")
	v.Set("state", "nonce")
	v.Set("timestamp", "1700000000")

	v.Set("hmac", signForTest(v, secret))

	if err := ValidateHMAC(v, secret); err != nil {
		t.Fatalf("expected ok, got err: %v", err)
	}
}

func TestValidateHMAC_Tampered(t *testing.T) {
	secret := "test_secret"

	v := url.Values{}
	v.Set("shop", "test-store.myshopify.com")
	v.Set("code", "abc")
	v.Set("state", "nonce")
	v.Set("timestamp", "1700000000")

	v.Set("hmac", signForTest(v, secret))

	v.Set("code", "evil")

	if err := ValidateHMAC(v, secret); err == nil {
		t.Fatalf("expected error, got nil")
	}
}
