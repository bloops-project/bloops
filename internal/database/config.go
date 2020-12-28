package database

type Config struct {
	FilePath string `envconfig:"BLOOP_DB_FILE" default:"./db"`
}
