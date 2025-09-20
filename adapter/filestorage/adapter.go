package filestorage

import (
	"io"
	"os"

	"go.uber.org/zap"

	"github.com/RichardKnop/ragserver"
)

type Adapter struct {
	dir    string
	logger *zap.Logger
}

type Option func(*Adapter)

func WithDir(dir string) Option {
	return func(a *Adapter) {
		a.dir = dir
	}
}

func WithLogger(logger *zap.Logger) Option {
	return func(a *Adapter) {
		a.logger = logger
	}
}

func New(opts ...Option) (*Adapter, error) {
	a := &Adapter{
		dir:    os.TempDir(),
		logger: zap.NewNop(),
	}

	for _, o := range opts {
		o(a)
	}

	_, err := os.Stat(a.dir)
	if err != nil {
		return nil, err
	}

	a.logger.Sugar().With(
		"directory", a.dir,
	).Info("init filestorage adapter")

	return a, nil
}

func (a *Adapter) NewTempFile() (ragserver.TempFile, error) {
	return os.CreateTemp("", "file*")
}

func (a *Adapter) DeleteTempFile(name string) error {
	return os.Remove(name)
}

func (a *Adapter) Write(filename string, data io.Reader) error {
	f, err := os.Create(a.dir + "/" + filename)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.Copy(f, data)
	if err != nil {
		return err
	}

	return nil
}

func (a *Adapter) Exists(filename string) (bool, error) {
	_, err := os.Stat(a.dir + "/" + filename)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (a *Adapter) Read(filename string) (io.ReadSeekCloser, error) {
	return os.Open(a.dir + "/" + filename)
}

func (a *Adapter) Delete(filename string) error {
	return os.Remove(a.dir + "/" + filename)
}
