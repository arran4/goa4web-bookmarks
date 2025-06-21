package a4webbm

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"strings"
)

// Config holds runtime configuration values.
type Config struct {
	Oauth2ClientID   string `json:"oauth2_client_id"`
	Oauth2Secret     string `json:"oauth2_secret"`
	Oauth2AuthURL    string `json:"oauth2_auth_url"`
	Oauth2TokenURL   string `json:"oauth2_token_url"`
	ExternalURL      string `json:"external_url"`
	CssColumns       bool   `json:"css_columns"`
	Namespace        string `json:"namespace"`
	Title            string `json:"title"`
	FaviconCacheDir  string `json:"favicon_cache_dir"`
	FaviconCacheSize int64  `json:"favicon_cache_size"`
	NoFooter         bool   `json:"no_footer"`
	SessionKey       string `json:"session_key"`
}

// LoadConfigFile loads configuration from the given path.
// It returns the loaded Config, a boolean indicating if the file existed,
// and any error that occurred while reading or parsing the file.
func LoadConfigFile(path string) (Config, bool, error) {
	var c Config

	log.Printf("attempting to load config from %s", path)

	data, err := ioutil.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			log.Printf("config file %s not found", path)
			return c, false, err
		}
		return c, false, fmt.Errorf("unable to read config file: %w", err)
	}

	if err := json.Unmarshal(data, &c); err != nil {
		return c, true, fmt.Errorf("unable to parse config file: %w", err)
	}

	log.Printf("successfully loaded config from %s (keys: %s)", path, strings.Join(loadedConfigKeys(c), ", "))

	return c, true, nil
}

func loadedConfigKeys(c Config) []string {
	var keys []string
	v := reflect.ValueOf(c)
	t := reflect.TypeOf(c)
	for i := 0; i < v.NumField(); i++ {
		if !v.Field(i).IsZero() {
			key := t.Field(i).Tag.Get("json")
			if key == "" {
				key = t.Field(i).Name
			}
			keys = append(keys, key)
		}
	}
	return keys
}

// MergeConfig copies values from src into dst if they are non-zero.
func MergeConfig(dst *Config, src Config) {
	if src.Oauth2ClientID != "" {
		dst.Oauth2ClientID = src.Oauth2ClientID
	}
	if src.Oauth2Secret != "" {
		dst.Oauth2Secret = src.Oauth2Secret
	}
	if src.Oauth2AuthURL != "" {
		dst.Oauth2AuthURL = src.Oauth2AuthURL
	}
	if src.Oauth2TokenURL != "" {
		dst.Oauth2TokenURL = src.Oauth2TokenURL
	}
	if src.ExternalURL != "" {
		dst.ExternalURL = src.ExternalURL
	}
	if src.CssColumns {
		dst.CssColumns = true
	}
	if src.Namespace != "" {
		dst.Namespace = src.Namespace
	}
	if src.Title != "" {
		dst.Title = src.Title
	}
	if src.FaviconCacheDir != "" {
		dst.FaviconCacheDir = src.FaviconCacheDir
	}
	if src.FaviconCacheSize != 0 {
		dst.FaviconCacheSize = src.FaviconCacheSize
	}
	if src.NoFooter {
		dst.NoFooter = true
	}
	if src.SessionKey != "" {
		dst.SessionKey = src.SessionKey
	}
}

// DefaultConfigPath returns the path to the config file depending on
// environment and the effective user. If running as a non-root user and
// XDG variables are set, the config lives under the XDG config directory.
// Otherwise it falls back to /etc/goa4web-bookmarks/config.json.
func DefaultConfigPath() string {
	if p := os.Getenv("GOBM_CONFIG_FILE"); p != "" {
		return p
	}
	if os.Geteuid() != 0 {
		xdg := os.Getenv("XDG_CONFIG_HOME")
		if xdg != "" {
			return filepath.Join(xdg, "goa4web-bookmarks", "config.json")
		}
		if home := os.Getenv("HOME"); home != "" {
			return filepath.Join(home, ".config", "goa4web-bookmarks", "config.json")
		}
	}
	return "/etc/goa4web-bookmarks/config.json"
}

// DefaultSessionKeyPath returns the location of the session key file.
// User installs store it under XDG state or ~/.local/state. System-wide
// installations use /var/lib/goa4web-bookmarks/session.key.
// DefaultSessionKeyPath returns the path used to read or write the
// session key depending on the value of writing. When writing it
// chooses the path appropriate for the current user. When reading it
// checks the usual locations and returns the first existing file,
// falling back to the writing location if none are found.
func DefaultSessionKeyPath(writing bool) string {
	var userPaths []string
	if xdg := os.Getenv("XDG_STATE_HOME"); xdg != "" {
		userPaths = append(userPaths, filepath.Join(xdg, "goa4web-bookmarks", "session.key"))
	}
	if home := os.Getenv("HOME"); home != "" {
		userPaths = append(userPaths, filepath.Join(home, ".local", "state", "goa4web-bookmarks", "session.key"))
	}

	systemPath := "/var/lib/goa4web-bookmarks/session.key"

	if !writing {
		if os.Geteuid() == 0 {
			if fileExists(systemPath) {
				return systemPath
			}
			for _, p := range userPaths {
				if fileExists(p) {
					return p
				}
			}
		} else {
			for _, p := range userPaths {
				if fileExists(p) {
					return p
				}
			}
			if fileExists(systemPath) {
				return systemPath
			}
		}
	}

	if os.Geteuid() != 0 {
		if len(userPaths) > 0 {
			return userPaths[0]
		}
	}
	return systemPath
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// Lines should be in KEY=VALUE format and may be commented with '#'.
func LoadEnvFile(path string) error {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])
		if os.Getenv(key) == "" {
			os.Setenv(key, val)
		}
	}
	return scanner.Err()
}
