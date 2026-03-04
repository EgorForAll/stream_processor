package mocks

import (
	"context"

	"stream_processor/internal/infra/data/models"
)

type MockCache struct {
	GetFn    func(ctx context.Context, url string) (*models.Document, error)
	GetCalls int
	SetFn    func(ctx context.Context, doc *models.Document) error
	SetCalls int
}

func (m *MockCache) QueryGet(ctx context.Context, url string) (*models.Document, error) {
	m.GetCalls++
	if m.GetFn != nil {
		return m.GetFn(ctx, url)
	}

	return nil, nil
}

func (m *MockCache) QuerySet(ctx context.Context, doc *models.Document) error {
	m.SetCalls++

	if m.SetFn != nil {
		return m.SetFn(ctx, doc)
	}
	return nil
}