# Trendstream

Trendstream — Go-сервис для виджета актуальных поисковых трендов. Он читает поток поисковых событий из Kafka, считает Top-N нормализованных запросов за последние 5 минут и отдаёт результат через быстрый HTTP/JSON API.

Главная идея: ручка чтения не пересчитывает топ на каждый запрос. События попадают в in-memory скользящее окно, а фоновый процесс раз в секунду публикует неизменяемый snapshot с заранее подготовленным JSON для популярных лимитов. Поэтому получение топа сводится к короткому пути:

```text
atomic load snapshot -> write precomputed JSON
````

Общая схема:

```text
Kafka topic search-events
    ↓
Kafka consumer
    ↓
decode / validate / normalize / privacy filter / stop-list / anti-abuse
    ↓
sharded exact in-memory sliding window за 5 минут
    ↓
periodic immutable snapshot
    ↓
GET /v1/trends?limit=N
```

## Что реализовано

* Kafka consumer на `franz-go` с ручным commit offset после обработки batch.
* HTTP/JSON API на стандартном `net/http`.
* Контракт входящего события `SearchEvent` JSON v1.
* Валидация `schema_version`, `event_id`, `occurred_at`, `query`, длины query и допустимого clock skew.
* Нормализация query: trim, lower-case, сжатие пробелов, удаление control characters.
* Privacy filter для очевидных email, phone/card-like и слишком числовых запросов.
* Dynamic stop-list через admin API без перезапуска.
* Stop-list применяется и при приёме новых событий, и при сборке snapshot, поэтому уже накопленный query исчезает из выдачи после следующего обновления snapshot.
* Точное шардированное скользящее окно за последние 5 минут по времени события.
* Per-actor per-query limit для базовой защиты от повторов одного источника.
* Лимиты на cardinality, чтобы поток уникального мусора не положил сервис.
* Atomic immutable snapshot и заранее подготовленный JSON для лимитов 10, 20, 50, 100.
* Prometheus metrics.
* `pprof` на admin port при `PPROF_ENABLED=true`.
* Unit-тесты ключевой бизнес-логики.
* Локальный запуск Kafka через Docker Compose.
* Smoke, load и profiling scripts.

## Быстрый запуск

### Вариант 1: Kafka в Docker, сервис на host

```bash
make up
make create-topic
make describe-topic

KAFKA_ENABLED=true PPROF_ENABLED=true make run-kafka
```

В другом терминале:

```bash
make smoke
make produce PRODUCE_RATE=1000 PRODUCE_DURATION=10s

curl -s 'http://localhost:8080/v1/trends?limit=20' | jq
```

Остановить Kafka:

```bash
make down
```

### Вариант 2: Kafka и сервис в Docker Compose

```bash
make up-service
make create-topic
make logs-service
```

Проверка:

```bash
make smoke
make produce PRODUCE_RATE=1000 PRODUCE_DURATION=10s

curl -s 'http://localhost:8080/v1/trends?limit=20' | jq
```

Остановить:

```bash
make down
```

## API

### `GET /healthz`

Проверяет, что процесс жив.

```bash
curl -i http://localhost:8080/healthz
```

### `GET /readyz`

Возвращает readiness-ответ процесса.

```bash
curl -i http://localhost:8080/readyz
```

### `GET /v1/trends?limit=N`

Возвращает Top-N поисковых запросов за последние 5 минут.

```bash
curl -i 'http://localhost:8080/v1/trends?limit=20'
```

Пример ответа:

```json
{
  "window_seconds": 300,
  "generated_at": "2026-05-23T12:00:00Z",
  "items": [
    {
      "query": "iphone 15",
      "count": 1842
    },
    {
      "query": "кроссовки женские",
      "count": 1720
    }
  ]
}
```

Правила `limit`:

* значение по умолчанию: `20`;
* максимум: `100`;
* `limit <= 0`, нечисловой `limit` и `limit > 100` возвращают `400 Bad Request`.

## Admin API

Admin API слушает отдельный порт, по умолчанию `:9090`, и требует bearer token.

Для локального запуска используется:

```text
Authorization: Bearer dev-token
```

В проде нужно задать свой `ADMIN_TOKEN` и не публиковать admin port наружу.

### `POST /admin/events`

Ручная отправка события без Kafka. Удобно для проверки сервиса.

```bash
NOW="$(date -u +"%Y-%m-%dT%H:%M:%SZ")"

curl -i \
  -H 'Authorization: Bearer dev-token' \
  -H 'Content-Type: application/json' \
  -d "{
    \"schema_version\": 1,
    \"event_id\": \"manual-1\",
    \"occurred_at\": \"${NOW}\",
    \"query\": \"manual iphone 15\",
    \"user_id_hash\": \"u_manual_1\"
  }" \
  http://localhost:9090/admin/events
```

Проверить, что query появился в топе:

```bash
sleep 2
curl -s 'http://localhost:8080/v1/trends?limit=20' | jq
```

### `GET /admin/stop-list`

```bash
curl -s \
  -H 'Authorization: Bearer dev-token' \
  http://localhost:9090/admin/stop-list | jq
```

### `POST /admin/stop-list`

Добавляет термин в stop-list. Значение нормализуется тем же способом, что и поисковый запрос.

```bash
curl -i \
  -H 'Authorization: Bearer dev-token' \
  -H 'Content-Type: application/json' \
  -d '{"term":"manual iphone 15"}' \
  http://localhost:9090/admin/stop-list
```

Поведение stop-list:

* multi-token term, например `manual iphone 15`, работает как exact phrase;
* single-token term, например `casino`, работает как запрещённое слово и скрывает query вроде `best casino online`;
* уже накопленный query исчезает из `/v1/trends` после следующей сборки snapshot, обычно в пределах 1 секунды;
* накопленные counters физически не чистятся сразу, а естественно выходят из 5-минутного окна.

### `DELETE /admin/stop-list/{term}`

```bash
curl -i \
  -X DELETE \
  -H 'Authorization: Bearer dev-token' \
  'http://localhost:9090/admin/stop-list/manual%20iphone%2015'
```

## Контракт Kafka-события

Topic по умолчанию:

```text
search-events
```

Message key желательно делать равным нормализованному query. Сервис всё равно сам нормализует поле `query`, но key полезен для будущего масштабирования по партициям.

Message value — JSON:

```json
{
  "schema_version": 1,
  "event_id": "01HYZX5N6A6P6V6E8EV5V7C2SC",
  "occurred_at": "2026-05-23T10:15:30.123Z",
  "query": "  iPhone 15 Pro  ",
  "user_id_hash": "u_8f4a1c0b",
  "session_id": "s_91fd0f",
  "device_id_hash": "d_673aef",
  "ip_hash": "ip_d3a11e",
  "user_agent_hash": "ua_98a77b",
  "region": "local",
  "locale": "ru-RU",
  "platform": "desktop",
  "is_bot": false
}
```

Обязательные поля:

| Поле             | Зачем нужно                                                             |
| ---------------- | ----------------------------------------------------------------------- |
| `schema_version` | Версионирование контракта. Сейчас поддерживается `1`.                   |
| `event_id`       | Идентификатор события для трассировки и возможной будущей дедупликации. |
| `occurred_at`    | Время пользовательского поиска. По нему считается 5-минутное окно.      |
| `query`          | Поисковая строка, по которой считается популярность.                    |

Опциональные поля:

| Поле                           | Зачем нужно                                               |
| ------------------------------ | --------------------------------------------------------- |
| `user_id_hash`                 | Основной ключ источника события для anti-abuse rules.     |
| `device_id_hash`               | Запасной ключ источника события.                          |
| `ip_hash`                      | Запасной ключ, если нет user/device id.                   |
| `session_id`                   | Ещё один запасной ключ.                                   |
| `user_agent_hash`              | Сигнал для будущих anti-abuse правил.                     |
| `region`, `locale`, `platform` | Основа для будущих региональных или платформенных топов.  |
| `is_bot`                       | Если upstream уже распознал бота, событие не учитывается. |

Сырые `user_id`, IP, email, phone и другие персональные данные в payload передавать не нужно. Сервису достаточно хэшей и грубой технической метаинформации.

## Обработка события

Событие проходит такой путь:

```text
decode JSON
→ validate schema/time/query
→ normalize query
→ privacy filter
→ bot flag check
→ stop-list check
→ add to sharded sliding window
→ metrics
```

Основные причины отбрасывания события:

```text
invalid_event
empty_query
privacy_filter
stoplist
bot
too_old
from_future
cardinality_limit
bucket_cardinality_limit
actor_query_limit
```

Эти причины видны в метриках и в ответе ручки `POST /admin/events`.

## Семантика времени

Топ считается по времени события, а не по времени обработки:

```text
snapshot generated_at = T
учитываются события с occurred_at внутри [T - 5 минут, T]
```

Почему так: если Kafka consumer отстал, старые события не должны выглядеть как свежие только потому, что сервис обработал их позже.

События слишком далеко из будущего или старше окна отклоняются.

## Архитектура и структуры данных

### Точное in-memory окно

Окно короткое — 5 минут, а запросов на чтение топа ожидается намного больше, чем входящих событий. Поэтому состояние хранится в памяти процесса:

```text
query -> count за окно
ring buckets по секундам
actor counters для per-actor limit
```

Внешняя база данных не используется на критическом пути: она добавила бы сетевой вызов, задержку и лишнюю инфраструктурную сложность. По условию сервис может стартовать пустым, поэтому durable storage для счётчиков топа не является обязательным.

### Шардирование

Одна общая map под одним mutex плохо подходит для highload. Поэтому состояние разбито на shards:

```text
shard = hash(normalized_query) % SHARD_COUNT
```

Каждый query всегда попадает в один shard. Для сборки общего топа сервис берёт локальный топ из каждого shard и затем объединяет результаты.

Stop-list фильтруется до локального топа. Поэтому query, который должен попасть в общий Top-N после фильтрации, не потеряется из-за того, что в его shard выше него были stop-listed query.

### Snapshot

Раз в секунду фоновый процесс собирает новый snapshot и публикует его через `atomic.Pointer`.

После публикации snapshot не изменяется. HTTP-ручка только читает текущий snapshot и отдаёт заранее подготовленный JSON для популярных лимитов.

Это важный trade-off: топ может отставать примерно на 1 секунду, зато `/v1/trends` не сортирует map и не сканирует окно на каждый запрос.

## Stop-list

Stop-list хранится как copy-on-write snapshot внутри процесса и сохраняется в JSON-файл `STOPLIST_PATH`.

При обновлении:

```text
load current terms
→ normalize term
→ write temp file
→ rename
→ publish new in-memory snapshot
```

На приёме событий stop-list не даёт новым запрещённым query попасть в агрегатор. При сборке snapshot он скрывает query, которые уже были накоплены раньше. Поэтому изменение stop-list быстро отражается в API без дорогой чистки всех buckets.

## Anti-abuse

В текущей версии реализован простой и прозрачный набор правил:

1. `is_bot=true` — событие не учитывается.
2. Источник события выбирается из `user_id_hash`, затем `device_id_hash`, затем `ip_hash`, затем `session_id`.
3. Один источник может добавить один и тот же query не больше `PER_ACTOR_QUERY_LIMIT=3` раз за окно.

Это не полноценная anti-fraud система. В проде можно добавить diversity checks, top actor domination, shadow/enforce mode и более богатые сигналы. Для тестового задания важно, что текущие правила простые, детерминированные и объяснимые.

## Защита от высокой cardinality

Основные лимиты:

```text
MAX_UNIQUE_QUERIES=1000000
MAX_UNIQUE_QUERIES_PER_BUCKET=100000
PER_ACTOR_QUERY_LIMIT=3
```

Если shard достиг лимита уникальных query, новые неизвестные query отбрасываются с причиной `cardinality_limit`, но уже известные query продолжают считаться. Это защищает сервис от потока уникального мусора и не ломает уже накопленные популярные запросы.

Query не используется как Prometheus label, чтобы не получить взрыв cardinality и не утечь поисковые строки в метрики.

## Метрики и pprof

Prometheus metrics доступны на admin port:

```bash
curl -s http://localhost:9090/metrics | grep -E 'trendstream|http|kafka' | head -80
```

`pprof` включается явно:

```bash
PPROF_ENABLED=true KAFKA_ENABLED=true make run-kafka
```

Heap profile:

```bash
make profile-heap
```

CPU profile:

```bash
make profile-cpu CPU_PROFILE_SECONDS=30
```

Открыть профиль:

```bash
go tool pprof -http=:6060 bench/results/cpu.pprof
```

## Проверки

```bash
gofmt -w .
go mod tidy
go test ./...
go test -race ./...
go vet ./...
```

Coverage можно сгенерировать локально, но файлы `coverage.out` и `coverage.html` не нужно коммитить:

```bash
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html
```

## Нагрузочная проверка

Установить `vegeta`:

```bash
make install-vegeta
export PATH="$HOME/go/bin:$PATH"
```

Read-only:

```bash
make bench-read READ_RATE=1000 READ_DURATION=30s READ_LIMIT=20
make bench-read READ_RATE=5000 READ_DURATION=30s READ_LIMIT=20
```

Mixed ingestion + reads:

```bash
make bench-mixed PRODUCE_RATE=1000 PRODUCE_DURATION=30s READ_RATE=1000 READ_DURATION=30s READ_LIMIT=20
```

Результаты сохраняются в `bench/results/` и не должны попадать в git.

Пример локальных результатов:

| Сценарий                     |        Нагрузка | Длительность | Средняя задержка |     p99 | Успешно |
| ---------------------------- | --------------: | -----------: | ---------------: | ------: | ------: |
| read top-20                  |        1000 rps |          30s |           ~404µs |  ~751µs |    100% |
| read top-20                  |        5000 rps |          30s |           ~338µs | ~1.11ms |    100% |
| produce 1000/s + read 1000/s | 1000 rps чтения |          30s |           ~404µs |  ~858µs |    100% |

Эти цифры не являются универсальным SLA: они зависят от машины, версии Go, Docker/Kafka и текущей нагрузки. Их задача — показать, что ручка чтения не делает тяжёлую работу на каждый HTTP-запрос.

## Конфигурация

Основные переменные окружения:

```text
SERVICE_NAME=trendstream
HTTP_ADDR=:8080
ADMIN_ADDR=:9090
LOG_LEVEL=info
SHUTDOWN_TIMEOUT=5s

ADMIN_TOKEN=dev-token
STOPLIST_PATH=data/stoplist.json
PPROF_ENABLED=false

KAFKA_ENABLED=false
KAFKA_BROKERS=localhost:9092
KAFKA_TOPIC=search-events
KAFKA_GROUP_ID=trendstream
KAFKA_CLIENT_ID=trendstream-local

SHARD_COUNT=32
MAX_UNIQUE_QUERIES=1000000
MAX_UNIQUE_QUERIES_PER_BUCKET=100000
PER_ACTOR_QUERY_LIMIT=3
```

Пример локального файла есть в `.env.example`.

## Компромиссы и ограничения

### At-least-once обработка Kafka

Consumer коммитит offset после обработки batch. Это снижает риск потери события между чтением из Kafka и применением в агрегаторе, но при crash/rebalance возможна повторная обработка части сообщений.

Дедупликация по `event_id` не реализована, потому что для неё нужен отдельный 5-минутный dedup cache, а это дополнительная память и сложность.

### Пустой старт

Сервис стартует без восстановления предыдущего 5-минутного окна. Это допустимо по условию задачи.

На проде: при старте искать Kafka offsets по timestamp и дочитывать последние 5 минут.

### Один агрегирующий экземпляр

Нельзя просто запустить несколько одинаковых replicas в одной Kafka consumer group и ожидать, что каждая будет отдавать global top. Kafka распределит partitions между replicas, и каждый экземпляр увидит только часть событий.

Возможные варианты для прода:

1. один aggregator строит общий snapshot, read replicas получают готовый snapshot;
2. несколько shard aggregators считают локальные топы, отдельный merger собирает global top;
3. каждая replica читает весь поток в отдельной consumer group — проще, но дороже.

### Exact counting

Сервис использует точный подсчёт, а не Count-Min Sketch или Space-Saving. Это проще проверить и объяснить, но при экстремальной cardinality может потребоваться approximate heavy hitters mode.

### Stop-list persistence

Stop-list сохраняется в локальный JSON-файл. Для одного процесса этого достаточно. Для нескольких replicas нужен общий control plane или механизм распространения stop-list snapshot.

### Health/readiness

`/healthz` и `/readyz` сейчас простые process-level endpoints. На проде стоит добавить проверку Kafka assignment/lag и возраста последнего snapshot.