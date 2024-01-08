package mango

import (
	"regexp"

	"github.com/luevano/libmangal"
)

var ImageExtensionRegex = regexp.MustCompile(`^\.[a-zA-Z0-9][a-zA-Z0-9.]*[a-zA-Z0-9]$`)

var _ libmangal.Page = (*MangoPage)(nil)

type MangoPage struct {
	Extension string            `json:"extension"`
	URL       string            `json:"url"`
	Headers   map[string]string `json:"headers"`
	Cookies   map[string]string `json:"cookies"`

	Chapter_ *MangoChapter
}

func (p MangoPage) String() string {
	return p.URL
}

func (p MangoPage) GetExtension() string {
	return p.Extension
}

func (p MangoPage) Chapter() libmangal.Chapter {
	return p.Chapter_
}
