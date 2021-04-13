package ynabimporter

import (
	"regexp"
	"strings"

	"github.com/bcaldwell/selfops/pkg/financialimporter"
	"github.com/davidsteinsland/ynab-go/ynab"
)

type YnabTransaction struct {
	*ynab.TransactionDetail
	Regex         *regexp.Regexp
	CategoryIDMap map[string]category
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

func (t *YnabTransaction) CategoryGroup() string {
	if t.CategoryId == nil {
		return ""
	}
	return t.CategoryIDMap[*t.CategoryId].Group
}

func (t *YnabTransaction) Memo() string {
	if t.TransactionDetail.Memo == nil {
		return ""
	}
	return *t.TransactionDetail.Memo
}

func (t *YnabTransaction) Amount() float64 {
	return float64(t.TransactionDetail.Amount) / 1000.0
}

func (t *YnabTransaction) TransactionType() financialimporter.TransactionType {
	// check if its a transfer first so all transfer ins arent reported as income
	if t.TransferAccountId != nil {
		// transfers might be only counted in one account
		return financialimporter.Transfer
	}

	if t.Amount() >= 0 {
		return financialimporter.Income
	}

	return financialimporter.Expense
}

func (t *YnabTransaction) Tags() []string {
	return tagsList(t.Regex, t.Memo())
}

func (t *YnabTransaction) HasSubTransactions() bool {
	return len(t.TransactionDetail.SubTransactions) > 0
}

func (t *YnabTransaction) SubTransactions() []financialimporter.Transaction {
	transactions := make([]financialimporter.Transaction, len(t.TransactionDetail.SubTransactions))
	for i := range t.TransactionDetail.SubTransactions {
		transactions[i] = &YnabSubTransaction{&t.TransactionDetail.SubTransactions[i], t}
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
	if t.CategoryId == nil {
		return ""
	}
	return t.Parent.CategoryIDMap[*t.CategoryId].Name
}

func (t *YnabSubTransaction) CategoryGroup() string {
	if t.CategoryId == nil {
		return ""
	}
	return t.Parent.CategoryIDMap[*t.CategoryId].Group
}

func (t *YnabSubTransaction) Memo() string {
	if t.SubTransaction.Memo == nil || *t.SubTransaction.Memo == "" {
		return t.Parent.Memo()
	}

	return *t.SubTransaction.Memo
}

func (t *YnabSubTransaction) Amount() float64 {
	return float64(t.SubTransaction.Amount) / 1000.0
}

func (t *YnabSubTransaction) TransactionType() financialimporter.TransactionType {
	if t.Amount() >= 0 {
		return financialimporter.Income
	}

	if t.TransferAccountId != nil {
		// transfers might be only counted in one account
		return financialimporter.Transfer
	}

	return financialimporter.Expense
}

func (t *YnabSubTransaction) Tags() []string {
	return tagsList(t.Parent.Regex, t.Memo())
}

func (t *YnabSubTransaction) HasSubTransactions() bool {
	return false
}

func (t *YnabSubTransaction) SubTransactions() []financialimporter.Transaction {
	return []financialimporter.Transaction{}
}

func (t *YnabSubTransaction) Account() string {
	return t.Parent.AccountName
}

func (t *YnabSubTransaction) IndexKey() string {
	return t.Id
}

func tagsList(regex *regexp.Regexp, memo string) []string {
	var tags []string
	parts := strings.Split(memo, ",")
	for _, s := range parts {
		// remove spaces and conver to lowercase
		s = strings.ToLower(strings.TrimSpace(s))
		if regex.Match([]byte(s)) {
			tags = append(tags, s)
		}
	}
	return tags
}
