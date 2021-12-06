package ynabimporter

import (
	"context"
	"fmt"
	"time"

	"github.com/bcaldwell/selfops/internal/config"
	"github.com/bcaldwell/selfops/internal/postgresutils"
	"github.com/bcaldwell/selfops/pkg/financialimporter"
	"github.com/davidsteinsland/ynab-go/ynab"
	"github.com/uptrace/bun"
)

type ImportYNABRunner struct {
	ynabClient        *ynab.Client
	currencyConverter *financialimporter.CurrencyConverter
	db                *bun.DB
	budgets           map[string]ynab.BudgetDetail
	categories        map[string]map[string]category
}

type LastSeen struct {
	Endpoint string
	LastSeen string
}

func (importer *ImportYNABRunner) Run() error {
	return importer.importYNAB()
}

func (importer *ImportYNABRunner) Close() error {
	return importer.db.Close()
}

func NewImportYNABRunner() (*ImportYNABRunner, error) {
	ynabClient := ynab.NewDefaultClient(config.CurrentYnabSecrets().YnabAccessToken)

	db, err := postgresutils.CreatePostgresClient(config.CurrentYnabConfig().SQL.YnabDatabase)
	if err != nil {
		return nil, fmt.Errorf("Error connecting to postgres DB: %s", err)
	}

	return &ImportYNABRunner{
		ynabClient:        ynabClient,
		currencyConverter: financialimporter.NewCurrencyConverter(config.CurrentExchangeRateAPISecrets().AccessKey),
		db:                db,
		budgets:           make(map[string]ynab.BudgetDetail),
		categories:        make(map[string]map[string]category),
	}, nil
}

func (importer *ImportYNABRunner) importYNAB() error {
	err := importer.detectBudgetIDs(config.CurrentYnabConfig())
	if err != nil {
		return fmt.Errorf("Error detecting budget IDs: %s", err)
	}

	for _, b := range config.CurrentYnabConfig().Budgets {
		importer.budgets[b.ID], err = importer.ynabClient.BudgetService.Get(b.ID)
		if err != nil {
			return fmt.Errorf("Failed to get budget details for %s", err)
		}

		categoryGroupIDToName := make(map[string]string)
		for _, g := range importer.budgets[b.ID].CategoryGroups {
			categoryGroupIDToName[g.Id] = g.Name
		}

		importer.categories[b.Name] = make(map[string]category)

		for _, c := range importer.budgets[b.ID].Categories {
			importer.categories[b.Name][c.Id] = category{
				Id:    c.Id,
				Name:  c.Name,
				Group: categoryGroupIDToName[c.CategoryGroupId],
			}
		}
	}

	if err != nil {
		return err
	}

	_, err = importer.db.NewCreateTable().Model(&LastSeen{}).IfNotExists().Exec(context.Background())
	if err != nil {
		return err
	}

	i := financialimporter.NewTransactionImporter(importer.db, importer.currencyConverter, nil, nil, "", nil, time.Now(), config.CurrentYnabConfig().SQL.TransactionsTable)
	i.Migrate()

	for _, b := range config.CurrentYnabConfig().Budgets {
		err = importer.importTransactions(b, config.CurrentYnabConfig().Currencies)
		if err != nil {
			return err
		}

		err = importer.importAccounts(b, config.CurrentYnabConfig().Currencies)
		if err != nil {
			return err
		}

		err = importer.importBudgets(b, config.CurrentYnabConfig().Currencies)
		if err != nil {
			return err
		}
	}

	err = importer.importNetworth(config.CurrentYnabConfig().Budgets, config.CurrentYnabConfig().Currencies)
	if err != nil {
		return err
	}

	return nil
}

func (importer *ImportYNABRunner) detectBudgetIDs(conf *config.YnabConfig) error {
	budgets, err := importer.ynabClient.BudgetService.List()
	if err != nil {
		return err
	}

	for i, budgetConfig := range conf.Budgets {
		if budgetConfig.ID == "" {
			found := false

			for _, b := range budgets {
				if budgetConfig.Name == b.Name {
					conf.Budgets[i].ID = b.Id
					if budgetConfig.Currency == "" {
						conf.Budgets[i].Currency = b.CurrencyFormat.IsoCode
					}

					found = true

					break
				}
			}

			if !found {
				return fmt.Errorf("Unable to find ID for budget: %s", budgetConfig.Name)
			}
		}

		if conf.Budgets[i].Conversions == nil {
			conf.Budgets[i].Conversions = make(config.CurrencyConversion)
		}

		for _, currency := range conf.Currencies {
			if _, ok := conf.Budgets[i].Conversions[currency]; ok {
				continue
			}

			conf.Budgets[i].Conversions[currency], err = importer.currencyConverter.ConversionRate(conf.Budgets[i].Currency, currency)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (importer *ImportYNABRunner) deleteRowsByDate(table string, date string, filters map[string]string) error {
	queryString := fmt.Sprintf("DELETE FROM \"%s\" where date = ?", table)
	// i := 2

	parmas := []interface{}{date}

	for key, value := range filters {
		queryString += fmt.Sprintf(" AND \"%v\" = ?", key)
		parmas = append(parmas, value)
	}

	_, err := importer.db.Exec(queryString, parmas...)
	return err
}
