package importer

type Budget struct {
	Name        string             `json:"name"`
	ID          string             `json:"id"`
	Currency    string             `json:"currency"`
	Conversions CurrencyConversion `json:"conversions"`
}

type Config struct {
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

type Secrets struct {
	YnabAccessToken string `json:"ynab_access_token"`
	InfluxEndpoint  string `json:"influx_endpoint"`
	InfluxUser      string `json:"influx_user"`
	InfluxPassword  string `json:"influx_Password"`
}

type CurrencyConversion map[string]float64

type CurrencyConversionResponse struct {
	Query struct {
		Count int
	}
	Results map[string]struct {
		Id  string
		Val float64
		To  string
		Fr  string
	}
}
