package batoto

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	openssl "github.com/Luzifer/go-openssl/v4"
	"github.com/PuerkitoBio/goquery"
	"github.com/luevano/libmangal"
	"github.com/luevano/libmangal/metadata"
	mango "github.com/luevano/mangoprovider"
	"github.com/luevano/mangoprovider/scraper"
	"github.com/tj/go-naturaldate"
	"github.com/vineesh12344/gojsfuck/jsfuck"
)

var Info = libmangal.ProviderInfo{
	ID:          mango.BundleID + "-batoto",
	Name:        "Batoto",
	Version:     "0.1.0",
	Description: "Batoto scraper",
	Website:     "https://bato.to/",
}

var Config = &scraper.Configuration{
	Name:    Info.ID,
	Delay:   50 * time.Millisecond,
	BaseURL: Info.Website,
	GenerateSearchURL: func(baseUrl string, query string) (string, error) {
		// path is /search?word==<query>
		params := url.Values{}
		params.Set("word", query)

		u, _ := url.Parse(baseUrl)
		u.Path = "search"
		u.RawQuery = params.Encode()

		return u.String(), nil
	},
	GenerateSearchByIDURL: func(baseUrl string, id string) (string, error) {
		return fmt.Sprintf("%sseries/%s", baseUrl, id), nil
	},
	// TODO: make these more specific? (add class identifiers)
	MangaByIDExtractor: &scraper.MangaByIDExtractor{
		Selector: "div.mainer > div.container-fluid",
		Title: func(selection *goquery.Selection) string {
			return selection.Find("div.title-set > h3.item-title > a").Text()
		},
		Cover: func(selection *goquery.Selection) string {
			return selection.Find("div.detail-set > div.attr-cover > img").AttrOr("src", "")
		},
	},
	MangaExtractor: &scraper.MangaExtractor{
		Selector: "div.mainer > div.container-fluid > div.series-list > div.item",
		Title: func(selection *goquery.Selection) string {
			return selection.Find("div.item-text > a.item-title").Text()
		},
		URL: func(selection *goquery.Selection) string {
			return selection.Find("div.item-text > a.item-title").AttrOr("href", "")
		},
		Cover: func(selection *goquery.Selection) string {
			return selection.Find("a.item-cover > img").AttrOr("src", "")
		},
		ID: func(_url string) string {
			return strings.Join(strings.Split(_url, "/")[4:], "/")
		},
	},
	VolumeExtractor: &scraper.VolumeExtractor{
		Selector: "body",
		Number: func(selection *goquery.Selection) float32 {
			return 1.0
		},
	},
	ChapterExtractor: &scraper.ChapterExtractor{
		Selector: "div.episode-list > div.main > div.item",
		Title: func(selection *goquery.Selection) string {
			chapNum := selection.Find("a").First().Find("b").Text()
			chapTitle := selection.Find("a").First().Find("span").Text()
			return chapNum + chapTitle
		},
		URL: func(selection *goquery.Selection) string {
			return selection.Find("a").First().AttrOr("href", "")
		},
		ID: func(_url string) string {
			return strings.Split(_url, "/")[4]
		},
		Date: func(selection *goquery.Selection) metadata.Date {
			publishedDate := selection.Find("div.extra > i").Text()
			now := time.Now()
			date, err := naturaldate.Parse(publishedDate, now)
			if err != nil {
				// if failed to parse date, use scraping day
				date = now
			}
			return metadata.Date{
				Year:  date.Year(),
				Month: int(date.Month()),
				Day:   date.Day(),
			}
		},
		ScanlationGroup: func(_ *goquery.Selection) string {
			return Info.Name
		},
	},
	PageExtractor: &scraper.PageExtractor{
		Selector: "body > script:not([src])",
		URLs: func(selection *goquery.Selection) []string {
			// get the correct script tag that contains the img contents
			var script string
			selection.Each(func(_ int, s *goquery.Selection) {
				scriptTemp := s.Text()
				if strings.Contains(scriptTemp, "imgHttps") {
					script = scriptTemp
					return
				}
			})
			if script == "" {
				return nil
			}

			// get the img urls and password
			var (
				imgHttps []string
				batoPass string
				batoWord string
			)
			for _, s := range strings.Split(script, "\n") {
				s = mango.CleanString(s)
				if strings.Contains(s, "imgHttps") {
					// TODO: use json.Unmarshal lol???
					imgHttpsRaw := strings.Split(s, " ")[3]
					imgHttpsRaw = imgHttpsRaw[1 : len(imgHttpsRaw)-2]
					imgHttps = strings.Split(imgHttpsRaw, ",")
					for i, img := range imgHttps {
						imgHttps[i] = img[1 : len(img)-1]
					}
				}
				if strings.Contains(s, "batoPass") {
					batoPass = strings.Split(s, " ")[3]
					batoPass = batoPass[:len(batoPass)-1]
				}
				if strings.Contains(s, "batoWord") {
					batoWord = strings.Split(s, " ")[3]
					batoWord = batoWord[1 : len(batoWord)-2]
				}
			}

			// decrypt the 'batoWord' (access to each img)
			jsFuck := jsfuck.New()
			jsFuck.Init()
			batoPass = jsFuck.Decode(batoPass)
			o := openssl.New()
			dec, err := o.DecryptBytes(batoPass, []byte(batoWord), openssl.BytesToKeyMD5)
			if err != nil {
				mango.Log("error decrypting img passwords: %s", err.Error())
				return imgHttps
			}
			// get as slice of string
			var imgAcc []string
			err = json.Unmarshal(dec, &imgAcc)
			if err != nil {
				mango.Log("error unmarshaling img passwords: %s", err.Error())
				return imgHttps
			}

			// add the password to the img url
			if len(imgAcc) != len(imgHttps) {
				mango.Log("wrong len of img passwords (%d) given len of img urls (%d)", len(imgAcc), len(imgHttps))
				return imgHttps
			}
			for i, img := range imgHttps {
				imgHttps[i] = img + "?" + imgAcc[i]
			}
			return imgHttps
		},
	},
}
