package pdf

import (
	"context"
	"io"
	"log"
	"math"
	"strings"

	"github.com/neurosnap/sentences"

	"github.com/RichardKnop/ragserver/internal/ragserver/core/ragserver"
)

type Adapter struct {
	extractor *extractor
	training  *sentences.Storage
}

type Option func(*Adapter)

func New(training *sentences.Storage, options ...Option) *Adapter {
	a := &Adapter{
		extractor: &extractor{
			pageMin:         1,
			pageMax:         1000,
			xRangeMin:       math.Inf(-1),
			xRangeMax:       math.Inf(1),
			showPageNumbers: false,
		},
		training: training,
	}

	for _, o := range options {
		o(a)
	}

	return a
}

func (a *Adapter) Extract(ctx context.Context, tempFile io.ReadSeeker) ([]ragserver.Document, error) {
	pageBytes, numPages, err := a.extractor.extractText(tempFile)
	if err != nil {
		return nil, err
	}

	log.Printf("extracted text from PDF file, pages: %d", numPages)

	// Just saving the text to a file for now
	// f, err := os.Create("extracted_text.txt")
	// if err != nil {
	// 	log.Printf("error extracting text: %v", err)
	// 	http.Error(w, "error extracting text", http.StatusInternalServerError)
	// 	return
	// }
	// defer f.Close()
	// _, err = f.Write(text)
	// if err != nil {
	// 	log.Printf("error extracting text: %v", err)
	// 	http.Error(w, "error extracting text", http.StatusInternalServerError)
	// 	return
	// }

	// Create the default sentence tokenizer
	tokenizer := sentences.NewSentenceTokenizer(a.training)
	documents := make([]ragserver.Document, 0, 100)

	var (
		numTables       int
		scopeRelevant   int
		netZeroRelevant int
	)

	for i, page := range pageBytes {
		pageNum := i + 1
		log.Printf("processing page %d/%d", pageNum, numPages)
		for _, aSentence := range tokenizer.Tokenize(page.String()) {
			var (
				scopeRelated   = isScopeRelated(aSentence.Text)
				netZeroRelated = isNetZeroRelated(aSentence.Text)
			)
			if !scopeRelated && !netZeroRelated {
				continue
			}

			if netZeroRelated {
				netZeroRelevant += 1
				documents = append(documents, ragserver.Document{
					Text: aSentence.Text,
					Page: i + 1,
				})
				continue
			}

			scopeRelevant += 1

			// In case of scope-related sentence, we want to first try to extract yearly scope tables,
			// to get better context for the LLM. These are tables with years as columns and categories
			// as rows, with numeric values for each year.
			tables, err := NewTables(aSentence.Text)
			if err != nil {
				documents = append(documents, ragserver.Document{
					Text: aSentence.Text,
					Page: i + 1,
				})
				continue
			}

			if len(tables) > 0 {
				numTables += len(tables)
				for _, aTable := range tables {
					tableContexts := aTable.ToContexts()
					log.Printf("table title: %s, contexts: %d", aTable.Title, len(tableContexts))
					for _, aContext := range tableContexts {
						documents = append(documents, ragserver.Document{
							Text: aContext,
							Page: i + 1,
						})
					}
				}
			} else {
				documents = append(documents, ragserver.Document{
					Text: aSentence.Text,
					Page: i + 1,
				})
			}
		}
	}

	log.Printf("scope relevant sentences: %d", scopeRelevant)
	log.Printf("net zero relevant sentences: %d", netZeroRelevant)
	log.Printf("number of documents: %d", len(documents))

	return documents, nil
}

func isScopeRelated(s string) bool {
	if strings.Contains(strings.ToLower(s), "scope 1") {
		return true
	}
	if strings.Contains(strings.ToLower(s), "scope 2") {
		return true
	}
	if strings.Contains(strings.ToLower(s), "scope 3") {
		return true
	}
	return false
}

func isNetZeroRelated(s string) bool {
	if strings.Contains(strings.ToLower(s), "net-zero") {
		return true
	}
	if strings.Contains(strings.ToLower(s), "net zero") {
		return true
	}
	if strings.Contains(strings.ToLower(s), "net-zero target") {
		return true
	}
	if strings.Contains(strings.ToLower(s), "net zero target") {
		return true
	}
	return false
}
