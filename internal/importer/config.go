package importer

import (
	"encoding/json"
	"fmt"
	"io/ioutil"

	"github.com/davidsteinsland/ynab-go/ynab"
	"github.com/ghodss/yaml"
)

var config Config
var secrets Secrets

func ReadConfig(configFile, secretsFile string) error {
	_, err := readConfig(configFile)
	if err != nil {
		return err
	}

	_, err = readSecrets(secretsFile)
	if err != nil {
		return err
	}
	return nil
}

func CurrentConfig() *Config {
	return &config
}

func CurrentSecrets() *Secrets {
	return &secrets
}

func readConfig(filename string) (*Config, error) {
	raw, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	err = yaml.Unmarshal(raw, &config)

	return &config, err
}

func readSecrets(filename string) (*Secrets, error) {
	raw, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(raw, &secrets)

	return &secrets, err
}

// todo: handle my error
func detectBudgetIDs(ynabClient *ynab.Client, config *Config) error {
	budgets, err := ynabClient.BudgetService.List()
	if err != nil {
		return err
	}
	for i, budgetConfig := range config.Budgets {
		if budgetConfig.ID == "" {
			found := false
			for _, b := range budgets {
				if budgetConfig.Name == b.Name {
					config.Budgets[i].ID = b.Id
					if budgetConfig.Currency == "" {
						config.Budgets[i].Currency = b.CurrencyFormat.IsoCode
					}
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("Unable to find ID for budget: %s", budgetConfig.Name)
			}
		}

		if config.Budgets[i].Conversions == nil {
			config.Budgets[i].Conversions = make(CurrencyConversion)
		}
		for _, currency := range config.Currencies {
			if _, ok := config.Budgets[i].Conversions[currency]; ok {
				continue
			}

			config.Budgets[i].Conversions[currency], err = conversionRate(config.Budgets[i].Currency, currency)
			if err != nil {
				return err
			}

		}
	}
	return nil
}
