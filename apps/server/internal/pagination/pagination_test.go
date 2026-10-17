package pagination_test

import (
	"encoding/base64"
	"testing"

	"github.com/LucJosin/opid"
)

func newCodec(t *testing.T) *opid.Codec {
	t.Helper()

	codec, err := opid.NewCodec("col")
	if err != nil {
		t.Fatal(err)
	}

	return codec
}

func rawCursor(value string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(value))
}

// Filters supplies domain-specific values for pagination tests.
type Filters struct {
	Resource    string  `json:"resource"`
	WorkspaceID opid.ID `json:"workspace_id,omitzero"`
}
