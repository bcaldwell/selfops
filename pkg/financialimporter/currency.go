package financialimporter

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"
)

const CurrencyConversionEndpoint = "https://api.exchangeratesapi.io"

// {"rates":{"CAD":1.3259376651},"date":"2019-03-19","base":"USD"}
type CurrencyConversionResponse struct {
	Date  string
	Base  string
	Rates map[string]float64
}

func generateCurrencyConversions(baseCurrency string, currencies []string) (CurrencyConversion, error) {
	wg := sync.WaitGroup{}
	mutex := sync.Mutex{}

	conversions := make(CurrencyConversion)

	var conversionErr error = nil

	for _, currency := range currencies {
		wg.Add(1)

		go func(currency string) {
			conversion, err := conversionRate(baseCurrency, currency)
			if err != nil {
				conversionErr = err
				return
			}

			mutex.Lock()
			defer mutex.Unlock()

			conversions[currency] = conversion
			wg.Done()
		}(currency)
	}

	wg.Wait()

	if conversionErr != nil {
		return nil, conversionErr
	}

	return conversions, nil
}

func conversionRate(from, to string) (float64, error) {
	// latest query https://api.exchangeratesapi.io/latest?symbols=USD,CAD&base=USD
	// support for history queries...
	// https://api.exchangeratesapi.io/history?start_at=2018-01-01&end_at=2018-01-02&symbols=USD,GBP&base=USD
	conversionString := fmt.Sprintf("%s,%s", strings.ToUpper(from), strings.ToUpper(to))

	req, err := http.NewRequest("GET", CurrencyConversionEndpoint+"/latest", nil)
	if err != nil {
		return 0, err
	}

	q := req.URL.Query()
	q.Add("base", from)
	q.Add("symbols", conversionString)
	req.URL.RawQuery = q.Encode()

	rs, err := http.DefaultClient.Do(req)

	if err != nil {
		return 0, fmt.Errorf("error getting currency conversion: %s", err)
	}

	defer rs.Body.Close()

	bodyBytes, err := ioutil.ReadAll(rs.Body)
	if err != nil {
		return 0, fmt.Errorf("error parsing currency conversion response: %s", err)
	}

	var currencyConversionResponse CurrencyConversionResponse

	err = json.Unmarshal(bodyBytes, &currencyConversionResponse)
	if err != nil {
		return 0, err
	}

	if rate, ok := currencyConversionResponse.Rates[to]; ok {
		return rate, nil
	}

	return 0, fmt.Errorf("invalid currency")
}
