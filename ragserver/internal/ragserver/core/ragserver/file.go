package ragserver

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gofrs/uuid/v5"

	"github.com/RichardKnop/ai/ragserver/internal/pkg/authz"
)

type FileID struct{ uuid.UUID }

func NewFileID() FileID {
	return FileID{uuid.Must(uuid.NewV4())}
}

type File struct {
	ID        FileID     `json:"id"`
	FileName  string     `json:"file_name"`
	MimeType  string     `json:"mime_type"`
	Extension string     `json:"extension"`
	Size      int64      `json:"size"`
	CreatedAt time.Time  `json:"created_at"`
	Documents []Document `json:"-"`
}

func (rs *ragServer) CreateFile(ctx context.Context, principal authz.Principal, file io.ReadSeeker, header *multipart.FileHeader) (*File, error) {
	tempFile, err := os.CreateTemp("", "file*")
	if err != nil {
		return nil, fmt.Errorf("error creating temp file: %v", err)
	}
	defer tempFile.Close()

	contentType, ok, err := checkContentType(file)
	if err != nil {
		return nil, fmt.Errorf("error checking content type: %w", err)
	}
	if !ok {
		return nil, fmt.Errorf("invalid file type")
	}

	// Reset the temp file offset to the beginning for further reading
	_, err = file.Seek(0, io.SeekStart)
	if err != nil {
		return nil, fmt.Errorf("error seeking temp file to start: %w", err)
	}

	log.Printf("uploading file: %s, size: %d, mime header: %v", header.Filename, header.Size, header.Header)

	fileSize, err := io.Copy(bufio.NewWriter(tempFile), file)
	if err != nil {
		return nil, fmt.Errorf("error copying to temp file: %w", err)
	}

	aFile := &File{
		ID:        NewFileID(),
		FileName:  header.Filename,
		MimeType:  contentType,
		Extension: strings.TrimPrefix(contentType, "image/"),
		Size:      fileSize,
		CreatedAt: rs.now(),
	}

	switch contentType {
	case "application/pdf":
		var err error
		aFile.Documents, err = rs.pdf.Extract(ctx, tempFile)
		if err != nil {
			return nil, fmt.Errorf("error processing PDF file: %w", err)
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

		return nil, fmt.Errorf("image file processing not implemented yet")
	}

	// Use the batch embedding API to embed all documents at once.
	vectors, err := rs.embedDocuments(ctx, aFile.Documents)
	if err != nil {
		return nil, fmt.Errorf("error generating vectors: %v", err)
	}

	log.Printf("generated vectors: %d", len(vectors))

	if err := rs.saveEmbeddings(ctx, aFile.Documents, vectors); err != nil {
		return aFile, fmt.Errorf("error saving embeddings: %v", err)
	}

	// TODO - save file in the database as well

	return aFile, nil
}

func (rs *ragServer) ExtractPDF(ctx context.Context, contents io.ReadSeeker) ([]Document, error) {
	return rs.pdf.Extract(ctx, contents)
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
