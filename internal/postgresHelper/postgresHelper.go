package postgresHelper

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/bcaldwell/selfops/internal/config"
	_ "github.com/lib/pq"
	"k8s.io/klog"
)

func CreatePostgresClient(dbname string) (*sql.DB, error) {
	// bypass creating of db if database_url is set because we are likely running in heroku then
	if config.CurrentSecrets().DatabaseURL == "" {
		databaselessConnStr := fmt.Sprintf("host=%s user=%s password=%s dbname=%s sslmode=disable",
			config.CurrentSqlSecrets().SqlHost, config.CurrentSqlSecrets().SqlUsername, config.CurrentSqlSecrets().SqlPassword, "postgres")
		db, err := sql.Open("postgres", databaselessConnStr)
		if err != nil {
			return nil, fmt.Errorf("Failed to create db for databaseless connection: %s", err)
		}

		rows, err := db.Query(fmt.Sprintf("SELECT datname FROM pg_database where datname = '%s'", config.CurrentYnabConfig().SQL.YnabDatabase))
		if err != nil {
			return nil, fmt.Errorf("Failed to get list of databases: %s", err)
		}
		defer rows.Close()

		// next meaning there is a row, all we care about is if there is a row
		if !rows.Next() {
			klog.Infof("Creating database %s in postgres database\n", config.CurrentYnabConfig().SQL.YnabDatabase)
			_, err := db.Exec("CREATE DATABASE " + config.CurrentYnabConfig().SQL.YnabDatabase)
			if err != nil {
				return nil, err
			}
		}
	}

	db, err := sql.Open("postgres", getConnectionString(dbname))
	if err != nil {
		return db, err
	}
	err = db.Ping()
	return db, err
}

func getConnectionString(dbname string) string {
	if config.CurrentSecrets().DatabaseURL != "" {
		return config.CurrentSecrets().DatabaseURL
	}

	return fmt.Sprintf("host=%s user=%s password=%s dbname=%s sslmode=disable",
		config.CurrentSqlSecrets().SqlHost, config.CurrentSqlSecrets().SqlUsername, config.CurrentSqlSecrets().SqlPassword, dbname)
}

func CreateTable(db *sql.DB, tableName string, parameters map[string]string) error {
	bodystr := ""

	for key, value := range parameters {
		bodystr += fmt.Sprintf("\"%s\" %s,", key, value)
	}

	createstr := fmt.Sprintf(`
CREATE SEQUENCE IF NOT EXISTS "public"."%s_id_seq";

CREATE TABLE "public"."%s" (
	"id" int4 DEFAULT nextval('%s_id_seq'::regclass),
	%s
	PRIMARY KEY ("id")
);
	`, tableName, tableName, tableName, bodystr)
	_, err := db.Exec(createstr)
	return err
}

func DropTable(db *sql.DB, tableName string) error {
	dropStr := fmt.Sprintf("DROP TABLE IF EXISTS %s;", tableName)
	_, err := db.Exec(dropStr)
	return err
}

// func TableExist(db *sql.DB)

func insertStr(tableName string, parameters map[string]string) string {
	values := ""
	keys := ""
	first := true
	for key, value := range parameters {
		if value == "" {
			value = "EMPTY"
		} else {
			value = "'" + value + "'"
		}
		if first {
			keys = key
			values = value
			first = false
		} else {
			keys += "\", \"" + key
			values += ", " + value
		}
	}
	return fmt.Sprintf("INSERT INTO %s (\"%s\") VALUES (%s);", tableName, keys, values)
}

func Insert(db *sql.DB, tableName string, parameters map[string]string) error {
	queryStr := insertStr(tableName, parameters)
	_, err := db.Exec(queryStr)
	return err
}

func InsertRecords(db *sql.DB, tableName string, records []map[string]string) error {
	if len(records) == 0 {
		return nil
	}

	valueStr := ""
	keyStr := ""
	keys := []string{}
	first := true
	for key, _ := range records[0] {
		if first {
			keyStr = key
			first = false
		} else {
			keyStr += "\", \"" + key
		}
		keys = append(keys, key)
	}

	for _, record := range records {
		valueStr += "("
		for i, key := range keys {
			value := record[key]
			if value == "" {
				value = "''"
			} else {
				value = "'" + strings.Replace(value, "'", "", -1) + "'"
			}
			if i == 0 {
				valueStr += value
			} else {
				valueStr += ", " + value
			}
		}
		valueStr += "),\n"
	}

	recordsInsertStr := fmt.Sprintf(`
INSERT INTO %s ("%s") VALUES 
%s;`, tableName, keyStr, strings.TrimSuffix(valueStr, ",\n"))
	_, err := db.Exec(recordsInsertStr)
	return err
}
