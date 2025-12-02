package httpapi

import (
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"net/url"
	"regexp"
	"shopify-auth-app/internal/config"
	"shopify-auth-app/internal/repository"
	"shopify-auth-app/internal/shopify"
	"time"

	"github.com/gin-gonic/gin"
)

var shopRe = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9\-]*\.myshopify\.com$`)

type Handlers struct {
	cfg       config.Config
	shopRepo  *repository.ShopRepository
	stateRepo *repository.StateRepository
}

func NewHandlers(cfg config.Config, shopRepo *repository.ShopRepository, stateRepo *repository.StateRepository) *Handlers {
	return &Handlers{
		cfg:       cfg,
		shopRepo:  shopRepo,
		stateRepo: stateRepo,
	}
}

func (h *Handlers) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handlers) Login(c *gin.Context) {
	shop := c.Query("shop")
	if shop == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "missing shop query parameter. Example: /login?shop=your-store.myshopify.com",
		})
		return
	}
	if !shopRe.MatchString(shop) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid shop domain. Must match *.myshopify.com",
		})
		return
	}

	ctx := c.Request.Context()

	_, err := h.shopRepo.GetByDomain(ctx, shop)
	if err == nil {
		//  if shhop already exists in DB redirect to dashboard
		c.Redirect(http.StatusFound, "/dashboard?shop="+url.QueryEscape(shop))
		return
	}
	if err != repository.ErrNotFound {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "database error",
		})
		return
	}

	// 2) create nonce and register to db
	nonce, err := newNonce()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate nonce"})
		return
	}

	if err := h.stateRepo.Create(ctx, shop, nonce, 10*time.Minute); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to persist oauth state"})
		return
	}

	// 3) Shopify authorize redirect to url
	authURL, err := shopify.BuildAuthorizeURL(
		shop,
		h.cfg.ShopifyAPIKey,
		h.cfg.ShopifyScopes,
		h.cfg.CallbackURL,
		nonce,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to build authorize url"})
		return
	}

	c.Redirect(http.StatusFound, authURL)
}

func (h *Handlers) OAuthCallback(c *gin.Context) {
	shop := c.Query("shop")
	code := c.Query("code")
	hmacParam := c.Query("hmac")
	state := c.Query("state")
	_ = c.Query("timestamp")

	if shop == "" || code == "" || hmacParam == "" || state == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing required parameters"})
		return
	}

	if err := shopify.ValidateHMAC(c.Request.URL.Query(), h.cfg.ShopifyAPISecret); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid hmac signature"})
		return
	}

	ctx := c.Request.Context()

	//state validation, check nonce is valid and not expred
	valid, err := h.stateRepo.Consume(ctx, shop, state)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to validate state"})
		return
	}
	if !valid {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired state parameter"})
		return
	}

	//token exchange convert authorization code to access token
	tokenResp, err := shopify.ExchangeCodeForToken(shop, h.cfg.ShopifyAPIKey, h.cfg.ShopifyAPISecret, code)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to exchange token"})
		return
	}

	//save shop to database with the access token
	_, err = h.shopRepo.Upsert(ctx, shop, tokenResp.AccessToken, tokenResp.Scope)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save shop"})
		return
	}

	c.Redirect(http.StatusFound, "/dashboard?shop="+url.QueryEscape(shop))
}

// dummy dashboard it will be change
func (h *Handlers) Dashboard(c *gin.Context) {
	shop := c.Query("shop")
	if shop == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing shop"})
		return
	}
	if !shopRe.MatchString(shop) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid shop"})
		return
	}

	ctx := c.Request.Context()
	s, err := h.shopRepo.GetByDomain(ctx, shop)
	if err == repository.ErrNotFound {
		c.JSON(http.StatusNotFound, gin.H{"error": "shop not installed"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}

	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK,
		"<h1>Dashboard (demo)</h1><p>Shop: %s</p><p>Scopes: %s</p><p>Installed: %s</p>",
		s.ShopDomain, s.Scopes, s.InstalledAt.Format(time.RFC3339),
	)
}

func newNonce() (string, error) {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
