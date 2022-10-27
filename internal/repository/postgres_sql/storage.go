package postgres_sql

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
	"gitlab.ozon.dev/netrebinr/netrebin-roman/internal/repository"
)

type PostgresStorage struct {
	db *sql.DB
}

func New(db *sql.DB) *PostgresStorage {
	return &PostgresStorage{db}
}

// CreateSpending добавляет новую затрату в хранилище
func (ps *PostgresStorage) CreateSpending(ctx context.Context,
	userID int64, categoryName string, amount decimal.Decimal, date time.Time) error {

	category, inStor, err := ps.GetCategory(ctx, userID, categoryName)
	if err != nil {
		return err
	}

	var categoryID int64

	tx, err := ps.db.Begin()
	if err != nil {
		return err
	}

	if inStor {
		categoryID = category.ID
	} else {
		const query = `
			INSERT INTO categories(
				user_id,
				name,
				created_at,
				updated_at
			) VALUES (
				$1, $2, now(), now()
			) RETURNING id;
		`
		row := tx.QueryRowContext(ctx, query,
			userID,
			categoryName,
		)
		err := row.Scan(&categoryID)
		if err != nil {
			if tx.Rollback() != nil {
				return fmt.Errorf("%w, tx.Rollback() failed", err)
			}
			return err
		}
	}

	var unlimit bool
	const query2 = `
		SELECT amount
		FROM limits
		WHERE user_id = $1
	`
	var limit decimal.Decimal
	row := tx.QueryRowContext(ctx, query2, userID)
	err = row.Scan(&limit)
	if err != nil {
		if err == sql.ErrNoRows {
			unlimit = true
		} else {
			if tx.Rollback() != nil {
				return fmt.Errorf("%w, tx.Rollback() failed", err)
			}
			return err
		}
	}

	if !unlimit {
		now := time.Now()
		currentYear, currentMonth, _ := now.Date()
		currentLocation := now.Location()
		firstOfMonth := time.Date(currentYear, currentMonth, 1, 0, 0, 0, 0, currentLocation)

		const query = `
			SELECT SUM(sp.amount)
			FROM spendings sp
			WHERE sp.user_id = $1 AND
				(sp.date BETWEEN $2 AND $3);
		`
		row := tx.QueryRowContext(ctx, query,
			userID,
			firstOfMonth,
			now,
		)
		var summ decimal.Decimal
		err = row.Scan(&summ)
		if err != nil {
			if err != sql.ErrNoRows {
				if tx.Rollback() != nil {
					return fmt.Errorf("%w, tx.Rollback() failed", err)
				}
				return err
			}
		}
		if limit.LessThan(summ.Add(amount)) {
			if tx.Rollback() != nil {
				return errors.Wrap(repository.ErrLimitExceeded, "tx.Rollback() failed")
			}
			return repository.ErrLimitExceeded
		}
	}

	const query = `
		INSERT INTO spendings(
			user_id,
			category_id,
			amount,
			date,
			created_at,
			updated_at
		) VALUES (
			$1, $2, $3, $4, now(), now()
		);
	`
	_, err = tx.ExecContext(ctx, query,
		userID,
		categoryID,
		amount,
		date,
	)
	if err != nil {
		if tx.Rollback() != nil {
			return fmt.Errorf("%w, tx.Rollback() failed", err)
		}
		return err
	}

	return tx.Commit()
}

func (ps *PostgresStorage) GetCategory(ctx context.Context, userID int64, name string) (*repository.Category, bool, error) {
	const query = `
		SELECT id FROM categories
		WHERE user_id = $1 AND name = $2;
	`
	var cat repository.Category
	row := ps.db.QueryRowContext(ctx, query,
		userID,
		name,
	)
	err := row.Scan(&cat.ID)
	if err != nil {
		if err == sql.ErrNoRows {
			err = nil
		}
		return nil, false, err
	}

	return &cat, true, nil
}

func (ps *PostgresStorage) CreateCategory(ctx context.Context, userID int64, name string) error {
	_, inStor, err := ps.GetCategory(ctx, userID, name)
	if err != nil {
		return err
	}
	if inStor {
		return repository.ErrCategoryExists
	}

	const query = `
		INSERT INTO categories(
			user_id,
			name,
			created_at,
			updated_at
		) VALUES (
			$1, $2, now(), now()
		);
	`
	_, err = ps.db.ExecContext(ctx, query,
		userID,
		name,
	)
	return err
}

// GetAllCategories возвращает из хранилища все категории
func (ps *PostgresStorage) GetAllCategories(ctx context.Context, userID int64) ([]*repository.Category, error) {
	const query = `
		SELECT id, user_id, name, created_at, updated_at
		FROM categories
		WHERE user_id = $1
		ORDER BY name;
	`
	rows, err := ps.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var categories []*repository.Category
	for rows.Next() {
		var cat repository.Category
		err := rows.Scan(
			&cat.ID,
			&cat.UserID,
			&cat.Name,
			&cat.CreatedAt,
			&cat.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		categories = append(categories, &cat)
	}

	return categories, nil
}

// ReportPeriod возвращает отчет за период по каждой категории
func (ps *PostgresStorage) ReportPeriod(ctx context.Context,
	userID int64, dateFirst time.Time, dateLast time.Time) (
	[]*repository.ReportByCategory, error) {

	const query = `
		SELECT SUM(sp.amount), cat.name
		FROM spendings sp, categories cat
		WHERE sp.category_id = cat.id AND
		 	  sp.user_id = $1 AND
		 	  (sp.date BETWEEN $2 AND $3)
		GROUP BY cat.name
		HAVING SUM(sp.amount) > 0
		ORDER BY cat.name;
	`
	rows, err := ps.db.QueryContext(ctx, query,
		userID,
		dateFirst,
		dateLast,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	report := make([]*repository.ReportByCategory, 0)
	for rows.Next() {
		var amount decimal.Decimal
		var name string
		err := rows.Scan(
			&amount,
			&name,
		)
		if err != nil {
			return nil, err
		}
		report = append(report, &repository.ReportByCategory{
			CategoryName: name,
			Sum:          amount,
		})
	}

	return report, nil
}

// GetActiveCurrency возвращает используемую юзером валюту
func (ps *PostgresStorage) GetActiveCurrency(ctx context.Context,
	userID int64) (string, error) {

	tx, err := ps.db.Begin()
	if err != nil {
		return "", err
	}

	const query = `
		SELECT char_code
		FROM currencies
		WHERE user_id = $1;
	`
	var currCharCode string
	row := tx.QueryRowContext(ctx, query, userID)
	err = row.Scan(&currCharCode)
	if err == nil {
		return currCharCode, tx.Commit()
	}
	if err == sql.ErrNoRows {
		const query = `
			INSERT INTO currencies(
				user_id,
				char_code,
				created_at,
				updated_at
			) VALUES (
				$1, 'RUB', now(), now());
		`
		_, err = tx.ExecContext(ctx, query,
			userID,
		)
		if err == nil {
			return "RUB", tx.Commit()
		}
	}
	if tx.Rollback() != nil {
		return "", fmt.Errorf("%w, tx.Rollback() failed", err)
	}
	return "", err
}

// SetActiveCurrency устанавливает используемую юзером валюту
func (ps *PostgresStorage) SetActiveCurrency(ctx context.Context, userID int64, currCharCode string) error {
	const query = `
		INSERT INTO currencies(
			user_id,
			char_code,
			created_at,
			updated_at
		) VALUES (
			$1, $2, now(), now()
		) ON CONFLICT (user_id) DO UPDATE
			SET char_code = $2,
				updated_at = now();
	`
	_, err := ps.db.ExecContext(ctx, query,
		userID,
		currCharCode,
	)
	return err
}

// GetLimit возвращает лимит трат в месяц
func (ps *PostgresStorage) GetLimit(ctx context.Context, userID int64) (decimal.Decimal, error) {
	const query = `
		SELECT amount
		FROM limits
		WHERE user_id = $1
	`
	var limit decimal.Decimal
	row := ps.db.QueryRowContext(ctx, query, userID)
	err := row.Scan(&limit)
	if err != nil {
		if err == sql.ErrNoRows {
			return limit, repository.ErrLimitNotSet
		} else {
			return limit, err
		}
	}

	return limit, nil
}

// SetLimit устанавливает лимит трат в месяц
func (ps *PostgresStorage) SetLimit(ctx context.Context, userID int64, amount decimal.Decimal) error {
	const query = `
	INSERT INTO limits(
		user_id,
		amount,
		created_at,
		updated_at
	) VALUES (
		$1, $2, now(), now()
	) ON CONFLICT (user_id) DO UPDATE
		SET amount = $2,
			updated_at = now();
	`
	_, err := ps.db.ExecContext(ctx, query,
		userID,
		amount,
	)
	return err
}

// DropLimit устанавливает неограниченный лимит трат в месяц
func (ps *PostgresStorage) DropLimit(ctx context.Context, userID int64) error {
	const query = `
		DELETE FROM limits
		WHERE user_id = $1;
	`
	_, err := ps.db.ExecContext(ctx, query, userID)
	return err
}
