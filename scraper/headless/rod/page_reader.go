package rod

import (
	"io"
	"strings"

	"github.com/go-rod/rod"
)

var _ io.ReadCloser = (*pageReader)(nil)

type pageReader struct {
	page   *rod.Page
	reader io.Reader
}

func newPageReader(page *rod.Page) *pageReader {
	return &pageReader{
		reader: strings.NewReader(page.MustHTML()),
		page:   page,
	}
}

func (p pageReader) Close() error {
	return p.page.Close()
}

func (p pageReader) Read(buffer []byte) (n int, err error) {
	return p.reader.Read(buffer)
}
