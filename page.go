package mangoprovider

import (
	"regexp"

	"github.com/luevano/libmangal"
)

var ImageExtensionRegex = regexp.MustCompile(`^\.[a-zA-Z0-9][a-zA-Z0-9.]*[a-zA-Z0-9]$`)

var _ libmangal.Page = (*Page)(nil)

type Page struct {
	Extension string            `json:"-"`
	URL       string            `json:"url"`
	Headers   map[string]string `json:"-"`
	Cookies   map[string]string `json:"-"`

	Chapter_ *Chapter `json:"-"`
}

func (p *Page) String() string {
	return p.URL
}

func (p *Page) GetExtension() string {
	return p.Extension
}

func (p *Page) Chapter() libmangal.Chapter {
	return p.Chapter_
}
