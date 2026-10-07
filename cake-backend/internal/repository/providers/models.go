package providers

import "time"

type Provider struct {
	Id        int       `db:"id"`
	Uuid      string    `db:"uuid"`
	Name      string    `db:"name"`
	BaseUrl   string    `db:"base_url"`
	ApiKey    []byte    `db:"api_key"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}
