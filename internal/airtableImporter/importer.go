package airtableImporter

import (
	"fmt"

	"github.com/bcaldwell/selfops/internal/config"
	"github.com/bcaldwell/selfops/internal/influxHelper"
	"github.com/fabioberger/airtable-go"

	influx "github.com/influxdata/influxdb/client/v2"
)

type ImportAirtableRunner struct{}

func (ImportAirtableRunner) Run() error {
	return ImportAirtable()
}

func ImportAirtable() error {
	client, err := airtable.New(config.CurrentAirtableSecrets().AirtableAPIKey, config.CurrentAirtableConfig().AirtableBaseID)
	if err != nil {
		panic(err)
	}

	influxDB, err := influxHelper.CreateInfluxClient(*config.CurrentSecrets())
	if err != nil {
		return fmt.Errorf("Error creating InfluxDB Client: %s", err.Error())
	}

	err = influxHelper.DropTable(influxDB, config.CurrentAirtableConfig().AirtableDatabase)
	if err != nil {
		return fmt.Errorf("Error dropping DB: %s", err.Error())
	}
	err = influxHelper.CreateTable(influxDB, config.CurrentAirtableConfig().AirtableDatabase)
	if err != nil {
		return fmt.Errorf("Error creating DB: %s", err.Error())
	}

	type AirtableRecords struct {
		AirtableID string
		Fields     map[string]interface{}
	}

	bp, err := influx.NewBatchPoints(influx.BatchPointsConfig{
		Database:  config.CurrentConfig().AirtableDatabase,
		Precision: "h",
	})

	airtableRecords := []AirtableRecords{}
	if err := client.ListRecords(config.CurrentAirtableConfig().AirtableTableName, &airtableRecords); err != nil {
		return fmt.Errorf("Error getting airtable records: %s", err.Error())
	}

	for _, record := range airtableRecords {
		tags := map[string]string{}
		// for name, field := range record.Fields {
		// 	tags[name] = fmt.Sprintf("%s", field)
		// }
		pt, err := influx.NewPoint(config.CurrentConfig().AirtableDatabase, tags, record.Fields)
		if err != nil {
			return fmt.Errorf("Error adding new point: %s", err.Error())
		}
		bp.AddPoint(pt)

	}

	err = influxDB.Write(bp)
	if err != nil {
		return fmt.Errorf("Error writing to influx: %s", err.Error())
	}

	fmt.Printf("Wrote %d rows to influx from airtable\n", len(airtableRecords))

	return nil
}
