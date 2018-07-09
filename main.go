package main

import (
	"fmt"
	"os"

	"github.com/bcaldwell/ynab-influx-importer/internal/importer"
)

func main() {
	err := importer.ImportYNAB()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
