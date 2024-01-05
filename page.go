package mangoprovider

import (
	"regexp"

	"github.com/luevano/libmangal"
)

var fileExtensionRegex = regexp.MustCompile(`^\.[a-zA-Z0-9][a-zA-Z0-9.]*[a-zA-Z0-9]$`)

var _ libmangal.Page = (*mangoPage)(nil)

type mangoPage struct {
	Extension string            `json:"extension"`
	URL       string            `json:"url"`
	Headers   map[string]string `json:"headers"`
	Cookies   map[string]string `json:"cookies"`

	chapter *mangoChapter
}

func (p mangoPage) String() string {
	return p.URL
}

func (p mangoPage) GetExtension() string {
	return p.Extension
}

func (p mangoPage) Chapter() libmangal.Chapter {
	return p.chapter
}
