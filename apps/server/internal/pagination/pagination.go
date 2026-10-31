package pagination

const (
	// DefaultLimit is the page size when no limit is provided.
	DefaultLimit = 20
	// MaxLimit bounds the number of records returned per page.
	MaxLimit = 100
	// DefaultSort is the ordering when no sort is provided.
	DefaultSort = "created_at:desc"
)
