package pdf

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"strings"

	"github.com/neurosnap/sentences"

	"github.com/RichardKnop/ragserver"
)

type item struct {
	Left       float64 `json:"left"`
	Top        float64 `json:"top"`
	Width      float64 `json:"width"`
	Height     float64 `json:"height"`
	PageNumber int     `json:"page_number"`
	PageWidth  float64 `json:"page_width"`
	PageHeight float64 `json:"page_height"`
	Text       string  `json:"text"`
	Type       string  `json:"type"`
}

//	curl -X POST \
//	  -F 'file=@/Users/richardknop/Desktop/Statement on Emissions.pdf' \
//	  -F 'fast=true' \
//	  -F 'types=all' \
//	  http://localhost:5060
func (a *Adapter) Extract(ctx context.Context, fileName string, contents io.ReadSeeker, topics ragserver.RelevantTopics) ([]ragserver.Document, error) {
	items, err := a.extractItems(ctx, fileName, contents)
	if err != nil {
		return nil, err
	}

	// TODO - extract tables too

	var (
		// Create the default sentence tokenizer
		tokenizer  = sentences.NewSentenceTokenizer(a.training)
		documents  = make([]ragserver.Document, 0, 100)
		topicCount = map[string]int{}
	)

	for _, anItem := range items {
		if anItem.Type != "Text" && anItem.Type != "Footnote" && anItem.Type != "List item" && anItem.Type != "Table" {
			continue
		}

		for _, aSentence := range tokenizer.Tokenize(anItem.Text) {
			if len(topics) > 0 {
				aTopic, ok := topics.IsRelevant(aSentence.Text)
				if !ok {
					continue
				}
				if aTopic.Name != "" {
					_, ok := topicCount[aTopic.Name]
					if !ok {
						topicCount[aTopic.Name] = 0
					}
					topicCount[aTopic.Name] += 1
				}
			}

			documents = append(documents, ragserver.Document{
				Content: strings.TrimSpace(aSentence.Text),
				Page:    anItem.PageNumber,
			})
		}
	}

	for name, count := range topicCount {
		a.logger.Sugar().Infof("%s relevant sentences: %d", name, count)
	}

	a.logger.Sugar().Infof("number of documents: %d", len(documents))

	return documents, nil
}

func (a *Adapter) extractItems(ctx context.Context, fileName string, contents io.ReadSeeker) ([]item, error) {
	buf := new(bytes.Buffer)
	writer := multipart.NewWriter(buf)
	defer writer.Close()

	part, err := writer.CreateFormFile("file", fileName)
	if err != nil {
		return nil, err
	}
	_, err = io.Copy(part, contents)
	if err != nil {
		return nil, err
	}

	if err := writer.WriteField("fast", "true"); err != nil {
		return nil, err
	}
	if err := writer.WriteField("types", "text,list item"); err != nil {
		return nil, err
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", a.baseURL, buf)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		return nil, errors.New(string(respData))
	}

	items := []item{}
	if err := json.Unmarshal(respData, &items); err != nil {
		return nil, err
	}

	return items, nil
}

func (a *Adapter) extractHTMLTables(ctx context.Context, fileName string, contents io.ReadSeeker) ([]Table, error) {
	buf := new(bytes.Buffer)
	writer := multipart.NewWriter(buf)
	defer writer.Close()

	part, err := writer.CreateFormFile("file", fileName)
	if err != nil {
		return nil, err
	}
	_, err = io.Copy(part, contents)
	if err != nil {
		return nil, err
	}

	if err := writer.WriteField("fast", "true"); err != nil {
		return nil, err
	}
	if err := writer.WriteField("types", "table"); err != nil {
		return nil, err
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", a.baseURL+"/html", buf)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		respData, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		return nil, errors.New(string(respData))
	}

	return parseTables(a.logger, resp.Body)
}
