package airtableImporter

import (
	"fmt"
	"strings"
	"time"

	"github.com/bcaldwell/selfops/internal/config"
	"github.com/bcaldwell/selfops/internal/influxHelper"
	"github.com/crufter/airtable-go"

	influx "github.com/influxdata/influxdb/client/v2"
)

type ImportAirtableRunner struct{}

func (ImportAirtableRunner) Run() error {
	return ImportAirtable()
}

func (ImportAirtableRunner) Close() error {
	return nil
}

type AirtableRecords struct {
	ID     string
	Fields map[string]interface{}
}

func ImportAirtable() error {
	influxDB, err := influxHelper.CreateInfluxClient()
	if err != nil {
		return fmt.Errorf("Error creating InfluxDB Client: %s", err.Error())
	}

	err = influxHelper.DropDatabase(influxDB, config.CurrentAirtableConfig().AirtableDatabase)
	if err != nil {
		return fmt.Errorf("Error dropping DB: %s", err.Error())
	}
	err = influxHelper.CreateDatabase(influxDB, config.CurrentAirtableConfig().AirtableDatabase)
	if err != nil {
		return fmt.Errorf("Error creating DB: %s", err.Error())
	}

	for _, base := range config.CurrentAirtableConfig().AirtableBases {

		client, err := airtable.New(config.CurrentAirtableSecrets().AirtableAPIKey, base.BaseID)
		if err != nil {
			panic(err)
		}

		bp, err := influx.NewBatchPoints(influx.BatchPointsConfig{
			Database:  config.CurrentAirtableConfig().AirtableDatabase,
			Precision: "h",
		})

		airtableRecords := []AirtableRecords{}
		if err := client.ListRecords(base.AirtableTableName, &airtableRecords); err != nil {
			return fmt.Errorf("Error getting airtable records: %s", err.Error())
		}

		for _, record := range airtableRecords {
			tags := map[string]string{}
			// for name, field := range record.Fields {
			// 	tags["tag"+name] = strings.Replace(fmt.Sprintf("%v", field), "\n", ":", -1)
			// }

			date, err := parseAsDate(record.Fields["Date"])
			if err != nil {
				return fmt.Errorf("Error parsing date: %s", record.Fields["Date"])
			}
			fields := make(map[string]interface{})
			for key, field := range record.Fields {
				if stringInSlice(key, base.Fields.Blacklist) {
					continue
				}
				switch field.(type) {
				case int32, int64, float32, float64:
					fields[key] = field
				case bool:
					if base.Fields.ConvertBoolToInt {
						fields[key] = 0
						if field.(bool) {
							fields[key] = 1
						}
					} else {
						fields[key] = field
					}

				case string:
					if stringInSlice(key, base.Fields.ConvertToTimeFromMidnightList) {
						valueDate, err := parseAsDateTime(field)
						if err != nil {
							fmt.Printf("Error parsing date: %s\n", field)
							continue
						}
						offset := 0.0
						if record.Fields["Timezone Offset"] != nil {
							offset = record.Fields["Timezone Offset"].(float64)
						}
						duration := valueDate.Sub(date)
						fields[key] = duration.Minutes() + (offset * 60)
					} else {
						tags[key] = strings.Replace(fmt.Sprintf("%v", field), "\n", ":", -1)
					}
				default:
					fmt.Printf("Ignoring %v \n", key)
				}
				if value, ok := fields[key]; ok {
					tags["tag"+key] = fmt.Sprintf("%v", value)
				}
			}

			pt, err := influx.NewPoint(base.InfluxMeasurement, tags, fields, date)
			if err != nil {
				return fmt.Errorf("Error adding new point: %s", err.Error())
			}
			bp.AddPoint(pt)

		}

		err = influxDB.Write(bp)
		if err != nil {
			return fmt.Errorf("Error writing to influx: %s", err.Error())
		}

		fmt.Printf("Wrote %d rows to influx from airtable base %s:%s\n", len(airtableRecords), base.BaseID, base.AirtableTableName)
	}

	return nil
}

func stringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

func parseAsDate(a interface{}) (time.Time, error) {
	return time.Parse("2006-01-02", a.(string))
}

func parseAsDateTime(a interface{}) (time.Time, error) {
	return time.Parse(time.RFC3339, a.(string))
}

// func getFieldValueString(record AirtableRecords, field string, t string)
