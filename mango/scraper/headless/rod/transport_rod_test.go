package rod

// TODO: once mangoprovider implements tests this should also be translated from belphemur's code
/*
import (
	"net/http"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestTransportRod(t *testing.T) {
	transport := NewTransport()

	client := http.Client{
		Transport: transport,
	}
	defer client.CloseIdleConnections()
	Convey("Rod can go on google", t, func() {
		req, err := http.NewRequest("GET", "https://www.google.com", nil)
		So(err, ShouldBeNil)

		resp, err := client.Do(req)
		So(err, ShouldBeNil)
		defer resp.Body.Close()
		So(resp.StatusCode, ShouldEqual, 200)

		Convey("Rod should be closed when finished", func() {
			err := transport.Close()
			So(err, ShouldBeNil)
		})
	})
}
*/
