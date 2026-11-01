package artist

import "github.com/google/uuid"

type BecomeArtistInput struct {
	UserID uuid.UUID
	Name   string
	Bio    string
}
