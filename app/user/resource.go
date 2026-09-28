package user

import (
	"api-kms/utils"
)

type Resource struct {
	ID          string  `json:"id"`
	FullName    string  `json:"full_name"`
	FirstName   string  `json:"first_name"`
	LastName    *string `json:"last_name"`
	Email       string  `json:"email"`
	MarketingId *string `json:"marketing_id"`
	// Whether the account is allowed in at all.
	MembershipStatus string  `json:"membership_status"`
	DeactivatedAt    *string `json:"deactivated_at"`
	CreatedAt        string  `json:"created_at"`
	Roles            []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"roles"`
	// Flattened across every role, deduped. Requires Roles.Permissions to be
	// preloaded (AuthMiddleware does this) — nil otherwise, same as Roles
	// would be empty.
	Permissions []string `json:"permissions"`
}

func NewResourceCollection(users []User) []*Resource {
	result := make([]*Resource, len(users))
	for i := range users {
		result[i] = NewResource(&users[i])
	}
	return result
}

func NewResource(user *User) *Resource {
	if user == nil {
		return nil
	}

	fullName := user.FirstName
	if user.LastName != nil && *user.LastName != "" {
		fullName += " " + *user.LastName
	}
	roles := make([]struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}, len(user.Roles))
	for i, r := range user.Roles {
		roles[i].ID = r.ID.String()
		roles[i].Name = r.Name
	}
	seenPermissions := make(map[string]bool)
	permissions := make([]string, 0)
	for _, role := range user.Roles {
		for _, p := range role.Permissions {
			if !seenPermissions[p.Name] {
				seenPermissions[p.Name] = true
				permissions = append(permissions, p.Name)
			}
		}
	}

	return &Resource{
		ID:               user.ID.String(),
		FullName:         fullName,
		FirstName:        user.FirstName,
		LastName:         user.LastName,
		Email:            user.Email,
		MarketingId:      user.MarketingId,
		MembershipStatus: string(user.MembershipStatus),
		DeactivatedAt:    utils.FormatDateTime(user.DeactivatedAt),
		Roles:            roles,
		Permissions:      permissions,
		CreatedAt:        utils.FormatDateTimeVal(user.CreatedAt),
	}
}
