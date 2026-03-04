# stream_processor

Сервис на Go, который читает обновления документов из очереди (Kafka или аналог), агрегирует версии по `Url` по заданным правилам и отдаёт «нормализованные» документы.

## Описание задачи

На вход поступают сообщения формата:

```proto
message TDocument {
    string Url = 1;            // URL документа, уникальный идентификатор
    uint64 PubDate = 2;        // заявленное время публикации документа
    uint64 FetchTime = 3;      // время получения версии (идентификатор версии)
    string Text = 4;           // текст документа
    uint64 FirstFetchTime = 5; // изначально отсутствует, заполняется сервисом
}
```

Для всех сообщений с одинаковым `Url` сервис поддерживает инварианты:

- `Text` и `FetchTime` — всегда от **самой новой** версии (максимальный `FetchTime`).
- `PubDate` — от **самой первой** версии (минимальный `FetchTime`).
- `FirstFetchTime` — **минимальный** `FetchTime` среди всех полученных версий.

## Запуск проекта

Все команды выполняются из корня репозитория.

### Быстрый старт
```bash
go run ./cmd/main.go
```

### Сборка и запуск бинарника
```bash
go build -o ./bin/stream_processor ./cmd/main.go
```

### Сборка с escape-анализом
```bash
go build -o ./bin/stream_processor -gcflags="-m" ./cmd/main.go
```

### Запуск тестов
```bash
go test ./internal/domain/service ./internal/infra/data/repositories
```
