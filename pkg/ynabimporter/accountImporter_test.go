package ynabimporter

import (
	"testing"

	"github.com/davidsteinsland/ynab-go/ynab"
	"github.com/stretchr/testify/assert"
)

func TestAccountAppend(t *testing.T) {
	a := accountAggregator{
		balance:     1000 * balanceMultiplier,
		name:        "testing",
		accountType: "checking",
		onBudget:    true,
		currency:    "USD",
		budgetName:  "main",
		conversion: map[string]float64{
			"USD": 1.0,
			"CAD": 1.3,
		},
		sql: []SQLAccount{},
	}

	a.appendTransaction(ynab.TransactionSummary{
		Date:   "2024-01-01",
		Amount: 100 * balanceMultiplier,
	})
	a.appendTransaction(ynab.TransactionSummary{
		Date:   "2024-01-10",
		Amount: 500 * balanceMultiplier,
	})
	a.appendTransaction(ynab.TransactionSummary{
		Date:   "2024-01-15",
		Amount: 400 * balanceMultiplier,
	})

	// b, _ := json.MarshalIndent(a.sql, "", "  ")
	// fmt.Println(string(b))
	assert.Len(t, a.sql, 15)
	assert.Equal(t, a.sql[0].Balance, 100.0)
	assert.Equal(t, a.sql[3].Balance, 100.0)
	assert.Equal(t, a.sql[9].Balance, 600.0)
	assert.Equal(t, a.sql[11].Balance, 600.0)
	assert.Equal(t, a.sql[14].Balance, 1000.0)
}
