package browser

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestByNameIsCaseInsensitive(t *testing.T) {
	b, err := ByName("brave browser")
	if err != nil {
		t.Fatalf("ByName() error = %v, want nil", err)
	}
	if b.Name != "Brave Browser" {
		t.Errorf("Name = %q, want %q", b.Name, "Brave Browser")
	}
	if b.PrivateArg != "--incognito" {
		t.Errorf("PrivateArg = %q, want %q", b.PrivateArg, "--incognito")
	}
}

func TestByNameRejectsUnknownBrowser(t *testing.T) {
	_, err := ByName("Netscape Navigator")
	if !errors.Is(err, ErrUnknownBrowser) {
		t.Fatalf("ByName() error = %v, want ErrUnknownBrowser", err)
	}
	if !strings.Contains(err.Error(), "Google Chrome") {
		t.Errorf("erro = %q, want the known browsers listed", err)
	}
}

func TestByBundleIDMapsPrivateArguments(t *testing.T) {
	firefox, err := ByBundleID("org.mozilla.firefox")
	if err != nil {
		t.Fatalf("ByBundleID(firefox) error = %v, want nil", err)
	}
	if firefox.PrivateArg != "-private-window" {
		t.Errorf("Firefox PrivateArg = %q, want %q", firefox.PrivateArg, "-private-window")
	}

	chrome, err := ByBundleID("com.google.Chrome")
	if err != nil {
		t.Fatalf("ByBundleID(chrome) error = %v, want nil", err)
	}
	if chrome.Name != "Google Chrome" {
		t.Errorf("Name = %q, want %q", chrome.Name, "Google Chrome")
	}
}

func TestDefaultBrowserPicksTheHTTPSHandler(t *testing.T) {
	raw := `{"LSHandlers":[
      {"LSHandlerURLScheme":"mailto","LSHandlerRoleAll":"com.apple.mail"},
      {"LSHandlerURLScheme":"https","LSHandlerRoleAll":"com.brave.browser"}
    ]}`

	b, err := DefaultBrowser([]byte(raw))
	if err != nil {
		t.Fatalf("DefaultBrowser() error = %v, want nil", err)
	}
	if b.Name != "Brave Browser" {
		t.Errorf("Name = %q, want %q", b.Name, "Brave Browser")
	}
}

func TestDefaultBrowserErrorsWithoutHTTPSHandler(t *testing.T) {
	raw := `{"LSHandlers":[{"LSHandlerURLScheme":"mailto","LSHandlerRoleAll":"com.apple.mail"}]}`

	if _, err := DefaultBrowser([]byte(raw)); !errors.Is(err, ErrUnknownBrowser) {
		t.Errorf("DefaultBrowser() error = %v, want ErrUnknownBrowser", err)
	}
}

func TestDefaultBrowserErrorsWhenDefaultIsNotSupported(t *testing.T) {
	raw := `{"LSHandlers":[{"LSHandlerURLScheme":"https","LSHandlerRoleAll":"com.apple.Safari"}]}`

	if _, err := DefaultBrowser([]byte(raw)); !errors.Is(err, ErrUnknownBrowser) {
		t.Errorf("DefaultBrowser() error = %v, want ErrUnknownBrowser for Safari", err)
	}
}

func TestDefaultBrowserErrorsOnInvalidJSON(t *testing.T) {
	if _, err := DefaultBrowser([]byte("{broken")); err == nil {
		t.Error("DefaultBrowser() error = nil, want an error")
	}
}

func TestOpenCommandLaunchesPrivateWindow(t *testing.T) {
	b, err := ByName("Google Chrome")
	if err != nil {
		t.Fatalf("ByName() error = %v", err)
	}

	got := OpenCommand(b, "https://accounts.x.ai/oauth2/device?user_code=ABCD-1234")
	want := []string{
		"open", "-na", "Google Chrome",
		"--args", "--incognito",
		"https://accounts.x.ai/oauth2/device?user_code=ABCD-1234",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("OpenCommand() = %v, want %v", got, want)
	}
}
