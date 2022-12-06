package cbrcurrency

import (
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/cr00z/goSpendingBot/internal/currency"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
	"golang.org/x/text/encoding/ianaindex"
)

const (
	updateTimeout = time.Hour
	cbrXmlUrl     = "https://cbr.ru/scripts/XML_daily.asp"
)

type ValCurs struct {
	XMLName xml.Name `xml:"ValCurs"`
	Text    string   `xml:",chardata"`
	Date    string   `xml:"Date,attr"`
	Name    string   `xml:"name,attr"`
	Valute  []struct {
		Text     string `xml:",chardata"`
		ID       string `xml:"ID,attr"`
		NumCode  string `xml:"NumCode"`
		CharCode string `xml:"CharCode"`
		Nominal  string `xml:"Nominal"`
		Name     string `xml:"Name"`
		Value    string `xml:"Value"`
	} `xml:"Valute"`
}

func getValCursFromCBR() (*ValCurs, error) {
	var vc ValCurs

	resp, err := http.Get(cbrXmlUrl)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	xmlCurrenciesData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	decoder := xml.NewDecoder(bytes.NewBuffer(xmlCurrenciesData))
	decoder.CharsetReader = func(charset string, reader io.Reader) (io.Reader, error) {
		enc, err := ianaindex.IANA.Encoding(charset)
		if err != nil {
			return nil, fmt.Errorf("charset %s: %s", charset, err.Error())
		}
		if enc == nil {
			return reader, nil
		}
		return enc.NewDecoder().Reader(reader), nil
	}

	err = decoder.Decode(&vc)
	return &vc, err
}

type CbrCurrencyStorage struct {
	sync.RWMutex
	currencies map[string]decimal.Decimal
}

func NewCbrCurrencyStorage(ctx context.Context,
	wg *sync.WaitGroup, logger *zap.Logger) (*CbrCurrencyStorage, error) {

	wg.Add(1)
	defer wg.Done()

	var cbr CbrCurrencyStorage
	cbr.currencies = make(map[string]decimal.Decimal)
	cbr.currencies["RUB"] = decimal.NewFromInt(1)

	timer := time.NewTicker(updateTimeout)

	vc, err := getValCursFromCBR()
	if err == nil {
		cbr.valCursToCbrCurrencyStorage(vc)
	}

	go func() {
		var exit bool
		for !exit {
			select {
			case <-timer.C:
				vc, err = getValCursFromCBR()
				if err != nil {
					logger.Warn(
						"currency storage temporary failed",
						zap.Error(err),
					)
				} else {
					cbr.valCursToCbrCurrencyStorage(vc)
					logger.Info("currencies updated")
				}
			case <-ctx.Done():
				logger.Info("shutdown currency thread")
				exit = true
			}
		}
	}()

	return &cbr, err
}

func (cbr *CbrCurrencyStorage) valCursToCbrCurrencyStorage(vc *ValCurs) {
	cbr.RWMutex.Lock()
	defer cbr.RWMutex.Unlock()

	for _, v := range vc.Valute {
		valueStr := strings.Replace(v.Value, ",", ".", -1)
		valueDecimal, err := decimal.NewFromString(valueStr)
		if err != nil {
			continue
		}
		if v.Nominal != "" {
			nominal, err := decimal.NewFromString(v.Nominal)
			if err != nil {
				nominal = decimal.NewFromInt(1)
			}
			valueDecimal = valueDecimal.Div(nominal)
		}
		cbr.currencies[v.CharCode] = valueDecimal
	}
}

func (cbr *CbrCurrencyStorage) GetAllCurrencies() []currency.Currency {
	cbr.RWMutex.RLock()
	defer cbr.RWMutex.RUnlock()

	var result []currency.Currency

	keys := make([]string, 0, len(cbr.currencies))
	for k := range cbr.currencies {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		result = append(result, currency.Currency{
			CharCode: k,
			Value:    cbr.currencies[k],
		})
	}

	return result
}

func (cbr *CbrCurrencyStorage) GetCurrencyValue(curr string) (value decimal.Decimal, err error) {
	cbr.RWMutex.RLock()
	value, inMap := cbr.currencies[curr]
	cbr.RWMutex.RUnlock()
	if !inMap {
		err = currency.ErrCurrencyNotSupported
	}
	return value, err
}
