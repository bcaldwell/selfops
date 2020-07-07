package ynabimporter

import (
	"fmt"
	"strconv"

	"github.com/bcaldwell/selfops/internal/config"
	"github.com/bcaldwell/selfops/internal/postgresHelper"
)

var baseBudgetSQLSchema = map[string]string{
	"ynabID":        "varchar",
	"category":      "varchar",
	"categoryGroup": "varchar",
	"month":         "timestamp",
	"name":          "varchar",
	"currency":      "varchar",
	"budgeted":      "float8",
	"activity":      "float8",
	"amount":        "float8",
}

func (importer *ImportYNABRunner) importBudgets(budget config.Budget, currencies []string) error {
	sqlRecords := make([]map[string]string, 0)

	// importer.budgets[budget.ID].Months[0].Categories[0].
	months := importer.budgets[budget.ID].Months
	// categories := importer.budgets[budget.ID].Categories

	for monthIndex := range months {
		for categoryIndex := range months[monthIndex].Categories {
			// for categoryIndex := range categories {
			category := months[monthIndex].Categories[categoryIndex]

			if category.Hidden {
				continue
			}

			budgeted := float64(category.Budgeted) / 1000.0
			activity := float64(category.Activity) / 1000.0

			row := map[string]string{
				"ynabID":        category.Id,
				"category":      category.Name,
				"categoryGroup": importer.categories[budget.Name][category.Id].Group,
				"budgeted":      strconv.FormatFloat(budgeted, 'f', 2, 64),
				"amount":        strconv.FormatFloat(budgeted, 'f', 2, 64),
				"activity":      strconv.FormatFloat(activity, 'f', 2, 64),
				"name":          budget.Name,
				"currency":      budget.Currency,
				// "month":      importer.budgets[budget.ID].,
				"month": months[monthIndex].Month,
			}

			for _, currency := range currencies {
				// convert budgeted
				convertedBudgeted := Round(budgeted*budget.Conversions[currency], 0.01)
				row[currency] = strconv.FormatFloat(convertedBudgeted, 'f', 2, 64)
				
				// convert activity
				convertedActivity := Round(activity*budget.Conversions[currency], 0.01)
				row["activity_" + currency] = strconv.FormatFloat(convertedActivity, 'f', 2, 64)
			}

			for _, field := range budget.CalculatedFields {
				calculateField := stringInSlice(category.Name, field.Category) || stringInSlice(row["categoryGroup"], field.CategoryGroup)
				if field.Inverted {
					calculateField = !calculateField
				}

				row[field.Name] = strconv.FormatBool(calculateField)
			}

			sqlRecords = append(sqlRecords, row)
		}
	}

	err := postgresHelper.InsertRecords(importer.db, config.CurrentYnabConfig().SQL.BudgetsTable, sqlRecords)
	if err != nil {
		return fmt.Errorf("Failed to write budgets to db: %v", err)
	}

	fmt.Printf("Wrote %v budgets for %s to sql\n", len(sqlRecords), budget.Name)

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

func (importer *ImportYNABRunner) recreateBudgetTable(calculatedFields []config.CalculatedField) error {
	err := postgresHelper.DropTable(importer.db, config.CurrentYnabConfig().SQL.BudgetsTable)
	if err != nil {
		return fmt.Errorf("Error dropping table: %s", err)
	}

	err = postgresHelper.CreateTable(importer.db, config.CurrentYnabConfig().SQL.BudgetsTable, importer.createBudgetSQLSchema(calculatedFields))
	if err != nil {
		return fmt.Errorf("Error creating table: %s", err)
	}

	return nil
}

func (importer *ImportYNABRunner) createBudgetSQLSchema(calculatedFields []config.CalculatedField) map[string]string {
	schema := baseBudgetSQLSchema

	for _, field := range calculatedFields {
		if _, ok := schema[field.Name]; !ok {
			schema[field.Name] = "boolean"
		}
	}

	for _, currency := range config.CurrentYnabConfig().Currencies {
		schema[currency] = "float8"
		schema["activity_" + currency] = "float8"
	}

	return schema
}
