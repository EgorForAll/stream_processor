package cache

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"stream_processor/internal/infra/customerr"
	"stream_processor/internal/infra/data/models"
)

const (
	retries        = 3
	retriesTimeout = 500 * time.Millisecond // ms
	clientTimeout  = 8 * time.Second        // ms
)

var ErrNoReplica = errors.New("no replica responded in time")

type ReqErr struct {
	addr string
	err  error
}

type InMemory struct {
	replicas map[string]string
	client   DBConn
}

//go:generate minimock -i DBConn
type DBConn interface {
	Do(req *http.Request) (*http.Response, error)
}

func NewInMemory(conn DBConn, replicas map[string]string) *InMemory {
	return &InMemory{replicas: replicas, client: conn}
}

/*
Получает первый успешный ответ из реплик.
Как только первый ответ приходит, остальные запросы отменяются.
Если ни одна реплика не ответила успешно, все накопленные ошибки объединяются в одну через errors.Join и возвращаются.
*/
func (in *InMemory) QueryGet(ctx context.Context, url string) (*models.Document, error) {
	if in.replicas == nil {
		return nil, ErrNoReplica
	}

	getCtx, cancel := context.WithTimeout(ctx, clientTimeout)
	defer cancel()

	treads := len(in.replicas)
	resultCh := make(chan *models.Document)
	errCh := make(chan error, treads)

	var wg sync.WaitGroup
	wg.Add(treads)

	for addr := range in.replicas {
		go func(addr string) {
			defer wg.Done()
			query := fmt.Sprintf("%s/%s", addr, url)
			doc, err := in.doQueryWorker(getCtx, query)
			if err != nil {
				errCh <- err
				return
			}
			if doc != nil {
				resultCh <- doc
			}
		}(addr)
	}

	go func() {
		wg.Wait()
		close(resultCh)
		close(errCh)
	}()

	var errs []error
	for resultCh != nil || errCh != nil {
		select {
		case doc, ok := <-resultCh:
			if !ok {
				resultCh = nil
			} else {
				cancel()
				return doc, nil
			}
		case e, ok := <-errCh:
			if !ok {
				errCh = nil
			} else {
				errs = append(errs, e)
			}
		case <-getCtx.Done():
			return nil, customerr.ErrCtxExeeded
		}
	}

	return nil, errors.Join(errs...)
}

/*
Делает POST запрос к мастеру с ретрайами
*/
func (in *InMemory) QuerySet(ctx context.Context, doc *models.Document) error {
	if in.replicas == nil {
		return ErrNoReplica
	}
	if doc == nil {
		return errors.New("doc cannot be nil")
	}
	if doc.Url == "" {
		return errors.New("url cannot be empty")
	}

	setCtx, cancel := context.WithTimeout(ctx, clientTimeout)
	defer cancel()

	master, ok := in.replicas[doc.Url]
	if !ok {
		return errors.New("no master replica found")
	}

	body, err := json.Marshal(doc)
	if err != nil {
		return fmt.Errorf("marshal doc: %w", err)
	}

	query := fmt.Sprintf("%s/%s", master, doc.Url)

	return in.doSetQueryWithRetries(setCtx, query, body)
}

func (in *InMemory) doQueryWorker(ctx context.Context, query string) (*models.Document, error) {
	select {
	case <-ctx.Done():
		// контекст истек, shutdown
		return nil, nil
	default:
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, query, nil)
		if err != nil {
			return nil, fmt.Errorf("query %s: build req error -> %w", query, err)
		}
		doc, err := in.doGetQueryWithRetries(ctx, req)
		if err != nil {
			return nil, fmt.Errorf("query %s: error req with retries -> %w", query, err)
		}
		
		return doc, nil
	}
}

func (in *InMemory) doSetQueryWithRetries(ctx context.Context, query string, body []byte) error {
	var lastErr error

	for i := 0; i <= retries; i++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, query, bytes.NewReader(body))
		if err != nil {
			return fmt.Errorf("create set request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := in.client.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("request failed: %w", err)

			if i < retries {
				if err := sleepWithContext(ctx, retriesTimeout); err != nil {
					return err
				}
				continue
			}

			return lastErr
		}

		resp.Body.Close()

		if resp.StatusCode >= http.StatusInternalServerError || resp.StatusCode == http.StatusTooManyRequests {
			lastErr = fmt.Errorf("temporary error status: %d", resp.StatusCode)

			if i < retries {
				if err := sleepWithContext(ctx, retriesTimeout); err != nil {
					return err
				}
				continue
			}

			return lastErr
		}

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return nil
		}

		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return lastErr
}

func (in *InMemory) doGetQueryWithRetries(ctx context.Context, req *http.Request) (*models.Document, error) {
	var err error

	for i := 0; i <= retries; i++ {
		cloned := req.Clone(ctx)

		var resp *http.Response
		resp, err = in.client.Do(cloned)

		if err == nil && resp != nil {
			if resp.StatusCode >= 500 || resp.StatusCode == 429 {
				resp.Body.Close()
				if i < retries {
					if err := sleepWithContext(ctx, retriesTimeout); err != nil {
						return nil, err
					}
					continue
				}
			}

			doc, fmtErr := in.formatResponse(resp)
			resp.Body.Close()
			if fmtErr != nil {
				return nil, fmtErr
			}
			return doc, nil
		}

		if i < retries {
			if err := sleepWithContext(ctx, retriesTimeout); err != nil {
				return nil, err
			}
			continue
		}
	}

	return nil, fmt.Errorf("all %d retries failed: %w", retries, err)
}

func (in *InMemory) formatResponse(resp *http.Response) (*models.Document, error) {
	var doc *models.Document
	if err := json.NewDecoder(resp.Body).Decode(&doc); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return doc, nil
}

func sleepWithContext(ctx context.Context, d time.Duration) error {
	select {
	case <-time.After(d):
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
