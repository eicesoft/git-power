package config

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestSettingsRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")

	// 设置文件不存在时应返回默认设置。
	defaults, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if defaults.RecentLimit != 10 || defaults.DiffFontSize != 12 || !defaults.ConfirmDiscard {
		t.Fatalf("unexpected default settings: %+v", defaults)
	}

	saved := Settings{
		ScanDirectories: []string{"/tmp/workspace", "/tmp/other"},
		DiffFontFamily:  "JetBrains Mono",
		DiffFontSize:    14,
		RecentLimit:     20,
		ConfirmDiscard:  false,
		GitPath:         "/usr/local/bin/git",
		AgentBaseURL:    "https://api.example.com/v1",
		AgentAPIKey:     "sk-test",
		AgentModel:      "example-model",
	}
	if err := Save(path, saved); err != nil {
		t.Fatal(err)
	}

	reloaded, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(reloaded, saved) {
		t.Fatalf("unexpected reloaded settings: %+v", reloaded)
	}
}

func TestLoadSettingsKeepsDefaultsForMissingFields(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	if err := os.WriteFile(path, []byte(`{"diffFontSize": 15, "confirmDiscard": false}`), 0o600); err != nil {
		t.Fatal(err)
	}

	settings, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if settings.DiffFontSize != 15 || settings.ConfirmDiscard {
		t.Fatalf("unexpected settings: %+v", settings)
	}
	// 文件中未出现的字段应保留默认值。
	if settings.RecentLimit != 10 || len(settings.ScanDirectories) != 0 {
		t.Fatalf("expected defaults for missing fields, got %+v", settings)
	}
}
