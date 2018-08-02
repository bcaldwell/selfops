package config

type Config struct {
	YnabConfig
	AirtableConfig
}

type Secrets struct {
	YnabSecrets
	AirtableSecrets
}

///////////////////////////////////////////////////////////////////////////////////////
// YNAB
///////////////////////////////////////////////////////////////////////////////////////

type YnabConfig struct {
	// YnabTable            string   `json:"ynabTable"`
	TransactionsDatabase string   `json:"transactionsDatabase"`
	AccountsDatabase     string   `json:"accountsDatabase"`
	UpdateFrequency      string   `json:"updateFrequency"`
	Currencies           []string `json:"currencies"`
	Budgets              []Budget `json:"budgets"`
	Tags                 struct {
		Enabled    bool
		RegexMatch string
	}
}

type YnabSecrets struct {
	YnabAccessToken string `json:"ynabAccessToken"`
	InfluxEndpoint  string `json:"influxEndpoint"`
	InfluxUser      string `json:"influxUser"`
	InfluxPassword  string `json:"influxPassword"`
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
	AirtableBaseID string `json:"airtableBaseId"`
	// UpdateFrequency  string `json:"updateFrequency"`
	AirtableDatabase  string `json:"airtableDatabase"`
	AirtableTableName string `json:"airtableTableName"`
}

type AirtableSecrets struct {
	AirtableAPIKey string `json:"airtableApiKey"`
}
