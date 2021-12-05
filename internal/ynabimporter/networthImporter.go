package ynabimporter

import (
	"context"
	"fmt"
	"time"

	"github.com/bcaldwell/selfops/internal/config"
	"github.com/uptrace/bun"
	"k8s.io/klog"
)

// INSERT INTO networth
// SELECT id, date, "USD", "CAD", json_build_object('USD', json_build_object('cad', "USD_CAD", 'usd', "USD_USD"), 'CAD', json_build_object('cad', "CAD_CAD", 'usd', "CAD_USD"))
// FROM networth_old;

type SQLNetWorth struct {
	bun.BaseModel   `bun:"table:networth"`
	ID              int64     `bun:",pk,autoincrement"`
	Date            time.Time `bun:",unique"`
	USD             float64
	CAD             float64
	BudgetBreakdown map[string]map[string]float64 `bun:"type:jsonb"`
}

func (importer *ImportYNABRunner) importNetworth(budgets []config.Budget, currencies []string) error {
	tableName := config.CurrentYnabConfig().SQL.NetworthTable
	// todo make this come from config
	_, err := importer.db.NewCreateTable().Model((*SQLNetWorth)(nil)).ModelTableExpr(tableName).IfNotExists().Exec(context.Background())
	if err != nil {
		return err
	}

	budgetNetworths := make(map[string]map[string]float64)

	for _, budget := range budgets {
		budgetNetworths[budget.Name] = make(map[string]float64)

		accounts := importer.budgets[budget.ID].Accounts

		for _, account := range accounts {
			if account.Closed {
				continue
			}

			balance := float64(account.Balance) / 1000.0

			budgetNetworths[budget.Name]["usd"] += Round(balance*budget.Conversions["USD"], 0.01)
			budgetNetworths[budget.Name]["cad"] += Round(balance*budget.Conversions["CAD"], 0.01)
		}
	}

	netWorthRow := SQLNetWorth{
		Date:            time.Now().UTC().Truncate(24 * time.Hour),
		BudgetBreakdown: budgetNetworths,
		USD:             0,
		CAD:             0,
	}

	for k, v := range budgetNetworths {
		budgetNetworths[k]["usd"] = Round(budgetNetworths[k]["usd"], 0.01)
		budgetNetworths[k]["cad"] = Round(budgetNetworths[k]["cad"], 0.01)

		netWorthRow.CAD += v["cad"]
		netWorthRow.USD += v["usd"]
	}

	netWorthRow.CAD = Round(netWorthRow.CAD, 0.01)
	netWorthRow.USD = Round(netWorthRow.USD, 0.01)

	_, err = importer.db.NewInsert().
		Model(&netWorthRow).
		ModelTableExpr(tableName).
		On("CONFLICT (date) DO UPDATE").
		Set("budget_breakdown = EXCLUDED.budget_breakdown").
		Set("usd = EXCLUDED.usd").
		Set("cad = EXCLUDED.cad").
		Exec(context.Background())
	if err != nil {
		return fmt.Errorf("Failed to write net worth to db: %v", err)
	}

	klog.Infof("Wrote net worth to sql\n")

	return nil
}
