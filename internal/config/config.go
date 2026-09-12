// Package config reads the process configuration from the environment.
package config

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"

	"github.com/dragunovartem99/chessdocs-api/internal/github"
)

// Config is everything the server needs to start. See .env.example.
type Config struct {
	// Addr is the TCP address the HTTP server listens on.
	Addr string
	// AllowedOrigin is the single browser origin CORS lets through.
	AllowedOrigin string
	// GitHub identifies the repository submissions are opened against.
	GitHub github.Config
}

// Load reads the environment, reporting every missing variable at once so a
// misconfigured deploy takes one restart to diagnose rather than six.
func Load() (Config, error) {
	var missing []string
	required := func(key string) string {
		value := strings.TrimSpace(os.Getenv(key))
		if value == "" {
			missing = append(missing, key)
		}
		return value
	}

	cfg := Config{
		AllowedOrigin: required("ALLOWED_ORIGIN"),
		GitHub: github.Config{
			Token:      required("GITHUB_TOKEN"),
			Owner:      required("GITHUB_OWNER"),
			Repo:       required("GITHUB_REPO"),
			BaseBranch: required("GITHUB_BASE_BRANCH"),
		},
	}
	port := required("PORT")

	if len(missing) > 0 {
		return Config{}, fmt.Errorf("missing environment variables: %s", strings.Join(missing, ", "))
	}
	if _, err := strconv.ParseUint(port, 10, 16); err != nil {
		return Config{}, fmt.Errorf("PORT must be a port number, got %q", port)
	}
	cfg.Addr = net.JoinHostPort("", port)

	return cfg, nil
}
