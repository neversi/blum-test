package config

import "fmt"

type Postgres struct {
	User     string `envconfig:"USER" required:"true"`
	Password string `envconfig:"PASSWORD" required:"true"`
	Name     string `envconfig:"NAME" required:"true"`
	Host     string `envconfig:"HOST" required:"true"`
}

func (p Postgres) DSN() string {
	dsn := "postgres://%s:%s@%s/%s?sslmode=disable"
	return fmt.Sprintf(
		dsn,
		p.User,
		p.Password,
		p.Host,
		p.Name,
	)
}
