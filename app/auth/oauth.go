package auth

import (
	"api-kms/config"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/endpoints"
)

// GoogleOauthConfig uses the pre-built Google endpoint (fixed, public URLs
// that never vary by environment) and a confidential client (client secret),
// the standard "Sign in with Google" server-side flow. Scope is limited to email: that is all completeLogin
// needs to match against an existing user, and nothing here ever creates one
// from it.
func GoogleOauthConfig() *oauth2.Config {
	return &oauth2.Config{
		ClientID:     config.AppConfig.OauthGoogleClientID,
		ClientSecret: config.AppConfig.OauthGoogleClientSecret,
		RedirectURL:  config.AppConfig.OauthGoogleRedirectURL,
		Scopes:       []string{"https://www.googleapis.com/auth/userinfo.email"},
		Endpoint:     endpoints.Google,
	}
}
