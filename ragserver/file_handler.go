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

		// sanitizedText := bytes.NewBuffer(nil)
		// r := bufio.NewReader(text)
		// for line, _, err := r.ReadLine(); err != io.EOF {

		// }

		// Just saving the text to a file for now
		f, err := os.Create("/Users/richardknop/Desktop/extracted_text.txt")
		if err != nil {
			http.Error(w, "error extracting text", http.StatusInternalServerError)
			return
		}
		defer f.Close()
		_, err = f.Write(text)
		if err != nil {
			http.Error(w, "error extracting text", http.StatusInternalServerError)
			return
		}

		// Create the default sentence tokenizer
		tokenizer := sentences.NewSentenceTokenizer(rs.training)
		sentences := tokenizer.Tokenize(string(text))
		contexts := make([]string, 0, len(sentences))

		for _, aSentence := range sentences {
			if !isRelevantSentence(aSentence.Text) {
				continue
			}
			// If we can parse the sentence as a table, do so and extract relevant contexts from it
			aTable, err := NewTable(aSentence.Text)
			if err == nil {
				tableContexts := aTable.ToContexts()
				for _, aContext := range tableContexts {
					log.Printf("tablecontext: %s", aContext)
				}
				contexts = append(contexts, tableContexts...)
				continue
			} else {
				log.Printf("could not parse sentence as table: %s : %v", aSentence.Text, err)
			}
			contexts = append(contexts, aSentence.Text)
		}

		for _, aContext := range contexts {
			log.Printf("context: %s", aContext)
		}

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

func isRelevantSentence(s string) bool {
	if strings.Contains(strings.ToLower(s), "scope 1") {
		return true
	}
	if strings.Contains(strings.ToLower(s), "scope 2") {
		return true
	}
	if strings.Contains(strings.ToLower(s), "scope 3") {
		return true
	}

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
	if strings.Contains(strings.ToLower(s), "absolute emissions") {
		return true
	}
	if strings.Contains(strings.ToLower(s), "emissions intensity") {
		return true
	}
	if strings.Contains(strings.ToLower(s), "ghg emissions") {
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
