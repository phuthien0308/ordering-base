package sample

import "time"

//go:generate daogen -struct User -file user.go
type User struct {
	ID        int64     `sql-col:"id" sql-identifier:"true"`
	Username  string    `sql-col:"username" sql-insert:"true"`
	Email     string    `sql-col:"email" sql-insert:"true" sql-update:"true"`
	Password  string    `sql-col:"password" sql-insert:"true" sql-update:"true"`
	CreatedAt time.Time `sql-col:"created_at" sql-skip:"true"`
	Internal  string    `sql-skip:"true"`
}

//go:generate daogen -struct User1 -file user.go
type User1 struct {
	ID        int64     `sql-col:"id" sql-identifier:"true"`
	Username  string    `sql-col:"username" sql-insert:"true"`
	Email     string    `sql-col:"email" sql-insert:"true" sql-update:"true"`
	Password  string    `sql-col:"password" sql-insert:"true" sql-update:"true"`
	CreatedAt time.Time `sql-col:"created_at" sql-skip:"true"`
	Internal  string    `sql-skip:"true"`
}
