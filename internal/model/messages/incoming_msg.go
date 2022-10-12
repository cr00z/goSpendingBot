package messages

import (
	"fmt"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
	"gitlab.ozon.dev/netrebinr/netrebin-roman/internal/currency"
	"gitlab.ozon.dev/netrebinr/netrebin-roman/internal/repository"
)

type MessageSender interface {
	SendMessage(text string, userID int64) error
}

type Model struct {
	tgClient   MessageSender
	store      repository.Storager
	currencies currency.CurrencyStorager
}

func New(tgClient MessageSender, store repository.Storager, currencies currency.CurrencyStorager) *Model {
	return &Model{
		tgClient:   tgClient,
		store:      store,
		currencies: currencies,
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
		commandCurrencySet + " <CUR> - set active currency"
)

func (s *Model) IncomingMessage(msg Message) error {
	switch {
	case msg.Text == commandStart:
		return s.tgClient.SendMessage(messageHello+"\n\n"+messageHelp, msg.UserID)

	case strings.HasPrefix(msg.Text, commandCreateSpending):
		return s.handleCommandCreateSpending(msg)

	case strings.HasPrefix(msg.Text, commandCreateCategory):
		return s.handleCommandCreateCategory(msg)

	case strings.HasPrefix(msg.Text, commandGetAllCategories):
		return s.handleCommandGetAllCategories(msg)

	case msg.Text == commandReportWeekly:
		dateLast := time.Now()
		dateFirst := dateLast.AddDate(0, 0, -7)
		return s.handleCommandReport(msg, dateFirst, dateLast)

	case msg.Text == commandReportMonthly:
		dateLast := time.Now()
		dateFirst := dateLast.AddDate(0, -1, 0)
		return s.handleCommandReport(msg, dateFirst, dateLast)

	case msg.Text == commandReportAnnual:
		dateLast := time.Now()
		dateFirst := dateLast.AddDate(-1, 0, 0)
		return s.handleCommandReport(msg, dateFirst, dateLast)

	case msg.Text == commandCurrencyAll:
		return s.handleCommandCurrencyAll(msg)

	case msg.Text == commandCurrencyActive:
		return s.handleCommandCurrencyActive(msg)

	case strings.HasPrefix(msg.Text, commandCurrencySet):
		return s.handleCommandCurrencySet(msg)

	default:
		return s.tgClient.SendMessage("Я не знаю эту команду", msg.UserID)
	}
}

// Обработчик команды создания траты
func (s *Model) handleCommandCreateSpending(msg Message) error {
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
		return err
	}
	lastIndex--
	categoryName = strings.Join(elements[1:lastIndex+1], " ")
	categoryName = strings.TrimSpace(categoryName)
	if categoryName == "" {
		return repository.ErrCategoryIsEmpty
	}

	curr := s.store.GetActiveCurrency(msg.UserID)
	value, err := s.currencies.GetCurrencyValue(curr)
	if err != nil {
		return err
	}
	amount = amount.Mul(value)

	err = s.store.CreateSpending(msg.UserID, categoryName, amount, date)
	if err != nil {
		return err
	}
	return s.tgClient.SendMessage("Exspense added", msg.UserID)
}

// Обработчик команды создания категории
func (s *Model) handleCommandCreateCategory(msg Message) error {
	var answer string

	elements := strings.Split(msg.Text, " ")
	category := strings.Join(elements[1:], " ")
	category = strings.TrimSpace(category)

	if category == "" {
		answer = "Category name must not be empty"
	} else {
		err := s.store.CreateCategory(msg.UserID, category)
		if err != nil {
			if errors.Is(err, repository.ErrCategoryExists) {
				answer = "Category '" + category + "' already exists"
			} else {
				return err
			}
		} else {
			answer = "Category '" + category + "' added"
		}
	}
	return s.tgClient.SendMessage(answer, msg.UserID)
}

// Обработчик команды списка всех категорий
func (s *Model) handleCommandGetAllCategories(msg Message) error {
	repCats, err := s.store.GetAllCategories(msg.UserID)
	if err != nil {
		return err
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

	return s.tgClient.SendMessage(header+catList, msg.UserID)
}

// Обработчик команд для формирования отчетов
func (s *Model) handleCommandReport(msg Message, dateFirst time.Time, dateLast time.Time) error {
	repCats, err := s.store.ReportPeriod(msg.UserID, dateFirst, dateLast)
	if err != nil {
		return err
	}

	curr := s.store.GetActiveCurrency(msg.UserID)
	value, err := s.currencies.GetCurrencyValue(curr)
	if err != nil {
		return err
	}

	header := "*Report:*"
	catList := " empty"
	if len(repCats) > 0 {
		categories := make([]string, 0, len(repCats))
		for _, cat := range repCats {
			sum := cat.Sum.Div(value)
			categories = append(categories, fmt.Sprintf("%s: %s %s", cat.CategoryName, sum.StringFixed(2), curr))
		}
		catList = "\n" + strings.Join(categories, "\n")
	}

	return s.tgClient.SendMessage(header+catList, msg.UserID)
}

// Обработчик команды списка всех валют
func (s *Model) handleCommandCurrencyAll(msg Message) error {
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

	return s.tgClient.SendMessage(header+currList, msg.UserID)
}

// Обработчик команды запроса активной валюты
func (s *Model) handleCommandCurrencyActive(msg Message) error {
	curr := s.store.GetActiveCurrency(msg.UserID)
	value, err := s.currencies.GetCurrencyValue(curr)
	if err != nil {
		return err
	}

	header := "*Active Currency:*"
	body := "\n" + curr + " " + value.String()
	currName, inMap := currency.CharCodeToName[curr]
	if inMap {
		body += " _" + currName + "_"
	}

	return s.tgClient.SendMessage(header+body, msg.UserID)
}

// Обработчик команды установки активной валюты
func (s *Model) handleCommandCurrencySet(msg Message) error {
	elements := strings.Split(msg.Text, " ")

	if len(elements) == 1 {
		return s.tgClient.SendMessage("Active currency not set", msg.UserID)
	}

	currCharCode := strings.ToUpper(elements[1])
	if _, err := s.currencies.GetCurrencyValue(currCharCode); err != nil {
		return s.tgClient.SendMessage("Unknown currency", msg.UserID)
	}

	err := s.store.SetActiveCurrency(msg.UserID, currCharCode)
	if err != nil {
		return nil
	}

	return s.handleCommandCurrencyActive(msg)
}
