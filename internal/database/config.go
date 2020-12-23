package database

type Config struct {
	FileName string `envconfig:"SOD_DB_FILE" default:"db"`
}
