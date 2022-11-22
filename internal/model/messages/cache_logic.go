package messages

import (
	"context"
	"strconv"
	"time"

	"gitlab.ozon.dev/netrebinr/netrebin-roman/internal/cache"
	"gitlab.ozon.dev/netrebinr/netrebin-roman/internal/observability"
	"gitlab.ozon.dev/netrebinr/netrebin-roman/internal/repository"
)

// Запрос из кеша кода активной валюты
func (s *Model) getActiveCurrencyFromCacheAndDB(ctx context.Context,
	userID int64) (curr string, err error) {

	key := strconv.FormatInt(userID, 10)
	result, err := s.currCache.Get(key)
	if err != nil {
		curr, err = s.store.GetActiveCurrency(ctx, userID)
		if err == nil {
			eviction := s.currCache.Add(key, curr)
			if eviction {
				// Метрики: вытеснение из кэша
				observability.CacheEvictionCountVec.WithLabelValues(s.currCache.Name()).Inc()
			} else {
				// Метрики: количество ключей - добавление ключа (без вытеснения)
				observability.CacheKeyCountVec.WithLabelValues(s.currCache.Name()).Inc()
			}
		}
	} else {
		// Метрики: попадание в кэш
		observability.CacheHitCountVec.WithLabelValues(s.currCache.Name()).Inc()
		curr = result.(string)
	}
	return curr, err
}

// Запрос из кеша рапорта за период
func (s *Model) getReportPeriodFromCacheAndDB(ctx context.Context,
	msg Message, period string, dateFirst time.Time, dateLast time.Time) (report *repository.Report, err error) {

	key := strconv.FormatInt(msg.UserID, 10) + "_" + period
	result, err := s.reportCache.Get(key)
	if err == nil {
		report = result.(*repository.Report)

		// Инвалидация кэша при запросе рапорта
		// минимальная дата рапорта из кэша еще влезает в запрошенный период?
		if report.MinDate.Before(dateFirst) {
			_ = s.reportCache.Delete(key)
			err = cache.ErrElementNotInCache

			// Метрики: количество ключей - удаление ключа
			observability.CacheKeyCountVec.WithLabelValues(s.reportCache.Name()).Dec()

		} else {
			// Метрики: попадание в кэш
			observability.CacheHitCountVec.WithLabelValues(s.reportCache.Name()).Inc()
		}
	}
	if err != nil {
		// report, err = s.store.ReportPeriod(ctx, msg.UserID, dateFirst, dateLast)
		// if err == nil {
		// 	eviction := s.reportCache.Add(key, report)
		// 	if eviction {
		// 		// Метрики: вытеснение из кэша
		// 		observability.CacheEvictionCountVec.WithLabelValues(s.reportCache.Name()).Inc()
		// 	} else {
		// 		// Метрики: количество ключей - добавление ключа (без вытеснения)
		// 		observability.CacheKeyCountVec.WithLabelValues(s.reportCache.Name()).Inc()
		// 	}
		// }
		err = s.reportService.SendMessage(msg.UserID, period, dateFirst, dateLast)
	}
	return report, err
}

// Инвалидация кэша при добавлении траты
// Если дата траты > (now()-год) -> протухает годовой
// Если дата траты > (now()-месяц) -> протухает месячный рапорт
// Если дата траты > (now()-неделя) -> протухает недельный рапорт
// Метрики: вытеснение из кэша
func (s *Model) invalidateReportPeriodInCache(userID int64, dateFirst time.Time) {
	key := strconv.FormatInt(userID, 10)
	dateLast := time.Now()

	if dateFirst.After(dateLast.AddDate(-1, 0, 0)) {
		_ = s.reportCache.Delete(key + "_Y")
		observability.CacheEvictionCountVec.WithLabelValues(s.reportCache.Name()).Inc()
	}

	if dateFirst.After(dateLast.AddDate(0, -1, 0)) {
		_ = s.reportCache.Delete(key + "_M")
		observability.CacheEvictionCountVec.WithLabelValues(s.reportCache.Name()).Inc()
	}

	if dateFirst.After(dateLast.AddDate(0, 0, -7)) {
		_ = s.reportCache.Delete(key + "_W")
		observability.CacheEvictionCountVec.WithLabelValues(s.reportCache.Name()).Inc()
	}
}
