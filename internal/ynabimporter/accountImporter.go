package ynabimporter

import (
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/bcaldwell/selfops/internal/config"
	"github.com/bcaldwell/selfops/internal/postgresHelper"
	"k8s.io/klog"
)

var baseAccountsSqlSchema = map[string]string{
	"name":       "varchar",
	"currency":   "varchar",
	"budgetName": "varchar",
	"onBudget":   "boolean",
	"type":       "varchar",
	"balance":    "varchar",
	"date":       "timestamp",
}

func (importer *ImportYNABRunner) importAccounts(budget config.Budget, currencies []string) error {
	err := importer.createAccountsTable()
	// if err != nil {
	// 	return err
	// }

	sqlRecords := make([]map[string]string, 0)

	currencyNetworths := make(map[string]float64)
	for _, currency := range currencies {
		currencyNetworths[currency] = 0
	}

	accounts := importer.budgets[budget.ID].Accounts

	date := time.Now().Format("01-02-2006")

	err = importer.deleteRowsByDate(config.CurrentYnabConfig().SQL.AccountsTable, date, map[string]string{"budgetName": budget.Name})

	if err != nil {
		return fmt.Errorf("Error getting deleting old account records for %s: %s", date, err.Error())
	}

	for _, account := range accounts {
		if account.Closed {
			continue
		}

		balance := float64(account.Balance) / 1000.0

		row := map[string]string{
			"balance":    strconv.FormatFloat(balance, 'f', 2, 64),
			"name":       account.Name,
			"type":       account.Type,
			"onBudget":   strconv.FormatBool(account.OnBudget),
			"currency":   budget.Currency,
			"budgetName": budget.Name,
			"date":       date,
		}

		for _, currency := range currencies {
			currencyBalance := Round(balance*budget.Conversions[currency], 0.01)
			row[currency] = strconv.FormatFloat(currencyBalance, 'f', 2, 64)
			currencyNetworths[currency] += currencyBalance
		}

		sqlRecords = append(sqlRecords, row)
	}

	err = postgresHelper.InsertRecords(importer.db, config.CurrentYnabConfig().SQL.AccountsTable, sqlRecords)
	if err != nil {
		return fmt.Errorf("Error writing accounts to sql: %s", err.Error())
	}

	klog.Infof("Wrote %d accounts to sql from budget %s\n", len(accounts), budget.Name)

	return nil
}

func (importer *ImportYNABRunner) createAccountsTable() error {
	err := postgresHelper.CreateTable(importer.db, config.CurrentYnabConfig().SQL.AccountsTable, importer.createAccountsSQLSchema())
	if err != nil {
		return fmt.Errorf("Error creating table: %s", err)
	}
	return nil
}

func (importer *ImportYNABRunner) createAccountsSQLSchema() map[string]string {
	schema := baseAccountsSqlSchema

	for _, currency := range config.CurrentYnabConfig().Currencies {
		schema[currency] = "float8"
	}
	return schema
}

func Round(x, unit float64) float64 {
	return math.Round(x/unit) * unit
}
