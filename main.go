package main

import (
	"fmt"
	"os"
	"time"

	"github.com/bcaldwell/ynab-influx-importer/internal/importer"
	"github.com/robfig/cron"
)

func main() {
	err := importer.ReadConfig()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	run()

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
