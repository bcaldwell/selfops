package importer

import (
	"fmt"

	"github.com/davidsteinsland/ynab-go/ynab"
)

const TableName = "transactions"

func ImportYNAB() error {
	config, err := readConfig("./config.yml")
	if err != nil {
		return err
	}

	secrets, err := readSecrets("./secrets.json")
	if err != nil {
		return err
	}

	ynabClient := ynab.NewDefaultClient(secrets.YnabAccessToken)

	influxClient, err := createInfluxClient(*secrets)
	if err != nil {
		return fmt.Errorf("Error creating InfluxDB Client: %s", err.Error())
	}

	dropTable(influxClient, TableName)
	createTable(influxClient, TableName)

	err = detectBudgetIDs(ynabClient, config)
	if err != nil {
		return fmt.Errorf("Error detecting budget IDs: %s", err)
	}

	for _, b := range config.Budgets {
		err = importTransactions(ynabClient, influxClient, b, config.Currencies)
		if err != nil {
			return err
		}
	}

	return nil
}
