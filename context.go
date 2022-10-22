package juice

import (
	"context"
	"sync"
)

// Context is a context of the middleware.
type Context struct {
	// Statement is the statement of current sql.
	Statement *Statement
	// Configuration is the configuration of the engine.
	Configuration *Configuration
	ctx           context.Context
}

// Context returns context.Context of current instance.
// if not found, return context.Background() instead.
func (c *Context) Context() context.Context {
	if c.ctx == nil {
		return context.Background()
	}
	return c.ctx
}

// WithContext set context.Context into current instance.
func (c *Context) WithContext(ctx context.Context) *Context {
	c.ctx = ctx
	return c
}

// release the context to the pool.
func (c *Context) release() {
	putContext(c)
}

var (
	// contextPool is a pool of context.
	// It is used to reduce the memory allocation.
	contextPool = sync.Pool{
		New: func() interface{} {
			return &Context{}
		},
	}
)

// newContext returns a context from the pool.
func newContext(stmt *Statement, cfg *Configuration) *Context {
	ctx := contextPool.Get().(*Context)
	ctx.Statement = stmt
	ctx.Configuration = cfg
	return ctx
}

// putContext returns a context to the pool.
func putContext(c *Context) {
	c.ctx = nil
	c.Statement = nil
	c.Configuration = nil
	contextPool.Put(c)
}
