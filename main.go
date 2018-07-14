package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/bcaldwell/ynab-influx-importer/internal/importer"
	"github.com/robfig/cron"
)

func main() {
	singleRun := flag.Bool("single-run", false, "run importer once (disable cron)")
	configFile := flag.String("config", "./config.yml", "configuration file")
	secretsFile := flag.String("secrets", "./secrets.yml", "secrets file")
	help := flag.Bool("help", false, "show command help")

	flag.Parse()

	if *help {
		fmt.Println("ynab influx importer")
		flag.PrintDefaults()
		return
	}

	err := importer.ReadConfig(*configFile, *secretsFile)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	run()

	if *singleRun {
		return
	}

	c := cron.New()
	c.AddFunc(importer.CurrentConfig().UpdateFrequency, run)

	c.Start()

	select {}

}

func run() {
	fmt.Println(time.Now().Format(time.RFC850))
	err := importer.ImportYNAB()
	if err != nil {
		fmt.Println(err)
	}
}
