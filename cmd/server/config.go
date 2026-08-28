package main

import (
	"cmp"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strconv"
)

// Config is every knob this binary has. After parseConfig returns, nothing
// else reads os.Getenv — the struct is the whole contract.
type Config struct {
	Host     string
	Port     string // string: net.JoinHostPort takes one
	LogLevel slog.Level
	Env      string // dev | prod — picks text vs JSON log output
}

// errUsage means the flag package already printed what was wrong, so main
// only picks the exit code.
var errUsage = errors.New("usage error")

// parseConfig reads flags, with environment variables as their defaults, and
// validates everything before anything else starts.
func parseConfig(args []string, stderr io.Writer) (Config, error) {
	fs := flag.NewFlagSet("server", flag.ContinueOnError)
	fs.SetOutput(stderr)

	var c Config
	fs.StringVar(&c.Host, "host", cmp.Or(os.Getenv("HOST"), "127.0.0.1"), "bind address (env HOST)")
	fs.StringVar(&c.Port, "port", cmp.Or(os.Getenv("PORT"), "8080"), "listener port (env PORT)")
	level := fs.String("log-level", cmp.Or(os.Getenv("LOG_LEVEL"), "info"), "debug|info|warn|error (env LOG_LEVEL)")

	// The variables that are not flags have nowhere else to be documented.
	fs.Usage = func() {
		fmt.Fprintf(stderr, "Usage of server:\n")
		fs.PrintDefaults()
		fmt.Fprintf(stderr, "\nRead from the environment only:\n"+
			"  ENV\n\tdev|prod, picks text vs JSON logs (default dev)\n"+
			"  CREDENTIALS_DIRECTORY\n\tnot read: this server has no secrets\n")
	}

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return Config{}, err
		}
		return Config{}, errUsage
	}

	c.Env = cmp.Or(os.Getenv("ENV"), "dev")

	if err := c.LogLevel.UnmarshalText([]byte(*level)); err != nil {
		return Config{}, fmt.Errorf("log-level %q: want debug, info, warn, or error", *level)
	}
	if _, err := strconv.ParseUint(c.Port, 10, 16); err != nil {
		return Config{}, fmt.Errorf("port %q: want a number from 0 to 65535", c.Port)
	}
	if c.Env != "dev" && c.Env != "prod" {
		return Config{}, fmt.Errorf("ENV %q: want dev or prod", c.Env)
	}
	return c, nil
}

// LogValue is what slog logs for a Config. There are no secrets to leave out
// yet; keep it an allowlist so a secret added later stays out by default.
func (c Config) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("host", c.Host),
		slog.String("port", c.Port),
		slog.String("env", c.Env),
		slog.String("log_level", c.LogLevel.String()),
	)
}
