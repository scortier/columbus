package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/golang-migrate/migrate/v4"

	// Register database postgres
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	// Register golang migrate source file
	_ "github.com/golang-migrate/migrate/v4/source/file"

	"github.com/odpf/columbus/record"
	esStore "github.com/odpf/columbus/store/elasticsearch"
	"github.com/odpf/columbus/store/postgres"
	"github.com/pkg/errors"
)

const esMigrationTimeout = 5 * time.Second

func Migrate() {
	fmt.Println("Preparing migration...")
	if err := loadConfig(); err != nil {
		panic(err)
	}

	rootLogger := initLogger(config.LogLevel)
	log = rootLogger.WithField("reporter", "main")
	log.Infof("columbus %s is migrating", Version)

	log.Info("Migrating Postgres...")
	if err := migratePostgres(); err != nil {
		panic(err)
	}
	log.Info("Migration Postgres done.")

	log.Info("Migrating ES...")
	if err := migrateElasticsearch(); err != nil {
		panic(err)
	}
	log.Info("Migration ES done.")
}

func migratePostgres() (err error) {
	log.Info("Initiating Postgres client...")

	pgConfig := postgres.NewConfig(config.ServerHost, config.DBPort, config.DBName, config.DBUser, config.DBPassword, config.DBSSLMode)

	m, err := migrate.New(
		"file://db/migrations",
		pgConfig.ConnectionURL("postgres"))
	if err != nil {
		log.Errorf("failed to prepare migration: %s", err)
		return err

	}

	log.Info("Migrating DB...")
	if err := m.Up(); err != nil {
		if err == migrate.ErrNoChange {
			log.Infof("migration - no changes")
			return nil
		}
		log.Errorf("migration failed: %s", err)
		return err
	}
	return nil
}

func migrateElasticsearch() (err error) {
	log.Info("Initiating ES client...")
	esClient := initElasticsearch(config)
	for _, supportedTypeName := range record.AllSupportedTypes {
		log.Infof("Migrating %q type\n", supportedTypeName)
		ctx, cancel := context.WithTimeout(context.Background(), esMigrationTimeout)
		defer cancel()
		err = esStore.Migrate(ctx, esClient, supportedTypeName)
		if err != nil {
			err = errors.Wrapf(err, "error creating/replacing type: %q", supportedTypeName)
			return
		}
		log.Infof("created/updated %q type\n", supportedTypeName)
	}
	return
}
