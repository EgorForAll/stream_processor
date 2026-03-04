package models

import "stream_processor/internal/domain/dto"

type Document struct {
	Url            string `json:"url"`
	PubDate        uint64 `json:"pub_date"`
	FetchTime      uint64 `json:"fetch_time"`
	Text           string `json:"text"`
	FirstFetchTime uint64 `json:"first_fetch_time"`
}

func FromDTO(doc dto.Document) *Document {
	return &Document{
		Url:            doc.Url,
		PubDate:        doc.PubDate,
		FetchTime:      doc.FetchTime,
		Text:           doc.Text,
		FirstFetchTime: doc.FirstFetchTime,
	}
}

func ToDTO(doc *Document) dto.Document {
	return dto.Document{
		Url:            doc.Url,
		PubDate:        doc.PubDate,
		FetchTime:      doc.FetchTime,
		Text:           doc.Text,
		FirstFetchTime: doc.FirstFetchTime,
	}
}

