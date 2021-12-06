package ynabimporter

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/bcaldwell/selfops/internal/config"
	"github.com/bcaldwell/selfops/internal/postgresutils"
	"github.com/uptrace/bun"
	"k8s.io/klog"
)

// INSERT INTO accounts(key, date, name, currency, budget_name, on_budget, type, balance, usd, cad)
// SELECT
//     FORMAT('%s::%s::%s', to_char(date, 'MM-DD-YYYY'), "budgetName", name) as key,
//     date,
// 	name,
// 	currency,
// 	"budgetName" as budget_name,
// 	"onBudget" as on_budget,
// 	type,
// 	balance,
// 	"USD" as usd,
// 	"CAD" as cad
// FROM accounts_old;

type SQLAccount struct {
	bun.BaseModel `bun:"table:accounts"`
	ID            int64  `bun:",pk,autoincrement"`
	Key           string `bun:",pk,unique"`
	Date          time.Time
	Name          string
	Currency      string
	BudgetName    string
	OnBudget      bool
	Type          string
	Balance       float64
	USD           float64
	CAD           float64
}

func (importer *ImportYNABRunner) importAccounts(budget config.Budget, currencies []string) error {
	model := (*SQLAccount)(nil)
	tableName := config.CurrentYnabConfig().SQL.AccountsTable
	// todo make this come from config
	_, err := importer.db.NewCreateTable().Model(model).ModelTableExpr(tableName).IfNotExists().Exec(context.Background())
	if err != nil {
		return err
	}

	sqlRecords := make([]SQLAccount, 0)

	currencyNetworths := make(map[string]float64)
	for _, currency := range currencies {
		currencyNetworths[currency] = 0
	}

	accounts := importer.budgets[budget.ID].Accounts

	date := time.Now().UTC().Truncate(24 * time.Hour)

	for _, account := range accounts {
		if account.Closed {
			continue
		}

		balance := float64(account.Balance) / 1000.0

		row := SQLAccount{
			Key:        fmt.Sprintf("%s::%s::%s", date.Format("01-02-2006"), budget.Name, account.Name),
			Balance:    balance,
			USD:        Round(balance*budget.Conversions["USD"], 0.01),
			CAD:        Round(balance*budget.Conversions["CAD"], 0.01),
			Name:       account.Name,
			Type:       account.Type,
			OnBudget:   account.OnBudget,
			Currency:   budget.Currency,
			BudgetName: budget.Name,
			Date:       date,
		}

		sqlRecords = append(sqlRecords, row)
	}

	_, err = importer.db.NewInsert().
		Model(&sqlRecords).
		ModelTableExpr(tableName).
		On("CONFLICT (key) DO UPDATE").
		Set(postgresutils.TableSetString(importer.db, model, "id", "key")).
		Exec(context.Background())
	if err != nil {
		return fmt.Errorf("Error writing accounts to sql: %s", err.Error())
	}

	klog.Infof("Wrote %d accounts to sql from budget %s\n", len(accounts), budget.Name)

	return nil
}

func Round(x, unit float64) float64 {
	return math.Round(x/unit) * unit
}
