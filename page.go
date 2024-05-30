package mangoprovider

import "github.com/luevano/libmangal/mangadata"

var _ mangadata.Page = (*Page)(nil)

type Page struct {
	Ext     string            `json:"-"`
	URL     string            `json:"url"`
	Headers map[string]string `json:"-"`
	Cookies map[string]string `json:"-"`

	Chapter_ *Chapter `json:"-"`
}

func (p *Page) String() string {
	return p.URL
}

func (p *Page) Extension() string {
	return p.Ext
}

func (p *Page) Chapter() mangadata.Chapter {
	return p.Chapter_
}
