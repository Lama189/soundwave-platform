package money

type Currency string

const (
	USD Currency = "USD"
	UZS Currency = "UZS"
	RUB Currency = "RUB"
)

func (c Currency) IsValid() bool {
	switch c {
	case USD, RUB, UZS:
		return true
	default:
		return false
	}
}

func (c Currency) String() string {
	return string(c)
}
