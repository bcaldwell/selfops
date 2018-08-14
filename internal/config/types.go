package config

type Config struct {
	Ynab     YnabConfig
	Airtable AirtableConfig
}

type Secrets struct {
	Ynab     YnabSecrets
	Airtable AirtableSecrets
	Influx   InfluxSecrets
}

///////////////////////////////////////////////////////////////////////////////////////
// YNAB
///////////////////////////////////////////////////////////////////////////////////////

type YnabConfig struct {
	YnabDatabase            string   `json:"ynabDatabase"`
	TransactionsMeasurement string   `json:"transactionsMeasurement"`
	AccountsMeasurement     string   `json:"accountsMeasurement"`
	UpdateFrequency         string   `json:"updateFrequency"`
	Currencies              []string `json:"currencies"`
	Budgets                 []Budget `json:"budgets"`
	Tags                    struct {
		Enabled    bool
		RegexMatch string
	}
}

type YnabSecrets struct {
	YnabAccessToken string `json:"ynabAccessToken"`
}

type InfluxSecrets struct {
	InfluxEndpoint string `json:"influxEndpoint"`
	InfluxUser     string `json:"influxUser"`
	InfluxPassword string `json:"influxPassword"`
}

type Budget struct {
	Name        string             `json:"name"`
	ID          string             `json:"id"`
	Currency    string             `json:"currency"`
	Conversions CurrencyConversion `json:"conversions"`
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
