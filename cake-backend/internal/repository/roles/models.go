package roles

type Role struct {
	Id   int    `db:"id"`
	Uuid string `db:"uuid"`
	Name string `db:"name"`
}
