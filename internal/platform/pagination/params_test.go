package pagination

import (
	"errors"
	"math"
	"testing"
)

func TestNewParamsValidatesBounds(t *testing.T) {
	cases := []struct {
		name          string
		page, perPage int
		want          error
	}{
		{"valid", 2, 10, nil},
		{"zero page", 0, 10, ErrInvalidPage},
		{"zero per page", 1, 0, ErrInvalidPerPage},
		{"per page above max", 1, MaxPerPage + 1, ErrInvalidPerPage},
		{"offset overflow", math.MaxInt, MaxPerPage, ErrInvalidPage},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			params, err := NewParams(tc.page, tc.perPage)
			if !errors.Is(err, tc.want) {
				t.Fatalf("err = %v, want %v", err, tc.want)
			}
			if err == nil && (params.Page() != tc.page || params.PerPage() != tc.perPage) {
				t.Fatalf("params = %d/%d, want %d/%d", params.Page(), params.PerPage(), tc.page, tc.perPage)
			}
		})
	}
}

func TestParseUsesDefaultsAndRejectsInvalidInput(t *testing.T) {
	cases := []struct {
		name, page, perPage string
		wantPage, wantPer   int
		want                error
	}{
		{"defaults", "", "", DefaultPage, DefaultPerPage, nil},
		{"explicit", "3", "50", 3, 50, nil},
		{"non numeric page", "x", "", 0, 0, ErrInvalidPage},
		{"negative page", "-1", "", 0, 0, ErrInvalidPage},
		{"non numeric per page", "", "x", 0, 0, ErrInvalidPerPage},
		{"per page too large", "", "101", 0, 0, ErrInvalidPerPage},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			params, err := Parse(tc.page, tc.perPage)
			if !errors.Is(err, tc.want) {
				t.Fatalf("err = %v, want %v", err, tc.want)
			}
			if err == nil && (params.Page() != tc.wantPage || params.PerPage() != tc.wantPer) {
				t.Fatalf("params = %d/%d, want %d/%d", params.Page(), params.PerPage(), tc.wantPage, tc.wantPer)
			}
		})
	}
}

func TestZeroParamsNormalizeToDefaults(t *testing.T) {
	var params Params
	if params.Page() != DefaultPage || params.Limit() != DefaultPerPage || params.Offset() != 0 {
		t.Fatalf("zero params = page %d limit %d offset %d", params.Page(), params.Limit(), params.Offset())
	}
	params, _ = NewParams(3, 25)
	if params.Offset() != 50 || params.Limit() != 25 {
		t.Fatalf("offset/limit = %d/%d, want 50/25", params.Offset(), params.Limit())
	}
}
