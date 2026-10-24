package user

type UpdateRequest struct {
	RoleIDs     []string `form:"role_ids" json:"role_ids"`
	MarketingId *string  `form:"marketing_id" json:"marketing_id"`
}

type UpdatePasswordRequest struct {
	CurrentPassword string `form:"current_password" json:"current_password" binding:"required"`
	Password        string `form:"password" json:"password" binding:"required,min=8,nefield=CurrentPassword"`
	// eqfield keeps the "typed it twice" check server-side too — the SPA does
	// it as well, but a mismatched pair reaching here would silently set the
	// password to whichever field the API happened to read.
	PasswordConfirmation string `form:"password_confirmation" json:"password_confirmation" binding:"required,eqfield=Password"`
}
