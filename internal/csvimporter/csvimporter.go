package csvimporter

import (
	"bufio"
	"database/sql"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

	"github.com/bcaldwell/selfops/internal/config"
	"github.com/bcaldwell/selfops/internal/postgresHelper"
	"github.com/bcaldwell/selfops/pkg/financialimporter"
	"github.com/sirupsen/logrus"
)

const LogLevelEnv = "SELFOPS_LOG_LEVEL"

var defaultRegex = "^[A-Za-z0-9]([A-Za-z0-9\\-\\_]+)?$"

type ImportCSVRunner struct {
	db      *sql.DB
	csvFile string
	log     *logrus.Logger
}

func (i *ImportCSVRunner) Run() error {
	csvFile, err := os.Open(i.csvFile)

	if err != nil {
		return fmt.Errorf("failed to open %s csv file %w", i.csvFile, err)
	}

	reader := csv.NewReader(bufio.NewReader(csvFile))

	header, err := reader.Read()
	if err != nil {
		return fmt.Errorf("failed to parse %s csv header %w", i.csvFile, err)
	}

	headerMap := generateHeaderMap(header)

	regexPattern := config.CurrentYnabConfig().Tags.RegexMatch
	if regexPattern == "" {
		regexPattern = defaultRegex
	}

	regex := regexp.MustCompile(regexPattern)
	transactionConfig := config.CurrentCSVConfig().Transactions

	transactions := []financialimporter.Transaction{}

	for {
		line, error := reader.Read()
		if error == io.EOF {
			break
		} else if error != nil {
			return fmt.Errorf("failed to parse %s csv row %w", i.csvFile, err)
		}
		transactions = append(transactions, &CSVTransaction{
			record:            line,
			headerMap:         headerMap,
			regex:             regex,
			transactionConfig: transactionConfig,
			log:               i.log,
		})
	}

	importer := financialimporter.NewTransactionImporter(
		i.db,
		transactions,
		transactionConfig.CalculatedFields,
		transactionConfig.Currency,
		transactionConfig.Currencies,
		transactionConfig.ImportAfterDate,
		transactionConfig.Table,
	)

	err = importer.DropSQLTable()
	if err != nil {
		return err
	}

	err = importer.CreateOrUpdateSQLTable()
	if err != nil {
		return err
	}

	written, err := importer.Import()
	if err != nil {
		return err
	}

	fmt.Printf("Wrote %d transactions to sql from csv file %s %s\n", written, i.csvFile)

	return nil
}

func (i *ImportCSVRunner) Close() error {
	return i.db.Close()
}

func NewImportCSVRunner(csvFile string) (*ImportCSVRunner, error) {
	log := logrus.New()
	log.SetReportCaller(true)

	level, err := logrus.ParseLevel(os.Getenv(LogLevelEnv))
	if err != nil {
		level = logrus.InfoLevel
	}

	log.SetLevel(level)

	db, err := postgresHelper.CreatePostgresClient(config.CurrentYnabConfig().SQL.YnabDatabase)
	if err != nil {
		return nil, fmt.Errorf("Error connecting to postgres DB: %s", err)
	}

	log.Infof("Connected to postgres database %v", config.CurrentYnabConfig().SQL.YnabDatabase)

	return &ImportCSVRunner{
		db: db, log: log, csvFile: csvFile,
	}, nil
}

// generateHeaderMap creates a header map from the passed in header row and saves it to the CSVTransaction struct
func generateHeaderMap(record []string) map[string]int {
	m := make(map[string]int)
	for i, r := range record {
		m[strings.ToLower(r)] = i
	}
	return m
}
