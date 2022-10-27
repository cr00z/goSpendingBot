package postgres_gorm

import (
	"errors"
	"time"

	"github.com/shopspring/decimal"
	"gitlab.ozon.dev/netrebinr/netrebin-roman/internal/repository"
	"gorm.io/gorm"
)

type PostgresStorage struct {
	db *gorm.DB
}

func New(db *gorm.DB) *PostgresStorage {
	return &PostgresStorage{
		db: db,
	}
}

// CreateSpending добавляет новую затрату в хранилище
func (ps *PostgresStorage) CreateSpending(userID int64, categoryName string, amount decimal.Decimal, date time.Time) error {
	category, inStor := ps.GetCategory(userID, categoryName)
	if !inStor {
		err := ps.CreateCategory(userID, categoryName)
		if err != nil {
			return err
		}
		category, _ = ps.GetCategory(userID, categoryName)
	}
	categoryID := category.ID

	spend := repository.Spending{
		UserID:     userID,
		CategoryId: int(categoryID),
		Amount:     amount,
		Date:       date,
	}
	result := ps.db.Create(&spend)

	return result.Error
}

func (ps *PostgresStorage) GetCategory(userID int64, name string) (*repository.Category, bool) {
	var cat repository.Category
	result := ps.db.Where(&repository.Category{UserID: userID, Name: name}).First(&cat)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, false
	}

	return &cat, true
}

// CreateCategory создает новую категорию в хранилище
func (ps *PostgresStorage) CreateCategory(userID int64, name string) error {
	if _, inStor := ps.GetCategory(userID, name); inStor {
		return repository.ErrCategoryExists
	}
	cat := repository.Category{
		UserID: userID,
		Name:   name,
	}
	result := ps.db.Create(&cat)

	return result.Error
}

// GetAllCategories возвращает из хранилища все категории
func (ps *PostgresStorage) GetAllCategories(userID int64) ([]*repository.Category, error) {
	var cats []repository.Category
	result := ps.db.Where(&repository.Category{UserID: userID}).Find(&cats)
	var retCats []*repository.Category
	for _, cat := range cats {
		retCats = append(retCats, &cat)
	}
	return retCats, result.Error
}

// ReportPeriod возвращает отчет за период по каждой категории
func (ps *PostgresStorage) ReportPeriod(userID int64, dateFirst time.Time, dateLast time.Time) ([]*repository.ReportByCategory, error) {
	var spends []repository.Spending
	result := ps.db.Where(&repository.Spending{UserID: userID}).Where("date BETWEEN ? AND ?", dateFirst, dateLast).Find(&spends)
	if result.Error != nil {
		return nil, result.Error
	}

	reportMap := make(map[int64]decimal.Decimal)
	for _, sp := range spends {
		reportMap[int64(sp.CategoryId)] = reportMap[int64(sp.CategoryId)].Add(sp.Amount)
	}

	categories, err := ps.GetAllCategories(userID)
	if err != nil {
		return nil, err
	}

	report := make([]*repository.ReportByCategory, 0, len(categories))
	for _, cat := range categories {
		if !reportMap[cat.ID].IsZero() {
			report = append(report, &repository.ReportByCategory{
				CategoryName: cat.Name,
				Sum:          reportMap[cat.ID],
			})
		}
	}

	return report, nil
}

// GetActiveCurrency возвращает используемую юзером валюту
func (ps *PostgresStorage) GetActiveCurrency(userID int64) (curr string) {
	var resp repository.Currency
	result := ps.db.Model(repository.Currency{UserID: userID}).First(&resp)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return "RUB"
	}
	return resp.CharCode
}

// SetActiveCurrency устанавливает используемую юзером валюту
func (ps *PostgresStorage) SetActiveCurrency(userID int64, curr string) error {
	var resp repository.Currency
	result := ps.db.Model(repository.Currency{UserID: userID}).First(&resp)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		result = ps.db.Save(&repository.Currency{
			UserID:   userID,
			CharCode: curr,
		})
	} else {
		result = ps.db.Model(&repository.Currency{}).Where("user_id = ?", userID).Update("char_code", curr)
	}

	return result.Error
}
