package track

import "github.com/google/uuid"

type UploadTrackInput struct {
	UserID   uuid.UUID
	AlbumID  *uuid.UUID
	Title    string
	FileName string
}
