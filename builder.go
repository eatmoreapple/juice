package juice

import (
	"strings"
	"sync"
)

var builderPool = sync.Pool{
	New: func() interface{} {
		return &strings.Builder{}
	},
}

func getBuilder() *strings.Builder {
	return builderPool.Get().(*strings.Builder)
}

func putBuilder(builder *strings.Builder) {
	builder.Reset()
	builderPool.Put(builder)
}
