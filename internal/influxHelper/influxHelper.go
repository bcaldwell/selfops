package influxHelper

import (
	"fmt"
	"strings"

	"github.com/bcaldwell/selfops/internal/config"
	influxdb "github.com/influxdata/influxdb/client/v2"
)

func CreateInfluxClient(secrets config.Secrets) (influxdb.Client, error) {
	return influxdb.NewHTTPClient(influxdb.HTTPConfig{
		Addr:     secrets.InfluxEndpoint,
		Username: secrets.InfluxUser,
		Password: secrets.InfluxPassword,
	})
}

func DropTable(influxClient influxdb.Client, name string) error {
	name = strings.Split(name, " ")[0]

	dropCommand := fmt.Sprintf("DROP DATABASE %s", name)

	q := influxdb.NewQuery(dropCommand, "", "")
	if response, err := influxClient.Query(q); err == nil && response.Error() != nil {
		return err
	}
	return nil
}

func CreateTable(influxClient influxdb.Client, name string) error {
	name = strings.Split(name, " ")[0]

	dropCommand := fmt.Sprintf("CREATE DATABASE %s", name)

	q := influxdb.NewQuery(dropCommand, "", "")
	if response, err := influxClient.Query(q); err == nil && response.Error() != nil {
		return err
	}
	return nil
}
