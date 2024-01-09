package mango

import (
	"regexp"

	"github.com/luevano/libmangal"
)

var ImageExtensionRegex = regexp.MustCompile(`^\.[a-zA-Z0-9][a-zA-Z0-9.]*[a-zA-Z0-9]$`)

var _ libmangal.Page = (*Page)(nil)

type Page struct {
	Extension string            `json:"extension"`
	URL       string            `json:"url"`
	Headers   map[string]string `json:"headers"`
	Cookies   map[string]string `json:"cookies"`

	Chapter_ *Chapter
}

func (p Page) String() string {
	return p.URL
}

func (p Page) GetExtension() string {
	return p.Extension
}

func (p Page) Chapter() libmangal.Chapter {
	return p.Chapter_
}
