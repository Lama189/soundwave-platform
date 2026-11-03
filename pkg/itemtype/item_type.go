package itemtype

import (
	"errors"
	"fmt"
)

var (
	ErrInvalidItemType = errors.New("invalid item type")
)

type ItemType string

const (
	Track ItemType = "track"
	Album ItemType = "album"
)

func (t ItemType) IsValid() bool {
	switch t {
	case Track, Album:
		return true
	default:
		return false
	}
}

func (t ItemType) String() string {
	return string(t)
}

func Parse(s string) (ItemType, error) {
	t := ItemType(s)
	if !t.IsValid() {
		return "", fmt.Errorf("%w %q: must be 'track' or 'album'", ErrInvalidItemType, s)
	}
	return t, nil
}
