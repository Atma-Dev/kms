package auth

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"api-kms/app/user"
	"api-kms/config"
	"api-kms/enums"
	"api-kms/errors"
	"api-kms/utils"
	"api-kms/validator"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/iqbalatma/gofortify"
)

type Handler struct{}

const (
	// Covers the 30-minute access token TTL with room to spare.
	accessTokenVerifierMaxAge = 3600
	// Matches JWT_REFRESH_TOKEN_TTL (10080 minutes); a shorter cookie would end
	// the session while the refresh token itself is still valid.
	refreshTokenMaxAge = 7 * 24 * 3600

	// dummyPasswordHash is a valid bcrypt hash of a value nobody can supply,
	// compared against when the email has no account (or no password). Without
	// it Login would skip bcrypt entirely on those paths and answer in
	// microseconds instead of ~60ms — a timing oracle for which addresses are
	// registered, which is exactly what the shared error message avoids.
	dummyPasswordHash = "$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy"
)

// setAuthCookies writes both cookies the auth flow depends on: the verifier that
// AuthMiddleware checks the access token against, and the refresh token itself.
// SameSite=None keeps them attached when the SPA calls this API cross-site; it
// requires Secure, which these cookies set in every environment but local.
// Browsers silently drop Secure cookies over plain http://localhost, so a local
// backend without TLS would otherwise never receive access_token_verifier back.
func setAuthCookies(c *gin.Context, accessTokenVerifier string, refreshToken string) {
	secure := os.Getenv("APP_ENV") != "local"
	if secure {
		c.SetSameSite(http.SameSiteNoneMode)
	} else {
		c.SetSameSite(http.SameSiteLaxMode)
	}
	c.SetCookie("access_token_verifier", accessTokenVerifier, accessTokenVerifierMaxAge, "/", "", secure, true)
	c.SetCookie("refresh_token", refreshToken, refreshTokenMaxAge, "/", "", secure, true)
}

// Logout godoc
// @Summary      Logout
// @Description  Revokes the caller's current access token (and its paired refresh token) via the JWT blacklist.
// @Tags         auth
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  utils.HTTPResponse
// @Failure      401  {object}  utils.HTTPError
// @Router       /auth/logout [post]
func (h *Handler) Logout(c *gin.Context) error {
	token := c.GetHeader("Authorization")
	gofortify.RemoveBearer(&token)
	payload, err := gofortify.Decode(token)
	if err != nil {
		return errors.InvalidAction("Failed to revoke token").WithCause(err)
	}
	gofortify.BlacklistTokenAndPairToken(payload)
	utils.ResponseJSON(c, enums.SUCCESS, "Logout successfully", nil)

	return nil
}

// Refresh godoc
// @Summary      Refresh access token
// @Description  Rotates the access/refresh token pair. Web clients send the refresh token via the httpOnly `refresh_token` cookie; mobile clients send it as a Bearer token instead (no cookie jar).
// @Tags         auth
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  utils.HTTPResponse{payload=object{data=auth.AuthenticateResource}}
// @Failure      401  {object}  utils.HTTPError
// @Router       /auth/refresh [post]
func (h *Handler) Refresh(c *gin.Context) error {
	userFromDB := c.MustGet("user").(*user.User)
	isWeb := utils.IsWebClient(c)
	accessTokenJTI := uuid.New().String()
	refreshTokenJTI := uuid.New().String()
	accessToken, atv, err := gofortify.Encode(userFromDB, gofortify.AccessToken, isWeb, c.Request.UserAgent(), accessTokenJTI, refreshTokenJTI)
	if err != nil {
		return err
	}

	refreshToken, _, err := gofortify.Encode(userFromDB, gofortify.RefreshToken, isWeb, c.Request.UserAgent(), refreshTokenJTI, accessTokenJTI)
	if err != nil {
		return err
	}

	// Web: without this the new access token has no matching verifier and the
	// very next request would 401 again, and the rotated refresh token would be
	// lost. Mobile has no cookie jar to rely on, so both tokens travel in the
	// body below instead (NewAuthenticateResource already includes both).
	if isWeb {
		setAuthCookies(c, atv, refreshToken)
	}

	utils.ResponseJSON(c, enums.SUCCESS, "Refresh Token successfully", NewAuthenticateResource(userFromDB, accessToken, refreshToken))

	return nil
}

// DevLogin godoc
// @Summary      Dev-only login bypass
// @Description  Issues a token pair for the given email with no password. Only enabled when APP_ENV=local; returns 404 otherwise.
// @Tags         auth
// @Produce      json
// @Param        email  query     string  true  "User email"
// @Success      200    {object}  utils.HTTPResponse
// @Failure      404    {object}  utils.HTTPResponse  "not local env, or user not found"
// @Router       /auth/dev-login [get]
func (h *Handler) DevLogin(c *gin.Context) error {
	if os.Getenv("APP_ENV") != "local" {
		utils.ResponseJSON(c, enums.ERR_NOT_FOUND, "Not found", nil)
		return nil
	}

	email := c.Query("email")
	if email == "" {
		utils.ResponseJSON(c, enums.ERR_BAD_REQUEST, "email query param is required", nil)
		return nil
	}

	var userFromDB user.User
	if err := config.DB.Where("email = ?", email).First(&userFromDB).Error; err != nil {
		utils.ResponseJSON(c, enums.ERR_NOT_FOUND, "User not found", nil)
		return nil
	}

	accessToken, refreshToken, err := issueTokens(c, &userFromDB)
	if err != nil {
		return err
	}

	// Dev-login is local-only tooling, so returning the refresh token here too
	// (regardless of client type) is harmless and saves testing mobile's
	// bearer-only refresh path from having to go through a real login.
	utils.ResponseJSON(c, enums.SUCCESS, "Dev login successful", gin.H{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
	})
	return nil
}

// WSToken godoc
// @Summary      Issue a WebSocket-only access token
// @Description  Issues a short-lived token scoped to WebSocket routes only (/notifications/ws, /management/audits/ws) — it carries no cookie requirement, so it does not depend on access_token_verifier crossing a different subdomain during the WS handshake, which browsers do not reliably do. AuthMiddleware refuses this token on every other route; it is otherwise usable nowhere else. Requires the same auth as any protected route (Bearer + access_token_verifier cookie for web clients).
// @Tags         auth
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  utils.HTTPResponse{payload=object{data=object{access_token=string}}}
// @Failure      401  {object}  utils.HTTPError
// @Router       /auth/ws-token [post]
func (h *Handler) WSToken(c *gin.Context) error {
	userFromDB := c.MustGet("user").(*user.User)

	jti := uuid.New().String()
	// No paired refresh token exists for this one-off credential, so pti has
	// nothing real to reference — a fresh id keeps the shape gofortify
	// expects without implying a pair that isn't there.
	pti := uuid.New().String()

	wsToken, _, err := gofortify.Encode(userFromDB, gofortify.AccessToken, false, c.Request.UserAgent(), jti, pti)
	if err != nil {
		return err
	}

	ttl := time.Duration(gofortify.Config.AccessTokenTTL) * time.Minute
	if err := utils.RegisterWSOnlyToken(c.Request.Context(), jti, ttl); err != nil {
		return errors.InternalServerError("failed to register ws token").WithCause(err)
	}

	utils.ResponseJSON(c, enums.SUCCESS, "WS token issued", gin.H{
		"access_token": wsToken,
	})
	return nil
}

// Login godoc
// @Summary      Log in with email and password
// @Description  Issues a JWT pair for a local account. For a web client both tokens also land in httpOnly cookies (access_token_verifier + refresh_token); a non-web client gets the refresh token in the body instead. A wrong email and a wrong password return the same 401 on purpose — telling them apart would let anyone enumerate registered addresses.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        payload  body      auth.LoginRequest  true  "Credentials"
// @Success      200  {object}  utils.HTTPResponse{payload=object{data=object{access_token=string,refresh_token=string}}}
// @Failure      401  {object}  utils.HTTPError
// @Failure      422  {object}  utils.HTTPError
// @Router       /auth/login [post]
func (h *Handler) Login(c *gin.Context) error {
	var request LoginRequest
	if err := validator.BindAndValidate(c, &request); err != nil {
		return err
	}

	var currentUser user.User
	err := config.DB.Where("email = ?", request.Email).First(&currentUser).Error

	// One failure path for "no such user", "account has no password set" and
	// "wrong password": three distinct answers would tell a caller which
	// addresses exist. CheckHash still runs against a dummy hash when the row
	// is missing so a rejection costs the same bcrypt work either way — a
	// fast "no" is its own oracle.
	hashed := dummyPasswordHash
	if err == nil && currentUser.Password != nil {
		hashed = *currentUser.Password
	}
	if !utils.CheckHash(hashed, request.Password) || err != nil || currentUser.Password == nil {
		return errors.InvalidCredential("Email or password is incorrect")
	}

	// Membership is checked only after the password proved the caller owns
	// this account, so an inactive one is a message they are entitled to.
	if currentUser.MembershipStatus != user.MembershipStatusActive {
		return errors.InvalidCredential("This account has been deactivated")
	}

	accessToken, refreshToken, err := issueTokens(c, &currentUser)
	if err != nil {
		return err
	}

	payload := gin.H{"access_token": accessToken}
	// Web already has the refresh token in its httpOnly cookie; repeating it
	// in the body would put it somewhere JavaScript can read.
	if !utils.IsWebClient(c) {
		payload["refresh_token"] = refreshToken
	}

	utils.ResponseJSON(c, enums.SUCCESS, "Login successful", payload)
	return nil
}

// GoogleRedirect godoc
// @Summary      Start Google login
// @Description  Redirects the browser to Google's consent screen. This is a confidential client (client secret, no PKCE) — the standard server-side "Sign in with Google" flow.
// @Tags         auth
// @Success      302  {string}  string  "redirect to Google"
// @Router       /auth/google/redirect [get]
func (h *Handler) GoogleRedirect(c *gin.Context) error {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return err
	}
	state := hex.EncodeToString(b)

	c.SetCookie("oauth_state", state, 300, "/", "", false, true)

	authURL := GoogleOauthConfig().AuthCodeURL(state)
	c.Redirect(http.StatusFound, authURL)
	return nil
}

// GoogleCallback godoc
// @Summary      Google login callback
// @Description  Backend-only callback: exchanges the authorization code for a Google token, resolves the local user by email, issues a Go JWT pair, and redirects to FRONTEND_URL/auth/callback with the access token (or an error reason). Never creates a user, and a Google account only ever proves an email — it does not itself grant access. See completeLogin.
// @Tags         auth
// @Param        code   query  string  true  "authorization code from Google"
// @Param        state  query  string  true  "state from GoogleRedirect, echoed back by Google"
// @Success      302  {string}  string  "redirect to FRONTEND_URL/auth/callback"
// @Router       /auth/google/callback [get]
func (h *Handler) GoogleCallback(c *gin.Context) error {
	savedState, err := c.Cookie("oauth_state")
	if err != nil || savedState != c.Query("state") {
		redirectToFrontend(c, "", "invalid_state")
		return nil
	}

	if errParam := c.Query("error"); errParam != "" {
		redirectToFrontend(c, "", errParam)
		return nil
	}

	code := c.Query("code")
	if code == "" {
		redirectToFrontend(c, "", "missing_code")
		return nil
	}

	token, err := GoogleOauthConfig().Exchange(c.Request.Context(), code)
	if err != nil {
		redirectToFrontend(c, "", "token_exchange_failed")
		return nil
	}

	googleUser, err := fetchGoogleUser(token.AccessToken)
	if err != nil {
		redirectToFrontend(c, "", "fetch_user_failed")
		return nil
	}

	email, _ := googleUser["email"].(string)
	completeLogin(c, email)
	return nil
}

// resolveMember resolves an email to an existing, active local user. Login
// never creates one. Unknown email and an inactive membership get distinct
// reasons — a caller reaching this point already proved a real identity via
// the provider's own login, so telling them apart is a support/UX question.
// Callers decide how the rejection surfaces (redirect reason vs HTTPError).
//
// Password login does NOT go through this: there the email is the thing being
// proved, so distinguishing "no such account" from "wrong password" would be
// an enumeration oracle. See Login.
func resolveMember(email string) (*user.User, string) {
	var currentUser user.User
	if email == "" || config.DB.Where("email = ?", email).First(&currentUser).Error != nil {
		return nil, "no_account"
	}

	if currentUser.MembershipStatus != user.MembershipStatusActive {
		return nil, "membership_inactive"
	}

	return &currentUser, ""
}

// issueTokens mints the access/refresh pair for an authenticated user and,
// for a web client, writes both cookies. Every login path ends here — Login,
// DevLogin and completeLogin — so the cookie rules and the jti pairing live
// in exactly one place.
func issueTokens(c *gin.Context, currentUser *user.User) (accessToken string, refreshToken string, err error) {
	isWeb := utils.IsWebClient(c)
	accessTokenJTI := uuid.New().String()
	refreshTokenJTI := uuid.New().String()

	accessToken, atv, err := gofortify.Encode(currentUser, gofortify.AccessToken, isWeb, c.Request.UserAgent(), accessTokenJTI, refreshTokenJTI)
	if err != nil {
		return "", "", err
	}

	refreshToken, _, err = gofortify.Encode(currentUser, gofortify.RefreshToken, isWeb, c.Request.UserAgent(), refreshTokenJTI, accessTokenJTI)
	if err != nil {
		return "", "", err
	}

	if isWeb {
		setAuthCookies(c, atv, refreshToken)
	}
	return accessToken, refreshToken, nil
}

// completeLogin is GoogleCallback's tail: resolve the member, issue a token
// pair, and redirect — or redirect with a rejection reason.
func completeLogin(c *gin.Context, email string) {
	currentUser, reason := resolveMember(email)
	if reason != "" {
		redirectToFrontend(c, "", reason)
		return
	}

	accessToken, refreshToken, err := issueTokens(c, currentUser)
	if err != nil {
		redirectToFrontend(c, "", "issue_token_failed")
		return
	}

	// Web keeps the refresh token in its httpOnly cookie, never in the URL. The
	// mobile webview can't read that cookie once it hands off to the native app,
	// so it gets the refresh token appended to the deep-link redirect instead.
	if utils.IsWebClient(c) {
		redirectToFrontend(c, accessToken, "")
	} else {
		redirectToFrontend(c, accessToken, "", refreshToken)
	}
}

func redirectToFrontend(c *gin.Context, accessToken string, errMsg string, refreshToken ...string) {
	base := config.AppConfig.FrontendURL + "/auth/callback"
	if errMsg != "" {
		c.Redirect(http.StatusFound, base+"?error="+errMsg)
		return
	}
	url := base + "?access_token=" + accessToken
	if len(refreshToken) > 0 && refreshToken[0] != "" {
		url += "&refresh_token=" + refreshToken[0]
	}
	c.Redirect(http.StatusFound, url)
}

// googleUserInfoURL is fixed: Google's endpoints are public URLs that never
// vary by environment, so there is nothing to configure.
const googleUserInfoURL = "https://www.googleapis.com/oauth2/v2/userinfo"

func fetchGoogleUser(accessToken string) (map[string]any, error) {
	req, err := http.NewRequest(http.MethodGet, googleUserInfoURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("google userinfo returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var googleUser map[string]any
	if err := json.Unmarshal(body, &googleUser); err != nil {
		return nil, err
	}

	return googleUser, nil
}

func NewHandler() *Handler {
	return &Handler{}
}
