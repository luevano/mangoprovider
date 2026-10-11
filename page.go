package mangoprovider

import "github.com/luevano/libmangal/mangadata"

var _ mangadata.Page = (*Page)(nil)

type Page struct {
	Ext     string            `json:"-"`
	URL     string            `json:"url"`
	Headers map[string]string `json:"-"`
	Cookies map[string]string `json:"-"`
	// Fallbacks: other urls of the same image (e.g. on other cdns), for the provider's
	// GetPageImage to try when URL fails; GenericGetPageImage ignores them.
	Fallbacks []string `json:"-"`

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
