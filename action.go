package juice

// Action defines a sql action.
type Action string

const (
	// Select is an action for query
	Select Action = "select"

	// Insert is an action for insert
	Insert Action = "insert"

	// Update is an action for update
	Update Action = "update"

	// Delete is an action for delete
	Delete Action = "delete"
)

func (a Action) String() string {
	return string(a)
}

func (a Action) ForRead() bool {
	return a == Select
}

func (a Action) ForWrite() bool {
	return a == Insert || a == Update || a == Delete
}
