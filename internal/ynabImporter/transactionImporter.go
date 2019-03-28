package ynabImporter

import (
	"database/sql"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/bcaldwell/selfops/internal/config"
	"github.com/bcaldwell/selfops/internal/postgresHelper"
	"github.com/davidsteinsland/ynab-go/ynab"
	influx "github.com/influxdata/influxdb/client/v2"
)

type TransactionType string
type category struct {
	Name  string
	Group string
	Id    string
}

var baseTransactionsSqlSchema = map[string]string{
	"transactionDate":  "timestamp",
	"transactionMonth": "timestamp",
	"category":         "varchar",
	"categoryGroup":    "varchar",
	"payee":            "varchar",
	"account":          "varchar",
	"memo":             "text",
	"currency":         "varchar",
	"amount":           "float8",
	"transactionType":  "varchar",
	"tags":             "varchar[]",
	"updatedAt":        "timestamp",
}

const (
	expense  TransactionType = "expense"
	income                   = "income"
	transfer                 = "transfer"
)

var default_Regex = "^[A-Za-z0-9]([A-Za-z0-9\\-\\_]+)?$"

// http://www.postgresqltutorial.com/postgresql-array/

func importTransactions(ynabClient *ynab.Client, influxClient influx.Client, sqlClient *sql.DB, budget config.Budget, currencies []string) error {

	sqlRecords := make([]map[string]string, 0)

	categoryGroups, err := ynabClient.CategoriesService.List(budget.ID)
	if err != nil {
		return fmt.Errorf("Unable to get category from id: %s", err.Error())
	}

	categories := make(map[string]category)

	for _, categoryGroup := range categoryGroups {
		for _, c := range categoryGroup.Categories {
			categories[c.Id] = category{
				Id:    c.Id,
				Name:  c.Name,
				Group: categoryGroup.CategoryGroup.Name,
			}
		}
	}

	// map[tag]{
	// 	category,
	//  categoryGroup
	// }

	regexPattern := config.CurrentYnabConfig().Tags.RegexMatch
	if regexPattern == "" {
		regexPattern = default_Regex
	}
	regex := regexp.MustCompile(regexPattern)

	bp, err := influx.NewBatchPoints(influx.BatchPointsConfig{
		Database:  config.CurrentYnabConfig().Influx.YnabDatabase,
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
			pt, sqlInsert, err := createPtForTransaction(regex, budget, currencies, categories, transaction)
			if err != nil {
				return err
			}
			sqlRecords = append(sqlRecords, sqlInsert)
			bp.AddPoint(pt)
		} else {
			for _, t := range transaction.SubTransactions {
				// todo: fix fallback here
				var transactionCategory string
				var transactionCategoryId *string

				if t.CategoryId != nil {
					transactionCategory = categories[*t.CategoryId].Name
					transactionCategoryId = t.CategoryId
				} else {
					transactionCategory = categories[*transaction.CategoryId].Name
					transactionCategoryId = transaction.CategoryId
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

				pt, sqlInsert, err := createPtForTransaction(regex, budget, currencies, categories, ynab.TransactionDetail{
					CategoryName: transactionCategory,
					PayeeName:    payeeName,
					AccountName:  transaction.AccountName,
					TransactionSummary: ynab.TransactionSummary{
						Memo:              memo,
						Amount:            t.Amount,
						TransferAccountId: transaction.TransferAccountId,
						Date:              transaction.Date,
						CategoryId:        transactionCategoryId,
					},
				})
				if err != nil {
					return err
				}
				sqlRecords = append(sqlRecords, sqlInsert)
				bp.AddPoint(pt)
			}
		}
	}

	err = influxClient.Write(bp)
	if err != nil {
		return fmt.Errorf("Error writing to influx: %s", err.Error())
	}

	err = postgresHelper.InsertRecords(sqlClient, "transactions", sqlRecords)
	if err != nil {
		return fmt.Errorf("Error writing to sql: %s", err.Error())
	}

	fmt.Printf("Wrote %d transactions to influx from budget %s\n", len(transactions), budget.Name)

	return nil
}

func createPtForTransaction(regex *regexp.Regexp, budget config.Budget, currencies []string, categories map[string]category, transaction ynab.TransactionDetail) (*influx.Point, map[string]string, error) {
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

	var categoryGroup string
	if transaction.CategoryId != nil {
		categoryGroup = categories[*transaction.CategoryId].Group
	}

	t, err := time.Parse("2006-01-02", transaction.Date)
	if err != nil {
		return nil, nil, fmt.Errorf("Unable to parse date: %s", err.Error())
	}

	transactionMonth := time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, t.Location())
	// fmt.Println(t.String() + "  " + transactionMonth.String())

	tags := map[string]string{
		"category":         transaction.CategoryName,
		"categoryGroup":    categoryGroup,
		"payee":            transaction.PayeeName,
		"account":          transaction.AccountName,
		"memo":             memo,
		"currency":         budget.Currency,
		"amount":           strconv.FormatFloat(amount, 'f', 2, 64),
		"transactionType":  string(transactionType),
		"transactionMonth": transactionMonth.Format("2006-01-02"),
	}

	for _, field := range budget.CalculatedFields {
		tags[field.Name] = strconv.FormatBool(calculateField(field, transaction, categoryGroup))
	}

	memoTags := tagsList(regex, memo)
	for _, t := range memoTags {
		tags[t] = "true"
	}

	sqlInsert := tags
	if len(memoTags) != 0 {
		sqlInsert["tags"] = fmt.Sprintf("{\"%s\"}", strings.Join(memoTags, "\",\""))
	} else {
		sqlInsert["tags"] = ""
	}

	sqlInsert["transactionDate"] = transaction.Date
	sqlInsert["updatedAt"] = time.Now().Format(time.UnixDate)

	fields := map[string]interface{}{
		"amount": amount,
	}

	for _, currency := range currencies {
		value := Round(amount*budget.Conversions[currency], 0.01)
		fields[currency] = value
		sqlInsert[currency] = strconv.FormatFloat(value, 'f', 2, 64)
	}

	pt, err := influx.NewPoint(config.CurrentYnabConfig().Influx.TransactionsMeasurement, tags, fields, t)
	if err != nil {
		return nil, nil, fmt.Errorf("Error adding new point: %s", err.Error())
	}
	return pt, sqlInsert, nil
}

func calculateField(field config.CalculatedField, transaction ynab.TransactionDetail, categoryGroup string) bool {
	return stringInSlice(transaction.CategoryName, field.Category) ||
		stringInSlice(categoryGroup, field.CategoryGroup) ||
		stringInSlice(transaction.PayeeName, field.Payee)
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

func createTransactionsSQLSchema() map[string]string {
	schema := baseTransactionsSqlSchema

	for _, budget := range config.CurrentYnabConfig().Budgets {
		for _, field := range budget.CalculatedFields {
			if _, ok := schema[field.Name]; !ok {
				schema[field.Name] = "boolean"
			}
		}
	}
	for _, currency := range config.CurrentYnabConfig().Currencies {
		schema[currency] = "float8"
	}
	return schema
}

func stringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}
