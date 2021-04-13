package financialimporter

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
)

const CurrencyConversionEndpoint = "http://api.exchangeratesapi.io"

type cacheItem struct {
	expiration time.Time
	rate       float64
}

type CurrencyConverter struct {
	accessKey string
	cache     map[string]map[string]cacheItem
}

func NewCurrencyConverter(accessKey string) *CurrencyConverter {
	cache := make(map[string]map[string]cacheItem)

	return &CurrencyConverter{
		accessKey: accessKey,
		cache:     cache,
	}
}

// {"rates":{"CAD":1.3259376651},"date":"2019-03-19","base":"USD"}
type CurrencyConversionResponse struct {
	Date      string
	Timestamp int
	Base      string
	Rates     map[string]float64
}

func (c *CurrencyConverter) ConversionRate(from, to string) (float64, error) {
	// latest query https://api.exchangeratesapi.io/latest?symbols=USD,CAD&base=USD
	// support for history queries...
	// https://api.exchangeratesapi.io/history?start_at=2018-01-01&end_at=2018-01-02&symbols=USD,GBP&base=USD
	// newest plan, query /latest and use the constant base to cache all conversions

	cacheRate, err := c.getRateFromCache(from, to)
	if err == nil {
		return cacheRate, nil
	}

	req, err := http.NewRequest("GET", CurrencyConversionEndpoint+"/latest", nil)
	if err != nil {
		return 0, err
	}

	q := req.URL.Query()
	q.Add("access_key", c.accessKey)
	req.URL.RawQuery = q.Encode()

	rs, err := http.DefaultClient.Do(req)

	if err != nil {
		return 0, fmt.Errorf("Error getting currency conversion: %s", err)
	}
	defer rs.Body.Close()

	bodyBytes, err := ioutil.ReadAll(rs.Body)
	if err != nil {
		return 0, fmt.Errorf("Error parsing currency conversion response: %s", err)
	}

	var currencyConversionResponse CurrencyConversionResponse

	err = json.Unmarshal(bodyBytes, &currencyConversionResponse)
	if err != nil {
		return 0, err
	}

	cacheExpiry := time.Now().Truncate(24 * time.Hour).Add(24 * time.Hour)

	for dest, destRate := range currencyConversionResponse.Rates {
		for src, srcRate := range currencyConversionResponse.Rates {
			if _, ok := c.cache[dest]; !ok {
				c.cache[dest] = map[string]cacheItem{}
			}

			c.cache[dest][src] = cacheItem{
				rate:       srcRate / destRate,
				expiration: cacheExpiry,
			}
		}
	}

	return c.getRateFromCache(from, to)
}

func (c *CurrencyConverter) getRateFromCache(from, to string) (float64, error) {
	fromCache, ok := c.cache[from]
	if !ok {
		return 0, fmt.Errorf("unable to find conversion from %s to %s", from, to)
	}
	item, ok := fromCache[to]
	if !ok {
		return 0, fmt.Errorf("unable to find conversion from %s to %s", from, to)
	}

	if time.Now().After(item.expiration) {
		return 0, fmt.Errorf("item in currency cache expired")
	}

	return item.rate, nil
}

func generateCurrencyConversions(converter *CurrencyConverter, baseCurrency string, currencies []string) (CurrencyConversion, error) {
	conversions := make(CurrencyConversion)

	for _, currency := range currencies {
		conversion, err := converter.ConversionRate(baseCurrency, currency)
		if err != nil {
			return conversions, err
		}

		conversions[currency] = conversion
	}

	return conversions, nil
}
