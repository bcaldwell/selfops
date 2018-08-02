package ynabImporter

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
)

const CurrencyConversionEndpoint = "https://free.currencyconverterapi.com/api/v5/convert"

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

func conversionRate(from, to string) (float64, error) {
	conversionString := fmt.Sprintf("%s_%s", strings.ToUpper(from), strings.ToUpper(to))

	req, err := http.NewRequest("GET", CurrencyConversionEndpoint, nil)
	if err != nil {
		return 0, err
	}

	q := req.URL.Query()
	q.Add("q", conversionString)
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

	if conversionResponse, ok := currencyConversionResponse.Results[conversionString]; ok {
		return conversionResponse.Val, nil
	}

	return 0, fmt.Errorf("Invalid currency")
}
