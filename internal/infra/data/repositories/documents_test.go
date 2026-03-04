package repositories

import (
	"context"
	"errors"
	"stream_processor/internal/domain/dto"
	customerr "stream_processor/internal/infra/customerr"
	"stream_processor/internal/infra/data/models"
	"stream_processor/internal/infra/mocks"
	"stream_processor/internal/infra/utils"
	
	"testing"
)

func TestDocumentsRepository_Save(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name         string
		doc          *dto.Document
		setupMock    func(*mocks.MockCache)
		wantErr      bool
		errContains string
	}{
		{
			name: "ok - valid doc, cache set success",
			doc: &dto.Document{
				Url:       "https://example.com",
				Text:      "hello",
				PubDate:   654321,
				FetchTime: 123456,
			},
			setupMock: func(m *mocks.MockCache) {
				m.SetFn = func(ctx context.Context, doc *models.Document) error {
					if doc.Url != "https://example.com" {
						t.Errorf("unexpected url in document: %s", doc.Url)
					}
					return nil
				}
			},
			wantErr: false,
		},
		{
			name: "error - url is empty",
			doc: &dto.Document{
				Url:       "",
				Text:      "hello",
				PubDate:   654321,
				FetchTime: 123456,
			},
			setupMock:    nil,
			wantErr:      true,
			errContains: "url is emty",
		},
		{
			name: "error - cache set fails",
			doc: &dto.Document{
				Url:  "https://example.com",
				Text: "hello",
			},
			setupMock: func(m *mocks.MockCache) {
				m.SetFn = func(ctx context.Context, doc *models.Document) error {
					return errors.New("some cache error")
				}
			},
			wantErr:     true,
			errContains: "cache save",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cache := &mocks.MockCache{}
			if tt.setupMock != nil {
				tt.setupMock(cache)
			}
			repo := NewRepo(cache)

			err := repo.Save(ctx, *tt.doc)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				if tt.errContains != "" && !errors.Is(err, customerr.ErrDocumentNotFound) {
					if !utils.Contains(err.Error(), tt.errContains) {
						t.Fatalf("expected error to contain %q, got %q", tt.errContains, err.Error())
					}
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestRepo_Get(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name        string
		url         string
		setupMock   func(m *mocks.MockCache)
		wantDoc     *dto.Document
		wantErr     bool
		errIs       error
		errContains string
	}{
		{
			name: "ok - document found and mapped",
			url:  "https://example.com",
			setupMock: func(m *mocks.MockCache) {
				m.GetFn = func(ctx context.Context, url string) (*models.Document, error) {
					if url != "https://example.com" {
						t.Errorf("unexpected url in QueryGet: %s", url)
					}
					return &models.Document{
						Url:       "https://example.com",
						Text:      "hello",
						PubDate:   1,
						FetchTime: 2,
					}, nil
				}
			},
			wantDoc: &dto.Document{
				Url:       "https://example.com",
				Text:      "hello",
				PubDate:   1,
				FetchTime: 2,
			},
			wantErr: false,
		},
		{
			name: "error - empty url",
			url:  "",
			setupMock: func(m *mocks.MockCache) {
				m.GetFn = func(ctx context.Context, url string) (*models.Document, error) {
					t.Fatal("QueryGet should not be called for empty url")
					return nil, nil
				}
			},
			wantDoc:     nil,
			wantErr:     true,
			errContains: "url is empty",
		},
		{
			name: "error - cache returns error -> ErrDocumentNotFound",
			url:  "https://missing.com",
			setupMock: func(m *mocks.MockCache) {
				m.GetFn = func(ctx context.Context, url string) (*models.Document, error) {
					return nil, errors.New("low-level cache error")
				}
			},
			wantDoc: nil,
			wantErr: true,
			errIs:   customerr.ErrDocumentNotFound,
		},
		{
			name: "error - cache returns nil doc",
			url:  "https://example.com",
			setupMock: func(m *mocks.MockCache) {
				m.GetFn = func(ctx context.Context, url string) (*models.Document, error) {
					return nil, nil
				}
			},
			wantDoc:     nil,
			wantErr:     true,
			errContains: "cache returned nil document",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := &mocks.MockCache{}
			if tt.setupMock != nil {
				tt.setupMock(mc)
			}

			repo := NewRepo(mc)

			doc, err := repo.Get(ctx, tt.url)

			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				if tt.errIs != nil && !errors.Is(err, tt.errIs) {
					t.Fatalf("expected error %v, got %v", tt.errIs, err)
				}
				if tt.errContains != "" && !utils.Contains(err.Error(), tt.errContains) {
					t.Fatalf("expected error to contain %q, got %q", tt.errContains, err.Error())
				}
				if doc != nil {
					t.Fatalf("expected nil doc, got %+v", doc)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.wantDoc == nil {
				if doc != nil {
					t.Fatalf("expected nil doc, got %+v", doc)
				}
				return
			}
			if doc == nil {
				t.Fatalf("expected non-nil doc")
			}
			if *doc != *tt.wantDoc {
				t.Fatalf("unexpected doc.\n got: %+v\nwant: %+v", doc, tt.wantDoc)
			}
		})
	}
}