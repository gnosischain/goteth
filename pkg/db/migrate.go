package db

import (
	_ "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/clickhouse"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func (s *DBService) makeMigrations() error {
	log.Info("will try to apply migrations ...")

	m, err := migrate.New(
		"file://pkg/db/migrations",
		s.migrationUrl)

	if err != nil {
		log.Errorf("could not create DB migrator: %s", err.Error())
		return err
	}

	log.Infof("now applying database migrations ...")
	if err := m.Up(); err != nil {
		if err != migrate.ErrNoChange {
			log.Errorf("there was an error while applying migrations: %s", err.Error())
			return err
		}
	}
	connErr, dbErr := m.Close()

	if connErr != nil {
		log.Errorf("there was an error closing migrator connection: %s", connErr.Error())
		return connErr
	}

	if dbErr != nil {
		log.Errorf("there was an error with DB migrator: %s", dbErr.Error())
		return dbErr
	}

	return err
}
