package pagination

type Page[T any] struct {
	Items   []T
	HasMore bool
}

// NewPage removes the extra row used to detect continuation and preserves empty arrays.
func NewPage[T any](items []T, limit int) Page[T] {
	if items == nil {
		items = make([]T, 0)
	}
	hasMore := len(items) > limit
	if hasMore {
		items = items[:limit]
	}
	page := Page[T]{
		Items:   items,
		HasMore: hasMore,
	}
	return page
}
