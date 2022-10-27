# Задание на индексы

## Заполнение тестовыми данными

Пытался сделать через процедуры postgres - времени не хватило разобраться. Сделал через скрипт.

Настройка через константы, запуск:

`go run cmd/test_data/main.go`

## Чекаем и добавляем индексы

### Без индекса: Выборка за неделю для моего id

```
postgres=# set max_parallel_workers_per_gather = 0;
postgres=# EXPLAIN (analyze, timing off)
SELECT sp.amount, cat.name
FROM spendings sp,
    categories cat
WHERE sp.category_id = cat.id
    AND sp.user_id = 893762098
    AND (
        sp.date BETWEEN DATE('2022-10-12') AND now()
    );
                                                     QUERY PLAN
---------------------------------------------------------------------------------------------------------------------
 Nested Loop  (cost=0.28..998.28 rows=6 width=14) (actual rows=6 loops=1)
   ->  Seq Scan on spendings sp  (cost=0.00..952.50 rows=6 width=12) (actual rows=6 loops=1)
         Filter: ((date >= '2022-10-12'::date) AND (user_id = 893762098) AND (date <= now()))
         Rows Removed by Filter: 31419
   ->  Index Scan using categories_pkey on categories cat  (cost=0.28..7.63 rows=1 width=18) (actual rows=1 loops=6)
         Index Cond: (id = sp.category_id)
 Planning Time: 0.156 ms
 Execution Time: 2.018 ms
(8 rows)
```

Пробую ускорить фильтр

### С индексом на user_id: Выборка за неделю для моего id

`CREATE INDEX ON spendings(user_id);`

```
postgres=# EXPLAIN (analyze, timing off)
SELECT sp.amount, cat.name
FROM spendings sp,
    categories cat
WHERE sp.category_id = cat.id
    AND sp.user_id = 893762098
    AND (
        sp.date BETWEEN DATE('2022-10-12') AND now()
    );
                                                      QUERY PLAN
----------------------------------------------------------------------------------------------------------------------
 Nested Loop  (cost=6.89..391.39 rows=6 width=14) (actual rows=6 loops=1)
   ->  Bitmap Heap Scan on spendings sp  (cost=6.61..345.62 rows=6 width=12) (actual rows=6 loops=1)
         Recheck Cond: (user_id = 893762098)
         Filter: ((date >= '2022-10-12'::date) AND (date <= now()))
         Rows Removed by Filter: 306
         Heap Blocks: exact=200
         ->  Bitmap Index Scan on spendings_user_id_idx  (cost=0.00..6.61 rows=310 width=0) (actual rows=312 loops=1)
               Index Cond: (user_id = 893762098)
   ->  Index Scan using categories_pkey on categories cat  (cost=0.28..7.63 rows=1 width=18) (actual rows=1 loops=6)
         Index Cond: (id = sp.category_id)
 Planning Time: 0.345 ms
 Execution Time: 0.279 ms
(12 rows)
```
`drop index spendings_user_id_idx;`

### С индексом на user_id и date: Выборка за неделю для моего id

`CREATE INDEX ON spendings(user_id,date);`

```
CREATE INDEX
postgres=# EXPLAIN (analyze, timing off)
SELECT sp.amount, cat.name
FROM spendings sp,
    categories cat
WHERE sp.category_id = cat.id
    AND sp.user_id = 893762098
    AND (
        sp.date BETWEEN DATE('2022-10-12') AND now()
    );
                                                          QUERY PLAN
-------------------------------------------------------------------------------------------------------------------------------
 Nested Loop  (cost=0.57..69.37 rows=6 width=14) (actual rows=6 loops=1)
   ->  Index Scan using spendings_user_id_date_idx on spendings sp  (cost=0.29..23.59 rows=6 width=12) (actual rows=6 loops=1)
         Index Cond: ((user_id = 893762098) AND (date >= '2022-10-12'::date) AND (date <= now()))
   ->  Index Scan using categories_pkey on categories cat  (cost=0.28..7.63 rows=1 width=18) (actual rows=1 loops=6)
         Index Cond: (id = sp.category_id)
 Planning Time: 0.366 ms
 Execution Time: 0.072 ms
(7 rows)
```

Ускорился в 28 раз, брать другой тип индекса не стал, т.к индекс надо строить по двум столбцам, тут b-tree скорее всего без вариантов.
