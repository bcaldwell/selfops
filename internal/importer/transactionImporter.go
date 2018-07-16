package importer

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/davidsteinsland/ynab-go/ynab"
	influx "github.com/influxdata/influxdb/client/v2"
)

type TransactionType string

const (
	expense  TransactionType = "expense"
	income                   = "income"
	transfer                 = "transfer"
)

func importTransactions(ynabClient *ynab.Client, influxClient influx.Client, budget Budget, currencies []string) error {
	bp, err := influx.NewBatchPoints(influx.BatchPointsConfig{
		Database:  config.TransactionsDatabase,
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
		memoTags := tagsList(memo)
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
			return fmt.Errorf("Unable to parse date: %s", err.Error())
		}

		pt, err := influx.NewPoint(config.AccountsDatabase, tags, fields, t)
		if err != nil {
			return fmt.Errorf("Error adding new point: %s", err.Error())
		}

		bp.AddPoint(pt)
	}

	err = influxClient.Write(bp)
	if err != nil {
		return fmt.Errorf("Error writing to influx: %s", err.Error())
	}

	fmt.Printf("Wrote %d transactions to influx from budget %s\n", len(transactions), budget.Name)

	return nil
}

func tagsList(memo string) []string {
	var tags []string
	parts := strings.Split(memo, ",")
	for _, s := range parts {
		if match, err := regexp.Match(config.Tags.RegexMatch, []byte(s)); match {
			s = strings.ToLower(s)
			tags = append(tags, s)
		} else if err != nil {
			fmt.Println(err)
		}
	}
	return tags
}
