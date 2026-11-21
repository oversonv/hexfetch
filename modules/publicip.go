package modules

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

type publicip struct{}

func init() { Register(publicip{}) }

func (publicip) Name() string { return "publicip" }

func (publicip) OptionSchema() map[string]OptionDefinition {
	return map[string]OptionDefinition{
		"url": {Kind: OptionString, URL: true},
	}
}

const defaultPublicIPURL = "https://api.ipify.org"

func (publicip) Run(ctx Context) ([]Entry, error) {
	endpoint := ctx.OptionString("url", os.Getenv("NIXFETCH_PUBLIC_IP_URL"))
	if endpoint == "" {
		endpoint = defaultPublicIPURL
	}
	parsed, err := url.Parse(endpoint)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return nil, errors.New("public IP endpoint must be an http(s) URL")
	}
	ip, err := fetchPublicIP(endpoint, 3*time.Second)
	if err != nil {
		return nil, Unsupported("public IP unavailable: %v", err)
	}
	return Single(ctx, "Public IP", ip), nil
}

func fetchPublicIP(endpoint string, timeout time.Duration) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
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
		return "", errors.New("public IP request failed with " + response.Status)
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, 4096))
	if err != nil {
		return "", err
	}
	value := strings.TrimSpace(string(body))
	if net.ParseIP(value) == nil {
		return "", errors.New("public IP endpoint returned an invalid address")
	}
	return value, nil
}
