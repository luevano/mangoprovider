package flaresolverr

type response struct {
	Solution       solution `json:"solution"`
	Status         string   `json:"status"`
	Message        string   `json:"message"`
	StartTimestamp int64    `json:"startTimestamp"`
	EndTimestamp   int64    `json:"endTimestamp"`
	Version        string   `json:"version"`
}

type solution struct {
	URL       string            `json:"url"`
	Status    int               `json:"status"`
	Headers   map[string]string `json:"headers"`
	Response  string            `json:"response"`
	Cookies   []cookie          `json:"cookies"`
	UserAgent string            `json:"userAgent"`
}

type cookie struct {
	Name     string  `json:"name"`
	Value    string  `json:"value"`
	Domain   string  `json:"domain"`
	Path     string  `json:"path"`
	Expires  float64 `json:"expires"`
	Size     int     `json:"size"`
	HTTPOnly bool    `json:"httpOnly"`
	Secure   bool    `json:"secure"`
	Session  bool    `json:"session"`
	SameSite string  `json:"sameSite"`
}
