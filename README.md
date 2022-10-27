# goSpendingBot

Телеграмм-бот для контроля расходов

Сквозной проект на курсе "Продвинутая разработка микросервисов на Go" от Ozon (Route256)

> В процессе разработки

* [Пояснение к третьему заданию](README3.md)
* [Пояснение к четвертому заданию](README4.md)
* [Пояснения к пятому заданию](README5.md)

<tr>
    <td> <img src="images/screenshot1.jpg" alt="Demo" style="width: 350px;"/> </td>
    <td> <img src="images/screenshot2.jpg" alt="Demo" style="width: 350px;"/> </td>
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

## Заметки

```
docker run --name=gospend-db -e POSTGRES_PASSWORD='qwerty' -p 5432:5432 -d --rm postgres
goose -dir migrations postgres "host=localhost port=5432 user=postgres password=qwerty" create init_db sql
goose -dir migrations postgres "host=localhost port=5432 user=postgres password=qwerty sslmode=disable" up
```
