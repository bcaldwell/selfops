package ynabimporter

import (
	"database/sql"
	"fmt"

	"github.com/bcaldwell/selfops/internal/config"
	"github.com/bcaldwell/selfops/internal/postgresHelper"
	"github.com/davidsteinsland/ynab-go/ynab"
)

type ImportYNABRunner struct {
	ynabClient *ynab.Client
	db         *sql.DB
	budgets    map[string]ynab.BudgetDetail
	categories map[string]map[string]category
}

func (i *ImportYNABRunner) Run() error {
	return i.importYNAB()
}

func (i *ImportYNABRunner) Close() error {
	return i.db.Close()
}

func NewImportYNABRunner() (*ImportYNABRunner, error) {
	ynabClient := ynab.NewDefaultClient(config.CurrentYnabSecrets().YnabAccessToken)

	db, err := postgresHelper.CreatePostgresClient()
	if err != nil {
		return nil, fmt.Errorf("Error connecting to postgres DB: %s", err)
	}

	return &ImportYNABRunner{
		ynabClient, db, make(map[string]ynab.BudgetDetail), make(map[string]map[string]category),
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

		importer.categories[b.Name] = make(map[string]category)

		categoryGroups, err := importer.ynabClient.CategoriesService.List(b.ID)
		if err != nil {
			return fmt.Errorf("Unable to get categories for budget %s: %s", b.Name, err.Error())
		}

		for _, categoryGroup := range categoryGroups {
			for _, c := range categoryGroup.Categories {
				importer.categories[b.Name][c.Id] = category{
					Id:    c.Id,
					Name:  c.Name,
					Group: categoryGroup.CategoryGroup.Name,
				}
			}
		}
	}

	// var waitGroup sync.WaitGroup
  // waitGroup.Add(len(config.CurrentYnabConfig().Budgets))
	for _, b := range config.CurrentYnabConfig().Budgets {
		// go func() {
			err = importer.importTransactions(b, config.CurrentYnabConfig().Currencies)
			if err != nil {
				// fmt.Println(err)
				return err
			}
			err = importer.importAccounts(b, config.CurrentYnabConfig().Currencies)
			if err != nil {
				// fmt.Println(err)
				return err
			}
			// waitGroup.Done()
		// }()
	}

	// waitGroup.Wait()

	importer.importNetworth(config.CurrentYnabConfig().Budgets, config.CurrentYnabConfig().Currencies)

	return nil
}

// todo: handle my error
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

			conf.Budgets[i].Conversions[currency], err = conversionRate(conf.Budgets[i].Currency, currency)
			if err != nil {
				return err
			}

		}
	}
	return nil
}

func (importer *ImportYNABRunner) deleteRowsByDate(table string, date string, filters map[string]string) error {
	// fmt.Println("DELETE FROM \"" + table + "\" where date=" + date)
	queryString := fmt.Sprintf("DELETE FROM \"%s\" where date = $1", table)
	i := 2
	parmas := []interface{}{date}
	for key, value := range filters {
		queryString += fmt.Sprintf(" AND \"%v\" = $%v", key, i)
		parmas = append(parmas, value)
		i++
	}
	_, err := importer.db.Exec(queryString, parmas...)
	return err
}
