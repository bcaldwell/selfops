package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	airtableImporter "github.com/bcaldwell/selfops/internal/airtableimporter"
	"github.com/bcaldwell/selfops/internal/config"
	"github.com/bcaldwell/selfops/internal/ynabimporter"
	"github.com/robfig/cron"
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

	err := config.ReadConfig(*configFile, *secretsFile)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	if len(os.Args) == 1 {
		fmt.Println("No task passed in")
		return
	}
	taskIndex := len(os.Args) - 1

	switch os.Args[taskIndex] {
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
	default:
		fmt.Println("No task passed in")
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
	c.AddFunc(frequency, run)

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
