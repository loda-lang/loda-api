package shared

import (
	"fmt"
)

// OperationType represents a LODA operation type with its statistics
type OperationType struct {
	Name  string `json:"name"`
	RefId int    `json:"refId"`
	Count int    `json:"count"`
}

// List of all operation types with their ref_id as index (ref_id starts at 1)
var OperationTypeList = []string{
	"", // index 0, unused (ref_ids start at 1)
	"mov", "add", "sub", "trn", "mul", "div", "dif", "dir", "mod", "pow",
	"gcd", "lex", "bin", "fac", "log", "nrt", "dgs", "dgr", "equ", "neq",
	"leq", "geq", "min", "max", "ban", "bor", "bxo", "lpb", "lpe", "clr",
	"fil", "rol", "ror", "seq",
}

// Map for fast lookup: operation type -> ref_id (used as bit index)
var operationTypeToBit = func() map[string]uint {
	m := make(map[string]uint)
	for i, op := range OperationTypeList {
		if op != "" {
			m[op] = uint(i)
		}
	}
	return m
}()

// IsOperationType returns true if the given string is a valid operation type
func IsOperationType(s string) bool {
	_, ok := operationTypeToBit[s]
	return ok
}

// EncodeOperationTypes encodes a list of operation types into a uint64 bitmask
func EncodeOperationTypes(ops []string) (uint64, error) {
	var bits uint64
	for _, op := range ops {
		bit, ok := operationTypeToBit[op]
		if !ok {
			return 0, fmt.Errorf("unknown operation type: %s", op)
		}
		bits |= 1 << bit
	}
	return bits, nil
}

// DecodeOperationTypes decodes a uint64 bitmask into a list of operation types
func DecodeOperationTypes(bits uint64) []string {
	var result []string
	for i := 1; i < len(OperationTypeList); i++ {
		if bits&(1<<uint(i)) != 0 {
			result = append(result, OperationTypeList[i])
		}
	}
	return result
}

// HasOperationType returns true if the given operation type is present in the bits
func HasOperationType(bits uint64, op string) bool {
	bit, ok := operationTypeToBit[op]
	return ok && bits&(1<<bit) != 0
}

// HasAllOperationTypes returns true if all operation types in bits2 are present in bits1
func HasAllOperationTypes(bits1, bits2 uint64) bool {
	return bits1&bits2 == bits2
}

// HasNoOperationTypes returns true if none of the operation types in bits2 are present in bits1
func HasNoOperationTypes(bits1, bits2 uint64) bool {
	return bits1&bits2 == 0
}

// MergeOperationTypes merges two operation type bitmasks into one
func MergeOperationTypes(bits1, bits2 uint64) uint64 {
	return bits1 | bits2
}
