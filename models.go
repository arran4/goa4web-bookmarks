// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.21.0

package a4webbm

import (
	"database/sql"
)

type Bookmark struct {
	Idbookmarks   int32
	Userreference string
	List          sql.NullString
}
