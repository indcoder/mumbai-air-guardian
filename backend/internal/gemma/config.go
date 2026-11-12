package gemma

import (
	"errors"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	DefaultBaseURL = "http://localhost:11434"
	DefaultModel   = "gemma4:e4b"
	maxTimeout     = 30 * time.Second
)

type Config struct {
	BaseURL string
	Model   string
	Timeout time.Duration
}

func ConfigFromEnvironment() (Config, error) {
	baseURL := os.Getenv("OLLAMA_BASE_URL")
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	model := os.Getenv("OLLAMA_MODEL")
	if model == "" {
		model = DefaultModel
	}
	timeoutSeconds := 30
	if rawTimeout := os.Getenv("OLLAMA_TIMEOUT_SECONDS"); rawTimeout != "" {
		parsed, err := strconv.Atoi(rawTimeout)
		if err != nil {
			return Config{}, errors.New("OLLAMA_TIMEOUT_SECONDS must be an integer from 1 to 30")
		}
		timeoutSeconds = parsed
	}

	config := Config{BaseURL: baseURL, Model: model, Timeout: time.Duration(timeoutSeconds) * time.Second}
	if err := config.Validate(); err != nil {
		return Config{}, err
	}
	return config, nil
}

func (c Config) Validate() error {
	if c.Model != DefaultModel {
		return errors.New("OLLAMA_MODEL must be gemma4:e4b; model substitution is not supported")
	}
	if c.Timeout <= 0 || c.Timeout > maxTimeout {
		return errors.New("OLLAMA_TIMEOUT_SECONDS must be from 1 to 30")
	}
	if strings.TrimSpace(c.BaseURL) != c.BaseURL {
		return errors.New("OLLAMA_BASE_URL must be an absolute loopback HTTP URL")
	}
	parsed, err := url.Parse(c.BaseURL)
	if err != nil || parsed.Scheme != "http" || parsed.Host == "" ||
		parsed.User != nil || (parsed.Path != "" && parsed.Path != "/") ||
		parsed.RawQuery != "" || parsed.Fragment != "" {
		return errors.New("OLLAMA_BASE_URL must be an absolute loopback HTTP URL without credentials or a path")
	}
	host := strings.ToLower(parsed.Hostname())
	if host != "localhost" && net.ParseIP(host) == nil {
		return errors.New("OLLAMA_BASE_URL must target localhost or a loopback IP")
	}
	if ip := net.ParseIP(host); ip != nil && !ip.IsLoopback() {
		return errors.New("OLLAMA_BASE_URL must target localhost or a loopback IP")
	}
	if parsed.Port() != "" {
		port, err := strconv.Atoi(parsed.Port())
		if err != nil || port < 1 || port > 65535 {
			return errors.New("OLLAMA_BASE_URL has an invalid port")
		}
	}
	return nil
}
