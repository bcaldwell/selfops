package ynabImporter

import (
	"fmt"

	"github.com/bcaldwell/selfops/internal/config"
	"github.com/bcaldwell/selfops/internal/influxHelper"
	"github.com/bcaldwell/selfops/internal/postgresHelper"
	"github.com/davidsteinsland/ynab-go/ynab"
)

type ImportYNABRunner struct{}

func (ImportYNABRunner) Run() error {
	return ImportYNAB()
}

func ImportYNAB() error {
	ynabClient := ynab.NewDefaultClient(config.CurrentYnabSecrets().YnabAccessToken)

	influxClient, err := influxHelper.CreateInfluxClient()
	if err != nil {
		return fmt.Errorf("Error creating InfluxDB Client: %s", err.Error())
	}

	err = influxHelper.DropMeasurement(influxClient, config.CurrentYnabConfig().Influx.YnabDatabase, config.CurrentYnabConfig().Influx.TransactionsMeasurement)
	if err != nil {
		return fmt.Errorf("Error dropping measurement: %s", err.Error())
	}
	err = influxHelper.CreateDatabase(influxClient, config.CurrentYnabConfig().Influx.YnabDatabase)
	if err != nil {
		return fmt.Errorf("Error creating DB: %s", err.Error())
	}

	// db, err := gorm.Open("postgres", "host=localhost port=5432 user=postgres dbname=ynab password=password sslmode=disable")
	db, err := postgresHelper.CreatePostgresClient()
	if err != nil {
		return fmt.Errorf("Error connecting to postgres DB: %s", err)
	}
	defer db.Close()

	err = postgresHelper.DropTable(db, config.CurrentYnabConfig().Sql.TransactionsTable)
	if err != nil {
		return fmt.Errorf("Error dropping table: %s", err)
	}

	err = postgresHelper.CreateTable(db, config.CurrentYnabConfig().Sql.TransactionsTable, createTransactionsSQLSchema())
	if err != nil {
		return fmt.Errorf("Error creating table: %s", err)
	}

	err = detectBudgetIDs(ynabClient, config.CurrentYnabConfig())
	if err != nil {
		return fmt.Errorf("Error detecting budget IDs: %s", err)
	}

	for _, b := range config.CurrentYnabConfig().Budgets {
		err = importTransactions(ynabClient, influxClient, db, b, config.CurrentYnabConfig().Currencies)
		if err != nil {
			return err
		}
		err = importAccounts(ynabClient, influxClient, b, config.CurrentYnabConfig().Currencies)
		if err != nil {
			return err
		}
	}

	return nil
}

// todo: handle my error
func detectBudgetIDs(ynabClient *ynab.Client, conf *config.YnabConfig) error {
	budgets, err := ynabClient.BudgetService.List()
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
