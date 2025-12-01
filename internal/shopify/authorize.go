package shopify

import (
	"fmt"
	"net/url"
)

func BuildAuthorizeURL(shopDomain, apiKey, scopes, redirectURI, state string) (string, error) {
	u := url.URL{
		Scheme: "https",
		Host:   shopDomain,
		Path:   "/admin/oauth/authorize",
	}

	q := u.Query()
	q.Set("client_id", apiKey)
	q.Set("scope", scopes)
	q.Set("redirect_uri", redirectURI)
	q.Set("state", state)
	u.RawQuery = q.Encode()

	if shopDomain == "" || apiKey == "" || scopes == "" || redirectURI == "" || state == "" {
		return "", fmt.Errorf("missing required oauth input")
	}
	return u.String(), nil
}
