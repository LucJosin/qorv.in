package pagination_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/lucjosin/qorv.in/internal/pagination"
)

func TestNewPage_Boundaries(t *testing.T) {
	tests := []struct {
		name  string
		items []int
		want  pagination.Page[int]
	}{
		{
			name: "empty",
			want: pagination.Page[int]{
				Items: []int{},
			},
		},
		{
			name:  "exact limit",
			items: []int{3, 2},
			want: pagination.Page[int]{
				Items: []int{3, 2},
			},
		},
		{
			name:  "extra row",
			items: []int{3, 2, 1},
			want: pagination.Page[int]{
				Items:   []int{3, 2},
				HasMore: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := pagination.NewPage(tt.items, 2)
			want := tt.want
			if diff := cmp.Diff(want, got); diff != "" {
				t.Fatalf("page mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
