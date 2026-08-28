package main

import (
	"bytes"
	"errors"
	"flag"
	"io"
	"log/slog"
	"strings"
	"testing"
)

func TestParseConfig(t *testing.T) {
	tests := []struct {
		name    string
		env     map[string]string
		args    []string
		want    Config
		wantErr string
	}{
		{
			name: "an empty environment uses the defaults",
			want: Config{Host: "127.0.0.1", Port: "8080", LogLevel: slog.LevelInfo, Env: "dev"},
		},
		{
			name: "the environment sets the defaults",
			env:  map[string]string{"HOST": "0.0.0.0", "PORT": "9000", "LOG_LEVEL": "debug", "ENV": "prod"},
			want: Config{Host: "0.0.0.0", Port: "9000", LogLevel: slog.LevelDebug, Env: "prod"},
		},
		{
			name: "a flag beats its environment variable",
			env:  map[string]string{"PORT": "9000"},
			args: []string{"-port", "9100"},
			want: Config{Host: "127.0.0.1", Port: "9100", LogLevel: slog.LevelInfo, Env: "dev"},
		},
		{
			name:    "a port that is not a number is refused",
			args:    []string{"-port", "http"},
			wantErr: `port "http"`,
		},
		{
			name:    "a port above 65535 is refused",
			args:    []string{"-port", "70000"},
			wantErr: `port "70000"`,
		},
		{
			name:    "an unknown log level is refused",
			args:    []string{"-log-level", "loud"},
			wantErr: `log-level "loud"`,
		},
		{
			name:    "an unknown ENV is refused",
			env:     map[string]string{"ENV": "staging"},
			wantErr: `ENV "staging"`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, k := range []string{"HOST", "PORT", "LOG_LEVEL", "ENV"} {
				t.Setenv(k, "")
			}
			for k, v := range tt.env {
				t.Setenv(k, v)
			}

			got, err := parseConfig(tt.args, io.Discard)

			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("err = %v, want one containing %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("err = %v", err)
			}
			if got != tt.want {
				t.Errorf("config = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestParseConfigHelp(t *testing.T) {
	var out bytes.Buffer
	_, err := parseConfig([]string{"-h"}, &out)
	if !errors.Is(err, flag.ErrHelp) {
		t.Fatalf("err = %v, want flag.ErrHelp", err)
	}
	for _, want := range []string{"env HOST", "env PORT", "env LOG_LEVEL", "ENV", "CREDENTIALS_DIRECTORY"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("usage lacks %q", want)
		}
	}
}

func TestParseConfigUnknownFlag(t *testing.T) {
	_, err := parseConfig([]string{"-bogus"}, io.Discard)
	if !errors.Is(err, errUsage) {
		t.Fatalf("err = %v, want errUsage", err)
	}
}

func TestConfigLogValue(t *testing.T) {
	t.Parallel()
	var out bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&out, nil))
	cfg := Config{Host: "127.0.0.1", Port: "8080", LogLevel: slog.LevelWarn, Env: "prod"}

	logger.Info("boot", "config", cfg)

	for _, want := range []string{`"host":"127.0.0.1"`, `"port":"8080"`, `"env":"prod"`, `"log_level":"WARN"`} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("log lacks %s: %s", want, out.String())
		}
	}
}
