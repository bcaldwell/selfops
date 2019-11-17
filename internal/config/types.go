package config

type Config struct {
	Ynab     YnabConfig
	Airtable AirtableConfig
	CSV      CSVConfig
}

type Secrets struct {
	Ynab     YnabSecrets
	Airtable AirtableSecrets
	Influx   InfluxSecrets
	SQL      SQLSecrets `json:"sql"`

	// Altternative to Sql struct, also specifies table name which will be used for all importer
	// designed to be used with heroku env variable
	DatabaseURL string `env:"DATABASE_URL"`
}

///////////////////////////////////////////////////////////////////////////////////////
// YNAB
///////////////////////////////////////////////////////////////////////////////////////

type YnabConfig struct {
	UpdateFrequency string
	Currencies      []string
	Budgets         []Budget
	SQL             struct {
		YnabDatabase      string
		TransactionsTable string
		AccountsTable     string
		BudgetsTable      string
		NetworthTable     string
	}
	Tags struct {
		Enabled    bool
		RegexMatch string
	}
}

type Budget struct {
	Name string `json:"name"`
	// Date to import transactions after
	ImportAfterDate  string             `json:"importAfterDate"`
	ID               string             `json:"id"`
	Currency         string             `json:"currency"`
	Conversions      CurrencyConversion `json:"conversions"`
	CalculatedFields []CalculatedField
	// CalculatedFields []financialimporter.CalculatedField
}

type CalculatedField struct {
	Name          string
	Category      []string
	CategoryGroup []string
	Payee         []string
	Inverted      bool
}

type CurrencyConversion map[string]float64

type YnabSecrets struct {
	YnabAccessToken string `json:"ynabAccessToken" env:"YNAB_ACCESS_TOKEN"`
}

type InfluxSecrets struct {
	InfluxEndpoint string
	InfluxUsername string
	InfluxPassword string
}

type SQLSecrets struct {
	SQLHost     string `json:"sqlHost" env:"SQL_HOST"`
	SQLUsername string `json:"sqlUsername" env:"SQL_USERNAME"`
	SQLPassword string `json:"sqlPassword" env:"SQL_PASSWORD"`
}

///////////////////////////////////////////////////////////////////////////////////////
// Airtable
///////////////////////////////////////////////////////////////////////////////////////

type AirtableConfig struct {
	UpdateFrequency  string               `json:"updateFrequency"`
	AirtableDatabase string               `json:"airtableDatabase"`
	AirtableBases    []AirtableBaseConfig `json:"airtableBases"`
}

type AirtableBaseConfig struct {
	BaseID            string `json:"airtableBaseId"`
	AirtableTableName string
	InfluxMeasurement string
	Fields            AirtableFieldsConfig
}

type AirtableFieldsConfig struct {
	ConvertToTimeFromMidnightList []string
	ConvertBoolToInt              bool
	Blacklist                     []string
}

type AirtableSecrets struct {
	AirtableAPIKey string `json:"airtableApiKey"`
}

///////////////////////////////////////////////////////////////////////////////////////
// CSV
///////////////////////////////////////////////////////////////////////////////////////

type CSVConfig struct {
	Transactions CSVTransactionConfig `json:"transactions"`
}

type CSVTransactionConfig struct {
	ColumnTranslation map[string]string `json:"columnTransactions"`
	Table             string
	CalculatedFields  []CalculatedField
	Currency          string
	Currencies        []string
	ImportAfterDate   string
}
