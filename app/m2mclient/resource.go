package m2mclient

import "api-kms/utils"

type M2MClientResource struct {
	ID        string   `json:"id"`
	Client    string   `json:"client"`
	Scope     []string `json:"scope"`
	RevokedAt *string  `json:"revoked_at"`
	CreatedAt string   `json:"created_at"`
	UpdatedAt string   `json:"updated_at"`
}

func NewM2MClientResource(client *M2MClient) M2MClientResource {
	return M2MClientResource{
		ID:        client.ID.String(),
		Client:    client.Client,
		Scope:     client.Scope,
		RevokedAt: utils.FormatDateTime(client.RevokedAt),
		CreatedAt: utils.FormatDateTimeVal(client.CreatedAt),
		UpdatedAt: utils.FormatDateTimeVal(client.UpdatedAt),
	}
}

func NewM2MClientResourceCollection(clients []M2MClient) []M2MClientResource {
	result := make([]M2MClientResource, len(clients))
	for i := range clients {
		result[i] = NewM2MClientResource(&clients[i])
	}
	return result
}

// M2MClientCreatedResource is Store's response only: the plaintext secret is
// generated once and never recoverable after this, since only its bcrypt hash
// is persisted.
type M2MClientCreatedResource struct {
	M2MClientResource
	Secret string `json:"secret"`
}

func NewM2MClientCreatedResource(client *M2MClient, plaintextSecret string) M2MClientCreatedResource {
	return M2MClientCreatedResource{
		M2MClientResource: NewM2MClientResource(client),
		Secret:            plaintextSecret,
	}
}
