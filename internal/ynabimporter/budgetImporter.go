package ynabimporter

import (
	"context"
	"strconv"
	"time"

	"github.com/bcaldwell/selfops/internal/config"
	"github.com/uptrace/bun"
	"k8s.io/klog"
)

// var baseBudgetSQLSchema = map[string]string{
// 	"ynabID":        "varchar",
// 	"category":      "varchar",
// 	"categoryGroup": "varchar",
// 	"month":         "timestamp",
// 	"name":          "varchar",
// 	"currency":      "varchar",
// 	"budgeted":      "float8",
// 	"activity":      "float8",
// 	"balance":       "float8",
// 	"amount":        "float8",
// }

type SQLBudget struct {
	bun.BaseModel
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

func (importer *ImportYNABRunner) importBudgets(budget config.Budget, currencies []string) error {
	// todo make this come from config
	_, err := importer.db.NewCreateTable().Model((*SQLBudget)(nil)).ModelTableExpr("budgets").IfNotExists().Exec(context.Background())
	if err != nil {
		return err
	}

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

			// for _, currency := range currencies {
			// 	// convert budgeted
			// 	convertedBudgeted := Round(budgeted*budget.Conversions[currency], 0.01)
			// 	row[currency] = strconv.FormatFloat(convertedBudgeted, 'f', 2, 64)

			// 	// convert activity
			// 	convertedActivity := Round(activity*budget.Conversions[currency], 0.01)
			// 	row["activity_"+currency] = strconv.FormatFloat(convertedActivity, 'f', 2, 64)

			// 	// convert balance
			// 	convertedBalance := Round(balance*budget.Conversions[currency], 0.01)
			// 	row["balance_"+currency] = strconv.FormatFloat(convertedBalance, 'f', 2, 64)
			// }

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

	_, err = importer.db.NewInsert().
		Model(&sqlRecords).
		ModelTableExpr("budgets").
		On("CONFLICT (key) DO UPDATE").
		Set("category = EXCLUDED.category").
		Exec(context.Background())

	klog.Infof("Wrote %v budgets for %s to sql\n", len(sqlRecords), budget.Name)

	return err
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
	// err := postgresHelper.DropTable(importer.db.DB, config.CurrentYnabConfig().SQL.BudgetsTable)
	// if err != nil {
	// 	return fmt.Errorf("Error dropping table: %s", err)
	// }

	// err = postgresHelper.CreateTable(importer.db.DB, config.CurrentYnabConfig().SQL.BudgetsTable, importer.createBudgetSQLSchema(calculatedFields))
	// if err != nil {
	// 	return fmt.Errorf("Error creating table: %s", err)
	// }

	return nil
}

// func (importer *ImportYNABRunner) createBudgetSQLSchema(calculatedFields []config.CalculatedField) map[string]string {
// 	schema := baseBudgetSQLSchema

// 	for _, field := range calculatedFields {
// 		if _, ok := schema[field.Name]; !ok {
// 			schema[field.Name] = "boolean"
// 		}
// 	}

// 	for _, currency := range config.CurrentYnabConfig().Currencies {
// 		schema[currency] = "float8"
// 		schema["activity_"+currency] = "float8"
// 		schema["balance_"+currency] = "float8"
// 	}

// 	return schema
// }
