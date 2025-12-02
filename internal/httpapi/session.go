package httpapi

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

type sessionPayload struct {
	Shop string `json:"shop"`
	Exp  int64  `json:"exp"` // unix seconds
}

func signSession(shop, secret string, ttl time.Duration) (string, error) {
	p := sessionPayload{
		Shop: shop,
		Exp:  time.Now().Add(ttl).Unix(),
	}
	b, err := json.Marshal(p)
	if err != nil {
		return "", err
	}

	payload := base64.RawURLEncoding.EncodeToString(b)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(payload))
	sig := hex.EncodeToString(mac.Sum(nil))

	return payload + "." + sig, nil
}

func verifySession(value, secret string) (string, error) {
	parts := strings.Split(value, ".")
	if len(parts) != 2 {
		return "", errors.New("invalid session format")
	}
	payload, sigHex := parts[0], parts[1]

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(payload))
	expected := mac.Sum(nil)

	got, err := hex.DecodeString(sigHex)
	if err != nil {
		return "", errors.New("invalid session signature")
	}
	if !hmac.Equal(expected, got) {
		return "", errors.New("invalid session signature")
	}

	raw, err := base64.RawURLEncoding.DecodeString(payload)
	if err != nil {
		return "", errors.New("invalid session payload")
	}

	var p sessionPayload
	if err := json.Unmarshal(raw, &p); err != nil {
		return "", errors.New("invalid session payload")
	}
	if time.Now().Unix() > p.Exp {
		return "", errors.New("session expired")
	}
	return p.Shop, nil
}
