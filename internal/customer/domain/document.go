package domain

import (
	"errors"
	"strings"
)

var ErrInvalidDocument = errors.New("invalid document")

type Document struct {
	value string
}

func NewDocument(value string) (Document, error) {
	digits, ok := digitsOnly(value)
	if !ok {
		return Document{}, ErrInvalidDocument
	}
	if !validCPF(digits) && !validCNPJ(digits) {
		return Document{}, ErrInvalidDocument
	}
	return Document{value: digits}, nil
}

func (d Document) String() string { return d.value }

func digitsOnly(value string) (string, bool) {
	var digits strings.Builder
	digits.Grow(len(value))
	for _, r := range value {
		if r >= '0' && r <= '9' {
			digits.WriteRune(r)
			continue
		}
		switch r {
		case '.', '-', '/', ' ', '\t', '\n', '\r':
		default:
			return "", false
		}
	}
	return digits.String(), true
}

func validCPF(value string) bool {
	if len(value) != 11 || repeatedDigits(value) {
		return false
	}
	digits := []int{}
	for _, r := range value {
		digits = append(digits, int(r-'0'))
	}
	first := cpfDigit(digits[:9], 10)
	second := cpfDigit(digits[:10], 11)
	return digits[9] == first && digits[10] == second
}

func cpfDigit(digits []int, weight int) int {
	sum := 0
	for _, digit := range digits {
		sum += digit * weight
		weight--
	}
	result := (sum * 10) % 11
	if result == 10 {
		return 0
	}
	return result
}

func validCNPJ(value string) bool {
	if len(value) != 14 || repeatedDigits(value) {
		return false
	}
	digits := []int{}
	for _, r := range value {
		digits = append(digits, int(r-'0'))
	}
	first := cnpjDigit(digits[:12], []int{5, 4, 3, 2, 9, 8, 7, 6, 5, 4, 3, 2})
	second := cnpjDigit(digits[:13], []int{6, 5, 4, 3, 2, 9, 8, 7, 6, 5, 4, 3, 2})
	return digits[12] == first && digits[13] == second
}

func cnpjDigit(digits, weights []int) int {
	sum := 0
	for i, digit := range digits {
		sum += digit * weights[i]
	}
	remainder := sum % 11
	if remainder < 2 {
		return 0
	}
	return 11 - remainder
}

func repeatedDigits(value string) bool {
	for i := 1; i < len(value); i++ {
		if value[i] != value[0] {
			return false
		}
	}
	return true
}
