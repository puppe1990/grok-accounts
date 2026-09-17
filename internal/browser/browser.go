package browser

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

var ErrUnknownBrowser = errors.New("navegador desconhecido")

type Browser struct {
	Name       string
	BundleID   string
	PrivateArg string
}

var known = []Browser{
	{Name: "Brave Browser", BundleID: "com.brave.browser", PrivateArg: "--incognito"},
	{Name: "Google Chrome", BundleID: "com.google.Chrome", PrivateArg: "--incognito"},
	{Name: "Google Chrome Canary", BundleID: "com.google.Chrome.canary", PrivateArg: "--incognito"},
	{Name: "Microsoft Edge", BundleID: "com.microsoft.edgemac", PrivateArg: "--inprivate"},
	{Name: "Firefox", BundleID: "org.mozilla.firefox", PrivateArg: "-private-window"},
	{Name: "Opera", BundleID: "com.operasoftware.Opera", PrivateArg: "--private"},
}

func ByName(name string) (Browser, error) {
	for _, b := range known {
		if strings.EqualFold(b.Name, strings.TrimSpace(name)) {
			return b, nil
		}
	}
	return Browser{}, fmt.Errorf("%w: %q (conhecidos: %s)", ErrUnknownBrowser, name, names())
}

func ByBundleID(id string) (Browser, error) {
	for _, b := range known {
		if b.BundleID == id {
			return b, nil
		}
	}
	return Browser{}, fmt.Errorf("%w: %q (conhecidos: %s)", ErrUnknownBrowser, id, names())
}

func DefaultBrowser(launchServicesJSON []byte) (Browser, error) {
	var plist struct {
		Handlers []struct {
			Scheme string `json:"LSHandlerURLScheme"`
			Role   string `json:"LSHandlerRoleAll"`
		} `json:"LSHandlers"`
	}
	if err := json.Unmarshal(launchServicesJSON, &plist); err != nil {
		return Browser{}, fmt.Errorf("LaunchServices ilegível: %w", err)
	}

	for _, h := range plist.Handlers {
		if !strings.EqualFold(h.Scheme, "https") || h.Role == "" {
			continue
		}
		return ByBundleID(h.Role)
	}
	return Browser{}, fmt.Errorf("%w: nenhum handler https na LaunchServices", ErrUnknownBrowser)
}

func OpenCommand(b Browser, url string) []string {
	return []string{"open", "-na", b.Name, "--args", b.PrivateArg, url}
}

func names() string {
	list := make([]string, 0, len(known))
	for _, b := range known {
		list = append(list, b.Name)
	}
	return strings.Join(list, ", ")
}
