package queryx_test

import (
	"errors"
	"net/url"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/lucjosin/qorv.in/internal/errs"
	"github.com/lucjosin/qorv.in/internal/queryx"
	"github.com/lucjosin/qorv.in/internal/validatorx"
)

func TestString_Values(t *testing.T) {
	empty := ""
	category := "Research Notes"
	whitespace := " Design "
	unicode := "研究"
	tests := []struct {
		name   string
		values []string
		want   *string
	}{
		{
			name: "absent",
		},
		{
			name:   "empty",
			values: []string{""},
			want:   &empty,
		},
		{
			name:   "text",
			values: []string{category},
			want:   &category,
		},
		{
			name:   "whitespace preserved",
			values: []string{whitespace},
			want:   &whitespace,
		},
		{
			name:   "unicode preserved",
			values: []string{unicode},
			want:   &unicode,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			values := url.Values{
				"category": tt.values,
			}
			parser := queryx.New(values)

			got, err := parser.String("category")
			if err != nil {
				t.Fatal(err)
			}

			want := tt.want
			if diff := cmp.Diff(want, got); diff != "" {
				t.Fatalf("value mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestString_RepeatedValues(t *testing.T) {
	tests := []struct {
		name   string
		values []string
	}{
		{
			name:   "different",
			values: []string{"Design", "Tools"},
		},
		{
			name:   "identical",
			values: []string{"Design", "Design"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			values := url.Values{
				"category": tt.values,
			}
			parser := queryx.New(values)

			got, err := parser.String("category")
			assertFieldError(t, err, "category", "must be provided once")

			var want *string
			if diff := cmp.Diff(want, got); diff != "" {
				t.Fatalf("value mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestBool_Values(t *testing.T) {
	trueValue := true
	falseValue := false
	tests := []struct {
		name   string
		values []string
		want   *bool
	}{
		{
			name: "absent",
		},
		{
			name:   "true",
			values: []string{"true"},
			want:   &trueValue,
		},
		{
			name:   "false",
			values: []string{"false"},
			want:   &falseValue,
		},
		{
			name:   "numeric true",
			values: []string{"1"},
			want:   &trueValue,
		},
		{
			name:   "numeric false",
			values: []string{"0"},
			want:   &falseValue,
		},
		{
			name:   "uppercase true",
			values: []string{"TRUE"},
			want:   &trueValue,
		},
		{
			name:   "uppercase false",
			values: []string{"FALSE"},
			want:   &falseValue,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			values := url.Values{
				"pinned": tt.values,
			}
			parser := queryx.New(values)

			got, err := parser.Bool("pinned")
			if err != nil {
				t.Fatal(err)
			}

			want := tt.want
			if diff := cmp.Diff(want, got); diff != "" {
				t.Fatalf("value mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestBool_InvalidValues(t *testing.T) {
	tests := []struct {
		name    string
		values  []string
		message string
	}{
		{
			name:    "empty",
			values:  []string{""},
			message: "must be a boolean",
		},
		{
			name:    "malformed",
			values:  []string{"yes"},
			message: "must be a boolean",
		},
		{
			name:    "whitespace",
			values:  []string{" true "},
			message: "must be a boolean",
		},
		{
			name:    "repeated different",
			values:  []string{"true", "false"},
			message: "must be provided once",
		},
		{
			name:    "repeated identical",
			values:  []string{"false", "false"},
			message: "must be provided once",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			values := url.Values{
				"pinned": tt.values,
			}
			parser := queryx.New(values)

			got, err := parser.Bool("pinned")
			assertFieldError(t, err, "pinned", tt.message)

			var want *bool
			if diff := cmp.Diff(want, got); diff != "" {
				t.Fatalf("value mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestParser_ZeroValue(t *testing.T) {
	var parser queryx.Parser

	gotString, err := parser.String("category")
	if err != nil {
		t.Fatal(err)
	}
	gotBool, err := parser.Bool("pinned")
	if err != nil {
		t.Fatal(err)
	}

	var wantString *string
	if diff := cmp.Diff(wantString, gotString); diff != "" {
		t.Fatalf("string mismatch (-want +got):\n%s", diff)
	}
	var wantBool *bool
	if diff := cmp.Diff(wantBool, gotBool); diff != "" {
		t.Fatalf("boolean mismatch (-want +got):\n%s", diff)
	}
}

func assertFieldError(t *testing.T, err error, name, message string) {
	t.Helper()

	isInvalid := errors.Is(err, errs.ErrInvalid)
	if !isInvalid {
		t.Fatalf("error = %v, want invalid input", err)
	}
	validation, ok := errors.AsType[*validatorx.ValidationError](err)
	if !ok {
		t.Fatalf("error = %v, want field validation errors", err)
	}

	want := []validatorx.FieldError{
		{
			Field:   name,
			Tag:     "invalid",
			Message: message,
		},
	}
	got := validation.Fields
	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("fields mismatch (-want +got):\n%s", diff)
	}
}
