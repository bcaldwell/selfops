package ynabImporter

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/bcaldwell/selfops/internal/config"
	"github.com/davidsteinsland/ynab-go/ynab"
	influx "github.com/influxdata/influxdb/client/v2"
)

type TransactionType string

const (
	expense  TransactionType = "expense"
	income                   = "income"
	transfer                 = "transfer"
)

var default_Regex = "^[A-Za-z0-9]([A-Za-z0-9\\-\\_]+)?$"

func importTransactions(ynabClient *ynab.Client, influxClient influx.Client, budget config.Budget, currencies []string) error {
	regexPattern := config.CurrentConfig().Tags.RegexMatch
	if regexPattern == "" {
		regexPattern = default_Regex
	}
	regex := regexp.MustCompile(regexPattern)

	bp, err := influx.NewBatchPoints(influx.BatchPointsConfig{
		Database:  config.CurrentConfig().TransactionsDatabase,
		Precision: "h",
	})

	if err != nil {
		return fmt.Errorf("Error creating InfluxDB point batch: %s", err.Error())
	}

	transactions, err := ynabClient.TransactionsService.List(budget.ID)
	if err != nil {
		return fmt.Errorf("Error getting transactions: %s", err.Error())

	}

	for _, transaction := range transactions {
		if len(transaction.SubTransactions) == 0 {
			pt, err := createPtForTransaction(regex, budget, currencies, transaction)
			if err != nil {
				return err
			}
			bp.AddPoint(pt)
		} else {
			for _, t := range transaction.SubTransactions {
				var category string
				if t.CategoryId != nil {
					c, err := ynabClient.CategoriesService.Get(budget.ID, *t.CategoryId)
					category = c.Name
					if err != nil {
						return fmt.Errorf("Unable to get category from id: %s", err.Error())
					}
				}
				memo := transaction.Memo
				if t.Memo != nil {
					memo = t.Memo
				}
				payeeName := transaction.PayeeName
				if t.PayeeId != nil {
					p, err := ynabClient.PayeesService.Get(budget.ID, *t.PayeeId)
					payeeName = p.Name
					if err != nil {
						return fmt.Errorf("Unable to get payee from id: %s", err.Error())
					}
				}

				pt, err := createPtForTransaction(regex, budget, currencies, ynab.TransactionDetail{
					CategoryName: category,
					PayeeName:    payeeName,
					AccountName:  transaction.AccountName,
					TransactionSummary: ynab.TransactionSummary{
						Memo:              memo,
						Amount:            t.Amount,
						TransferAccountId: transaction.TransferAccountId,
						Date:              transaction.Date,
					},
				})
				if err != nil {
					return err
				}
				bp.AddPoint(pt)
			}
		}
	}

	err = influxClient.Write(bp)
	if err != nil {
		return fmt.Errorf("Error writing to influx: %s", err.Error())
	}

	fmt.Printf("Wrote %d transactions to influx from budget %s\n", len(transactions), budget.Name)
	return nil
}

func createPtForTransaction(regex *regexp.Regexp, budget config.Budget, currencies []string, transaction ynab.TransactionDetail) (*influx.Point, error) {
	// Create a point and add to batch
	var memo string
	if transaction.Memo != nil {
		memo = *transaction.Memo
	}

	amount := float64(transaction.Amount) / 1000.0

	transactionType := expense
	if amount >= 0 {
		transactionType = income
	}
	if transaction.TransferAccountId != nil {
		// transfers might be only counted in one account
		transactionType = transfer
	}

	tags := map[string]string{
		"category":        transaction.CategoryName,
		"payee":           transaction.PayeeName,
		"account":         transaction.AccountName,
		"memo":            memo,
		"currency":        budget.Currency,
		"amount":          strconv.FormatFloat(amount, 'f', 2, 64),
		"transactionType": string(transactionType),
	}
	memoTags := tagsList(regex, memo)
	for _, t := range memoTags {
		tags[t] = "true"
	}

	fields := map[string]interface{}{
		"amount": amount,
	}

	for _, currency := range currencies {
		fields[currency] = Round(amount*budget.Conversions[currency], 0.01)
	}

	t, err := time.Parse("2006-01-02", transaction.Date)
	if err != nil {
		return nil, fmt.Errorf("Unable to parse date: %s", err.Error())
	}

	pt, err := influx.NewPoint(config.CurrentConfig().TransactionsDatabase, tags, fields, t)
	if err != nil {
		return nil, fmt.Errorf("Error adding new point: %s", err.Error())
	}
	return pt, nil
}

func tagsList(regex *regexp.Regexp, memo string) []string {
	var tags []string
	parts := strings.Split(memo, ",")
	for _, s := range parts {
		if regex.Match([]byte(s)) {
			s = strings.ToLower(s)
			tags = append(tags, s)
		}
	}
	return tags
}
