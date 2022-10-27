package memory

import (
	"sort"
	"sync"
	"time"

	"github.com/shopspring/decimal"
	"gitlab.ozon.dev/netrebinr/netrebin-roman/internal/repository"
)

type MemoryStorage struct {
	sync.Mutex
	categories     map[int64]*repository.Category
	spendings      map[int64]*repository.Spending
	currency       map[int64]string
	nextCategoryID int64
	nextSpendingID int64
}

func NewMemoryStorage() *MemoryStorage {
	ms := &MemoryStorage{}
	ms.categories = make(map[int64]*repository.Category)
	ms.spendings = make(map[int64]*repository.Spending)
	ms.currency = make(map[int64]string)
	ms.nextCategoryID = 0
	ms.nextSpendingID = 0
	return ms
}

// CreateSpending добавляет новую затрату в хранилище
func (ms *MemoryStorage) CreateSpending(userID int64, categoryName string, amount decimal.Decimal, date time.Time) error {
	category, inStor := ms.GetCategory(userID, categoryName)
	categoryID := category.ID

	ms.Lock()
	defer ms.Unlock()

	if !inStor {
		ms.categories[ms.nextCategoryID] = &repository.Category{
			ID:     ms.nextCategoryID,
			UserID: userID,
			Name:   categoryName,
		}
		categoryID = ms.nextCategoryID
		ms.nextCategoryID++
	}

	ms.spendings[ms.nextSpendingID] = &repository.Spending{
		ID:         ms.nextSpendingID,
		UserID:     userID,
		CategoryId: int(categoryID),
		Amount:     amount,
		Date:       date,
	}
	ms.nextSpendingID++

	return nil
}

func (ms *MemoryStorage) GetCategory(userID int64, name string) (*repository.Category, bool) {
	ms.Lock()
	defer ms.Unlock()

	for _, cat := range ms.categories {
		if cat.UserID == userID && cat.Name == name {
			return cat, true
		}
	}
	return &repository.Category{}, false
}

// CreateCategory создает новую категорию в хранилище
func (ms *MemoryStorage) CreateCategory(userID int64, name string) error {
	if _, inStor := ms.GetCategory(userID, name); inStor {
		return repository.ErrCategoryExists
	}

	ms.Lock()
	defer ms.Unlock()

	ms.categories[ms.nextCategoryID] = &repository.Category{
		ID:     ms.nextCategoryID,
		UserID: userID,
		Name:   name,
	}
	ms.nextCategoryID++

	return nil
}

// GetAllCategories возвращает из хранилища все категории
func (ms *MemoryStorage) GetAllCategories(userID int64) ([]*repository.Category, error) {
	ms.Lock()
	defer ms.Unlock()

	ids := make([]int, 0)
	for id := range ms.categories {
		if ms.categories[id].UserID == userID {
			ids = append(ids, int(id))
		}
	}
	sort.Ints(ids)

	result := make([]*repository.Category, 0, len(ids))
	for id := range ids {
		result = append(result, ms.categories[int64(id)])
	}
	return result, nil
}

// ReportPeriod возвращает отчет за период по каждой категории
func (ms *MemoryStorage) ReportPeriod(userID int64, dateFirst time.Time, dateLast time.Time) ([]*repository.ReportByCategory, error) {
	reportMap := make(map[int64]decimal.Decimal)

	ms.Lock()
	for _, sp := range ms.spendings {
		if sp.UserID == userID && sp.Date.After(dateFirst) && sp.Date.Before(dateLast) {
			reportMap[int64(sp.CategoryId)] = reportMap[int64(sp.CategoryId)].Add(sp.Amount)
		}
	}
	ms.Unlock()

	categories, err := ms.GetAllCategories(userID)
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
func (ms *MemoryStorage) GetActiveCurrency(userID int64) (curr string) {
	ms.Lock()
	defer ms.Unlock()
	curr, inMap := ms.currency[userID]
	if !inMap {
		curr = "RUB"
		ms.currency[userID] = "RUB"
	}
	return curr
}

// SetActiveCurrency устанавливает используемую юзером валюту
func (ms *MemoryStorage) SetActiveCurrency(userID int64, curr string) error {
	ms.Lock()
	ms.currency[userID] = curr
	ms.Unlock()
	return nil
}
