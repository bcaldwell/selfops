package ynabimporter

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"time"

	"github.com/bcaldwell/selfops/pkg/config"
	"github.com/uptrace/bun"
)

// INSERT INTO networth(date, usd, cad, budget_breakdown)
// SELECT date, "USD", "CAD", json_build_object('USD', json_build_object('cad', "USD_CAD", 'usd', "USD_USD"), 'CAD', json_build_object('cad', "CAD_CAD", 'usd', "CAD_USD"))
// FROM networth_old;

type SQLNetWorth struct {
	bun.BaseModel   `bun:"table:networth"`
	ID              int64     `bun:",pk,autoincrement"`
	Date            time.Time `bun:",unique"`
	USD             float64
	CAD             float64
	BudgetBreakdown map[string]map[string]float64 `bun:"type:jsonb"`
}

func (s SQLNetWorth) ItemDate() time.Time {
	return s.Date
}

func addAccountToRow(row *SQLNetWorth, account SQLAccount) {
	row.USD += account.USD
	row.CAD += account.CAD

	if _, ok := row.BudgetBreakdown[account.BudgetName]; !ok {
		row.BudgetBreakdown[account.BudgetName] = map[string]float64{
			"usd": 0,
			"cad": 0,
		}
	}
	row.BudgetBreakdown[account.BudgetName]["usd"] += account.USD
	row.BudgetBreakdown[account.BudgetName]["cad"] += account.CAD
}

func (importer *ImportYNABRunner) importNetworth(accounts []SQLAccount) error {
	slog.Info("starting net worth import")
	tableName := config.CurrentYnabConfig().SQL.NetworthTable
	model := (*SQLNetWorth)(nil)
	// todo make this come from config
	// easiest way to handle deleted transactions, with the speed at which it works not too bad
	_, err := importer.db.NewDropTable().Model(model).ModelTableExpr(tableName).Exec(context.Background())
	if err != nil && !strings.Contains(err.Error(), fmt.Sprintf("ERROR: table \"%s\" does not exist (SQLSTATE=42P01)", tableName)) {
		return fmt.Errorf("failed to drop %s table: %w", tableName, err)
	}
	_, err = importer.db.NewCreateTable().Model(model).ModelTableExpr(tableName).IfNotExists().Exec(context.Background())
	if err != nil {
		return err
	}

	slices.SortFunc(accounts, func(a, b SQLAccount) int {
		return a.Date.Compare(b.Date)
	})

	rows := []SQLNetWorth{}

	for _, account := range accounts {
		rows = ensureOrderedSqlRecordsForDate(account.Date, func(t time.Time, last *SQLNetWorth) *SQLNetWorth {
			return &SQLNetWorth{
				Date:            t,
				BudgetBreakdown: map[string]map[string]float64{},
			}
		}, rows)
		addAccountToRow(&rows[len(rows)-1], account)
	}

	// clean up values
	for i, row := range rows {
		for j, budgetBreakdown := range row.BudgetBreakdown {
			budgetBreakdown["usd"] = Round(budgetBreakdown["usd"], 0.01)
			budgetBreakdown["cad"] = Round(budgetBreakdown["cad"], 0.01)
			rows[i].BudgetBreakdown[j] = budgetBreakdown
		}

		rows[i].CAD = Round(row.CAD, 0.01)
		rows[i].USD = Round(row.USD, 0.01)
	}

	slog.Info("About to write net worth to sql", "rows", len(rows))
	_, err = importer.db.NewInsert().
		Model(&rows).
		ModelTableExpr(tableName).
		On("CONFLICT (date) DO UPDATE").
		Set("budget_breakdown = EXCLUDED.budget_breakdown").
		Set("usd = EXCLUDED.usd").
		Set("cad = EXCLUDED.cad").
		Exec(context.Background())
	if err != nil {
		return fmt.Errorf("Failed to write net worth to db: %v", err)
	}

	slog.Info("Wrote net worth to sql", "rows", len(rows))

	return nil
}
