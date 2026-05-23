package shared

var (
	ErrNegativeAmount   = &DomainError{Code: "MONEY_NEGATIVE_AMOUNT", Message: "amount must not be negative"}
	ErrInvalidCurrency  = &DomainError{Code: "MONEY_INVALID_CURRENCY", Message: "currency must not be empty"}
	ErrCurrencyMismatch = &DomainError{Code: "MONEY_CURRENCY_MISMATCH", Message: "currency mismatch"}
)

type Money struct {
	amount   int64
	currency string
}

func NewMoney(amount int64, currency string) (Money, error) {
	if amount < 0 {
		return Money{}, ErrNegativeAmount
	}
	if currency == "" {
		return Money{}, ErrInvalidCurrency
	}
	return Money{amount: amount, currency: currency}, nil
}

func (m Money) Amount() int64        { return m.amount }
func (m Money) Currency() string     { return m.currency }
func (m Money) Add(o Money) (Money, error) {
	if m.currency != o.currency {
		return Money{}, ErrCurrencyMismatch
	}
	return Money{amount: m.amount + o.amount, currency: m.currency}, nil
}
func (m Money) Multiply(factor int) Money {
	return Money{amount: m.amount * int64(factor), currency: m.currency}
}
func (m Money) Equals(o Money) bool {
	return m.amount == o.amount && m.currency == o.currency
}
