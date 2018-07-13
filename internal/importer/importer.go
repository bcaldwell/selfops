package importer

import (
	"fmt"

	"github.com/davidsteinsland/ynab-go/ynab"
)

func ImportYNAB() error {
	ynabClient := ynab.NewDefaultClient(secrets.YnabAccessToken)

	influxClient, err := createInfluxClient(secrets)
	if err != nil {
		return fmt.Errorf("Error creating InfluxDB Client: %s", err.Error())
	}

	dropTable(influxClient, config.TransactionsDatabase)
	createTable(influxClient, config.TransactionsDatabase)
	createTable(influxClient, config.AccountsDatabase)

	err = detectBudgetIDs(ynabClient, &config)
	if err != nil {
		return fmt.Errorf("Error detecting budget IDs: %s", err)
	}

	for _, b := range config.Budgets {
		err = importTransactions(ynabClient, influxClient, b, config.Currencies)
		if err != nil {
			return err
		}
		err = importAccounts(ynabClient, influxClient, b, config.Currencies)
		if err != nil {
			return err
		}
	}

	return nil
}
