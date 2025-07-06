package ynabimporter

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/bcaldwell/selfops/pkg/config"
	"github.com/bcaldwell/selfops/pkg/postgresutils"
	"github.com/uptrace/bun"
	"k8s.io/klog"
)

type SQLBudget struct {
	bun.BaseModel `bun:"table:budgets"`
	ID            int64  `bun:",pk,autoincrement"`
	Key           string `bun:",pk,unique"`
	Category      string
	CategoryGroup string
	Month         time.Time
	Name          string
	Currency      string
	Budgeted      float64
	Activity      float64
	ActivityUSD   float64
	ActivityCAD   float64
	Balance       float64
	BalanceUSD    float64
	BalanceCAD    float64
	Amount        float64
	USD           float64
	CAD           float64
	Fields        map[string]interface{} `bun:"type:jsonb"`
}

func (importer *ImportYNABRunner) migrateBudgets() error {
	tableName := config.CurrentYnabConfig().SQL.BudgetsTable
	_, err := importer.db.NewCreateTable().Model((*SQLBudget)(nil)).ModelTableExpr(tableName).IfNotExists().Exec(context.Background())
	return err
}

func (importer *ImportYNABRunner) importBudgets(budget config.Budget, currencies []string) error {
	// todo make this come from config
	model := (*SQLBudget)(nil)
	tableName := config.CurrentYnabConfig().SQL.BudgetsTable

	sqlRecords := make([]SQLBudget, 0)

	// importer.budgets[budget.ID].Months[0].Categories[0].
	months := importer.budgets[budget.ID].Months
	// categories := importer.budgets[budget.ID].Categories

	for monthIndex := range months {
		for categoryIndex := range months[monthIndex].Categories {
			// for categoryIndex := range categories {
			category := months[monthIndex].Categories[categoryIndex]
			categoryGroup := importer.categories[budget.Name][category.Id].Group

			if category.Hidden {
				continue
			}

			budgeted := float64(category.Budgeted) / 1000.0
			activity := float64(category.Activity) / 1000.0
			balance := float64(category.Balance) / 1000.0

			month, err := time.Parse("2006-01-02", months[monthIndex].Month)
			if err != nil {
				return err
			}

			row := SQLBudget{
				Key:           months[monthIndex].Month + "-" + category.Id,
				Category:      category.Name,
				CategoryGroup: categoryGroup,
				Budgeted:      budgeted,
				Amount:        budgeted,
				USD:           Round(budgeted*budget.Conversions["USD"], 0.01),
				CAD:           Round(budgeted*budget.Conversions["CAD"], 0.01),
				Activity:      activity,
				ActivityUSD:   Round(activity*budget.Conversions["USD"], 0.01),
				ActivityCAD:   Round(activity*budget.Conversions["CAD"], 0.01),
				Balance:       balance,
				BalanceUSD:    Round(balance*budget.Conversions["USD"], 0.01),
				BalanceCAD:    Round(balance*budget.Conversions["CAD"], 0.01),
				Name:          budget.Name,
				Currency:      budget.Currency,
				Month:         month,
				Fields:        make(map[string]interface{}),
			}

			for _, field := range budget.CalculatedFields {
				calculateField := stringInSlice(category.Name, field.Category) || stringInSlice(categoryGroup, field.CategoryGroup)
				if field.Inverted {
					calculateField = !calculateField
				}

				row.Fields[field.Name] = strconv.FormatBool(calculateField)
			}

			sqlRecords = append(sqlRecords, row)
		}
	}

	batchSize := config.CurrentYnabConfig().SQL.BatchSize
	if batchSize == 0 {
		batchSize = 1000
	}

	for i := 0; i < len(sqlRecords); i += batchSize {
		endIndex := min(len(sqlRecords), i+batchSize)

		records := sqlRecords[i:endIndex]
		_, err := importer.db.NewInsert().
			Model(&records).
			ModelTableExpr(tableName).
			On("CONFLICT (key) DO UPDATE").
			Set(postgresutils.TableSetString(importer.db, model, "id", "key")).
			Exec(context.Background())

		if err != nil {
			return fmt.Errorf("error writing budgets: %s", err.Error())
		}
	}

	klog.Infof("Wrote %v budgets for %s to sql\n", len(sqlRecords), budget.Name)

	return nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func stringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}

	return false
}
