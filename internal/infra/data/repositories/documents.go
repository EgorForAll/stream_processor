package repositories

import (
	"context"
	"errors"
	"fmt"

	"stream_processor/internal/domain/dto"
	customerr "stream_processor/internal/infra/customerr"
	"stream_processor/internal/infra/data/models"
)

type ICache interface {
	QueryGet(ctx context.Context, url string) (*models.Document, error)
	QuerySet(ctx context.Context, doc *models.Document) error
}

type Repo struct {
	Cache ICache
}

func NewRepo(cache ICache) *Repo {
	return &Repo{Cache: cache}
}

func (r *Repo) Save(ctx context.Context, doc dto.Document) error {
	if doc.Url == "" {
		return errors.New("url is emty")
	}

	docEntity := models.FromDTO(doc)

	if err := r.Cache.QuerySet(ctx, docEntity); err != nil {
		return fmt.Errorf("cache save: %w", err)
	}

	return nil
}

func (r *Repo) Get(ctx context.Context, url string) (*dto.Document, error) {
	if url == "" {
		return nil, errors.New("url is empty")
	}

	docEntity, err := r.Cache.QueryGet(ctx, url)
	if err != nil {
		switch err {
		case customerr.ErrDocumentNotFound:
			return nil, customerr.ErrDocumentNotFound // тут должна быть какая-то обертка
		case customerr.ErrCtxExeeded:
			return nil, customerr.ErrCtxExeeded // тут должна быть какая-то обертка
		default:
			return nil, fmt.Errorf("cache get: %w", err)
		}
	}

	if docEntity == nil {
		return nil, errors.New("cache returned nil document")
	}

	doc := models.ToDTO(docEntity)

	return &doc, nil
}
