package docker_client

import (
	"log"

	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/db_client"
	"github.com/spf13/viper"
	"gorm.io/gorm"
)

func NewTestDockerClient() (testDockerClient dockertest.Pool, cleanerContainer func()) {
	pool, poolErr := dockertest.NewPool("")
	if poolErr != nil {
		log.Fatalf("Could not construct pool: %s", poolErr)
	}

	if err := pool.Client.Ping(); err != nil {
		log.Fatalf("Could not connect to Docker: %s", err)
	}
	opts := dockertest.RunOptions{
		Name:         viper.GetString("docker.containerName"),
		Repository:   "postgres",
		Tag:          "13",
		Env:          viper.GetStringSlice("docker.environment"),
		ExposedPorts: []string{"5432"},
		PortBindings: map[docker.Port][]docker.PortBinding{
			"5432": {
				{HostIP: "0.0.0.0", HostPort: "5433"},
			},
		},
	}

	resource, err := pool.RunWithOptions(&opts,
		func(config *docker.HostConfig) {
			config.AutoRemove = true
			config.RestartPolicy = docker.NeverRestart()
		},
	)
	if err != nil {
		log.Fatalf("Could not start resource: %s", err)
	}

	resource.Expire(viper.GetUint("docker.container_lifetime"))
	cleanerContainer = func() {
		// purge the container
		err = pool.Purge(resource)
		if err != nil {
			log.Panicf("Could not purge resource: %s", err)
		}
	}
	return
}

// NewTestPostgresClient connects to the dockertest Postgres and migrates it.
func NewTestPostgresClient(_logger *log.Logger, testDockerClient dockertest.Pool) (db_client.PostgresClient, error) {
	var gdb *gorm.DB
	if err := testDockerClient.Retry(func() error {
		var err error
		gdb, err = db_client.OpenByDSN(viper.GetString("postgres.postgresDsn"))
		if err != nil {
			log.Println("Test database not ready yet (it is booting up, wait for a few tries)...")
			return err
		}
		db, sqlErr := gdb.DB()
		if sqlErr != nil {
			return sqlErr
		}
		return db.Ping()
	}); err != nil {
		log.Fatalf("Could not connect to docker: %s", err)
	}
	client := db_client.WrapDB(gdb, _logger)
	return client, client.Migrate()
}
