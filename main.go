package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	airtableImporter "github.com/bcaldwell/selfops/internal/airtableimporter"
	"github.com/bcaldwell/selfops/internal/config"
	"github.com/bcaldwell/selfops/internal/csvimporter"
	"github.com/bcaldwell/selfops/internal/ynabimporter"
	"github.com/robfig/cron"
)

const (
	ConfigEnvName = "IMPORTERS_CONFIG"
)

type Runner interface {
	Run() error
	Close() error
}

var runner Runner

func main() {
	var frequency string
	singleRun := flag.Bool("single-run", false, "run importer once (disable cron)")
	once := flag.Bool("once", false, "run importer once (disable cron)")
	configFile := flag.String("config", "./config.yml", "configuration file")
	secretsFile := flag.String("secrets", "./secrets.ejson", "secrets ejson file")
	help := flag.Bool("help", false, "show command help")

	flag.Parse()

	if *help {
		fmt.Println("ynab influx importer")
		fmt.Println("selfops [options] task")
		flag.PrintDefaults()
		return
	}

	err := config.ReadConfig(ConfigEnvName, *configFile, *secretsFile)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	argsWithoutFlags := flag.Args()

	if len(argsWithoutFlags) < 1 {
		fmt.Println("No task passed in")
		return
	}

	switch argsWithoutFlags[0] {
	case "ynab":
		runner, err = ynabimporter.NewImportYNABRunner()
		if err != nil {
			fmt.Printf("Failed to create ynab importer: %s\n", err)
			return
		}
		frequency = config.CurrentYnabConfig().UpdateFrequency
	case "airtable":
		runner = airtableImporter.ImportAirtableRunner{}
		frequency = config.CurrentAirtableConfig().UpdateFrequency
	case "csv":
		if len(argsWithoutFlags) < 2 {
			fmt.Println("csv file name not passed in")
			return
		}

		runner, err = csvimporter.NewImportCSVRunner(argsWithoutFlags[1])
		if err != nil {
			fmt.Printf("Failed to create csv importer: %s\n", err)
			return
		}

		*singleRun = true
	default:
		fmt.Printf("Task %s not found\n", argsWithoutFlags[0])
		return
	}

	defer runner.Close()

	run()
	if *singleRun || *once {
		return
	}

	if frequency == "" {
		frequency = "@every 1h"
	}

	c := cron.New()
	err = c.AddFunc(frequency, run)
	if err != nil {
		fmt.Println("Failed to create con job")
		return
	}

	c.Start()

	select {}

}

func run() {
	fmt.Println(time.Now().Format(time.RFC850))
	err := runner.Run()
	if err != nil {
		fmt.Println(err)
	}
}
