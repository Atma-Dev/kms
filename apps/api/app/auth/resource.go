package auth

import (
	"fmt"
	"api-kms/app/user"
	"api-kms/utils"
)

type AuthenticateResource struct {
	Id              string        `json:"id"`
	FirstName       string        `json:"first_name"`
	LastName        *string       `json:"last_name"`
	FullName        string        `json:"full_name"`
	ActiveCompanyId interface{}   `json:"active_company_id"`
	Roles           []interface{} `json:"roles"`
	Email           string        `json:"email"`
	Tokens          struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
	} `json:"tokens"`
}

func NewAuthenticateResource(user *user.User, accessToken string, refreshToken string) *AuthenticateResource {
	return &AuthenticateResource{
		Id:        user.ID.String(),
		FirstName: user.FirstName,
		LastName:  user.LastName,
		Email:     user.Email,
		FullName:  fmt.Sprintf("%s %s", user.FirstName, utils.Deref(user.LastName)),
		Roles:     []interface{}{},
		Tokens: struct {
			AccessToken  string `json:"access_token"`
			RefreshToken string `json:"refresh_token"`
		}{
			AccessToken:  accessToken,
			RefreshToken: refreshToken,
		},
	}
}
