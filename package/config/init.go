package config

import (
	"os"
	"strings"

	"github.com/spf13/viper"
)

func Init() error {
	viper.SetConfigName("config")
	viper.SetConfigType("yml")
	viper.AddConfigPath("./package/config")

	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()
	if os.Getenv("POSTGRES_HOST") != "" {
		viper.Set("postgres.postgresDsn", "host="+os.Getenv("POSTGRES_HOST")+" port=5432 user="+os.Getenv("POSTGRES_USER")+" password="+os.Getenv("POSTGRES_PASSWORD")+" dbname="+os.Getenv("POSTGRES_DB")+" sslmode=disable")
	}
	if os.Getenv("PROJECTS_POSTGRES_DSN") != "" {
		viper.Set("projectsPostgres.postgresDsn", os.Getenv("PROJECTS_POSTGRES_DSN"))
	}

	err := viper.ReadInConfig()
	return err
}

func InitForTests() error {
	viper.SetConfigName("config-test")
	viper.SetConfigType("yml")
	viper.AddConfigPath("../../package/config")

	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	err := viper.ReadInConfig()
	return err
}
