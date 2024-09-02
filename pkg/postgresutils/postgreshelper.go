package postgresutils

import (
	"database/sql"
	"fmt"
	"reflect"
	"strings"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
	"k8s.io/klog"

	"github.com/bcaldwell/selfops/pkg/config"
)

func CreatePostgresClient(dbname string) (*bun.DB, error) {
	var pgconn *pgdriver.Connector

	// bypass creating of db if database_url is set because we are likely running in heroku then
	if config.CurrentSecrets().DatabaseURL == "" {
		err := ensureDBExistsInPostgres(config.CurrentYnabConfig().SQL.YnabDatabase)
		if err != nil {
			return nil, err
		}

		sqlHost := config.CurrentSecrets().SQL.SqlHost
		// slightly silly logic to add port if missing
		if !strings.Contains(sqlHost, ":") {
			sqlHost += ":5432"
		}

		pgconn = pgdriver.NewConnector(
			pgdriver.WithAddr(sqlHost),
			pgdriver.WithInsecure(true),
			pgdriver.WithUser(config.CurrentSqlSecrets().SqlUsername),
			pgdriver.WithPassword(config.CurrentSqlSecrets().SqlPassword),
			pgdriver.WithDatabase(config.CurrentYnabConfig().SQL.YnabDatabase),
		)
	} else {
		// this panics if its invalid
		pgconn = pgdriver.NewConnector(pgdriver.WithDSN(config.CurrentSecrets().DatabaseURL))
	}

	db := sql.OpenDB(pgconn)
	err := db.Ping()

	return bun.NewDB(db, pgdialect.New()), err
}

func ensureDBExistsInPostgres(table string) error {
	pgconn := pgdriver.NewConnector(
		pgdriver.WithAddr(config.CurrentSqlSecrets().SqlHost),
		pgdriver.WithInsecure(true),
		pgdriver.WithUser(config.CurrentSqlSecrets().SqlUsername),
		pgdriver.WithPassword(config.CurrentSqlSecrets().SqlPassword),
		pgdriver.WithDatabase("postgres"),
	)

	db := sql.OpenDB(pgconn)
	rows, err := db.Query(fmt.Sprintf("SELECT datname FROM pg_database where datname = '%s'", config.CurrentYnabConfig().SQL.YnabDatabase))
	if err != nil {
		return fmt.Errorf("Failed to get list of databases: %s", err)
	}
	defer rows.Close()

	// next meaning there is a row, all we care about is if there is a row
	if !rows.Next() {
		klog.Infof("Creating database %s in postgres database\n", config.CurrentYnabConfig().SQL.YnabDatabase)
		_, err := db.Exec("CREATE DATABASE " + config.CurrentYnabConfig().SQL.YnabDatabase)
		if err != nil {
			return fmt.Errorf("failed to create database %s: %w", config.CurrentYnabConfig().SQL.YnabDatabase, err)
		}
	}

	return nil
}

func TableSetString(db *bun.DB, model interface{}, exclude ...string) string {
	t := db.Dialect().Tables().Get(reflect.TypeOf(model).Elem())
	if t == nil {
		return ""
	}

	parts := []string{}

	for _, f := range t.FieldMap {
		if isInArray(exclude, f.Name) {
			continue
		}

		parts = append(parts, fmt.Sprintf("%s = EXCLUDED.%s", f.Name, f.Name))
	}

	return strings.Join(parts, ", ")
}

func isInArray(arr []string, s string) bool {
	for _, i := range arr {
		if i == s {
			return true
		}
	}

	return false
}
