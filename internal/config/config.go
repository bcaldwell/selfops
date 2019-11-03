package config

import (
	"encoding/json"
	"io/ioutil"
	"os"

	"github.com/Shopify/ejson"
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
	return &config.Ynab
}

func CurrentYnabSecrets() *YnabSecrets {
	return &secrets.Ynab
}

func CurrentAirtableConfig() *AirtableConfig {
	return &config.Airtable
}

func CurrentAirtableSecrets() *AirtableSecrets {
	return &secrets.Airtable
}

func CurrentInfluxSecrets() *InfluxSecrets {
	return &secrets.Influx
}

func CurrentSqlSecrets() *SqlSecrets {
	return &secrets.Sql
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
	ejsonKeyFile := os.Getenv("YNAB_IMPORTER_EJSON_SECRET_KEY")
	ejsonKey := []byte{}
	var err error

	if ejsonKeyFile != "" {
		ejsonKey, err = ioutil.ReadFile(ejsonKeyFile)
		if err != nil {
			return nil, err
		}
	}
	raw, err := ejson.DecryptFile(filename, "/opt/ejson/keys", string(ejsonKey))
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(raw, &secrets)

	return &secrets, err
}
