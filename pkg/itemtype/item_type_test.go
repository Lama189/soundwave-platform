package itemtype_test

import (
	"errors"
	"testing"

	"github.com/Lama189/soundwave-platform/pkg/itemtype"
)

func TestItemType_IsValid(t *testing.T) {
	tests := []struct {
		name     string
		itemType itemtype.ItemType
		want     bool
	}{
		{
			name:     "valid track",
			itemType: itemtype.Track,
			want:     true,
		},
		{
			name:     "valid album",
			itemType: itemtype.Album,
			want:     true,
		},
		{
			name:     "empty string",
			itemType: "",
			want:     false,
		},
		{
			name:     "unknown type",
			itemType: "podcast",
			want:     false,
		},
		{
			name:     "uppercase track",
			itemType: "TRACK",
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.itemType.IsValid(); got != tt.want {
				t.Errorf("ItemType.IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestItemType_String(t *testing.T) {
	if got := itemtype.Track.String(); got != "track" {
		t.Errorf("ItemType.String() = %q, want %q", got, "track")
	}
	if got := itemtype.Album.String(); got != "album" {
		t.Errorf("ItemType.String() = %q, want %q", got, "album")
	}
}

func TestParse(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    itemtype.ItemType
		wantErr bool
	}{
		{
			name:    "parse track",
			input:   "track",
			want:    itemtype.Track,
			wantErr: false,
		},
		{
			name:    "parse album",
			input:   "album",
			want:    itemtype.Album,
			wantErr: false,
		},
		{
			name:    "parse empty",
			input:   "",
			want:    "",
			wantErr: true,
		},
		{
			name:    "parse invalid",
			input:   "merch",
			want:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := itemtype.Parse(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Parse(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
			if tt.wantErr {
				if !errors.Is(err, itemtype.ErrInvalidItemType) {
					t.Errorf("expected ErrInvalidItemType, got %v", err)
				}
				return
			}
			if got != tt.want {
				t.Errorf("Parse(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}
