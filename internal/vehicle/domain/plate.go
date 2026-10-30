package domain

import (
	"errors"
	"strings"
)

var ErrInvalidPlate = errors.New("invalid plate")

// Plate accepts the legacy Brazilian format (ABC1234) and the Mercosul format (ABC1D23).
type Plate struct {
	value string
}

func NewPlate(value string) (Plate, error) {
	normalized := strings.ToUpper(strings.NewReplacer("-", "", " ", "").Replace(strings.TrimSpace(value)))
	if !validLegacyPlate(normalized) && !validMercosulPlate(normalized) {
		return Plate{}, ErrInvalidPlate
	}
	return Plate{value: normalized}, nil
}

func (p Plate) String() string { return p.value }

func validLegacyPlate(value string) bool {
	return len(value) == 7 && letters(value[:3]) && digits(value[3:])
}

func validMercosulPlate(value string) bool {
	return len(value) == 7 && letters(value[:3]) && digits(value[3:4]) && letters(value[4:5]) && digits(value[5:])
}

func letters(value string) bool {
	for _, r := range value {
		if r < 'A' || r > 'Z' {
			return false
		}
	}
	return true
}

func digits(value string) bool {
	for _, r := range value {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
