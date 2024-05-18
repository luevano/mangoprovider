package mangaplus

import (
	"context"
	"encoding/hex"
	"net/http"
	"net/url"

	mango "github.com/luevano/mangoprovider"
)

func (p *plus) GetPageImage(ctx context.Context, client *http.Client, page mango.Page) ([]byte, error) {
	// Everything is the same, except that the image needs to be decoded at the end.
	image, err := mango.GenericGetPageImage(ctx, client, page)
	if err != nil {
		return nil, err
	}

	url, err := url.Parse(page.URL)
	if err != nil {
		return nil, err
	}
	key := url.Fragment
	if key == "" {
		return image, nil
	}
	decodeXorCipher(&image, key)
	return image, nil
}

// Only catch here is that the key is encoded in hex
func decodeXorCipher(data *[]byte, key string) {
	keyBytes, err := hex.DecodeString(key)
	if err != nil {
		mango.Log("Error while decoding encryption key for image.")
		return
	}
	keyLen := len(keyBytes)

	for i, byte := range *data {
		(*data)[i] = byte ^ keyBytes[i%keyLen]
	}
}
