package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/bcaldwell/selfops/internal/airtableImporter"
	"github.com/bcaldwell/selfops/internal/config"
	"github.com/bcaldwell/selfops/internal/ynabImporter"
	"github.com/robfig/cron"
)

type Runner interface {
	Run() error
}

var runner Runner

func main() {
	var frequency string
	singleRun := flag.Bool("single-run", false, "run importer once (disable cron)")
	configFile := flag.String("config", "./config.yml", "configuration file")
	secretsFile := flag.String("secrets", "./secrets.json", "secrets file")
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

	switch os.Args[1] {
	case "ynab":
		runner = ynabImporter.ImportYNABRunner{}
		frequency = config.CurrentYnabConfig().UpdateFrequency
	case "airtable":
		runner = airtableImporter.ImportAirtableRunner{}
		frequency = config.CurrentAirtableConfig().UpdateFrequency
	default:
		fmt.Println("No task passed in")
		return
	}

	run()

	if *singleRun {
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
