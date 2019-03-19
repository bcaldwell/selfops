package config

type Config struct {
	Ynab     YnabConfig
	Airtable AirtableConfig
}

type Secrets struct {
	Ynab     YnabSecrets
	Airtable AirtableSecrets
	Influx   InfluxSecrets
	Sql      SqlSecrets
}

///////////////////////////////////////////////////////////////////////////////////////
// YNAB
///////////////////////////////////////////////////////////////////////////////////////

type YnabConfig struct {
	UpdateFrequency string
	Currencies      []string
	Budgets         []Budget
	Influx          struct {
		Enabled                 bool
		YnabDatabase            string
		TransactionsMeasurement string
		AccountsMeasurement     string
	}
	Sql struct {
		Enabled           bool
		YnabDatabase      string
		TransactionsTable string
		AccountsTable     string
	}
	Tags struct {
		Enabled    bool
		RegexMatch string
	}
}

type YnabSecrets struct {
	YnabAccessToken string `json:"ynabAccessToken"`
}

type InfluxSecrets struct {
	InfluxEndpoint string
	InfluxUsername string
	InfluxPassword string
}

type SqlSecrets struct {
	SqlHost     string
	SqlUsername string
	SqlPassword string
}

type Budget struct {
	Name        string             `json:"name"`
	ID          string             `json:"id"`
	Currency    string             `json:"currency"`
	Conversions CurrencyConversion `json:"conversions"`
	// EssentialCategories    []string
	// EssentialCategoryGroup []string
	CalculatedFields []CalculatedField
}

type CalculatedField struct {
	Name          string
	Category      []string
	CategoryGroup []string
	Payee         []string
	inverted      bool
}

type CurrencyConversion map[string]float64

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
