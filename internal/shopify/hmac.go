package shopify

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/url"
	"sort"
	"strings"
)

func ValidateHMAC(queryParams url.Values, secret string) error {
	receivedHMAC := queryParams.Get("hmac")
	if receivedHMAC == "" {
		return fmt.Errorf("missing hmac parameter")
	}

	// remove hmac and signature from params because they shouldn't be included in calculation
	paramsToSign := url.Values{}
	for key, values := range queryParams {
		if key != "hmac" && key != "signature" {
			paramsToSign[key] = values
		}
	}

	keys := make([]string, 0, len(paramsToSign))
	for key := range paramsToSign {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	var pairs []string
	for _, key := range keys {
		for _, value := range paramsToSign[key] {
			pairs = append(pairs, fmt.Sprintf("%s=%s", key, value))
		}
	}
	message := strings.Join(pairs, "&")

	// Calculate HMAC-SHA256
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(message))
	calculatedHMAC := hex.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(calculatedHMAC), []byte(receivedHMAC)) {
		return fmt.Errorf("hmac validation failed: signature mismatch")
	}

	return nil
}
