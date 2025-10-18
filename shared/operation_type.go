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

// OperationTypeIndex provides efficient encoding/decoding of operation types to bitmasks.
// It is initialized from loaded operation type data and validates uniqueness and ref ID ranges.
type OperationTypeIndex struct {
	types          []*OperationType   // All operation types indexed by ref_id
	nameToBit      map[string]uint    // Map from operation name to bit index
	maxRefId       int                // Maximum ref_id value
}

// Global operation type index (initialized from loaded data)
var globalOperationTypeIndex *OperationTypeIndex

// NewOperationTypeIndex creates a new OperationTypeIndex from loaded operation types.
// It validates that ref IDs are unique and range from 1..N.
func NewOperationTypeIndex(operationTypes []*OperationType) (*OperationTypeIndex, error) {
	if len(operationTypes) == 0 {
		return nil, fmt.Errorf("operation types list is empty")
	}

	// Find max ref_id and validate uniqueness
	maxRefId := 0
	refIdSeen := make(map[int]bool)
	nameSeen := make(map[string]bool)
	
	for _, ot := range operationTypes {
		if ot.RefId <= 0 {
			return nil, fmt.Errorf("invalid ref_id %d for operation type %s: must be >= 1", ot.RefId, ot.Name)
		}
		if refIdSeen[ot.RefId] {
			return nil, fmt.Errorf("duplicate ref_id %d", ot.RefId)
		}
		if nameSeen[ot.Name] {
			return nil, fmt.Errorf("duplicate operation type name %s", ot.Name)
		}
		refIdSeen[ot.RefId] = true
		nameSeen[ot.Name] = true
		if ot.RefId > maxRefId {
			maxRefId = ot.RefId
		}
	}

	// Validate that ref IDs are continuous from 1..N
	for i := 1; i <= maxRefId; i++ {
		if !refIdSeen[i] {
			return nil, fmt.Errorf("missing ref_id %d: ref IDs must be continuous from 1 to %d", i, maxRefId)
		}
	}

	// Build indexed structures
	types := make([]*OperationType, maxRefId+1) // index 0 is unused, ref_ids start at 1
	nameToBit := make(map[string]uint)
	
	for _, ot := range operationTypes {
		types[ot.RefId] = ot
		nameToBit[ot.Name] = uint(ot.RefId)
	}

	idx := &OperationTypeIndex{
		types:     types,
		nameToBit: nameToBit,
		maxRefId:  maxRefId,
	}
	
	// Set global instance for convenience functions
	globalOperationTypeIndex = idx
	
	return idx, nil
}

// IsOperationType returns true if the given string is a valid operation type
func (idx *OperationTypeIndex) IsOperationType(s string) bool {
	_, ok := idx.nameToBit[s]
	return ok
}

// EncodeOperationTypes encodes a list of operation types into a uint64 bitmask
func (idx *OperationTypeIndex) EncodeOperationTypes(ops []string) (uint64, error) {
	var bits uint64
	for _, op := range ops {
		bit, ok := idx.nameToBit[op]
		if !ok {
			return 0, fmt.Errorf("unknown operation type: %s", op)
		}
		bits |= 1 << bit
	}
	return bits, nil
}

// DecodeOperationTypes decodes a uint64 bitmask into a list of operation types
func (idx *OperationTypeIndex) DecodeOperationTypes(bits uint64) []string {
	var result []string
	for i := 1; i <= idx.maxRefId; i++ {
		if bits&(1<<uint(i)) != 0 {
			result = append(result, idx.types[i].Name)
		}
	}
	return result
}

// HasOperationType returns true if the given operation type is present in the bits
func (idx *OperationTypeIndex) HasOperationType(bits uint64, op string) bool {
	bit, ok := idx.nameToBit[op]
	return ok && bits&(1<<bit) != 0
}

// HasAllOperationTypes returns true if all operation types in bits2 are present in bits1
func (idx *OperationTypeIndex) HasAllOperationTypes(bits1, bits2 uint64) bool {
	return bits1&bits2 == bits2
}

// HasNoOperationTypes returns true if none of the operation types in bits2 are present in bits1
func (idx *OperationTypeIndex) HasNoOperationTypes(bits1, bits2 uint64) bool {
	return bits1&bits2 == 0
}

// MergeOperationTypes merges two operation type bitmasks into one
func (idx *OperationTypeIndex) MergeOperationTypes(bits1, bits2 uint64) uint64 {
	return bits1 | bits2
}

// GetOperationTypes returns all operation types (excluding index 0)
func (idx *OperationTypeIndex) GetOperationTypes() []*OperationType {
	result := make([]*OperationType, 0, idx.maxRefId)
	for i := 1; i <= idx.maxRefId; i++ {
		result = append(result, idx.types[i])
	}
	return result
}

// Global convenience functions that use the global index

// IsOperationType returns true if the given string is a valid operation type (uses global index)
func IsOperationType(s string) bool {
	if globalOperationTypeIndex == nil {
		return false
	}
	return globalOperationTypeIndex.IsOperationType(s)
}

// EncodeOperationTypes encodes a list of operation types into a uint64 bitmask (uses global index)
func EncodeOperationTypes(ops []string) (uint64, error) {
	if globalOperationTypeIndex == nil {
		return 0, fmt.Errorf("operation type index not initialized")
	}
	return globalOperationTypeIndex.EncodeOperationTypes(ops)
}

// DecodeOperationTypes decodes a uint64 bitmask into a list of operation types (uses global index)
func DecodeOperationTypes(bits uint64) []string {
	if globalOperationTypeIndex == nil {
		return nil
	}
	return globalOperationTypeIndex.DecodeOperationTypes(bits)
}

// HasOperationType returns true if the given operation type is present in the bits (uses global index)
func HasOperationType(bits uint64, op string) bool {
	if globalOperationTypeIndex == nil {
		return false
	}
	return globalOperationTypeIndex.HasOperationType(bits, op)
}

// HasAllOperationTypes returns true if all operation types in bits2 are present in bits1 (uses global index)
func HasAllOperationTypes(bits1, bits2 uint64) bool {
	if globalOperationTypeIndex == nil {
		return false
	}
	return globalOperationTypeIndex.HasAllOperationTypes(bits1, bits2)
}

// HasNoOperationTypes returns true if none of the operation types in bits2 are present in bits1 (uses global index)
func HasNoOperationTypes(bits1, bits2 uint64) bool {
	if globalOperationTypeIndex == nil {
		return true
	}
	return globalOperationTypeIndex.HasNoOperationTypes(bits1, bits2)
}

// MergeOperationTypes merges two operation type bitmasks into one (uses global index)
func MergeOperationTypes(bits1, bits2 uint64) uint64 {
	if globalOperationTypeIndex == nil {
		return 0
	}
	return globalOperationTypeIndex.MergeOperationTypes(bits1, bits2)
}
