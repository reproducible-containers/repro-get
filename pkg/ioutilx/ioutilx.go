package ioutilx

import (
	"io"
	"os"
)

type catReader struct {
	io.Reader
	files []*os.File
}

func (cr *catReader) Close() error {
	var firstErr error
	for _, f := range cr.files {
		if err := f.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

func CatReader(filenames ...string) (io.ReadCloser, error) {
	var (
		cr      catReader
		readers []io.Reader
	)
	for _, name := range filenames {
		f, err := os.Open(name)
		if err != nil {
			cr.Close()
			return nil, err
		}
		cr.files = append(cr.files, f)
		readers = append(readers, f)
	}
	cr.Reader = io.MultiReader(readers...)
	return &cr, nil
}
