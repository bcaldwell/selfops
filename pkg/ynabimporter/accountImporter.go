package ynabimporter

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/bcaldwell/selfops/pkg/config"
	"github.com/bcaldwell/selfops/pkg/postgresutils"
	"github.com/davidsteinsland/ynab-go/ynab"
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
//

const hoursInDay = 24
const balanceMultiplier = 1000.0

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

type accountAggregator struct {
	balance     float64
	name        string
	accountType string
	onBudget    bool
	currency    string
	budgetName  string
	closed      bool
	conversion  map[string]float64
	sql         []SQLAccount
}

func (a *accountAggregator) appendTransaction(transaction ynab.TransactionSummary) error {
	t, err := time.Parse("2006-01-02", transaction.Date)
	if err != nil {
		return fmt.Errorf("failed to parse transaction date: %w", err)
	}

	amount := float64(transaction.Amount) / balanceMultiplier
	i := a.ensureSqlForDate(t)
	addToBalance(&a.sql[i], amount, a.conversion)
	return nil
}

// ensureSqlForDate ensures that there is a sql account for a date. It will add one and any missing ones if needed. Returns the index of the created sql (last in array)
func (a *accountAggregator) ensureSqlForDate(date time.Time) int {
	if len(a.sql) == 0 {
		a.sql = append(a.sql, *a.newSql(date, 0))
		return 0
	}

	last := a.sql[len(a.sql)-1]
	daysToAdd := int(date.Sub(last.Date).Hours() / hoursInDay)
	for i := 1; i <= daysToAdd; i++ {
		h := hoursInDay * i
		a.sql = append(a.sql, *a.newSql(last.Date.Add(time.Hour*time.Duration(h)), last.Balance))
	}

	return len(a.sql) - 1
}

func (a *accountAggregator) newSql(date time.Time, balance float64) *SQLAccount {
	s := &SQLAccount{
		Key:        fmt.Sprintf("%s::%s::%s", date.Format("01-02-2006"), a.budgetName, a.name),
		Balance:    0,
		USD:        0,
		CAD:        0,
		Name:       a.name,
		Type:       a.accountType,
		OnBudget:   a.onBudget,
		Currency:   a.currency,
		BudgetName: a.name,
		Date:       date,
	}
	addToBalance(s, balance, a.conversion)
	return s
}

func addToBalance(s *SQLAccount, balance float64, conversion map[string]float64) {
	s.Balance += balance
	s.USD = Round(s.Balance*conversion["USD"], 0.01)
	s.CAD = Round(s.Balance*conversion["CAD"], 0.01)
}

func (importer *ImportYNABRunner) importAccounts(budget config.Budget, currencies []string) (map[string]*accountAggregator, error) {
	model := (*SQLAccount)(nil)
	tableName := config.CurrentYnabConfig().SQL.AccountsTable
	// todo make this come from config
	// easiest way to handle deleted transactions, with the speed at which it works not too bad
	_, err := importer.db.NewDropTable().Model(model).ModelTableExpr(tableName).Exec(context.Background())
	if err != nil && !strings.Contains(err.Error(), fmt.Sprintf("ERROR: table \"%s\" does not exist (SQLSTATE=42P01)", tableName)) {
		return nil, fmt.Errorf("failed to drop %s table: %w", tableName, err)
	}

	_, err = importer.db.NewCreateTable().Model(model).ModelTableExpr(tableName).IfNotExists().Exec(context.Background())
	if err != nil {
		return nil, err
	}

	sqlRecords := make([]SQLAccount, 0)

	currencyNetworths := make(map[string]float64)
	for _, currency := range currencies {
		currencyNetworths[currency] = 0
	}

	accounts := importer.budgets[budget.ID].Accounts

	// map of accountid to account info
	accountsMap := map[string]*accountAggregator{}

	for _, account := range accounts {
		balance := float64(account.Balance) / balanceMultiplier

		accountsMap[account.Id] = &accountAggregator{
			name:        account.Name,
			accountType: account.Type,
			onBudget:    account.OnBudget,
			currency:    budget.Currency,
			budgetName:  budget.Name,
			balance:     balance,
			conversion:  budget.Conversions,
			closed:      account.Closed,
			sql:         []SQLAccount{},
		}
	}

	for _, transaction := range importer.budgets[budget.ID].Transactions {
		accountsMap[transaction.AccountId].appendTransaction(transaction)
	}

	date := time.Now().UTC().Truncate(24 * time.Hour)
	for _, account := range accountsMap {
		if account.closed {
			continue
		}
		account.ensureSqlForDate(date)
	}

	for _, account := range accountsMap {

		_, err = importer.db.NewInsert().
			Model(&sqlRecords).
			ModelTableExpr(tableName).
			On("CONFLICT (key) DO UPDATE").
			Set(postgresutils.TableSetString(importer.db, model, "id", "key")).
			Exec(context.Background())
		if err != nil {
			return accountsMap, fmt.Errorf("Error writing accounts to sql: %s", err.Error())
		}

		klog.Infof("Wrote %d accounts to sql from budget %s account %s\n", len(accounts), budget.Name, account.name)
	}

	return accountsMap, nil
}

func Round(x, unit float64) float64 {
	return math.Round(x/unit) * unit
}