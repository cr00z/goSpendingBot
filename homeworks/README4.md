# Нарисовать три схемы на (1000, 100 000, 1 000 000 пользователей).

> Нагрузка получилась очень небольшая, поэтому при рисовании схем больше ориентировался на идею "монолит" -> "небольшой сервис на микросервисах" -> "высоконагруженный сервис"

* Draw.io: https://drive.google.com/file/d/19W7A8hPp0Fl2mqzjmUGzphWSvR7l0wIT/view?usp=sharing
* PDF: https://drive.google.com/file/d/1GhYHPBC8glkc9m0UFj-OdJMLQzRUHoWk/view?usp=sharing

# Функциональные требования

Для пользователей

- учет финансовых затрат (с выбором валюты и лимитов на траты)
- формирование отчета за неделю/месяц/год

Для администраторов

- предоставляет статистику по использованию сервиса
- предоставляет метрики для управления сервисом
- предоставляем логи и трейсы для разработчиков

Не функциональные требования:

- высокая скорость работы
- высокая отказоустойчивость

Дополнительные требования

- использование разных каналов (веб-сайт, телеграм etc)
- получение отчета на почту

# Нагрузка

В среднем пользователь сохраняет 10 трат в день. Максимальную нагрузку возмем как среднюю х 3.

Для 1000 пользователей:

- Средняя: 10000 сообщений / 24 / 60 / 60 = 0.116 RPS
- Максимальная: 0.347 RPS

Для 100 000 пользователей:

- Средняя: 11.574 RPS
- Максимальная: 34.722 RPS

Для 1 000 000 пользователей:

- Средняя: 115.740 RPS
- Максимальная: 347.222 RPS

# Оценка хранилища

На основе уже имеющихся данных можно приблизительно оценить размер БД (размер чистой базы ~7631663 bytes):

```
postgres=# SELECT pg_database_size('postgres');
 pg_database_size
------------------
         18207279
(1 row)

postgres=# select count(*) from spendings;
 count
-------
 52851
(1 row)
```

Одно сообщение приблизительно занимает (18207279 - 7631663) / 52851 = 200.1 байт (~ 200)

Буду рассчитывать для 1 000 000 пользователей:

за год мы отправим 115.74 * 365 * 24 * 60 * 60 = 3 649 976 640  сообщений
- в год мы ожидаем прирост на 730 Гб :) для хранения данных
- бекапы (x3 к размеру данных) = 2.2 Тб

Оценка размера оперативной памяти:

- Базе данных требуется: max_connections * work_mem * N + shared_buffers
значение по умолчанию для work_mem составляет 4 МБ, рекомендуется 16-32
N - сложность выборки, для простых запросов = 1
shared_buffers = обычно около 25% - 50% от общего объема оперативной памяти postgres

32 Мб * 347.222 RPS * 1 * 1.5 ~= 16 Гб на инстанс

- Кеш хранит 20% запросов за 24 часа, в среднем сообщение занимает 200 byte, необходимо 400 мегабайт памяти.