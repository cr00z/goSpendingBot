# Observability

> Я во все цели добавил sudo, где-то может не хватить - если так, надо добавить. Работал на терминале без sudo (так получилось), оттестить не получилось

Использовал всю инфраструктуру с воркшопа, немного поменял логирование (вместо отдельного контейнера file.d запускается вместе с ботом)

## Запуск бота

`make bot`

file.d пытается отправить логи на docker-host:12201 (GELF TCP в graylog)

## Запуcк graylog

1. `make logs`
2. Graylog: http://127.0.0.1:7555/ (admin/admin)
3. System->Inputs, добавляем GELF tcp, все значения по-умолчанию

## Metrics

1. `make metrics`
2. Prometheus: http://127.0.0.1:9090/
3. Grafana: http://127.0.0.1:3000/ (admin/admin)
4. Data sources->Prometheus, адрес `http://prometheus:9090`

## Tracing

1. `make tracing`
2. Jaeger: http://127.0.0.1:16686/