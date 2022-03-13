package config

type Config struct {
	Ynab     YnabConfig
	Airtable AirtableConfig
}

type Secrets struct {
	Ynab            YnabSecrets
	Airtable        AirtableSecrets
	Influx          InfluxSecrets
	SQL             SqlSecrets
	ExchangerateAPI ExchangerateAPISecrets `json:"exchangeratesapi"`

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
		BatchSize         int
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

type SqlSecrets struct {
	SqlHost     string `env:"SQL_HOST"`
	SqlUsername string `env:"SQL_USERNAME"`
	SqlPassword string `env:"SQL_PASSWORD"`
}

type ExchangerateAPISecrets struct {
	AccessKey string `json:"accessKey" env:"EXCHANGE_RATES_API_ACCESS_KEY"`
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
