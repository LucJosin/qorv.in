package pagination

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"
	"uuid"

	"github.com/LucJosin/opid"
)

const maxCursorSize = 2048

type Cursor struct {
	CreatedAt time.Time
	ID        uuid.UUID
}

type cursorPayload[F any] struct {
	Version   int       `json:"version"`
	Filters   F         `json:"filters"`
	CreatedAt time.Time `json:"created_at"`
	ID        opid.ID   `json:"id"`
}

// Encode stores a domain's JSON filter fields and a record OpID. Tokens are not authorization credentials.
func Encode[F any](position Cursor, filters F, codec *opid.Codec) (string, error) {
	request := Query{
		Limit: DefaultLimit,
		After: &position,
	}
	err := request.Validate()
	if err != nil {
		return "", err
	}

	id, err := codec.Encode(position.ID)
	if err != nil {
		return "", fmt.Errorf("encoding cursor ID: %w", err)
	}

	value := cursorPayload[F]{
		Version:   1,
		Filters:   filters,
		CreatedAt: position.CreatedAt.UTC(),
		ID:        id,
	}

	data, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(data), nil
}

func decode[F any](value string, filters F, codec *opid.Codec) (Cursor, error) {
	if len(value) > maxCursorSize {
		return Cursor{}, errors.New("invalid cursor")
	}

	data, err := base64.RawURLEncoding.Strict().DecodeString(value)
	if err != nil {
		return Cursor{}, err
	}

	var c cursorPayload[F]
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	err = decoder.Decode(&c)
	if err != nil {
		return Cursor{}, err
	}
	err = decoder.Decode(new(any))
	if err != io.EOF {
		return Cursor{}, errors.New("cursor must contain one object")
	}

	expectedFilters, err := json.Marshal(filters)
	if err != nil {
		return Cursor{}, err
	}
	actualFilters, err := json.Marshal(c.Filters)
	if err != nil {
		return Cursor{}, err
	}
	filtersMatch := bytes.Equal(actualFilters, expectedFilters)
	if c.Version != 1 || !filtersMatch {
		return Cursor{}, errors.New("cursor version or filters mismatch")
	}
	id, err := codec.Decode(c.ID)
	if err != nil {
		return Cursor{}, err
	}

	position := Cursor{
		CreatedAt: c.CreatedAt,
		ID:        id,
	}
	return position, nil
}
