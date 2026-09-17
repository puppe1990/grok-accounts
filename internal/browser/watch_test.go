package browser

import (
	"bytes"
	"io"
	"strings"
	"testing"
)

const deviceAuthOutput = `To sign in, open this URL in your browser:

  https://accounts.x.ai/oauth2/device?user_code=PBB9-333T

Confirm this code in your browser:

  PBB9-333T

Waiting for authorization...
`

func collectOpened() (*[]string, func(string)) {
	var opened []string
	return &opened, func(url string) { opened = append(opened, url) }
}

func TestWatchURLsCopiesStreamAndOpensFirstURL(t *testing.T) {
	var dst bytes.Buffer
	opened, open := collectOpened()

	err := WatchURLs(strings.NewReader(deviceAuthOutput), &dst, open)

	if err != nil {
		t.Fatalf("WatchURLs() error = %v, want nil", err)
	}
	if dst.String() != deviceAuthOutput {
		t.Errorf("stream was not copied through:\n%q", dst.String())
	}
	want := "https://accounts.x.ai/oauth2/device?user_code=PBB9-333T"
	if len(*opened) != 1 || (*opened)[0] != want {
		t.Errorf("opened = %v, want [%q]", *opened, want)
	}
}

func TestWatchURLsOpensOnlyOnceWhenURLRepeats(t *testing.T) {
	stream := "visit https://example.com/one now\nand again https://example.com/two\n"
	opened, open := collectOpened()

	if err := WatchURLs(strings.NewReader(stream), io.Discard, open); err != nil {
		t.Fatalf("WatchURLs() error = %v, want nil", err)
	}
	if len(*opened) != 1 || (*opened)[0] != "https://example.com/one" {
		t.Errorf("opened = %v, want only the first URL", *opened)
	}
}

func TestWatchURLsStaysQuietWithoutURL(t *testing.T) {
	opened, open := collectOpened()

	if err := WatchURLs(strings.NewReader("Waiting for authorization...\n"), io.Discard, open); err != nil {
		t.Fatalf("WatchURLs() error = %v, want nil", err)
	}
	if len(*opened) != 0 {
		t.Errorf("opened = %v, want none", *opened)
	}
}

func TestWatchURLsTrimsTrailingProsePunctuation(t *testing.T) {
	opened, open := collectOpened()

	if err := WatchURLs(strings.NewReader("see https://accounts.x.ai/device.\n"), io.Discard, open); err != nil {
		t.Fatalf("WatchURLs() error = %v, want nil", err)
	}
	if len(*opened) != 1 || (*opened)[0] != "https://accounts.x.ai/device" {
		t.Errorf("opened = %v, want the URL without the trailing period", *opened)
	}
}

func TestWatchURLsHandlesFinalLineWithoutNewline(t *testing.T) {
	opened, open := collectOpened()

	if err := WatchURLs(strings.NewReader("https://example.com/device"), io.Discard, open); err != nil {
		t.Fatalf("WatchURLs() error = %v, want nil", err)
	}
	if len(*opened) != 1 || (*opened)[0] != "https://example.com/device" {
		t.Errorf("opened = %v, want the URL of the last line", *opened)
	}
}
