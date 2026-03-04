package main

import (
	"context"
	"stream_processor/internal/domain/dto"
	"stream_processor/internal/domain/service"
	"stream_processor/internal/infra/data/cache"
	 repo "stream_processor/internal/infra/data/repositories"
	"log"
)

type Processor interface {
	Process(ctx context.Context, doc *dto.Document) (*dto.Document, error)
}

func main() {
	replicas := map[string]string{
		"master":    "db-master.example.com:5432",
		"replica-1": "db-replica-1.example.com:5432",
		"replica-2": "db-replica-2.example.com:5432",
		"replica-3": "db-replica-3.example.com:5432",
		"replica-4": "db-replica-4.example.com:5432",
	}

	cache := cache.NewInMemory(replicas)
	repo := repo.NewRepo(cache)

	processor := service.NewService(repo)
	
	// имитация работы сервиса
	ctx := context.Background()
	
	
	doc := &dto.Document{
		Url:       "https://example.com",
		PubDate:   123,
		FetchTime: 456,
		Text:      "hello",
	}

	out, err := processor.Process(ctx, doc)
	if err != nil {
		log.Printf("process error: %v", err)
		return
	}

	if out == nil {
		log.Println("nothing to send to Kafka")
		return
	}

	log.Printf("processed doc: %+v", out)
}
