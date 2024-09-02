package financialimporter

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/bcaldwell/selfops/pkg/config"
	"github.com/bcaldwell/selfops/pkg/postgresutils"
	"github.com/uptrace/bun"
)

type SQLTransaction struct {
	bun.BaseModel    `bun:"table:transactions"`
	ID               int64  `bun:",pk,autoincrement"`
	Key              string `bun:",pk,unique"`
	TransactionDate  time.Time
	TransactionMonth time.Time
	Category         string
	CategoryGroup    string
	Payee            string
	Account          string
	Memo             string `bun:"type:text"`
	Currency         string
	Amount           float64
	USD              float64
	CAD              float64
	TransactionType  string
	Tags             []string               `bun:",array"`
	Fields           map[string]interface{} `bun:"type:jsonb"`
	UpdatedAt        time.Time
}

func NewTransactionImporter(db *bun.DB, currencyConverter *CurrencyConverter, transactions []Transaction, calculatedFields []config.CalculatedField, transactionCurrency string, currencies []string, importAfterDate time.Time, sqlTable string) FinancialImporter {
	return &TransactionImporter{
		db:                  db,
		currencyConverter:   currencyConverter,
		calculatedFields:    calculatedFields,
		transactions:        transactions,
		transactionCurrency: transactionCurrency,
		currencies:          currencies,
		importAfterDate:     importAfterDate,
		sqlTable:            sqlTable,
	}
}

type TransactionImporter struct {
	db                  *bun.DB
	currencyConverter   *CurrencyConverter
	calculatedFields    []config.CalculatedField
	transactions        []Transaction
	importAfterDate     time.Time
	transactionCurrency string
	currencies          []string
	currencyConversions CurrencyConversion
	sqlTable            string
}

// server will return that a transaction is deleted
// {
// 	"data": {
// 	  "transactions": [
// 		{
// 		  "date": "2021-12-05",
// 		  "amount": -1000,
// 		  "memo": "TESTING",
// 		  "cleared": "cleared",
// 		  "approved": true,
// 		  "flag_color": null,
// 		  "account_name": "Cash",
// 		  "payee_name": "Interest",
// 		  "category_name": "Inflow: Ready to Assign",
// 		  "transfer_account_id": null,
// 		  "transfer_transaction_id": null,
// 		  "matched_transaction_id": null,
// 		  "import_id": null,
// 		  "deleted": true,
// 		  "subtransactions": []
// 		}
// 	  ],
// 	  "server_knowledge": 50996
// 	}
//   }

func (importer *TransactionImporter) Migrate() error {
	model := (*SQLTransaction)(nil)
	tableName := config.CurrentYnabConfig().SQL.TransactionsTable

	// easiest way to handle deleted transactions, with the speed at which it works not too bad
	_, err := importer.db.NewDropTable().Model(model).ModelTableExpr(tableName).Exec(context.Background())
	if err != nil && !strings.Contains(err.Error(), fmt.Sprintf("ERROR: table \"%s\" does not exist (SQLSTATE=42P01)", tableName)) {
		return fmt.Errorf("failed to drop %s table: %w", tableName, err)
	}

	_, err = importer.db.NewCreateTable().Model(model).ModelTableExpr(tableName).IfNotExists().Exec(context.Background())
	if err != nil {
		return fmt.Errorf("failed to create %s table: %w", tableName, err)
	}

	return nil
}

func (importer *TransactionImporter) Import() (int, error) {
	var err error

	model := (*SQLTransaction)(nil)
	tableName := config.CurrentYnabConfig().SQL.TransactionsTable

	importer.currencyConversions, err = generateCurrencyConversions(importer.currencyConverter, importer.transactionCurrency, importer.currencies)
	if err != nil {
		return 0, err
	}

	// sqlRecords holds a record(map) representing the sql rows to be added
	// It will be roughly the size of importer.transactions + number of sub transactions
	// set the initial size to 0 so append works but set cap to a good guess
	sqlRecords := make([]SQLTransaction, 0, len(importer.transactions))

	for _, transaction := range importer.transactions {
		// check if transaction is before cutoff date
		t, err := time.Parse("2006-01-02", transaction.Date())
		if err != nil {
			return 0, fmt.Errorf("unable to parse date: %s", err.Error())
		}

		if t.Before(importer.importAfterDate) {
			continue
		}

		sqlRow, err := importer.createSQLForTransaction(transaction)
		if err != nil {
			return 0, err
		}

		if transaction.HasSubTransactions() {
			for _, t := range transaction.SubTransactions() {
				sqlRow, err := importer.createSQLForTransaction(t)
				if err != nil {
					return 0, err
				}

				sqlRecords = append(sqlRecords, *sqlRow)
			}
		} else {
			sqlRecords = append(sqlRecords, *sqlRow)
		}
	}

	batchSize := config.CurrentYnabConfig().SQL.BatchSize
	if batchSize == 0 {
		batchSize = 1000
	}

	for i := 0; i < len(sqlRecords); i += batchSize {
		endIndex := min(len(sqlRecords), i+batchSize)

		records := sqlRecords[i:endIndex]

		_, err = importer.db.NewInsert().
			Model(&records).
			ModelTableExpr(tableName).
			On("CONFLICT (key) DO UPDATE").
			Set(postgresutils.TableSetString(importer.db, model, "id", "key")).
			Exec(context.Background())

		if err != nil {
			return 0, fmt.Errorf("error writing to sql, transaction batch start index %i: %w", i, err)
		}
	}

	return len(sqlRecords), nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (importer *TransactionImporter) createSQLForTransaction(transaction Transaction) (*SQLTransaction, error) {
	amount := transaction.Amount()

	t, err := time.Parse("2006-01-02", transaction.Date())
	if err != nil {
		return nil, fmt.Errorf("unable to parse date: %s", err.Error())
	}

	transactionMonth := time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, t.Location())

	sqlRow := SQLTransaction{
		Key:              transaction.IndexKey(),
		Category:         transaction.Category(),
		CategoryGroup:    transaction.CategoryGroup(),
		Payee:            transaction.Payee(),
		Account:          transaction.Account(),
		Memo:             transaction.Memo(),
		Currency:         importer.transactionCurrency,
		Amount:           amount,
		USD:              Round(amount*importer.currencyConversions["USD"], 0.01),
		CAD:              Round(amount*importer.currencyConversions["CAD"], 0.01),
		TransactionType:  transaction.TransactionType().String(),
		TransactionMonth: transactionMonth,
		TransactionDate:  t,
		UpdatedAt:        time.Now(),
		Fields:           make(map[string]interface{}),
	}

	for _, field := range importer.calculatedFields {
		sqlRow.Fields[field.Name] = strconv.FormatBool(calculateField(field, transaction))
	}

	sqlRow.Tags = transaction.Tags()

	return &sqlRow, nil
}

func calculateField(field config.CalculatedField, transaction Transaction) bool {
	boolean := stringInSlice(transaction.Category(), field.Category) ||
		stringInSlice(transaction.CategoryGroup(), field.CategoryGroup) ||
		stringInSlice(transaction.Payee(), field.Payee)

	if field.Inverted {
		return !boolean
	}

	return boolean
}

func stringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}

	return false
}

func Round(x, unit float64) float64 {
	return math.Round(x/unit) * unit
}
