package messages

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/opentracing/opentracing-go"
	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
	"gitlab.ozon.dev/netrebinr/netrebin-roman/internal/cache"
	"gitlab.ozon.dev/netrebinr/netrebin-roman/internal/currency"
	producer "gitlab.ozon.dev/netrebinr/netrebin-roman/internal/kafka/producers"
	"gitlab.ozon.dev/netrebinr/netrebin-roman/internal/observability"
	"gitlab.ozon.dev/netrebinr/netrebin-roman/internal/repository"
)

var (
	serviceErrorStr  = "Service error, try again later"
	currencyErrorStr = "Currency service error, try again later"
)

type MessageSender interface {
	SendMessage(ctx context.Context, text string, userID int64) error
}

type Model struct {
	tgClient      MessageSender
	store         repository.Storager
	currCache     cache.Storager
	reportCache   cache.Storager
	currencies    currency.CurrencyStorager
	reportService producer.ReportProducer
}

func New(tgClient MessageSender, store repository.Storager, currCache cache.Storager,
	reportCache cache.Storager, currencies currency.CurrencyStorager, reportService producer.ReportProducer) *Model {
	return &Model{
		tgClient:      tgClient,
		store:         store,
		currCache:     currCache,
		reportCache:   reportCache,
		currencies:    currencies,
		reportService: reportService,
	}
}

type Message struct {
	Text   string
	UserID int64
}

const (
	commandStart            = "/start"
	commandCreateSpending   = "/addexp"
	commandCreateCategory   = "/newcat"
	commandGetAllCategories = "/listcat"
	commandReportWeekly     = "/repw"
	commandReportMonthly    = "/repm"
	commandReportAnnual     = "/repa"
	commandCurrencyAll      = "/curall"
	commandCurrencyActive   = "/curget"
	commandCurrencySet      = "/curset"
	commandLimitGet         = "/limitget"
	commandLimitSet         = "/limitset"

	messageHello = "Hello! I can help you manage your spendings."
	messageHelp  = "You can control me by sending these commands:\n\n" +
		"*Expenses*\n" +
		commandCreateSpending + ` <category name> <amount> \[dd/mm/yy]  - add new expense` + "\n\n" +
		"*Edit Categories*\n" +
		commandCreateCategory + " <category name> - create a new expense category\n" +
		commandGetAllCategories + " - get a list of your expense categories\n\n" +
		"*Reports*\n" +
		commandReportWeekly + " - get a weekly report by category\n" +
		commandReportMonthly + " - get a monthly report by category\n" +
		commandReportAnnual + " - get the annual report by category\n\n" +
		"*Currencies*\n" +
		commandCurrencyAll + " - get currency list\n" +
		commandCurrencyActive + " - get active currency\n" +
		commandCurrencySet + " <CUR> - set active currency\n\n" +
		"*Limits*\n" +
		commandLimitGet + " - get month expense limit\n" +
		commandLimitSet + ` \[amount] - set month expense limit. If the value is` +
		` not set, then there will be no limit.`
)

func (s *Model) proceedCommand(ctx context.Context,
	command string, msg Message) (string, error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, "proceed message")
	span.SetTag("command", command[1:])
	defer span.Finish()

	var message string
	var err error

	switch command {
	case commandStart:
		message = messageHello + "\n\n" + messageHelp

	case commandCreateSpending:
		message, err = s.handleCommandCreateSpending(ctx, msg)

	case commandCreateCategory:
		message, err = s.handleCommandCreateCategory(ctx, msg)

	case commandGetAllCategories:
		message, err = s.handleCommandGetAllCategories(ctx, msg)

	case commandReportWeekly:
		dateLast := time.Now()
		dateFirst := dateLast.AddDate(0, 0, -7)
		message, err = s.handleCommandReport(ctx, msg, "W", dateFirst, dateLast)

	case commandReportMonthly:
		dateLast := time.Now()
		dateFirst := dateLast.AddDate(0, -1, 0)
		message, err = s.handleCommandReport(ctx, msg, "M", dateFirst, dateLast)

	case commandReportAnnual:
		dateLast := time.Now()
		dateFirst := dateLast.AddDate(-1, 0, 0)
		message, err = s.handleCommandReport(ctx, msg, "Y", dateFirst, dateLast)

	case commandCurrencyAll:
		message, err = s.handleCommandCurrencyAll(ctx, msg)

	case commandCurrencyActive:
		message, err = s.handleCommandCurrencyActive(ctx, msg)

	case commandCurrencySet:
		message, err = s.handleCommandCurrencySet(ctx, msg)

	case commandLimitGet:
		message, err = s.handleCommandLimitGet(ctx, msg)

	case commandLimitSet:
		message, err = s.handleCommandLimitSet(ctx, msg)

	default:
		message = "Я не знаю эту команду"
	}

	return message, err
}

func (s *Model) IncomingMessage(ctx context.Context, msg Message) error {
	startTime := time.Now()

	var command string
	if msg.Text != "" {
		command = strings.Split(msg.Text, " ")[0]
	}
	if command != commandStart &&
		command != commandCreateSpending &&
		command != commandCreateCategory &&
		command != commandGetAllCategories &&
		command != commandReportWeekly &&
		command != commandReportMonthly &&
		command != commandReportAnnual &&
		command != commandCurrencyAll &&
		command != commandCurrencyActive &&
		command != commandCurrencySet &&
		command != commandLimitGet &&
		command != commandLimitSet {
		command = "/unknown"
	}

	// Метрика на количество запросов, можно заменить на histogram_count
	observability.RequestsCount.WithLabelValues(command[1:]).Inc()

	// Первый спан
	span, ctx := opentracing.StartSpanFromContext(ctx, "incoming message")
	span.SetTag("command", command[1:])
	defer span.Finish()

	message, err := s.proceedCommand(ctx, command, msg)

	// Метрика на длительность обработки запроса внутри сервиса
	duration := time.Since(startTime)
	observability.HistogramCommandTimeVec.WithLabelValues(command[1:]).Observe(duration.Seconds())

	if message != "" {
		startTime = time.Now()

		err = s.tgClient.SendMessage(ctx, message, msg.UserID)

		// Метрика на длительность обработки запроса в телеграмм
		duration = time.Since(startTime)
		result := "success"
		if err != nil {
			result = "error"
		}
		observability.HistogramTgapiTimeVec.WithLabelValues(result).Observe(duration.Seconds())
	}

	return err
}

// Обработчик команды создания траты
func (s *Model) handleCommandCreateSpending(ctx context.Context, msg Message) (string, error) {
	var categoryName string
	elements := strings.Split(msg.Text, " ")
	lastIndex := len(elements) - 1

	date, err := time.Parse("02/01/06", elements[lastIndex])

	if err != nil {
		date = time.Now()
	} else {
		lastIndex--
	}
	amount, err := decimal.NewFromString(elements[lastIndex])
	if err != nil {
		return "Unknown amount", err
	}
	lastIndex--
	categoryName = strings.Join(elements[1:lastIndex+1], " ")
	categoryName = strings.TrimSpace(categoryName)
	if categoryName == "" {
		return "Unknown category", repository.ErrCategoryIsEmpty
	}

	// Конвертация в валюту

	curr, err := s.getActiveCurrencyFromCacheAndDB(ctx, msg.UserID)
	if err != nil {
		return serviceErrorStr, err
	}
	value, err := s.currencies.GetCurrencyValue(curr)
	if err != nil {
		return currencyErrorStr, err
	}
	amount = amount.Mul(value)

	// Проверка лимита

	err = s.store.CreateSpending(ctx, msg.UserID, categoryName, amount, date)
	if err != nil {
		if err == repository.ErrLimitExceeded {
			return "Limit exceeded", nil
		}
		return serviceErrorStr, err
	}

	s.invalidateReportPeriodInCache(msg.UserID, date)

	return "Exspense added", nil
}

// Обработчик команды создания категории
func (s *Model) handleCommandCreateCategory(ctx context.Context, msg Message) (string, error) {
	var answer string

	elements := strings.Split(msg.Text, " ")
	category := strings.Join(elements[1:], " ")
	category = strings.TrimSpace(category)

	if category == "" {
		answer = "Category name must not be empty"
	} else {
		err := s.store.CreateCategory(ctx, msg.UserID, category)
		if err != nil {
			if errors.Is(err, repository.ErrCategoryExists) {
				answer = "Category '" + category + "' already exists"
			} else {
				return serviceErrorStr, err
			}
		} else {
			answer = "Category '" + category + "' added"
		}
	}
	return answer, nil
}

// Обработчик команды списка всех категорий
func (s *Model) handleCommandGetAllCategories(ctx context.Context, msg Message) (string, error) {
	repCats, err := s.store.GetAllCategories(ctx, msg.UserID)
	if err != nil {
		return serviceErrorStr, err
	}

	header := "*Categories:*"
	catList := " empty"

	if len(repCats) > 0 {
		categories := make([]string, 0, len(repCats))
		for id := range repCats {
			categories = append(categories, repCats[id].Name)
		}
		catList = "\n" + strings.Join(categories, "\n")
	}

	return header + catList, nil
}

func (s *Model) ProceedCommandReport(ctx context.Context,
	userID int64, report *repository.Report) (string, error) {

	curr, err := s.getActiveCurrencyFromCacheAndDB(ctx, userID)
	if err != nil {
		return serviceErrorStr, err
	}
	value, err := s.currencies.GetCurrencyValue(curr)
	if err != nil {
		return currencyErrorStr, err
	}

	header := "*Report:*"
	catList := " empty"
	if len(report.ReportByCategory) > 0 {
		categories := make([]string, 0, len(report.ReportByCategory))
		for _, cat := range report.ReportByCategory {
			sum := cat.Sum.Div(value)
			categories = append(categories,
				fmt.Sprintf("%s: %s %s", cat.CategoryName, sum.StringFixed(2), curr))
		}
		catList = "\n" + strings.Join(categories, "\n")
	}

	return header + catList, nil
}

// Обработчик команд для формирования отчетов
func (s *Model) handleCommandReport(ctx context.Context,
	msg Message, period string, dateFirst time.Time, dateLast time.Time) (string, error) {

	report, err := s.getReportPeriodFromCacheAndDB(ctx, msg, period, dateFirst, dateLast)
	if err != nil {
		return serviceErrorStr, err
	}
	if report == nil {
		return "Report proceeed", nil
	}

	return s.ProceedCommandReport(ctx, msg.UserID, report)
}

// Обработчик команды списка всех валют
func (s *Model) handleCommandCurrencyAll(ctx context.Context, msg Message) (string, error) {
	currs := s.currencies.GetAllCurrencies()

	var currAllStrs []string
	for _, curr := range currs {
		currStr := curr.CharCode + " " + curr.Value.String()
		if currName, inMap := currency.CharCodeToName[curr.CharCode]; inMap {
			currStr += " _" + currName + "_"
		}
		currAllStrs = append(currAllStrs, currStr)
	}

	header := "*Currency List:*"
	currList := " empty"
	if len(currAllStrs) > 0 {
		currList = "\n" + strings.Join(currAllStrs, "\n")
	}

	return header + currList, nil
}

// Обработчик команды запроса активной валюты
func (s *Model) handleCommandCurrencyActive(ctx context.Context, msg Message) (string, error) {
	curr, err := s.getActiveCurrencyFromCacheAndDB(ctx, msg.UserID)
	if err != nil {
		return serviceErrorStr, err
	}
	value, err := s.currencies.GetCurrencyValue(curr)
	if err != nil {
		return currencyErrorStr, err
	}

	header := "*Active Currency:*"
	body := "\n" + curr + " " + value.String()
	currName, inMap := currency.CharCodeToName[curr]
	if inMap {
		body += " _" + currName + "_"
	}

	return header + body, nil
}

// Обработчик команды установки активной валюты
func (s *Model) handleCommandCurrencySet(ctx context.Context, msg Message) (string, error) {
	elements := strings.Split(msg.Text, " ")

	if len(elements) == 1 {
		return "Active currency not set", nil
	}

	currCharCode := strings.ToUpper(elements[1])
	if _, err := s.currencies.GetCurrencyValue(currCharCode); err != nil {
		return "Unknown currency", err
	}

	err := s.store.SetActiveCurrency(ctx, msg.UserID, currCharCode)
	if err != nil {
		return serviceErrorStr, err
	}

	s.currCache.Add(strconv.FormatInt(msg.UserID, 10), currCharCode)

	return s.handleCommandCurrencyActive(ctx, msg)
}

// Обработчик команды запроса лимита
func (s *Model) handleCommandLimitGet(ctx context.Context, msg Message) (string, error) {
	header := "*Month limit:* "
	var body string
	limit, err := s.store.GetLimit(ctx, msg.UserID)
	if err != nil {
		if err == repository.ErrLimitNotSet {
			body = "not set"
		} else {
			return serviceErrorStr, err
		}
	} else {
		currCharCode, err := s.getActiveCurrencyFromCacheAndDB(ctx, msg.UserID)
		if err != nil {
			return serviceErrorStr, err
		}
		value, err := s.currencies.GetCurrencyValue(currCharCode)
		if err != nil {
			return currencyErrorStr, err
		}
		body = limit.Div(value).String() + " " + currCharCode
	}

	return header + body, nil
}

// Обработчик команды установки лимита
func (s *Model) handleCommandLimitSet(ctx context.Context, msg Message) (string, error) {
	elements := strings.Split(msg.Text, " ")

	if len(elements) == 1 {
		err := s.store.DropLimit(ctx, msg.UserID)
		if err != nil {
			return serviceErrorStr, err
		}
	} else {
		amount, err := decimal.NewFromString(elements[1])
		if err != nil {
			return "Unknown amount", err
		}

		currCharCode, err := s.getActiveCurrencyFromCacheAndDB(ctx, msg.UserID)
		if err != nil {
			return serviceErrorStr, err
		}
		value, err := s.currencies.GetCurrencyValue(currCharCode)
		if err != nil {
			return currencyErrorStr, err
		}

		err = s.store.SetLimit(ctx, msg.UserID, amount.Mul(value))
		if err != nil {
			return serviceErrorStr, err
		}
	}

	return s.handleCommandLimitGet(ctx, msg)
}
