package ynabimporter

import (
	"fmt"
	"strconv"
	"time"

	"github.com/bcaldwell/selfops/internal/config"
	"github.com/bcaldwell/selfops/internal/postgresHelper"
)

var baseNetworthSqlSchema = map[string]string{
	"date": "timestamp",
}

func (importer *ImportYNABRunner) importNetworth(budgets []config.Budget, currencies []string) error {
	err := importer.createNetworthTable(budgets)
	// if err != nil {

	// }
	currencyNetworths := make(map[string]float64)

	for _, currency := range currencies {
		currencyNetworths[currency] = 0
	}

	for _, budget := range budgets {
		for _, currency := range currencies {
			currencyNetworths[budget.Name+"_"+currency] = 0
		}

		accounts := importer.budgets[budget.ID].Accounts

		date := time.Now().Format("01-02-2006")

		err = importer.deleteRowsByDate(config.CurrentYnabConfig().SQL.NetworthTable, date, nil)

		if err != nil {
			return fmt.Errorf("Error deleting old networth records for %s: %s", date, err.Error())
		}

		for _, account := range accounts {
			if account.Closed {
				continue
			}

			balance := float64(account.Balance) / 1000.0

			for _, currency := range currencies {
				currencyBalance := Round(balance*budget.Conversions[currency], 0.01)
				currencyNetworths[currency] += currencyBalance
				currencyNetworths[budget.Name+"_"+currency] += currencyBalance
			}
		}
	}

	netWorthRow := make(map[string]string)
	date := time.Now().Format("01-02-2006")
	netWorthRow["date"] = date
	for k, v := range currencyNetworths {
		netWorthRow[k] = strconv.FormatFloat(v, 'f', 2, 64)
	}

	err = postgresHelper.Insert(importer.db, config.CurrentYnabConfig().SQL.NetworthTable, netWorthRow)
	if err != nil {
		return fmt.Errorf("Failed to write net worth to db: %v", err)
	}

	fmt.Printf("Wrote net worth to sql\n")

	return nil
}

func (importer *ImportYNABRunner) createNetworthTable(budgets []config.Budget) error {
	err := postgresHelper.CreateTable(importer.db, config.CurrentYnabConfig().SQL.NetworthTable, importer.createNetworthSQLSchema(budgets))
	if err != nil {
		return fmt.Errorf("Error creating table: %s", err)
	}
	return nil
}

func (importer *ImportYNABRunner) createNetworthSQLSchema(budgets []config.Budget) map[string]string {
	schema := baseNetworthSqlSchema

	for _, currency := range config.CurrentYnabConfig().Currencies {
		schema[currency] = "float8"
	}
	for _, budget := range budgets {
		for _, currency := range config.CurrentYnabConfig().Currencies {
			schema[budget.Name+"_"+currency] = "float8"
		}
	}
	return schema
}
