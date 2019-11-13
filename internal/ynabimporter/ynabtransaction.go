package ynabimporter

import (
	"regexp"
	"strings"

	"github.com/davidsteinsland/ynab-go/ynab"
)

type YnabTransaction struct {
	*ynab.TransactionDetail
	BudgetCurrency string
	Regex          *regexp.Regexp
	CategoryIdMap  map[string]category
}

func (t *YnabTransaction) Date() string {
	return t.TransactionSummary.Date
}

func (t *YnabTransaction) Payee() string {
	return t.PayeeName
}

func (t *YnabTransaction) Category() string {
	return t.CategoryName
}

func (t *YnabTransaction) Memo() string {
	return *t.TransactionDetail.Memo
}

func (t *YnabTransaction) Currency() string {
	return t.BudgetCurrency
}

func (t *YnabTransaction) Amount() float64 {
	return float64(t.TransactionDetail.Amount) / 1000.0
}

func (t *YnabTransaction) Tags() []string {
	var tags []string
	parts := strings.Split(t.Memo(), ",")

	for _, s := range parts {
		// remove spaces and conver to lowercase
		s = strings.ToLower(strings.TrimSpace(s))
		if t.Regex.Match([]byte(s)) {
			tags = append(tags, s)
		}
	}
	return tags
}

func (t *YnabTransaction) SubTransactions() []Transaction {
	transactions := []Transaction{}
	for _, transaction := range t.TransactionDetail.SubTransactions {
		transactions = append(transactions, &YnabSubTransaction{&transaction, t})
	}
	return transactions
}

func (t *YnabTransaction) Account() string {
	return t.AccountName
}

func (t *YnabTransaction) IndexKey() string {
	return t.Id
}

type YnabSubTransaction struct {
	*ynab.SubTransaction
	Parent *YnabTransaction
}

func (t *YnabSubTransaction) Date() string {
	return t.Parent.TransactionSummary.Date
}

func (t *YnabSubTransaction) Payee() string {
	return t.Parent.PayeeName
}

func (t *YnabSubTransaction) Category() string {
	return t.Parent.CategoryIdMap[*t.CategoryId].Name
}

func (t *YnabSubTransaction) Memo() string {
	if t.SubTransaction.Memo != nil {
		return *t.SubTransaction.Memo
	}
	return t.Parent.Memo()
}

func (t *YnabSubTransaction) Currency() string {
	return t.Parent.BudgetCurrency
}

func (t *YnabSubTransaction) Amount() float64 {
	return float64(t.SubTransaction.Amount) / 1000.0
}

func (t *YnabSubTransaction) Tags() []string {
	var tags []string
	parts := strings.Split(t.Memo(), ",")

	for _, s := range parts {
		// remove spaces and conver to lowercase
		s = strings.ToLower(strings.TrimSpace(s))
		if t.Parent.Regex.Match([]byte(s)) {
			tags = append(tags, s)
		}
	}
	return tags
}

func (t *YnabSubTransaction) SubTransactions() []Transaction {
	return []Transaction{}
}

func (t *YnabSubTransaction) Account() string {
	return t.Parent.AccountName
}

func (t *YnabSubTransaction) IndexKey() string {
	return t.Id
}
