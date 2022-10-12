package repository

import (
	"errors"
	"time"

	"github.com/shopspring/decimal"
)

var (
	ErrCategoryExists  = errors.New("category exists")
	ErrCategoryIsEmpty = errors.New("category is empty")
)

type Storager interface {
	CreateSpending(userID int64, categoryName string, amount decimal.Decimal, date time.Time) error
	CreateCategory(userID int64, name string) error
	GetAllCategories(userID int64) ([]*Category, error)
	ReportPeriod(userID int64, dateFirst time.Time, dateLast time.Time) ([]*ReportByCategory, error)
	GetActiveCurrency(userID int64) string
	SetActiveCurrency(userID int64, curr string) error
}

type Category struct {
	Id     int
	UserID int64
	Name   string
}

type Spending struct {
	Id         int
	UserID     int64
	CategoryId int
	Amount     decimal.Decimal
	Date       time.Time
}

type ReportByCategory struct {
	CategoryName string
	Sum          decimal.Decimal
}
