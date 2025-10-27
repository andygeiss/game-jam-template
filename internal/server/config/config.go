package config

import (
	"embed"
	"log/slog"

	"github.com/andygeiss/cloud-native-utils/messaging"
)

// Config is the configuration for the application.
type Config struct {
	Efs       embed.FS             `json:"-"`
	Logging   *slog.Logger         `json:"-"`
	Messaging messaging.Dispatcher `json:"-"`
	Templates string               `json:"templates"`
}
