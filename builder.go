package juice

import (
	"strings"
	"sync"
)

// builderPool is a pool of strings.Builder.
// It is used to reduce the memory allocation.
var builderPool = sync.Pool{
	New: func() any {
		return &strings.Builder{}
	},
}

// getBuilder returns a strings.Builder from the pool.
func getBuilder() *strings.Builder {
	return builderPool.Get().(*strings.Builder)
}

// putBuilder puts a strings.Builder back to the pool.
func putBuilder(builder *strings.Builder) {
	builder.Reset()
	builderPool.Put(builder)
}
