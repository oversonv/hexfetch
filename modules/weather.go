package modules

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

type weather struct{}

func init() { Register(weather{}) }

func (weather) Name() string { return "weather" }

func (weather) OptionSchema() map[string]OptionDefinition {
	return map[string]OptionDefinition{
		"url": {Kind: OptionString, URL: true},
	}
}

const defaultWeatherURL = "https://wttr.in/?format=%l:+%c+%t"

func (weather) Run(ctx Context) ([]Entry, error) {
	endpoint := ctx.OptionString("url", os.Getenv("NIXFETCH_WEATHER_URL"))
	if endpoint == "" {
		endpoint = defaultWeatherURL
	}
	parsed, err := url.Parse(endpoint)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return nil, errors.New("weather endpoint must be an http(s) URL")
	}
	body, err := fetchText(endpoint, 3*time.Second)
	if err != nil {
		return nil, Unsupported("weather unavailable: %v", err)
	}
	value := weatherReport(body)
	if value == "" {
		return nil, errors.New("weather report unavailable")
	}
	return Single(ctx, "Weather", value), nil
}

func fetchText(endpoint string, timeout time.Duration) (string, error) {
	requestCtx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	request, err := http.NewRequestWithContext(requestCtx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", err
	}
	client := &http.Client{Timeout: timeout}
	response, err := client.Do(request)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return "", errors.New("weather request failed with " + response.Status)
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, 4096))
	if err != nil {
		return "", err
	}
	return string(body), nil
}

// weatherReport takes the first non-empty line of the response.
func weatherReport(body string) string {
	for _, line := range strings.Split(body, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			return line
		}
	}
	return ""
}
