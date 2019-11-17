package financialimporter

import (
	"database/sql"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/bcaldwell/selfops/internal/config"
	"github.com/bcaldwell/selfops/internal/postgresHelper"
)

var baseTransactionsSQLSchema = map[string]string{
	"transactionDate":  "timestamp",
	"transactionMonth": "timestamp",
	"category":         "varchar",
	"categoryGroup":    "varchar",
	"payee":            "varchar",
	"account":          "varchar",
	"memo":             "text",
	"currency":         "varchar",
	"amount":           "float8",
	"transactionType":  "varchar",
	"tags":             "varchar[]",
	"updatedAt":        "timestamp",
}

// http://www.postgresqltutorial.com/postgresql-array/

func NewTransactionImporter(db *sql.DB, transactions []Transaction, calculatedFields []config.CalculatedField, transactionCurrency string, currencies []string, importAfterDate string, sqlTable string) FinancialImporter {
	return &TransactionImporter{
		db:                  db,
		calculatedFields:    calculatedFields,
		transactions:        transactions,
		transactionCurrency: transactionCurrency,
		currencies:          currencies,
		importAfterDate:     importAfterDate,
		sqlTable:            sqlTable,
	}
}

type TransactionImporter struct {
	db                  *sql.DB
	calculatedFields    []config.CalculatedField
	transactions        []Transaction
	importAfterDate     string
	transactionCurrency string
	currencies          []string
	currencyConversions CurrencyConversion
	sqlTable            string
}

func (importer *TransactionImporter) Import() (int, error) {
	var err error

	importAfterDate := time.Time{}
	if importer.importAfterDate != "" {
		importAfterDate, err = time.Parse("01-02-2006", importer.importAfterDate)
		if err != nil {
			return 0, fmt.Errorf("Failed to parse import after date %s: %v", importer.importAfterDate, err)
		}
	}

	importer.currencyConversions, err = generateCurrencyConversions(importer.transactionCurrency, importer.currencies)
	if err != nil {
		return 0, err
	}

	// sqlRecords holds a record(map) representing the sql rows to be added
	// It will be roughly the size of importer.transactions + number of sub transactions
	// set the initial size to 0 so append works but set cap to a good guess
	sqlRecords := make([]map[string]string, 0, len(importer.transactions))

	for _, transaction := range importer.transactions {
		// check if transaction is before cutoff date
		t, err := time.Parse("2006-01-02", transaction.Date())
		if err != nil {
			return 0, fmt.Errorf("unable to parse date: %s", err.Error())
		}

		if t.Before(importAfterDate) {
			continue
		}

		sqlRow, err := importer.createSQLForTransaction(transaction)
		if err != nil {
			return 0, err
		}

		if transaction.HasSubTransactions() {
			for _, t := range transaction.SubTransactions() {
				sqlRow, err := importer.createSQLForTransaction(t)
				if err != nil {
					return 0, err
				}

				sqlRecords = append(sqlRecords, sqlRow)
			}
		} else {
			sqlRecords = append(sqlRecords, sqlRow)
		}
	}

	err = postgresHelper.InsertRecords(importer.db, importer.sqlTable, sqlRecords)
	if err != nil {
		return 0, fmt.Errorf("error writing to sql: %w", err)
	}

	return len(sqlRecords), nil
}

func (importer *TransactionImporter) createSQLForTransaction(transaction Transaction) (map[string]string, error) {
	amount := transaction.Amount()

	t, err := time.Parse("2006-01-02", transaction.Date())
	if err != nil {
		return nil, fmt.Errorf("unable to parse date: %s", err.Error())
	}

	transactionMonth := time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, t.Location())

	sqlRow := map[string]string{
		"category":         transaction.Category(),
		"categoryGroup":    transaction.CategoryGroup(),
		"payee":            transaction.Payee(),
		"account":          transaction.Account(),
		"memo":             transaction.Memo(),
		"currency":         importer.transactionCurrency,
		"amount":           strconv.FormatFloat(amount, 'f', 2, 64),
		"transactionType":  transaction.TransactionType().String(),
		"transactionMonth": transactionMonth.Format("2006-01-02"),
	}

	for _, field := range importer.calculatedFields {
		sqlRow[field.Name] = strconv.FormatBool(calculateField(field, transaction))
	}

	memoTags := transaction.Tags()

	if len(memoTags) != 0 {
		sqlRow["tags"] = fmt.Sprintf("{\"%s\"}", strings.Join(memoTags, "\",\""))
	} else {
		sqlRow["tags"] = "{}"
	}

	sqlRow["transactionDate"] = transaction.Date()
	sqlRow["updatedAt"] = time.Now().Format(time.UnixDate)

	for _, currency := range importer.currencies {
		value := Round(amount*importer.currencyConversions[currency], 0.01)
		sqlRow[currency] = strconv.FormatFloat(value, 'f', 2, 64)
	}

	return sqlRow, nil
}

func calculateField(field config.CalculatedField, transaction Transaction) bool {
	boolean := stringInSlice(transaction.Category(), field.Category) ||
		stringInSlice(transaction.CategoryGroup(), field.CategoryGroup) ||
		stringInSlice(transaction.Payee(), field.Payee)

	if field.Inverted {
		return !boolean
	}

	return boolean
}

func stringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}

	return false
}

func (importer *TransactionImporter) CreateOrUpdateSQLTable() error {
	err := postgresHelper.CreateTable(importer.db, importer.sqlTable, importer.createTransactionsSQLSchema())
	if err != nil {
		return fmt.Errorf("error creating table: %s", err)
	}

	return nil
}

func (importer *TransactionImporter) DropSQLTable() error {
	err := postgresHelper.DropTable(importer.db, importer.sqlTable)
	if err != nil {
		return fmt.Errorf("error dropping table: %s", err)
	}

	return err
}

func (importer *TransactionImporter) createTransactionsSQLSchema() map[string]string {
	schema := baseTransactionsSQLSchema

	for _, field := range importer.calculatedFields {
		if _, ok := schema[field.Name]; !ok {
			schema[field.Name] = "boolean"
		}
	}

	for _, currency := range importer.currencies {
		schema[currency] = "float8"
	}

	return schema
}

func Round(x, unit float64) float64 {
	return math.Round(x/unit) * unit
}
