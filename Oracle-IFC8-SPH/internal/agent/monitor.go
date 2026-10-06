package agent

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"time"
)

type health struct{ status, reason string }

var monItems = regexp.MustCompile(`(?is)<MonItem\b[^>]*>[^<]*</MonItem\s*>`)

func logStates(content []byte) map[string]string {
	states := make(map[string]string)
	for _, fragment := range monItems.FindAll(content, -1) {
		var item struct {
			Object string `xml:"ObjType,attr"`
			Type   string `xml:"Type,attr"`
			Value  string `xml:",chardata"`
		}
		if xml.Unmarshal(fragment, &item) != nil {
			continue
		}
		object, kind := strings.ToLower(item.Object), strings.ToLower(item.Type)
		if (object == "ifc" || object == "pms") && (kind == "statelink" || kind == "statecomm") {
			states[object+"."+strings.TrimPrefix(kind, "state")] = strings.TrimSpace(item.Value)
		}
	}
	return states
}

func collectHealth(ctx context.Context, cfg config) health {
	running, err := processRunning(ctx, cfg.configPath)
	return inspectHealth(cfg, running, err, time.Now())
}

func inspectHealth(cfg config, running bool, processErr error, now time.Time) health {
	if processErr != nil {
		return health{"INDETERMINADO", "Nao foi possivel consultar os processos do IFC8."}
	}
	if !running {
		return health{"OFFLINE", "Processo IFC8 da instancia configurada ausente."}
	}
	file, err := os.Open(cfg.logPath)
	if err != nil {
		return health{"INDETERMINADO", "Nao foi possivel abrir o log IFC8."}
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() {
		return health{"INDETERMINADO", "Log IFC8 nao e um arquivo legivel."}
	}
	if cfg.maxLogAge > 0 && now.Sub(info.ModTime()) > cfg.maxLogAge {
		return health{"INDETERMINADO", "Log sem atualizacao dentro do prazo configurado."}
	}
	const maxLogBytes = 16 * 1024 * 1024
	content, err := io.ReadAll(io.LimitReader(file, maxLogBytes+1))
	if err != nil || len(content) > maxLogBytes {
		return health{"INDETERMINADO", "Log ilegivel ou maior que 16 MiB."}
	}
	return healthFromStates(logStates(content))
}

func healthFromStates(states map[string]string) health {
	checks := []struct{ key, expected string }{
		{"ifc.link", "Alive"}, {"ifc.comm", "Sync"}, {"pms.link", "Alive"}, {"pms.comm", "Sync"},
	}
	var missing, offline []string
	for _, check := range checks {
		value, present := states[check.key]
		if !present {
			missing = append(missing, check.key)
		} else if !strings.EqualFold(value, check.expected) {
			offline = append(offline, check.key+"="+value)
		}
	}
	if len(offline) > 0 {
		return health{"OFFLINE", strings.Join(offline, "; ")}
	}
	if len(missing) > 0 {
		return health{"INDETERMINADO", "Estados ausentes no log: " + strings.Join(missing, ", ")}
	}
	return health{"ONLINE", "IFC e PMS: Alive/Sync (ultimos estados registrados)."}
}

type alertState struct {
	status      string
	lastSuccess time.Time
}

func (state *alertState) notify(ctx context.Context, sample health, cfg config, hostname string, now time.Time, send func(context.Context, string) error) error {
	recovery := sample.status == "ONLINE" && state.status != "" && state.status != "ONLINE"
	problem := sample.status != "ONLINE" && (sample.status != state.status || (cfg.repeat > 0 && now.Sub(state.lastSuccess) >= cfg.repeat))
	if !problem && !recovery {
		return nil
	}
	reason := []rune(sample.reason)
	if len(reason) > 3000 {
		reason = reason[:3000]
	}
	message := fmt.Sprintf("IFC8 [%s]\nHost: %s\nHorario: %s\n%s", sample.status, hostname, now.Format("2006-01-02 15:04:05"), string(reason))
	if err := send(ctx, message); err != nil {
		return err
	}
	state.status, state.lastSuccess = sample.status, now
	return nil
}
