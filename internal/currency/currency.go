package currency

import (
	"errors"

	"github.com/shopspring/decimal"
)

var CharCodeToName = map[string]string{
	"AUD": "Австралийский доллар",
	"AZN": "Азербайджанский манат",
	"GBP": "Фунт стерлингов Соединенного королевства",
	"AMD": "Армянская драма",
	"BYN": "Белорусский рубль",
	"BGN": "Болгарский лев",
	"BRL": "Бразильский реал",
	"HUF": "Венгерский форинт",
	"HKD": "Гонконгский доллар",
	"DKK": "Датская крона",
	"USD": "Доллар США",
	"EUR": "Евро",
	"INR": "Индийская рупия",
	"KZT": "Казахстанский тенге",
	"CAD": "Канадский доллар",
	"KGS": "Киргизский сом",
	"CNY": "Китайский юань",
	"MDL": "Молдавский лей",
	"NOK": "Норвежская крона",
	"PLN": "Польский злотый",
	"RON": "Румынский лей",
	"RUB": "Российский рубль",
	"XDR": "СДР (специальные права заимствования)",
	"SGD": "Сингапурский доллар",
	"TJS": "Таджикский сомони",
	"TRY": "Турецкая лира",
	"TMT": "Новый туркменский манат",
	"UZS": "Узбекский сум",
	"UAH": "Украинская гривна",
	"CZK": "Чешская крона",
	"SEK": "Шведская крона",
	"CHF": "Швейцарский франк",
	"ZAR": "Южноафриканский рэнд",
	"KRW": "Вона Республики Корея",
	"JPY": "Японская иена",
}

var ErrCurrencyNotSupported = errors.New("currency not supported")

type Currency struct {
	CharCode string
	Value    decimal.Decimal
}

type CurrencyStorager interface {
	GetAllCurrencies() []Currency
	GetCurrencyValue(curr string) (decimal.Decimal, error)
}
