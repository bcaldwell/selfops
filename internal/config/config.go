package config

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/Shopify/ejson"
	"github.com/caarlos0/env/v6"
	"github.com/ghodss/yaml"
	"github.com/imdario/mergo"
)

var config Config
var secrets Secrets

func ReadConfig(configEnvVar, configFile, secretsFile string) error {
	_, err := readConfig(configEnvVar, configFile)
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
	return &secrets.SQL
}

func readConfig(envName, filename string) (*Config, error) {
	var raw []byte
	var err error

	rawEnv := os.Getenv(envName)
	if rawEnv != "" {
		fmt.Printf("Reading config from environment variable %s\n", envName)
		raw = []byte(rawEnv)
	} else {
		raw, err = ioutil.ReadFile(filename)
		if err != nil {
			return nil, err
		}
	}

	err = yaml.Unmarshal(raw, &config)

	return &config, err
}

func readSecrets(filename string) (*Secrets, error) {
	ejsonSecrets, ejsonErr := readEjsonSecrets(filename)

	envSecrets, envErr := readEnvSecrets()

	if ejsonErr == nil && envErr == nil {
		err := mergo.Merge(envSecrets, *ejsonSecrets)
		secrets = *envSecrets
		if err != nil {
			return nil, fmt.Errorf("Failed to merge secrets: %v", err)
		}
	} else if ejsonErr != nil && envErr == nil {
		secrets = *envSecrets
	} else if ejsonErr == nil && envErr != nil {
		secrets = *ejsonSecrets
	} else {
		return nil, fmt.Errorf("Failed to parse secrets. Ejson error: %v. Env error: %v", ejsonErr, envErr)
	}

	return &secrets, nil
}

func readEjsonSecrets(filename string) (*Secrets, error) {
	ejsonSecrets := Secrets{}
	ejsonKeyFile := os.Getenv("IMPORTERS_EJSON_SECRET_KEY")
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

	err = json.Unmarshal(raw, &ejsonSecrets)
	return &ejsonSecrets, err
}

func readEnvSecrets() (*Secrets, error) {
	envSecrets := Secrets{}
	err := env.Parse(&envSecrets)
	return &envSecrets, err
}
