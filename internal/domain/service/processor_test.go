package service

import (
	"context"
	"errors"
	"testing"

	"stream_processor/internal/domain/dto"
	"stream_processor/internal/infra/mocks"
	"stream_processor/internal/infra/utils"
	customerr "stream_processor/internal/infra/customerr"
)

func TestService_Process(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name        string
		inDoc       *dto.Document
		setupMock   func(m *mocks.MockRepo)
		wantDoc     *dto.Document
		wantErr     bool
		errContains string
		wantGetCalls  int
		wantSaveCalls int
	}{
		{
			name:  "error - nil document",
			inDoc: nil,
			setupMock: func(m *mocks.MockRepo) {
				// не должен вызываться ни Get, ни Save
			},
			wantDoc:       nil,
			wantErr:       true,
			errContains:   "document is nil",
			wantGetCalls:  0,
			wantSaveCalls: 0,
		},
		{
			name: "ok - first document for url (not found in repo)",
			inDoc: &dto.Document{
				Url:       "https://example.com",
				Text:      "v1",
				PubDate:   10,
				FetchTime: 100,
			},
			setupMock: func(m *mocks.MockRepo) {
				m.GetFn = func(ctx context.Context, url string) (*dto.Document, error) {
					return nil, customerr.ErrDocumentNotFound
				}
				m.SaveFn = func(ctx context.Context, d dto.Document) error {
					// проверяем, что билдера применили по правилам:
					// PubDate = входной PubDate,
					// FetchTime = входной FetchTime,
					// FirstFetchTime = FetchTime первой версии
					if d.Url != "https://example.com" {
						t.Errorf("unexpected Url: %s", d.Url)
					}
					if d.Text != "v1" {
						t.Errorf("unexpected Text: %s", d.Text)
					}
					if d.PubDate != 10 {
						t.Errorf("unexpected PubDate: %d", d.PubDate)
					}
					if d.FetchTime != 100 {
						t.Errorf("unexpected FetchTime: %d", d.FetchTime)
					}
					if d.FirstFetchTime != 100 {
						t.Errorf("unexpected FirstFetchTime: %d", d.FirstFetchTime)
					}
					return nil
				}
			},
			wantDoc: &dto.Document{
				Url:       "https://example.com",
				Text:      "v1",
				PubDate:   10,
				FetchTime: 100,
			},
			wantErr:       false,
			wantGetCalls:  1,
			wantSaveCalls: 1,
		},
		{
			name: "ok - duplicate with same FetchTime -> no output",
			inDoc: &dto.Document{
				Url:       "https://example.com",
				Text:      "v1-dup",
				PubDate:   10,
				FetchTime: 100,
			},
			setupMock: func(m *mocks.MockRepo) {
				m.GetFn = func(ctx context.Context, url string) (*dto.Document, error) {
					return &dto.Document{
						Url:            "https://example.com",
						Text:           "v1",
						PubDate:        10,
						FetchTime:      100,
						FirstFetchTime: 100,
					}, nil
				}
				m.SaveFn = func(ctx context.Context, d dto.Document) error {
					t.Fatal("Save should not be called for duplicate FetchTime")
					return nil
				}
			},
			wantDoc:       nil, // ничего в очередь не пишем
			wantErr:       false,
			wantGetCalls:  1,
			wantSaveCalls: 0,
		},
		{
			name: "ok - newer document updates Text & FetchTime, keeps earliest PubDate/FirstFetchTime",
			inDoc: &dto.Document{
				Url:       "https://example.com",
				Text:      "v2",
				PubDate:   20,
				FetchTime: 200, // новый
			},
			setupMock: func(m *mocks.MockRepo) {
				m.GetFn = func(ctx context.Context, url string) (*dto.Document, error) {
					// existing — более старая версия
					return &dto.Document{
						Url:            "https://example.com",
						Text:           "v1",
						PubDate:        10,  // предыдущий pubdate
						FetchTime:      100, // предыдущий FetchTime
						FirstFetchTime: 100, // предыдущий FetchTime
					}, nil
				}
				m.SaveFn = func(ctx context.Context, d dto.Document) error {
					if d.Url != "https://example.com" {
						t.Errorf("unexpected Url: %s", d.Url)
					}
					if d.Text != "v2" {
						t.Errorf("unexpected Text: %s", d.Text)
					}
					if d.PubDate != 10 {
						t.Errorf("unexpected PubDate: %d", d.PubDate)
					}
					if d.FetchTime != 200 {
						t.Errorf("unexpected FetchTime: %d", d.FetchTime)
					}
					if d.FirstFetchTime != 100 {
						t.Errorf("unexpected FirstFetchTime: %d", d.FirstFetchTime)
					}
					return nil
				}
			},
			wantDoc: &dto.Document{
				Url:       "https://example.com",
				Text:      "v2",
				PubDate:   10,
				FetchTime: 200,
				FirstFetchTime: 100,
			},
			wantErr:       false,
			wantGetCalls:  1,
			wantSaveCalls: 1,
		},
		{
			name: "ok - older document than existing -> ignored",
			inDoc: &dto.Document{
				Url:       "https://example.com",
				Text:      "old",
				PubDate:   5,
				FetchTime: 50, // меньше, чем в existing
			},
			setupMock: func(m *mocks.MockRepo) {
				m.GetFn = func(ctx context.Context, url string) (*dto.Document, error) {
					return &dto.Document{
						Url:            "https://example.com",
						Text:           "v2",
						PubDate:        10,
						FetchTime:      200,
						FirstFetchTime: 100,
					}, nil
				}
				m.SaveFn = func(ctx context.Context, d dto.Document) error {
					t.Fatal("Save should not be called for older document")
					return nil
				}
			},
			wantDoc:       nil,
			wantErr:       false,
			wantGetCalls:  1,
			wantSaveCalls: 0,
		},
		{
			name: "error - repo Get returns unexpected error",
			inDoc: &dto.Document{
				Url:       "https://example.com",
				Text:      "v1",
				PubDate:   10,
				FetchTime: 100,
			},
			setupMock: func(m *mocks.MockRepo) {
				m.GetFn = func(ctx context.Context, url string) (*dto.Document, error) {
					return nil, errors.New("db down")
				}
			},
			wantDoc:       nil,
			wantErr:       true,
			errContains:   "repo get",
			wantGetCalls:  1,
			wantSaveCalls: 0,
		},
		{
			name: "error - repo Save fails for first document",
			inDoc: &dto.Document{
				Url:       "https://example.com",
				Text:      "v1",
				PubDate:   10,
				FetchTime: 100,
			},
			setupMock: func(m *mocks.MockRepo) {
				m.GetFn = func(ctx context.Context, url string) (*dto.Document, error) {
					return nil, customerr.ErrDocumentNotFound
				}
				m.SaveFn = func(ctx context.Context, d dto.Document) error {
					return errors.New("insert failed")
				}
			},
			wantDoc:       nil,
			wantErr:       true,
			errContains:   "save error",
			wantGetCalls:  1,
			wantSaveCalls: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mr := &mocks.MockRepo{}
			if tt.setupMock != nil {
				tt.setupMock(mr)
			}

			svc := NewService(mr)

			gotDoc, err := svc.Process(ctx, tt.inDoc)

			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				if tt.errContains != "" && !utils.Contains(err.Error(), tt.errContains) {
					t.Fatalf("expected error to contain %q, got %q", tt.errContains, err.Error())
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
			}

			if tt.wantDoc == nil {
				if gotDoc != nil {
					t.Fatalf("expected nil doc, got %+v", gotDoc)
				}
			} else {
				if gotDoc == nil {
					t.Fatalf("expected non-nil doc")
				}
				if *gotDoc != *tt.wantDoc {
					t.Fatalf("unexpected doc.\n got: %+v\nwant: %+v", gotDoc, tt.wantDoc)
				}
			}

			if mr.GetCalls != tt.wantGetCalls {
				t.Fatalf("unexpected Get calls: got %d, want %d", mr.GetCalls, tt.wantGetCalls)
			}
			if mr.SaveCalls != tt.wantSaveCalls {
				t.Fatalf("unexpected Save calls: got %d, want %d", mr.SaveCalls, tt.wantSaveCalls)
			}
		})
	}
}
