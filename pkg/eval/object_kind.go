// pkg/eval/object_kind.go
package eval

// ObjectKind represents the type of an object using an enum for faster comparisons.
type ObjectKind uint8

const (
	KindInvalid ObjectKind = iota
	KindInteger
	KindFloat
	KindString
	KindBoolean
	KindNull
	KindArray
	KindMap
	KindReturnValue
	KindError
	KindFunction
	KindBuiltin
	KindTask
	KindStructInstance
	KindModule
	KindNative
)

func (k ObjectKind) String() string {
	switch k {
	case KindInteger:
		return "INTEGER"
	case KindFloat:
		return "FLOAT"
	case KindString:
		return "STRING"
	case KindBoolean:
		return "BOOLEAN"
	case KindNull:
		return "NULL"
	case KindArray:
		return "ARRAY"
	case KindMap:
		return "MAP"
	case KindReturnValue:
		return "RETURN_VALUE"
	case KindError:
		return "ERROR"
	case KindFunction:
		return "FUNCTION"
	case KindBuiltin:
		return "BUILTIN"
	case KindTask:
		return "TASK"
	case KindStructInstance:
		return "STRUCT"
	case KindModule:
		return "MODULE"
	case KindNative:
		return "NATIVE"
	default:
		return "INVALID"
	}
}

// Integer cache for common integers (-4096 to 4096)
// Expanded from [-128, 127] to reduce allocations for loop counters
// and common numeric values. Uses ~66KB of memory.
const (
	MinCachedInt = -4096
	MaxCachedInt = 4096
	intCacheSize = MaxCachedInt - MinCachedInt + 1
)

var (
	IntCache [intCacheSize]*Integer // Exported for VM inlining

	NULL  *Null
	TRUE  *Boolean
	FALSE *Boolean
	ZERO  *Integer
	ONE   *Integer
)

// ============================================================================
// Identifier Interning System
// ============================================================================
// Identifier interning maintains a canonical pool of identifier strings,
// allowing fast lookups via pointer comparison instead of string comparison.
// This dramatically reduces overhead in environments with many local variables.

var (
	// internedIdentifiers maps string identifiers to their canonical interned reference
	internedIdentifiers = make(map[string]*InternedString)
	// nextInternID is an incrementing ID for each interned string
	nextInternID uint32 = 0
)

// InternedString represents a canonicalized identifier with a unique ID
type InternedString struct {
	Value string
	ID    uint32
}

// InternIdentifier returns a canonical interned string for the given identifier.
// Multiple calls with the same string return the same pointer, enabling fast comparisons.
func InternIdentifier(id string) *InternedString {
	if interned, exists := internedIdentifiers[id]; exists {
		return interned
	}

	interned := &InternedString{
		Value: id,
		ID:    nextInternID,
	}
	nextInternID++
	internedIdentifiers[id] = interned
	return interned
}

// GetInternedID retrieves the ID of an already-interned identifier, or returns 0 if not found
func GetInternedID(id string) uint32 {
	if interned, exists := internedIdentifiers[id]; exists {
		return interned.ID
	}
	return 0
}

// Initialize the integer cache and common singletons
func init() {
	for i := 0; i < intCacheSize; i++ {
		IntCache[i] = &Integer{Value: int64(i) + MinCachedInt, kind: KindInteger}
	}

	NULL = &Null{kind: KindNull}
	TRUE = &Boolean{Value: true, kind: KindBoolean}
	FALSE = &Boolean{Value: false, kind: KindBoolean}
	ZERO = NewInteger(0)
	ONE = NewInteger(1)
}

// NewInteger returns a cached integer for small values or allocates a new one.
func NewInteger(value int64) *Integer {
	if value >= MinCachedInt && value <= MaxCachedInt {
		return IntCache[value-MinCachedInt]
	}
	return &Integer{Value: value, kind: KindInteger}
}
