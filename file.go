package ragserver

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"time"

	"github.com/gofrs/uuid/v5"

	"github.com/RichardKnop/ragserver/pkg/authz"
)

const (
	MB          = 1 << 20
	MaxFileSize = 20 * MB
)

type FileID struct{ uuid.UUID }

func NewFileID() FileID {
	return FileID{uuid.Must(uuid.NewV4())}
}

type AuthorID struct{ uuid.UUID }

func NewAuthorID() AuthorID {
	return AuthorID{uuid.Must(uuid.NewV4())}
}

type FileStatus string

const (
	FileStatusUploaded              FileStatus = "UPLOADED"
	FileStatusProcessing            FileStatus = "PROCESSING"
	FileStatusProcessedSuccessfully FileStatus = "PROCESSED_SUCCESSFULLY"
	FileStatusProcessingFailed      FileStatus = "PROCESSING_FAILED"
)

type File struct {
	ID            FileID
	AuthorID      AuthorID
	FileName      string
	ContentType   string
	Extension     string
	Size          int64
	Hash          string
	Embedder      string // adapter used to generate embeddings for this file
	Retriever     string // adapter used to store/retrieve embeddings for this file
	Status        FileStatus
	StatusMessage string
	Created       time.Time
	Updated       time.Time
	Documents     []Document
}

// CompleteWithStatus changes the status of a file to a completion status,
// either FileStatusProcessedSuccessfully or FileStatusProcessingFailed.
func (f *File) CompleteWithStatus(newStatus FileStatus, message string, updatedAt time.Time) error {
	if f.Status != FileStatusProcessing {
		return fmt.Errorf("cannot change status from %s to %s", f.Status, newStatus)
	}

	f.Status = newStatus
	f.StatusMessage = message
	f.Updated = updatedAt

	return nil
}

type FileFilter struct {
	Embedder          string
	Retriever         string
	Status            FileStatus
	LastUpdatedBefore time.Time
	ScreeningID       ScreeningID
	Lock              bool
}

type TempFile interface {
	io.ReadSeekCloser
	io.Writer
	Name() string
}

func (rs *ragServer) CreateFile(ctx context.Context, principal authz.Principal, file io.ReadSeeker, header *multipart.FileHeader) (*File, error) {
	tempFile, err := rs.filestorage.NewTempFile()
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

	rs.logger.Sugar().With("filename", header.Filename, "size", header.Size, "header", header.Header).Infof("uploading file")

	hashWriter := sha256.New()
	newReader := io.TeeReader(file, hashWriter)
	fileSize, err := io.Copy(tempFile, newReader)
	if err != nil {
		return nil, fmt.Errorf("error copying to temp file: %w", err)
	}

	fileHash := hex.EncodeToString(hashWriter.Sum(nil))

	exists, err := rs.filestorage.Exists(fileHash)
	if err != nil {
		return nil, fmt.Errorf("error checking if file exists: %w", err)
	}
	if !exists {
		if err := rs.filestorage.Write(fileHash, tempFile); err != nil {
			return nil, fmt.Errorf("error writing to file storage: %w", err)
		}
	}
	defer rs.filestorage.DeleteTempFile(tempFile.Name())

	aFile := &File{
		ID:          NewFileID(),
		AuthorID:    AuthorID{principal.ID().UUID},
		FileName:    header.Filename,
		ContentType: contentType,
		Size:        fileSize,
		Hash:        fileHash,
		Embedder:    rs.embedder.Name(),
		Retriever:   rs.retriever.Name(),
		Status:      FileStatusUploaded,
		Created:     rs.now(),
		Updated:     rs.now(),
	}

	switch contentType {
	case "application/pdf":
		aFile.Extension = strings.TrimPrefix(contentType, "application/")
	case "image/jpeg", "image/png":
		return nil, fmt.Errorf("image file processing not implemented yet")
	}

	if err := rs.store.Transactional(ctx, &sql.TxOptions{}, func(ctx context.Context) error {
		if err := rs.store.SavePrincipal(ctx, principal); err != nil {
			return fmt.Errorf("error saving principal: %w", err)
		}

		if err := rs.store.SaveFiles(ctx, aFile); err != nil {
			return fmt.Errorf("error saving file: %w", err)
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
		files, err = rs.store.ListFiles(ctx, FileFilter{}, rs.filePpartial(), SortParams{})
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
	var aFile *File
	if err := rs.store.Transactional(ctx, &sql.TxOptions{}, func(ctx context.Context) error {
		var err error
		aFile, err = rs.store.FindFile(ctx, id, rs.filePpartial())
		if err != nil {
			return err
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return aFile, nil
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
