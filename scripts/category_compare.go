package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/bcaldwell/selfops/internal/config"
	"github.com/davidsteinsland/ynab-go/ynab"
)

func main() {
	configFile := flag.String("config", "./config.yml", "configuration file")
	secretsFile := flag.String("secrets", "./secrets.json", "secrets file")

	err := config.ReadConfig(*configFile, *secretsFile)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	ynabClient := ynab.NewDefaultClient(config.CurrentYnabSecrets().YnabAccessToken)

	budgetSummaries, _ := ynabClient.BudgetService.List()

	budgets := make(map[string]string)
	for _, budget := range budgetSummaries {
		if strings.Contains(budget.Name, "(Archived on") {
			continue
		}
		budgets[budget.Name] = budget.Id
	}

	categories := make(map[string][]string)

	for name, id := range budgets {
		cad, _ := ynabClient.CategoriesService.List(id)

		budgetCategories := []string{}

		for _, catGroup := range cad {
			if catGroup.Name == "Credit Card Payments" {
				continue
			}
			for _, category := range catGroup.Categories {
				if !category.Hidden {
					budgetCategories = append(budgetCategories, category.Name)
				}
			}
		}
		categories[name] = budgetCategories
	}

	for name, budgetCategories := range categories {
		PrettyPrint(name, budgetCategories)
	}

	for name, budgetCategories := range categories {
		for name2, budgetCategories2 := range categories {
			if name == name2 {
				continue
			}
			PrettyPrint(name+"-"+name2, difference(budgetCategories, budgetCategories2))
		}

	}

}

func difference(slice1 []string, slice2 []string) []string {
	var diff []string

	// Loop two times, first to find slice1 strings not in slice2,
	// second loop to find slice2 strings not in slice1
	for _, s1 := range slice1 {
		found := false
		for _, s2 := range slice2 {
			if s1 == s2 {
				found = true
				break
			}
		}
		// String not found. We add it to return slice
		if !found {
			diff = append(diff, s1)
		}
	}

	return diff
}

func PrettyPrint(prefix string, v interface{}) (err error) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err == nil {
		fmt.Println(prefix + ": " + string(b))
	}
	return
}
