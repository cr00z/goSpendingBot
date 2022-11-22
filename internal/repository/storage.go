package repository

import (
	"context"
	"errors"
	"time"

	"github.com/shopspring/decimal"
)

var (
	ErrCategoryExists  = errors.New("category exists")
	ErrCategoryIsEmpty = errors.New("category is empty")
	ErrLimitNotSet     = errors.New("month limit not set")
	ErrLimitExceeded   = errors.New("month limit exceeded")
)

type Storager interface {
	CreateSpending(ctx context.Context, userID int64, categoryName string, amount decimal.Decimal, date time.Time) error
	CreateCategory(ctx context.Context, userID int64, name string) error
	GetAllCategories(ctx context.Context, userID int64) ([]*Category, error)
	ReportPeriod(ctx context.Context, userID int64, dateFirst time.Time, dateLast time.Time) (*Report, error)
	GetActiveCurrency(ctx context.Context, userID int64) (string, error)
	SetActiveCurrency(ctx context.Context, userID int64, curr string) error
	GetLimit(ctx context.Context, userID int64) (decimal.Decimal, error)
	SetLimit(ctx context.Context, userID int64, amount decimal.Decimal) error
	DropLimit(ctx context.Context, userID int64) error
}

type Category struct {
	ID        int64
	UserID    int64
	Name      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type Spending struct {
	ID         int64
	UserID     int64
	CategoryId int
	Amount     decimal.Decimal
	Date       time.Time
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type ReportByCategory struct {
	CategoryName string
	Sum          decimal.Decimal
}

type Report struct {
	ReportByCategory []*ReportByCategory
	MinDate          time.Time
}

type Currency struct {
	ID        int64
	UserID    int64
	CharCode  string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type Limit struct {
	ID        int64
	UserID    int64
	Amount    decimal.Decimal
	CreatedAt time.Time
	UpdatedAt time.Time
}
