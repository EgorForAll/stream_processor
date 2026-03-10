package cache

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"stream_processor/internal/infra/customerr"
	"stream_processor/internal/infra/data/models"
	"sync"
	"testing"
	"time"

	"github.com/gojuno/minimock/v3"
)

// Успешный QueryGet
func TestInMemory_QueryGet_OK(t *testing.T) {
	ctx := context.Background()
	mc := minimock.NewController(t)
	defer mc.Finish()

	dbMock := NewDBConnMock(mc)

	url := "https://example.com"
	replicas := map[string]string{
		"replica-1": "http://db-replica-1.example.com:5432",
		"replica-2": "http://db-replica-2.example.com:5432",
		"replica-3": "http://db-replica-3.example.com:5432",
	}

	// тело успешного ответа
	body, _ := json.Marshal(&models.Document{Url: url})
	respOK := &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(bytes.NewReader(body)),
	}

	dbMock.DoMock.Set(func(req *http.Request) (*http.Response, error) {
		return respOK, nil
	})

	in := NewInMemory(dbMock, replicas)

	doc, err := in.QueryGet(ctx, url)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if doc == nil || doc.Url != url {
		t.Fatalf("unexpected doc: %+v", doc)
	}

}

// QueryGet: несколько ошибок от Do, потом успешный ответ
func TestInMemory_QueryGet_ErrorThenSuccess(t *testing.T) {
	ctx := context.Background()
	mc := minimock.NewController(t)
	defer mc.Finish()

	dbMock := NewDBConnMock(mc)

	url := "https://example.com"
	replicas := map[string]string{
		"replica-1": "http://db-replica-1.example.com:5432",
	}

	// успешный ответ
	body, _ := json.Marshal(&models.Document{Url: url})
	respOK := &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(bytes.NewReader(body)),
	}

	var call int
	dbMock.DoMock.Set(func(req *http.Request) (*http.Response, error) {
		call++
		switch call {
		case 1, 2:
			// имитируем сетевую ошибку
			return nil, fmt.Errorf("temporary error %d", call)
		case 3:
			return respOK, nil
		default:
			t.Fatalf("unexpected extra Do call: %d", call)
			return nil, nil
		}
	})

	in := NewInMemory(dbMock, replicas)

	doc, err := in.QueryGet(ctx, url)
	if err != nil {
		t.Fatalf("expected success after retries, got %v", err)
	}
	if doc == nil || doc.Url != url {
		t.Fatalf("unexpected doc: %+v", doc)
	}

	if call != 3 {
		t.Fatalf("expected 3 Do calls (2 errors + success), got %d", call)
	}
	if calls := dbMock.DoAfterCounter(); calls != 3 {
		t.Fatalf("expected DoAfterCounter = 3, got %d", calls)
	}
}

func TestInMemory_ManyConcurrency(t *testing.T) {
	ctx := context.Background()
	mc := minimock.NewController(t)
	defer mc.Finish()

	dbMock := NewDBConnMock(mc)

	url := "https://example.com"
	replicas := map[string]string{
		"replica-1": "http://db-replica-1.example.com:5432",
		"replica-2": "http://db-replica-2.example.com:5432",
		"replica-3": "http://db-replica-3.example.com:5432",
	}

	bodyBytes, _ := json.Marshal(&models.Document{Url: url})

	dbMock.DoMock.Set(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewReader(bodyBytes)),
		}, nil
	})

	in := NewInMemory(dbMock, replicas)

	var wg sync.WaitGroup
	for i := 0; i < 10000; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			doc, err := in.QueryGet(ctx, url)
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if doc == nil || doc.Url != url {
				t.Fatalf("unexpected doc: %+v", doc)
			}
		}()
	}
	wg.Wait()
}

func TestInMemory_QueryGet_ContextExceeded(t *testing.T) {
	// Дадим немного времени, но меньше, чем нужно для ретраев
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	mc := minimock.NewController(t)
	defer mc.Finish()

	dbMock := NewDBConnMock(mc)

	url := "https://example.com"
	replicas := map[string]string{
		"replica-1": "http://db-replica-1.example.com:5432",
	}

	// Мок: просто ждём отмены контекста
	dbMock.
		DoMock.
		Optional().
		Set(func(req *http.Request) (*http.Response, error) {
		<-req.Context().Done()
		return nil, req.Context().Err()
	})

	in := NewInMemory(dbMock, replicas)

	_, err := in.QueryGet(ctx, url)

	if err == nil {
		t.Fatalf("expected error %v, got nil", customerr.ErrCtxExeeded)
	}
	if !errors.Is(err, customerr.ErrCtxExeeded) {
		t.Fatalf("expected error %v, got %v", customerr.ErrCtxExeeded, err)
	}
}
