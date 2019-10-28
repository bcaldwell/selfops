package ynabimporter

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/bcaldwell/selfops/internal/config"
	"github.com/bcaldwell/selfops/internal/postgresHelper"
	"github.com/davidsteinsland/ynab-go/ynab"
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

func (importer *ImportYNABRunner) importTransactions(budget config.Budget, currencies []string) error {

	sqlRecords := make([]map[string]string, 0)

	regexPattern := config.CurrentYnabConfig().Tags.RegexMatch
	if regexPattern == "" {
		regexPattern = default_Regex
	}
	regex := regexp.MustCompile(regexPattern)

	transactions, err := importer.ynabClient.TransactionsService.List(budget.ID)
	if err != nil {
		return fmt.Errorf("Error getting transactions: %s", err.Error())
	}

	for _, transaction := range transactions {
		if len(transaction.SubTransactions) == 0 {
			sqlRow, err := importer.createSqlForTransaction(regex, budget, currencies, transaction)
			if err != nil {
				return err
			}
			sqlRecords = append(sqlRecords, sqlRow)
		} else {
			for _, t := range transaction.SubTransactions {
				// todo: fix fallback here
				var transactionCategory string
				var transactionCategoryID *string

				if t.CategoryId != nil {
					transactionCategory = importer.categories[budget.Name][*t.CategoryId].Name
					transactionCategoryID = t.CategoryId
				} else {
					transactionCategory = importer.categories[budget.Name][*transaction.CategoryId].Name
					transactionCategoryID = transaction.CategoryId
				}
				memo := transaction.Memo
				if t.Memo != nil {
					memo = t.Memo
				}
				payeeName := transaction.PayeeName
				if t.PayeeId != nil {
					p, err := importer.ynabClient.PayeesService.Get(budget.ID, *t.PayeeId)
					payeeName = p.Name
					if err != nil {
						return fmt.Errorf("Unable to get payee from id: %s", err.Error())
					}
				}

				sqlRow, err := importer.createSqlForTransaction(regex, budget, currencies, ynab.TransactionDetail{
					CategoryName: transactionCategory,
					PayeeName:    payeeName,
					AccountName:  transaction.AccountName,
					TransactionSummary: ynab.TransactionSummary{
						Memo:              memo,
						Amount:            t.Amount,
						TransferAccountId: transaction.TransferAccountId,
						Date:              transaction.Date,
						CategoryId:        transactionCategoryID,
					},
				})
				if err != nil {
					return err
				}
				sqlRecords = append(sqlRecords, sqlRow)
			}
		}
	}

	err = postgresHelper.InsertRecords(importer.db, config.CurrentYnabConfig().SQL.TransactionsTable, sqlRecords)
	if err != nil {
		return fmt.Errorf("Error writing to sql: %s", err.Error())
	}

	fmt.Printf("Wrote %d transactions to influx from budget %s\n", len(transactions), budget.Name)

	return nil
}

func (importer *ImportYNABRunner) createSqlForTransaction(regex *regexp.Regexp, budget config.Budget, currencies []string, transaction ynab.TransactionDetail) (map[string]string, error) {
	importer.recreateTransactionTable()

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
		categoryGroup = importer.categories[budget.Name][*transaction.CategoryId].Group
	}

	t, err := time.Parse("2006-01-02", transaction.Date)
	if err != nil {
		return nil, fmt.Errorf("Unable to parse date: %s", err.Error())
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

	sqlRow := tags
	if len(memoTags) != 0 {
		sqlRow["tags"] = fmt.Sprintf("{\"%s\"}", strings.Join(memoTags, "\",\""))
	} else {
		sqlRow["tags"] = ""
	}

	sqlRow["transactionDate"] = transaction.Date
	sqlRow["updatedAt"] = time.Now().Format(time.UnixDate)

	fields := map[string]interface{}{
		"amount": amount,
	}

	for _, currency := range currencies {
		value := Round(amount*budget.Conversions[currency], 0.01)
		fields[currency] = value
		sqlRow[currency] = strconv.FormatFloat(value, 'f', 2, 64)
	}

	if err != nil {
		return nil, fmt.Errorf("Error adding new point: %s", err.Error())
	}
	return sqlRow, nil
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

func (importer *ImportYNABRunner) recreateTransactionTable() error {
	err := postgresHelper.DropTable(importer.db, config.CurrentYnabConfig().SQL.TransactionsTable)
	if err != nil {
		return fmt.Errorf("Error dropping table: %s", err)
	}

	err = postgresHelper.CreateTable(importer.db, config.CurrentYnabConfig().SQL.TransactionsTable, importer.createTransactionsSQLSchema())
	if err != nil {
		return fmt.Errorf("Error creating table: %s", err)
	}
	return nil
}

func (importer *ImportYNABRunner) createTransactionsSQLSchema() map[string]string {
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
