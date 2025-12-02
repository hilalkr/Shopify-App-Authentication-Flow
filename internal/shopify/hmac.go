package shopify

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/url"
)

func ValidateHMAC(queryParams url.Values, secret string) error {
	receivedHMAC := queryParams.Get("hmac")
	if receivedHMAC == "" {
		return fmt.Errorf("missing hmac parameter")
	}

	paramsToSign := url.Values{}
	for key, values := range queryParams {
		if key == "hmac" || key == "signature" {
			continue
		}
		for _, v := range values {
			paramsToSign.Add(key, v)
		}
	}

	message := paramsToSign.Encode()

	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(message))
	calculated := mac.Sum(nil)

	receivedBytes, err := hex.DecodeString(receivedHMAC)
	if err != nil {
		return fmt.Errorf("invalid hmac: not valid hex")
	}

	if !hmac.Equal(calculated, receivedBytes) {
		return fmt.Errorf("hmac validation failed: signature mismatch")
	}

	return nil
}
