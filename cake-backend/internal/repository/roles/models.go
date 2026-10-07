package roles

// Role represents a row from the user_roles table.
type Role struct {
	Id   int    `db:"id"`
	Uuid string `db:"uuid"`
	Name string `db:"name"`
}