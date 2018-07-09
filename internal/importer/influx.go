package importer

import (
	"fmt"
	"strings"

	influx "github.com/influxdata/influxdb/client/v2"
)

func createInfluxClient(secrets Secrets) (influx.Client, error) {
	return influx.NewHTTPClient(influx.HTTPConfig{
		Addr:     secrets.InfluxEndpoint,
		Username: secrets.InfluxUser,
		Password: secrets.InfluxPassword,
	})
}

func dropTable(influxClient influx.Client, name string) error {
	name = strings.Split(name, " ")[0]

	dropCommand := fmt.Sprintf("DROP DATABASE %s", name)

	q := influx.NewQuery(dropCommand, "", "")
	if response, err := influxClient.Query(q); err == nil && response.Error() != nil {
		return err
	}
	return nil
}

func createTable(influxClient influx.Client, name string) error {
	name = strings.Split(name, " ")[0]

	dropCommand := fmt.Sprintf("CREATE DATABASE %s", name)

	q := influx.NewQuery(dropCommand, "", "")
	if response, err := influxClient.Query(q); err == nil && response.Error() != nil {
		return err
	}
	return nil
}
