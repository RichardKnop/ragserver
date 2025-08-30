package pdf

import (
	"context"
	"io"
	"log"
	"strings"

	"github.com/neurosnap/sentences"

	"github.com/RichardKnop/ragserver"
)

func (a *Adapter) Extract(ctx context.Context, tempFile io.ReadSeeker, topics ragserver.RelevantTopics) ([]ragserver.Document, error) {
	pageBytes, numPages, err := a.extractor.extractText(tempFile)
	if err != nil {
		return nil, err
	}

	log.Printf("extracted text from PDF file, pages: %d", numPages)

	var (
		// Create the default sentence tokenizer
		tokenizer  = sentences.NewSentenceTokenizer(a.training)
		documents  = make([]ragserver.Document, 0, 100)
		numTables  int
		topicCount = map[string]int{}
	)

	for i, page := range pageBytes {
		pageNum := i + 1
		log.Printf("processing page %d/%d", pageNum, numPages)

		// // Just saving the text to a file for debugging purposes
		// f, err := os.Create(fmt.Sprintf("extracted_text_page_%d.txt", pageNum))
		// if err != nil {
		// 	return nil, fmt.Errorf("error extracting text: %w", err)
		// }
		// defer f.Close()
		// _, err = f.Write(page.Bytes())
		// if err != nil {
		// 	return nil, fmt.Errorf("error extracting text: %w", err)
		// }

		for _, aSentence := range tokenizer.Tokenize(page.String()) {
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

			// In case of scope-related sentence, we want to first try to extract yearly scope tables,
			// to get better context for the LLM. These are tables with years as columns and categories
			// as rows, with numeric values for each year.
			tables, err := NewTables(aSentence.Text)
			if err != nil {
				documents = append(documents, ragserver.Document{
					Content: strings.TrimSpace(aSentence.Text),
					Page:    i + 1,
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
							Content: aContext,
							Page:    i + 1,
						})
					}
				}
			} else {
				documents = append(documents, ragserver.Document{
					Content: strings.TrimSpace(aSentence.Text),
					Page:    i + 1,
				})
			}
		}
	}

	for name, count := range topicCount {
		log.Printf("%s relevant sentences: %d", name, count)
	}

	log.Printf("number of documents: %d", len(documents))

	return documents, nil
}
