package ynabimporter

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"github.com/bcaldwell/selfops/internal/config"
	"github.com/bcaldwell/selfops/pkg/financialimporter"
	"k8s.io/klog"
)

type (
	TransactionType string
	category        struct {
		Name  string
		Group string
		Id    string
	}
)

var defaultRegex = "^[A-Za-z0-9]([A-Za-z0-9\\-\\_]+)?$"

// http://www.postgresqltutorial.com/postgresql-array/

func (importer *ImportYNABRunner) importTransactions(budget config.Budget, currencies []string) error {
	regexPattern := config.CurrentYnabConfig().Tags.RegexMatch
	if regexPattern == "" {
		regexPattern = defaultRegex
	}

	regex := regexp.MustCompile(regexPattern)

	lastSeen := LastSeen{}
	_, err := importer.db.NewSelect().Model(&lastSeen).Where("endpoint = ?", "transactions").Exec(context.Background())
	if err != nil {
		return err
	}

	// need to get transactions from transaction service to have the sub transaction data
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

	i := financialimporter.NewTransactionImporter(importer.db, importer.currencyConverter, transactions, budget.CalculatedFields, budget.Currency, currencies, importAfterDate, config.CurrentYnabConfig().SQL.TransactionsTable)

	written, err := i.Import()
	if err != nil {
		return err
	}

	klog.Infof("Wrote %d transactions to sql from budget %s\n", written, budget.Name)

	return nil
}
