package main

import (
	"bufio"
	_ "embed"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"strings"

	"github.com/neurosnap/sentences"
)

const (
	MB          = 1 << 20
	MaxFileSize = 20 * MB
)

func (rs *ragServer) uploadFileHandler(w http.ResponseWriter, req *http.Request) {
	// Limit memory usage to 20MB, anythin over this limit will be stored in a temporary file.
	req.ParseMultipartForm(MaxFileSize)

	// Limit the size of the request body to prevent large uploads. This will return
	// io.MaxBytesError if the request body exceeds the limit while being read.
	req.Body = http.MaxBytesReader(w, req.Body, MaxFileSize)

	tempFile, err := os.CreateTemp("", "file*")
	if err != nil {
		log.Printf("error creating temp file: %v", err)
		http.Error(w, "file upload error", http.StatusInternalServerError)
	}

	_, contentType, err := rs.writeUploadedFileToTempFile(req, tempFile)
	if err != nil {
		log.Printf("error writing uploaded file to temp file: %v", err)
		http.Error(w, "file upload error", http.StatusInternalServerError)
		return
	}
	defer tempFile.Close()

	switch contentType {
	case "application/pdf":
		extractor := &extractor{
			pageMin:         1,
			pageMax:         1000,
			xRangeMin:       math.Inf(-1),
			xRangeMax:       math.Inf(1),
			showPageNumbers: false,
		}
		text, numPages, err := extractor.extractText(tempFile)
		if err != nil {
			http.Error(w, "error extracting text", http.StatusInternalServerError)
			return
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
		tokenizer := sentences.NewSentenceTokenizer(rs.training)
		sentences := tokenizer.Tokenize(string(text))
		contexts := make([]string, 0, len(sentences))

		log.Printf("tokenized sentences: %d", len(sentences))

		var (
			numTables       int
			scopeRelevant   int
			netZeroRelevant int
		)
		for _, aSentence := range sentences {
			var (
				scopeRelated   = isScopeRelated(aSentence.Text)
				netZeroRelated = isNetZeroRelated(aSentence.Text)
			)
			if !scopeRelated && !netZeroRelated {
				continue
			}

			if netZeroRelated {
				netZeroRelevant += 1
				contexts = append(contexts, aSentence.Text)
				continue
			}

			scopeRelevant += 1

			// In case of scope-related sentence, we want to first try to extract yearly scope tables,
			// to get better context for the LLM. These are tables with years as columns and categories
			// as rows, with numeric values for each year.
			tables, err := NewTables(aSentence.Text)
			if err != nil {
				contexts = append(contexts, aSentence.Text)
				continue
			}

			if len(tables) > 0 {
				numTables += len(tables)
				for _, aTable := range tables {
					tableContexts := aTable.ToContexts()
					log.Printf("table title: %s, contexts: %d", aTable.Title, len(tableContexts))
					contexts = append(contexts, tableContexts...)
				}
			} else {
				contexts = append(contexts, aSentence.Text)
			}
		}

		log.Printf("scope relevant sentences: %d", scopeRelevant)
		log.Printf("net zero relevant sentences: %d", netZeroRelevant)
		log.Printf("number of contexts: %d", len(contexts))

		// TODO - create a file entry in the database
		// - generate embeddings and store them in vector DB with link to the file
		// return file ID in the response so it can be used later for queries

	case "image/jpeg", "image/png":
		// client := gosseract.NewClient()
		// defer client.Close()

		// if err := client.SetImageFromBytes(fileBytes); err != nil {
		// 	log.Printf("client.SetImageFromBytes error: %v", err.Error())
		// 	http.Error(w, "file processing error", http.StatusInternalServerError)
		// }

		// text, err := client.Text()
		// if err != nil {
		// 	log.Printf("client.Text error: %v", err.Error())
		// 	http.Error(w, "file processing error", http.StatusInternalServerError)
		// 	return
		// }

		// log.Printf("file processed, text: %v", text)
	}

	w.WriteHeader(http.StatusCreated)
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

func (rs *ragServer) writeUploadedFileToTempFile(req *http.Request, tempFile io.Writer) (int64, string, error) {
	file, handler, err := req.FormFile("file")
	if err != nil {
		return 0, "", fmt.Errorf("error retrieving the file: %w", err)
	}
	defer file.Close()

	contentType, ok, err := checkContentType(file)
	if err != nil {
		return 0, "", fmt.Errorf("error checking content type: %w", err)
	}
	if !ok {
		return 0, "", fmt.Errorf("invalid file type")
	}

	// Reset the temp file offset to the beginning for further reading
	_, err = file.Seek(0, io.SeekStart)
	if err != nil {
		return 0, "", fmt.Errorf("error seeking temp file to start: %w", err)
	}

	log.Printf("uploading file: %s, size: %d, mime header: %v", handler.Filename, handler.Size, handler.Header)

	fileSize, err := io.Copy(bufio.NewWriter(tempFile), file)
	if err != nil {
		return 0, "", fmt.Errorf("error copying to temp file: %w", err)
	}

	return fileSize, contentType, nil
}

var allowedContentTypes = map[string]struct{}{
	"application/pdf": {},
	// "image/jpeg":      {},
	// "image/png":       {},
	// "image/gif":       {},
}

func checkContentType(reader io.Reader) (string, bool, error) {
	contentType, err := detectContentType(reader)
	if err != nil {
		return "", false, err
	}
	_, ok := allowedContentTypes[contentType]
	return contentType, ok, nil
}

func detectContentType(reader io.Reader) (string, error) {
	// At most the first 512 bytes of data are used:
	// https://golang.org/src/net/http/sniff.go?s=646:688#L11
	buff := make([]byte, 512)

	bytesRead, err := reader.Read(buff)
	if err != nil && err != io.EOF {
		return "", err
	}

	// Slice to remove fill-up zero values which cause a wrong content type detection in the next step
	// (for example a text file which is smaller than 512 bytes)
	buff = buff[:bytesRead]

	return http.DetectContentType(buff), nil
}
