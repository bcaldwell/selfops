package financialimporter

type Transaction interface {
	Date() string
	Payee() string
	Category() string
	Memo() string

	Currency() string
	Amount() float64

	Tags() []string
	SubTransactions() []Transaction

	Account() string

	IndexKey() string
}

type CurrencyConversion map[string]float64
