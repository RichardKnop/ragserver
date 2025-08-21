package ragserver

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gofrs/uuid/v5"

	"github.com/RichardKnop/ragserver/internal/pkg/authz"
)

const (
	MB          = 1 << 20
	MaxFileSize = 20 * MB
)

type FileID struct{ uuid.UUID }

func NewFileID() FileID {
	return FileID{uuid.Must(uuid.NewV4())}
}

type File struct {
	ID        FileID
	FileName  string
	MimeType  string
	Extension string
	Size      int64
	Hash      string
	CreatedAt time.Time
	Documents []Document
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
		return nil, fmt.Errorf("error seeking file to start: %w", err)
	}

	log.Printf("uploading file: %s, size: %d, mime header: %v", header.Filename, header.Size, header.Header)

	hashWriter := sha256.New()
	newReader := io.TeeReader(file, hashWriter)
	fileSize, err := io.Copy(tempFile, newReader)
	if err != nil {
		return nil, fmt.Errorf("error copying to temp file: %w", err)
	}

	aFile := &File{
		ID:        NewFileID(),
		FileName:  header.Filename,
		MimeType:  contentType,
		Size:      fileSize,
		Hash:      hex.EncodeToString(hashWriter.Sum(nil)),
		CreatedAt: rs.now(),
	}

	_, err = tempFile.Seek(0, io.SeekStart) // Reset the temp file offset to the beginning for further reading
	if err != nil {
		return nil, fmt.Errorf("error seeking temp file to start: %w", err)
	}

	switch contentType {
	case "application/pdf":
		aFile.Extension = strings.TrimPrefix(contentType, "application/")

		var err error
		documents, err := rs.extract.Extract(ctx, tempFile, rs.relevantTopics)
		if err != nil {
			return nil, fmt.Errorf("error processing PDF file: %w", err)
		}
		for i := 0; i < len(documents); i++ {
			documents[i].FileID = aFile.ID
		}
		aFile.Documents = documents
	case "image/jpeg", "image/png":
		aFile.Extension = strings.TrimPrefix(contentType, "image/")

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
	vectors, err := rs.genai.EmbedDocuments(ctx, aFile.Documents)
	if err != nil {
		return nil, fmt.Errorf("error generating vectors: %v", err)
	}

	log.Printf("generated vectors: %d", len(vectors))

	if err := rs.store.Transactional(ctx, &sql.TxOptions{}, func(ctx context.Context) error {
		if err := rs.weaviate.SaveEmbeddings(ctx, aFile.Documents, vectors); err != nil {
			return fmt.Errorf("error saving embeddings: %v", err)
		}

		if err := rs.store.SaveFile(ctx, aFile); err != nil {
			return err
		}

		return nil
	}); err != nil {
		return nil, fmt.Errorf("error saving file: %v", err)
	}

	return aFile, nil
}

func (rs *ragServer) ListFiles(ctx context.Context, principal authz.Principal) ([]*File, error) {
	var files []*File
	if err := rs.store.Transactional(ctx, &sql.TxOptions{}, func(ctx context.Context) error {
		var err error
		files, err = rs.store.ListFiles(ctx)
		if err != nil {
			return err
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return files, nil
}

func (rs *ragServer) FindFile(ctx context.Context, principal authz.Principal, id FileID) (*File, error) {
	var file *File
	if err := rs.store.Transactional(ctx, &sql.TxOptions{}, func(ctx context.Context) error {
		var err error
		file, err = rs.store.FindFile(ctx, id)
		if err != nil {
			return err
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return file, nil
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
