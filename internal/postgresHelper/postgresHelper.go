package postgresHelper

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/bcaldwell/selfops/internal/config"
	_ "github.com/lib/pq"
)

func CreatePostgresClient() (*sql.DB, error) {
	connStr := fmt.Sprintf("host=%s user=%s password=%s dbname=%s sslmode=disable",
		config.CurrentSqlSecrets().SqlHost, config.CurrentSqlSecrets().SqlUsername, config.CurrentSqlSecrets().SqlPassword, config.CurrentYnabConfig().Sql.YnabDatabase)
	return sql.Open("postgres", connStr)
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
	_, err := db.Query(createstr)
	return err
}

func DropTable(db *sql.DB, tableName string) error {
	dropStr := fmt.Sprintf("DROP TABLE IF EXISTS %s;", tableName)
	_, err := db.Query(dropStr)
	return err
}

func insertStr(tableName string, parameters map[string]string) string {
	values := ""
	keys := ""
	first := true
	for key, value := range parameters {
		if value == "" {
			value = "NULL"
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
	_, err := db.Query(queryStr)
	fmt.Println(err)
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
				value = "NULL"
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

	_, err := db.Query(recordsInsertStr)
	return err
}
