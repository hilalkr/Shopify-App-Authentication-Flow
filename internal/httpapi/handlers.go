package httpapi

import (
	"crypto/rand"
	"encoding/base64"
	"log/slog"
	"net/http"
	"net/url"
	"regexp"
	"shopify-auth-app/internal/config"
	"shopify-auth-app/internal/repository"
	"shopify-auth-app/internal/shopify"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

var shopRe = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9\-]*\.myshopify\.com$`)

func normalizeAndValidateShop(raw string) (string, bool) {
	s := strings.ToLower(strings.TrimSpace(raw))
	return s, shopRe.MatchString(s)
}

type Handlers struct {
	cfg       config.Config
	shopRepo  *repository.ShopRepository
	stateRepo *repository.StateRepository
	log       *slog.Logger
}

func NewHandlers(cfg config.Config, shopRepo *repository.ShopRepository, stateRepo *repository.StateRepository, logger *slog.Logger) *Handlers {
	return &Handlers{
		cfg:       cfg,
		shopRepo:  shopRepo,
		stateRepo: stateRepo,
		log:       logger,
	}
}

func (h *Handlers) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handlers) Login(c *gin.Context) {
	rawShop := c.Query("shop")
	shop, ok := normalizeAndValidateShop(rawShop)
	if rawShop == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "missing shop query parameter. Example: /login?shop=your-store.myshopify.com",
		})
		return
	}
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid shop domain. Must match *.myshopify.com",
		})
		return
	}

	ctx := c.Request.Context()

	// Validate HMAC if present,this prevents unauthorized access by typing shop domain directly
	hmacParam := c.Query("hmac")
	if hmacParam != "" {
		if err := shopify.ValidateHMAC(c.Request.URL.Query(), h.cfg.ShopifyAPISecret); err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid hmac signature"})
			return
		}
	}

	_, err := h.shopRepo.GetByDomain(ctx, shop)
	if err == nil {
		// Shop exists in database
		if hmacParam != "" {
			sess, sErr := signSession(shop, h.cfg.SessionSecret, 15*time.Minute)
			if sErr != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create session"})
				h.log.Error("failed to sign session", "shop", shop, "err", sErr)
				return
			}
			c.SetSameSite(http.SameSiteLaxMode)
			c.SetCookie("app_session", sess, 900, "/", "", false, true)

			c.Redirect(http.StatusFound, "/dashboard?shop="+url.QueryEscape(shop))
			return
		}
	}
	if err != nil && err != repository.ErrNotFound {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "database error",
		})
		h.log.Error("db error in login get shop", "shop", shop, "err", err)
		return
	}

	// 2) create nonce and register to db
	nonce, err := newNonce()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate nonce"})
		h.log.Error("failed to generate nonce", "err", err)
		return
	}

	if err := h.stateRepo.Create(ctx, shop, nonce, 10*time.Minute); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to persist oauth state"})
		h.log.Error("failed to persist oauth state", "shop", shop, "nonce", nonce, "err", err)
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
		h.log.Error("failed to build authorize url", "shop", shop, "err", err)
		return
	}

	c.Redirect(http.StatusFound, authURL)
}

func (h *Handlers) OAuthCallback(c *gin.Context) {
	rawShop := c.Query("shop")
	shop, ok := normalizeAndValidateShop(rawShop)
	code := c.Query("code")
	hmacParam := c.Query("hmac")
	state := c.Query("state")
	_ = c.Query("timestamp")

	if rawShop == "" || code == "" || hmacParam == "" || state == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing required parameters"})
		return
	}
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid shop"})
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
		h.log.Error("failed to validate oauth state", "shop", shop, "state", state, "err", err)
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
		h.log.Error("token exchange failed", "shop", shop, "err", err)
		return
	}

	//save shop to database with the access token
	_, err = h.shopRepo.Upsert(ctx, shop, tokenResp.AccessToken, tokenResp.Scope)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save shop"})
		h.log.Error("failed to save shop", "shop", shop, "err", err)
		return
	}

	sess, err := signSession(shop, h.cfg.SessionSecret, 15*time.Minute)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create session"})
		h.log.Error("failed to sign session", "shop", shop, "err", err)
		return
	}

	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie("app_session", sess, 900, "/", "", false, true)

	c.Redirect(http.StatusFound, "/dashboard?shop="+url.QueryEscape(shop))
}

// dummy dashboard
func (h *Handlers) Dashboard(c *gin.Context) {
	rawShop := c.Query("shop")
	shop, ok := normalizeAndValidateShop(rawShop)
	if rawShop == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing shop"})
		return
	}
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid shop"})
		return
	}

	cookie, err := c.Cookie("app_session")
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing session"})
		return
	}

	sessShop, err := verifySession(cookie, h.cfg.SessionSecret)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid session"})
		return
	}
	if sessShop != shop {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "session-shop mismatch"})
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
		h.log.Error("db error in dashboard get shop", "shop", shop, "err", err)
		return
	}

	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK,
		"<h1>Dashboard</h1><p>Shop: %s</p><p>Scopes: %s</p><p>Installed: %s</p>",
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
