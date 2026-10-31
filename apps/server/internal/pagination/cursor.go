package pagination

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"
	"uuid"
)

var ErrInvalidCursor = errors.New("invalid pagination cursor")

const maxCursorSize = 2048

type Cursor struct {
	Version   int       `json:"v"`
	CreatedAt time.Time `json:"ct"`
	ID        uuid.UUID `json:"id"`
}

func EncodeCursor(cursor Cursor) (string, error) {
	cBytes, err := json.Marshal(cursor)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(cBytes), nil
}

func DecodeCursor(cursor string) (Cursor, error) {
	cursor = strings.TrimSpace(cursor)
	if cursor == "" {
		return Cursor{}, nil
	}
	if len(cursor) > maxCursorSize {
		return Cursor{}, errors.New("invalid cursor")
	}

	data, err := base64.RawURLEncoding.Strict().DecodeString(cursor)
	if err != nil {
		return Cursor{}, err
	}

	var c Cursor
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	err = decoder.Decode(&c)
	if err != nil {
		return Cursor{}, err
	}

	return c, nil
}
