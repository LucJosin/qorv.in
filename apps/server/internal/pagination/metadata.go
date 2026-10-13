package pagination

import "github.com/LucJosin/opid"

type Metadata struct {
	Limit      int     `json:"limit"`
	HasMore    bool    `json:"has_more"`
	NextCursor *string `json:"next_cursor"`
}

// NewMetadata encodes the last item's cursor when the page has more results.
func NewMetadata[T, F any](page Page[T], query Query, filters F, codec *opid.Codec, cursorFor func(T) Cursor) (Metadata, error) {
	metadata := Metadata{
		Limit:   query.Limit,
		HasMore: page.HasMore,
	}
	if !page.HasMore || len(page.Items) == 0 {
		return metadata, nil
	}

	last := page.Items[len(page.Items)-1]
	position := cursorFor(last)
	cursor, err := Encode(position, filters, codec)
	if err != nil {
		return Metadata{}, err
	}
	metadata.NextCursor = &cursor
	return metadata, nil
}
