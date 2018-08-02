package config

import (
	"encoding/json"
	"io/ioutil"

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

func CurrentYnabConfig() *YnabConfig {
	return &config.YnabConfig
}

func CurrentYnabecrets() *YnabSecrets {
	return &secrets.YnabSecrets
}

func CurrentAirtableConfig() *AirtableConfig {
	return &config.AirtableConfig
}

func CurrentAirtableSecrets() *AirtableSecrets {
	return &secrets.AirtableSecrets
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
