package ynabimporter

import (
	"fmt"
	"regexp"
	"time"

	"github.com/bcaldwell/selfops/internal/config"
	"github.com/bcaldwell/selfops/internal/postgresHelper"
	"github.com/bcaldwell/selfops/pkg/financialimporter"
)

type TransactionType string
type category struct {
	Name  string
	Group string
	Id    string
}

var baseTransactionsSqlSchema = map[string]string{
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

var defaultRegex = "^[A-Za-z0-9]([A-Za-z0-9\\-\\_]+)?$"

// http://www.postgresqltutorial.com/postgresql-array/

func (importer *ImportYNABRunner) importTransactions(budget config.Budget, currencies []string) error {
	regexPattern := config.CurrentYnabConfig().Tags.RegexMatch
	if regexPattern == "" {
		regexPattern = defaultRegex
	}

	regex := regexp.MustCompile(regexPattern)

	ynabTransactions, err := importer.ynabClient.TransactionsService.List(budget.ID)
	if err != nil {
		return fmt.Errorf("Error getting transactions: %s", err.Error())
	}

	transactions := make([]financialimporter.Transaction, len(ynabTransactions))

	for i := range ynabTransactions {
		transactions[i] = &YnabTransaction{
			TransactionDetail: &ynabTransactions[i],
			Regex:             regex,
			CategoryIDMap:     importer.categories[budget.Name],
		}
	}

	importAfterDate := time.Time{}
	if budget.ImportAfterDate != "" {
		importAfterDate, err = time.Parse("01-02-2006", budget.ImportAfterDate)
		if err != nil {
			return fmt.Errorf("Failed to parse import after date %s: %v", budget.ImportAfterDate, err)
		}
	}

	i := financialimporter.NewTransactionImporter(importer.db, transactions, budget.CalculatedFields, budget.Currency, currencies, importAfterDate, config.CurrentYnabConfig().SQL.TransactionsTable)

	written, err := i.Import()
	if err != nil {
		return err
	}

	fmt.Printf("Wrote %d transactions to sql from budget %s\n", written, budget.Name)

	return nil
}

func (importer *ImportYNABRunner) recreateTransactionTable() error {
	err := postgresHelper.DropTable(importer.db, config.CurrentYnabConfig().SQL.TransactionsTable)
	if err != nil {
		return fmt.Errorf("Error dropping table: %s", err)
	}

	err = postgresHelper.CreateTable(importer.db, config.CurrentYnabConfig().SQL.TransactionsTable, importer.createTransactionsSQLSchema())
	if err != nil {
		return fmt.Errorf("Error creating table: %s", err)
	}

	return nil
}

func (importer *ImportYNABRunner) createTransactionsSQLSchema() map[string]string {
	schema := baseTransactionsSqlSchema

	for _, budget := range config.CurrentYnabConfig().Budgets {
		for _, field := range budget.CalculatedFields {
			if _, ok := schema[field.Name]; !ok {
				schema[field.Name] = "boolean"
			}
		}
	}

	for _, currency := range config.CurrentYnabConfig().Currencies {
		schema[currency] = "float8"
	}
	return schema
}
