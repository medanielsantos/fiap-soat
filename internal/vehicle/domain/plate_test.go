package domain

import (
	"errors"
	"testing"
)

func TestNewPlate(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
		err   error
	}{
		{name: "test_it_should_normalize_legacy_plate", input: " abc-1234 ", want: "ABC1234"},
		{name: "test_it_should_accept_mercosul_plate", input: "abc1d23", want: "ABC1D23"},
		{name: "test_it_should_reject_plate_with_wrong_length", input: "ABC123", err: ErrInvalidPlate},
		{name: "test_it_should_reject_plate_with_digit_prefix", input: "1BC1234", err: ErrInvalidPlate},
		{name: "test_it_should_reject_plate_with_letter_suffix", input: "ABC1D2E", err: ErrInvalidPlate},
		{name: "test_it_should_reject_plate_with_invalid_characters", input: "ABC.1234", err: ErrInvalidPlate},
		{name: "test_it_should_reject_empty_plate", input: "", err: ErrInvalidPlate},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			plate, err := NewPlate(test.input)
			if !errors.Is(err, test.err) {
				t.Fatalf("NewPlate() error = %v, want %v", err, test.err)
			}
			if plate.String() != test.want {
				t.Errorf("plate = %q, want %q", plate.String(), test.want)
			}
		})
	}
}
