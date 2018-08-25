package influxHelper

import (
	"fmt"
	"strings"

	"github.com/bcaldwell/selfops/internal/config"
	influxdb "github.com/influxdata/influxdb/client/v2"
)

func CreateInfluxClient() (influxdb.Client, error) {
	return influxdb.NewHTTPClient(influxdb.HTTPConfig{
		Addr:     config.CurrentInfluxSecrets().InfluxEndpoint,
		Username: config.CurrentInfluxSecrets().InfluxUser,
		Password: config.CurrentInfluxSecrets().InfluxPassword,
	})
}

func DropDatabase(influxClient influxdb.Client, name string) error {
	name = strings.Split(name, " ")[0]

	dropCommand := fmt.Sprintf("DROP DATABASE %s", name)

	q := influxdb.NewQuery(dropCommand, "", "")
	if response, err := influxClient.Query(q); err == nil && response.Error() != nil {
		return err
	}
	return nil
}

func DropMeasurement(influxClient influxdb.Client, dbName string, name string) error {
	name = strings.Split(name, " ")[0]

	dropCommand := fmt.Sprintf("DROP MEASUREMENT \"%s\"", name)

	q := influxdb.NewQuery(dropCommand, dbName, "")
	if response, err := influxClient.Query(q); err == nil && response.Error() != nil {
		return err
	}
	return nil
}

func CreateDatabase(influxClient influxdb.Client, name string) error {
	name = strings.Split(name, " ")[0]

	dropCommand := fmt.Sprintf("CREATE DATABASE %s", name)

	q := influxdb.NewQuery(dropCommand, "", "")
	if response, err := influxClient.Query(q); err == nil && response.Error() != nil {
		return err
	}
	return nil
}
