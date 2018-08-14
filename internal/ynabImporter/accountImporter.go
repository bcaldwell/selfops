package ynabImporter

import (
	"fmt"
	"math"
	"strconv"

	"github.com/bcaldwell/selfops/internal/config"
	"github.com/davidsteinsland/ynab-go/ynab"
	influx "github.com/influxdata/influxdb/client/v2"
)

func importAccounts(ynabClient *ynab.Client, influxClient influx.Client, budget config.Budget, currencies []string) error {
	bp, err := influx.NewBatchPoints(influx.BatchPointsConfig{
		Database:  config.CurrentYnabConfig().YnabDatabase,
		Precision: "s",
	})
	if err != nil {
		return fmt.Errorf("Error creating InfluxDB point batch: %s", err.Error())
	}

	accounts, err := ynabClient.AccountsService.List(budget.ID)
	if err != nil {
		return fmt.Errorf("Error getting accounts: %s", err.Error())

	}

	for _, account := range accounts {
		balance := float64(account.Balance) / 1000.0

		tags := map[string]string{
			"balance":  strconv.FormatFloat(balance, 'f', 2, 64),
			"name":     account.Name,
			"type":     account.Type,
			"onBudget": strconv.FormatBool(account.OnBudget),
			"currency": budget.Currency,
		}
		fields := map[string]interface{}{
			"balance": balance,
		}

		for _, currency := range currencies {
			fields[currency] = Round(balance*budget.Conversions[currency], 0.01)
		}

		pt, err := influx.NewPoint(config.CurrentYnabConfig().AccountsMeasurement, tags, fields)
		if err != nil {
			return fmt.Errorf("Error adding new point: %s", err.Error())
		}

		bp.AddPoint(pt)
	}

	err = influxClient.Write(bp)
	if err != nil {
		return fmt.Errorf("Error writing to influx: %s", err.Error())
	}

	fmt.Printf("Wrote %d accounts to influx from budget %s\n", len(accounts), budget.Name)

	return nil
}

func Round(x, unit float64) float64 {
	return math.Round(x/unit) * unit
}
