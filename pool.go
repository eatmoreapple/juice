package juice

import (
	"strings"
	"sync"
)

// stringBuilderPool is a pool of strings.Builder.
// It is used to reduce the memory allocation.
var stringBuilderPool = sync.Pool{
	New: func() any {
		return &strings.Builder{}
	},
}

// getStringBuilder returns a strings.Builder from the pool.
func getStringBuilder() *strings.Builder {
	return stringBuilderPool.Get().(*strings.Builder)
}

// putStringBuilder puts a strings.Builder back to the pool.
func putStringBuilder(builder *strings.Builder) {
	builder.Reset()
	stringBuilderPool.Put(builder)
}
