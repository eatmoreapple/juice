/*
Copyright 2024 eatmoreapple

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package stmt

import (
	"database/sql"
	"unsafe"
)

// Query extracts the SQL query string from a *sql.Stmt using unsafe pointer arithmetic.
//
// How it works:
// 1. sql.Stmt struct memory layout (only relevant fields):
//
//   - First field (*DB): 8 bytes (on 64-bit systems)
//
//   - Second field (query string): 16 bytes  <- we want this
//
// 2. To get the query string, we:
//
//	a. Start at the beginning of the struct (sql.Stmt)
//	b. Skip the first field (8 bytes) to reach query string
//	c. Read the string value
//
// Note: This implementation relies on the internal structure of sql.Stmt
// and may break if the struct layout changes in future Go versions.
func Query(s *sql.Stmt) string {
	return *(*string)(unsafe.Pointer(uintptr(unsafe.Pointer(s)) + unsafe.Sizeof(uintptr(0))))
}

/* Memory Layout Visualization:

32-bit system:
sql.Stmt:
┌─────────────┬──────────────────────────┐
│     *DB     │          query           │
│  (4 bytes)  │        (8 bytes)         │
└─────────────┴──────────────────────────┘
			  ▲    ┌────────┬────────┐
			  │    │  ptr   │  len   │
			  └────┤4 bytes │4 bytes │
				   └────────┴────────┘

64-bit system:
sql.Stmt:
┌─────────────┬──────────────────────────┐
│     *DB     │          query           │
│  (8 bytes)  │        (16 bytes)        │
└─────────────┴──────────────────────────┘
			  ▲    ┌────────┬────────┐
			  │    │  ptr   │  len   │
			  └────┤8 bytes │8 bytes │
				   └────────┴────────┘
*/
