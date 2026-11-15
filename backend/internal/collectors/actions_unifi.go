package collectors

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// unifiActor sends legacy controller commands (cmd/devmgr, cmd/stamgr) with
// the collector's API key and HTTP client settings.
type unifiActor struct {
	col *UnifiCollector
}

// unifiCommand maps an action to its legacy command manager and command.
func unifiCommand(action string) (manager, cmd string, ok bool) {
	switch action {
	case "restart_device":
		return "devmgr", "restart", true
	case "kick":
		return "stamgr", "kick-sta", true
	case "block":
		return "stamgr", "block-sta", true
	case "unblock":
		return "stamgr", "unblock-sta", true
	}
	return "", "", false
}

func (u *unifiActor) Run(ctx context.Context, a Action) (string, error) {
	manager, cmd, ok := unifiCommand(a.Name)
	if !ok {
		return "", fmt.Errorf("unsupported unifi action %q", a.Name)
	}
	body, err := json.Marshal(map[string]string{"cmd": cmd, "mac": a.Target})
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.col.legacyURL("cmd/"+manager), bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("X-API-Key", u.col.cfg.APIKey)
	req.Header.Set("Content-Type", "application/json")
	resp, err := u.col.newClient().Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var out struct {
		Meta struct {
			RC  string `json:"rc"`
			Msg string `json:"msg"`
		} `json:"meta"`
	}
	decodeErr := json.NewDecoder(resp.Body).Decode(&out)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 || out.Meta.RC != "ok" {
		msg := out.Meta.Msg
		if msg == "" && decodeErr != nil {
			msg = decodeErr.Error()
		}
		return "", fmt.Errorf("UniFi %s %s failed (%s): %s", manager, cmd, resp.Status, msg)
	}
	return "", nil
}
