package mangoprovider

import (
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/luevano/libmangal"
	"github.com/luevano/libmangal/logger"
)

// newTestLogger: libmangal logger that records the messages.
func newTestLogger() (*logger.Logger, *[]string) {
	var mu sync.Mutex
	var msgs []string
	l := logger.NewLogger()
	l.SetOnLog(func(format string, a ...any) {
		mu.Lock()
		defer mu.Unlock()
		msgs = append(msgs, fmt.Sprintf(format, a...))
	})
	return l, &msgs
}

func TestLoggerPerProvider(t *testing.T) {
	dex, plus := NewProviderLogger(libmangal.ProviderInfo{ID: "mango-mangadex"}), NewProviderLogger(libmangal.ProviderInfo{ID: "mango-mangaplus"})
	dexOut, dexMsgs := newTestLogger()
	plusOut, plusMsgs := newTestLogger()

	dex.Log("before the logger is set")
	dex.SetLogger(dexOut)
	plus.SetLogger(plusOut)
	dex.Log("search %q", "x")
	plus.Log("search %q", "y")

	got := strings.Join(*dexMsgs, "\n")
	if !strings.Contains(got, "[backlog] mango-mangadex: before the logger is set") || !strings.Contains(got, `mango-mangadex: search "x"`) {
		t.Errorf("mangadex logs: %q", got)
	}
	if len(*plusMsgs) != 1 || (*plusMsgs)[0] != `mango-mangaplus: search "y"` {
		t.Errorf("mangaplus logs: %q", *plusMsgs)
	}
}

func TestLoggerConcurrent(t *testing.T) {
	log := NewProviderLogger(libmangal.ProviderInfo{ID: "p"})
	out, msgs := newTestLogger()
	var wg sync.WaitGroup
	for i := range 20 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if i == 10 {
				log.SetLogger(out)
			}
			log.Log("message %d", i)
		}()
	}
	wg.Wait()
	if len(*msgs) != 20 {
		t.Errorf("wanted 20 messages, got %d", len(*msgs))
	}
}
