package util

const (
	USD = "USD"
	EUR = "EUR"
	KRW = "KRW"
)

func IsSupportedCurrency(currency string) bool {
	switch currency {
	case USD, EUR, KRW:
		return true
	}
	return false
}
