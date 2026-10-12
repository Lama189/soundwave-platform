package http

import "github.com/google/uuid"

type BecomeArtistRequest struct {
	Name string `json:"name"`
	Bio  string `json:"bio"`
}

type UploadTrackRequest struct {
	Title    string     `json:"title"`
	FileName string     `json:"file_name"`
	AlbumID  *uuid.UUID `json:"album_id,omitempty"`
}
