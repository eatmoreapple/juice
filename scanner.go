package juice

import (
	"database/sql"
)

// RowsScanner scan sql.Rows to dest.
type RowsScanner interface {
	ScanRows(rows *sql.Rows) error
}
