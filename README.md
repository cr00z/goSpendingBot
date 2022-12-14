# goSpendingBot

> В процессе разработки

Телеграмм-бот для контроля расходов

Сквозной проект на курсе "Продвинутая разработка микросервисов на Go" от Ozon (Route256).

## О проекте

- бот с использованием принципов SOLID (слабая связность, интерфейсы) и telegram-bot-api
- парсинг валют с cbr.ru, обработка xml
- memory, orm (gorm) и postgres native хранилища для данных
- миграции (goose)
- своя реализация LRU cache
- тесты (gomock, sqlmock)
- observability: логи graylog + zap, метрики prometheus/grafana + promauto/promhttp, трейсы jaeger + opentracing
- очереди на kafka (sarama)
- сервис отчетов через grpc (google.golang.org/grpc)

<tr>
    <td> <img src="https://raw.githubusercontent.com/cr00z/goSpendingBot/main/images/screenshot1.jpeg" alt="Demo" style="width: 250px;"/> </td>
    <td> <img src="https://raw.githubusercontent.com/cr00z/goSpendingBot/main/images/screenshot2.jpeg" alt="Demo" style="width: 250px;"/> </td>
</tr>

## Основные команды

**Expenses**

- /addexp <category name> <amount> [dd/mm/yy]  - add new expense

**Edit Categories**

- /newcat <category name> - create a new expense category
- /listcat - get a list of your expense categories

**Reports**

- /repw - get a weekly report by category
- /repm - get a monthly report by category
- /repa - get the annual report by category

**Currencies**

- /curall - get currency list
- /curget - get active currency
- /curset <CUR> - set active currency

**Limits**
- /limitget - get month expense limit
- /limitset [amount] - set month expense limit. If the value is not set, then there will be no limit

## Домашки

* [Пояснение к третьему заданию](homeworks/README3.md)
* [Пояснение к четвертому заданию](homeworks/README4.md)
* [Пояснения к пятому заданию](homeworks/README5.md)
* [Пояснения к шестому заданию](homeworks/README6.md)
* [Пояснения к седьмому заданию](homeworks/README7.md)

## Заметки

```
docker run --name=gospend-db -e POSTGRES_PASSWORD='qwerty' -p 5432:5432 -d --rm postgres
goose -dir migrations postgres "host=localhost port=5432 user=postgres password=qwerty" create init_db sql
goose -dir migrations postgres "host=localhost port=5432 user=postgres password=qwerty sslmode=disable" up
```
