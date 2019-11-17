package csvimporter

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/bcaldwell/selfops/internal/config"
	"github.com/bcaldwell/selfops/pkg/financialimporter"
	"github.com/sirupsen/logrus"
)

type CSVTransaction struct {
	record []string
	// map of header name to index. Header name needs to be lower case to be matched
	headerMap         map[string]int
	regex             *regexp.Regexp
	transactionConfig config.CSVTransactionConfig
	log               *logrus.Logger
}

func (t *CSVTransaction) Date() string {
	return t.getKey("date")
}

func (t *CSVTransaction) Payee() string {
	return t.getKey("payee")
}

func (t *CSVTransaction) Category() string {
	return t.getKey("category")
}

func (t *CSVTransaction) CategoryGroup() string {
	return t.getKey("category group", "Master Group")
}

func (t *CSVTransaction) Memo() string {
	return t.getKey("memo")
}

func (t *CSVTransaction) Amount() float64 {
	amountString := t.getKey("amount")

	amount, err := strconv.ParseFloat(amountString, 64)
	if err != nil {
		t.log.WithFields(logrus.Fields{"transaction": t.record}).Error("Failed to parse amount for transaction")
		return 0
	}

	return amount
}

func (t *CSVTransaction) TransactionType() financialimporter.TransactionType {
	if t.Amount() >= 0 {
		return financialimporter.Income
	}

	if strings.Contains(strings.ToLower(t.Payee()), "transfer") {
		return financialimporter.Transfer
	}

	return financialimporter.Expense
}

func (t *CSVTransaction) Tags() []string {
	tagsString := t.getKey("tags")
	if tagsString != "" {
		return strings.Split(tagsString, ",")
	}

	return t.tagsList(t.regex, t.Memo())
}

func (t *CSVTransaction) HasSubTransactions() bool {
	return false
}

func (t *CSVTransaction) SubTransactions() []financialimporter.Transaction {
	return []financialimporter.Transaction{}
}

func (t *CSVTransaction) Account() string {
	return t.getKey("account")
}

func (t *CSVTransaction) IndexKey() string {
	indexKeys := []string{
		t.Account(), t.getKey("amount"), t.Payee(),
	}

	return strings.Join(indexKeys, "-")
}

func (t *CSVTransaction) getKey(column string, defaultValue ...string) string {
	if columneNameFromConfig, ok := t.transactionConfig.ColumnTranslation[column]; ok {
		column = columneNameFromConfig
	}

	column = strings.ToLower(column)

	if i, ok := t.headerMap[column]; ok {
		return t.record[i]
	} else if len(defaultValue) > 0 {
		return defaultValue[0]
	} else {
		return ""
	}
}

func (t *CSVTransaction) tagsList(regex *regexp.Regexp, memo string) []string {
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
