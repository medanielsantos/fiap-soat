package domain

import (
	"errors"
	"testing"
)

func TestNewDocument(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr error
	}{
		{name: "test_it_should_normalize_valid_cpf_when_formatted", input: "529.982.247-25", want: "52998224725"},
		{name: "test_it_should_normalize_valid_cnpj_when_formatted", input: "11.222.333/0001-81", want: "11222333000181"},
		{name: "test_it_should_reject_invalid_check_digits", input: "529.982.247-24", wantErr: ErrInvalidDocument},
		{name: "test_it_should_reject_repeated_digits", input: "111.111.111-11", wantErr: ErrInvalidDocument},
		{name: "test_it_should_reject_letters", input: "529.982.247-2A", wantErr: ErrInvalidDocument},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			document, err := NewDocument(test.input)
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("NewDocument() error = %v, want %v", err, test.wantErr)
			}
			if err == nil && document.String() != test.want {
				t.Errorf("document = %q, want %q", document.String(), test.want)
			}
		})
	}
}
