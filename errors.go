package juice

import (
	"errors"
)

// ErrEmptyQuery is an error that is returned when the query is empty.
var ErrEmptyQuery = errors.New("empty query")
