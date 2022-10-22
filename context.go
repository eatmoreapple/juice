package juice

import "context"

// Context is a context of the middleware.
type Context struct {
	Statement     *Statement
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
