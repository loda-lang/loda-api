package util

import (
	"fmt"
	"strconv"
)

type UID struct {
	domain byte
	number int64
}

func NewUID(domain byte, number int64) (UID, error) {
	if domain < 'A' || domain > 'Z' {
		return UID{}, fmt.Errorf("invalid UID domain: %c", domain)
	}
	if number < 0 || number > 999999 {
		return UID{}, fmt.Errorf("invalid UID number: %d", number)
	}
	return UID{domain: domain, number: number}, nil
}

func NewUIDFromString(s string) (UID, error) {
	if len(s) < 2 || len(s) > 10 {
		return UID{}, fmt.Errorf("invalid UID length: %d", len(s))
	}
	domain := s[0]
	number, err := strconv.ParseInt(s[1:], 10, 64)
	if err != nil {
		return UID{}, fmt.Errorf("invalid UID number: %v", err)
	}
	return NewUID(domain, number)
}

func (u UID) Domain() byte {
	return u.domain
}

func (u UID) Number() int64 {
	return u.number
}

func (u UID) String() string {
	return fmt.Sprintf("%c%06d", u.domain, u.number)
}

func (u UID) Equals(other UID) bool {
	return u.domain == other.domain && u.number == other.number
}

func (u UID) IsLessThan(other UID) bool {
	if u.domain != other.domain {
		return u.domain < other.domain
	}
	return u.number < other.number
}

func (u UID) IsGreaterThan(other UID) bool {
	if u.domain != other.domain {
		return u.domain > other.domain
	}
	return u.number > other.number
}

func (u UID) IsZero() bool {
	return (u.domain == 0 || u.domain == 'A') && u.number == 0
}
