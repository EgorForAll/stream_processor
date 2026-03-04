package mocks

import (
	"context"
	"stream_processor/internal/domain/dto"
)

type MockRepo struct {
	GetFn    func(ctx context.Context, url string) (*dto.Document, error)
	GetCalls int

	SaveFn    func(ctx context.Context, doc dto.Document) error
	SaveCalls int
}

func (m *MockRepo) Get(ctx context.Context, url string) (*dto.Document, error) {
	m.GetCalls++
	if m.GetFn != nil {
		return m.GetFn(ctx, url)
	}
	return nil, nil
}

func (m *MockRepo) Save(ctx context.Context, doc dto.Document) error {
	m.SaveCalls++
	if m.SaveFn != nil {
		return m.SaveFn(ctx, doc)
	}
	return nil
}