package flaresolverr

// TODO: once mangoprovider implements tests this should also be translated from belphemur's code
/*
import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/belphemur/mangal/key"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/spf13/viper"
)

func init() {
	viper.Set(key.SourceHeadlessUseFlaresolverr, true)
	viper.Set(key.SourceHeadlessFlaresolverrURL, "http://localhost:8191/v1")
}

func TestTransportFlaresolevrr(t *testing.T) {
	transport := NewTransport()

	client := http.Client{
		Transport: transport,
	}
	defer client.CloseIdleConnections()
	Convey("Flaresolverr can go on NowSecure", t, func() {
		ctx, _ := context.WithTimeout(context.Background(), 120*time.Second)
		req, err := http.NewRequestWithContext(ctx, "GET", "https://nowsecure.nl/", nil)
		So(err, ShouldBeNil)

		resp, err := client.Do(req)
		So(err, ShouldBeNil)
		defer resp.Body.Close()
		So(resp.StatusCode, ShouldEqual, 200)

		Convey("Flaresolverr session should be destroyed when finished", func() {
			err := transport.Close()
			So(err, ShouldBeNil)
		})
	})
}
*/
