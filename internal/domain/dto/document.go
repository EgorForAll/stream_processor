package dto

type Document struct {
	Url            string `json:"url"`
	PubDate        uint64 `json:"pub_date"`
	FetchTime      uint64 `json:"fetch_time"`
	Text           string `json:"text"`
	FirstFetchTime uint64 `json:"first_fetch_time"`
}

type DocumentBuilder struct {
	doc Document
}

func NewDocumentBuilder() *DocumentBuilder {
	return &DocumentBuilder{}
}

func (b *DocumentBuilder) SetUrl(url string) *DocumentBuilder {
	b.doc.Url = url
	return b
}

func (b *DocumentBuilder) SetPubDate(date uint64) *DocumentBuilder {
	b.doc.PubDate = date
	return b
}

func (b *DocumentBuilder) SetFetchTime(date uint64) *DocumentBuilder {
	b.doc.FetchTime = date
	return b
}

func (b *DocumentBuilder) SetText(text string) *DocumentBuilder {
	b.doc.Text = text
	return b
}

func (b *DocumentBuilder) SetFirstFetchTime(date uint64) *DocumentBuilder {
	b.doc.FirstFetchTime = date
	return b
}

func (b *DocumentBuilder) Build() Document {
	return b.doc
}