package financialimporter

type FinancialImporter interface {
	Import() (int, error)
}

type TransactionType int

const (
	Expense TransactionType = iota
	Income
	Transfer
)

func (t TransactionType) String() string {
	if t < Expense || t > Transfer {
		return "Unknown"
	}

	return [...]string{"expense", "income", "transfer"}[t]
}

type Transaction interface {
	Date() string
	Payee() string
	Category() string
	CategoryGroup() string
	Memo() string

	// Currency() string
	Amount() float64
	TransactionType() TransactionType

	Tags() []string
	HasSubTransactions() bool
	SubTransactions() []Transaction

	Account() string

	IndexKey() string
}

type CurrencyConversion map[string]float64

// type CalculatedField struct {
// 	Name          string
// 	Category      []string
// 	CategoryGroup []string
// 	Payee         []string
// 	Inverted      bool
// }
