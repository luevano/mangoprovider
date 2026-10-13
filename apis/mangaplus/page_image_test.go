package mangaplus

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/luevano/libmangal/metadata"
	"github.com/luevano/mangoplus"
	mango "github.com/luevano/mangoprovider"
)

func TestDecodeXorCipher(t *testing.T) {
	data := []byte{0x00, 0xff, 0x0f, 0xf0, 0x12}
	if err := decodeXorCipher(data, "0f0f"); err != nil {
		t.Fatal(err)
	}
	// the key repeats over the data
	if want := []byte{0x0f, 0xf0, 0x00, 0xff, 0x1d}; !bytes.Equal(data, want) {
		t.Errorf("wanted %x, got %x", want, data)
	}
	if err := decodeXorCipher(data, "not hex"); err == nil {
		t.Error("wanted an error for a key that isn't hex")
	}
}

func TestGetPageImage(t *testing.T) {
	image := []byte("\x89PNG fake image data")
	key := []byte{0xde, 0xad, 0xbe, 0xef}
	encrypted := make([]byte, len(image))
	for i, b := range image {
		encrypted[i] = b ^ key[i%len(key)]
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Referer") != "https://ref" {
			t.Errorf("the page headers weren't sent: %v", r.Header)
		}
		if r.URL.Path == "/plain.png" {
			w.Write(image)
			return
		}
		w.Write(encrypted)
	}))
	defer server.Close()

	p := &plus{}
	headers := map[string]string{"Referer": "https://ref"}
	got, err := p.GetPageImage(context.Background(), server.Client(), mango.Page{URL: server.URL + "/1.png#deadbeef", Headers: headers})
	if err != nil || !bytes.Equal(got, image) {
		t.Errorf("encrypted: wanted %q, got %q, %v", image, got, err)
	}
	// no key, as is
	got, err = p.GetPageImage(context.Background(), server.Client(), mango.Page{URL: server.URL + "/plain.png", Headers: headers})
	if err != nil || !bytes.Equal(got, image) {
		t.Errorf("plain: wanted %q, got %q, %v", image, got, err)
	}
	if _, err := p.GetPageImage(context.Background(), server.Client(), mango.Page{URL: server.URL + "/1.png#zz", Headers: headers}); err == nil {
		t.Error("wanted an error for a bad key")
	}
}

func TestPlusToMetadata(t *testing.T) {
	detail := &mangoplus.TitleDetailView{
		Title:         &mangoplus.Title{TitleId: 100020, Name: "One Piece", Author: "Eiichiro Oda", PortraitImageUrl: "https://img/portrait.jpg"},
		TitleImageUrl: "https://img/banner.jpg",
		Overview:      "Pirates.",
		TitleLabels:   &mangoplus.TitleLabels{ReleaseSchedule: mangoplus.ReleaseSchedule_WEEKLY},
		// noon utc, the same day anywhere around the americas and europe
		ChapterListV2: []*mangoplus.Chapter{{StartTimeStamp: 1611748800}},
	}
	meta := (&plus{}).plusToMetadata(detail)
	if meta.Title() != "One Piece" || meta.Summary != "Pirates." || meta.Cover() != "https://img/portrait.jpg" ||
		meta.Banner() != "https://img/banner.jpg" || meta.URL() != website+"titles/100020" || meta.ProviderID != "100020" ||
		meta.ProviderIDCode != "mp" || len(meta.AuthorList) != 1 || meta.AuthorList[0] != "Eiichiro Oda" {
		t.Errorf("got %+v", meta)
	}
	if meta.StartDate().String() != "2021-01-27" || meta.Status() != metadata.StatusReleasing {
		t.Errorf("wanted 2021-01-27, releasing; got %v, %v", meta.StartDate(), meta.Status())
	}

	for schedule, want := range map[mangoplus.ReleaseSchedule]metadata.Status{
		mangoplus.ReleaseSchedule_COMPLETED: metadata.StatusFinished,
		mangoplus.ReleaseSchedule_ONE_SHOT:  metadata.StatusFinished,
		mangoplus.ReleaseSchedule_HIATUS:    metadata.StatusHiatus,
		mangoplus.ReleaseSchedule_DISABLED:  metadata.StatusHiatus,
	} {
		detail.TitleLabels.ReleaseSchedule = schedule
		if got := (&plus{}).plusToMetadata(detail).Status(); got != want {
			t.Errorf("%s: wanted %s, got %s", schedule, want, got)
		}
	}

	// no chapters, no date
	if got := (&plus{}).plusToMetadata(&mangoplus.TitleDetailView{Title: &mangoplus.Title{TitleId: 1}}).StartDate(); got.Year != 0 {
		t.Errorf("wanted no start date, got %v", got)
	}
}
