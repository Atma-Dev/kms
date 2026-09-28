package user

// MembershipStatus is whether the account is allowed to use this app at all.
// Login refuses anything but Active, and it is the only flag that gates access.
type MembershipStatus string

const (
	MembershipStatusActive   MembershipStatus = "ACTIVE"
	MembershipStatusInactive MembershipStatus = "INACTIVE"
)
