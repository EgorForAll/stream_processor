package service

import (
	"context"
	"errors"
	"fmt"
	"stream_processor/internal/domain/dto"
	customerr "stream_processor/internal/infra/customerr"
)

// На вход сервису поступают обновления документов
//
// в формате protobuf
// message TDocument {
//     string Url = 1;  // URL документа, его уникальный идентификатор
//     uint64 PubDate = 2;  // время заявляемой публикации документа
//     uint64 FetchTime = 3; // время получения данного обновления документа, может рассматриваться как идентификатор версии. Пара (Url, FetchTime) уникальна.
//     string Text = 4; // текст документа
//     uint64 FirstFetchTime = 5; // изначально отсутствует, необходимо заполнить
// }
//
// Документы могут поступать в произвольном порядке (не в том, как они обновлялись),
// также возможно дублирование отдельных сообщений.

// Необходимо на выходе формировать такие же сообщения, но с исправленными отдельными полями
// по следующим правилам (всё нижеуказанное - для группы документов с совпадающим полем Url):
// - Поле `Text` и `FetchTime` должны быть такими, какими были в документе с наибольшим `FetchTime` на данный момент
// - Поле `PubDate` должно быть таким, каким было у сообщения с наименьшим `FetchTime`
// - Поле `FirstFetchTime` должно быть равно минимальному значению `FetchTime`
//
// То есть в каждый момент времени мы берём `PubDate` и `FirstFetchTime` от самой первой из полученных
// на данный момент версий (если отсортировать их по `FetchTime`), а `Text` - от самой последней.
//
// Данный код будет работать в сервисе, читающим входные сообщения из очереди сообщений (Kafka или подобное),
// и записывающем результат также в очередь. Если `Process` возвращает `nil` - то в очередь ничего не пишется.
//
//
// Интерфейс в коде можно реализовать таким:

type Repository interface {
	Save(ctx context.Context, doc dto.Document) error
	Get(ctx context.Context, url string) (*dto.Document, error)
}

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{
		repo: repo,
	}
}

func (s *Service) Process(ctx context.Context, doc *dto.Document) (*dto.Document, error) {
	if doc == nil {
		return nil, errors.New("document is nil")
	}

	existing, err := s.repo.Get(ctx, doc.Url)
	if err != nil {
		if errors.Is(err, customerr.ErrDocumentNotFound) {
			existing = nil
		} else {
			return nil, fmt.Errorf("repo get: %w", err)
		}
	}

	if existing == nil {
		newDoc := dto.NewDocumentBuilder().
			SetUrl(doc.Url).
			SetText(doc.Text).
			SetPubDate(doc.PubDate).
			SetFetchTime(doc.FetchTime).
			SetFirstFetchTime(doc.FetchTime).
			Build()

		if err := s.repo.Save(ctx, newDoc); err != nil {
			return nil, fmt.Errorf("save error: %w", err)
		}

		return doc, nil
	}

	// для того чтобы не обновлять документ, если он уже есть в базе и его FetchTime не изменился
	if existing.FetchTime == doc.FetchTime {
		return nil, nil
	}

	if existing.FetchTime < doc.FetchTime {

		updatedDoc := dto.NewDocumentBuilder().
			SetUrl(doc.Url).
			SetText(doc.Text).
			SetPubDate(existing.PubDate).
			SetFetchTime(doc.FetchTime).
			SetFirstFetchTime(existing.FirstFetchTime).
			Build()

		if err := s.repo.Save(ctx, updatedDoc); err != nil {
			return nil, fmt.Errorf("repo save: %w", err)
		}

		return &updatedDoc, nil
	}

	return nil, nil
}

// Данный код будет работать в сервисе, читающим входные сообщения из очереди сообщений (Kafka или подобное),
// и записывающем результат также в очередь. Если Process возвращает Null - то в очередь ничего не пишется.
