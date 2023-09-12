package gontentful

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gosimple/slug"
	gh "github.com/moonwalker/moonbase/pkg/github"
	"gopkg.in/yaml.v3"
)

func getAccessToken() string {
	ght := os.Getenv("GITHUB_TOKEN")
	if len(ght) == 0 {
		ght = os.Getenv("GH_TOKEN")
	}
	return ght
}

func parseFileName(fn string) (string, string, error) {
	ext := filepath.Ext(fn)
	if ext != ".json" {
		return "", "", fmt.Errorf("incorrect file format: %s", ext)
	}

	basefn := strings.TrimSuffix(fn, ext)
	s := strings.Split(basefn, "_")
	if len(s) < 2 || len(s[0]) == 0 || len(s[len(s)-1]) == 0 {
		return "", "", fmt.Errorf("incorrect filename: %s", fn)
	}

	return strings.TrimSuffix(basefn, fmt.Sprintf("_%s", s[len(s)-1])), s[len(s)-1], nil
}

func getConfig(ctx context.Context, owner string, repo string, ref string) *Config {
	accessToken := getAccessToken()
	data, _, _ := gh.GetBlob(ctx, accessToken, owner, repo, ref, configPath)
	return parseConfig(data, accessToken)
}

func parseConfig(data []byte, token string) *Config {
	cfg := &Config{
		Token: token,
	}

	err := yaml.Unmarshal(data, cfg)
	if err != nil {
		json.Unmarshal(data, cfg)
	}

	return cfg
}

func mergeMaps[M ~map[K]V, K comparable, V any](dst M, src M) {
	for k, v := range src {
		dst[k] = v
	}
}

func extractContentype(path string) string {
	items := strings.Split(path, "/")
	if len(items) > 1 {
		return items[len(items)-2]
	}
	return ""
}

func GetImageFileName(fileName string, sysId string, locale string) string {
	ext := filepath.Ext(fileName)
	return slug.Make(fmt.Sprintf("%s_%s-%s", fileName[:len(fileName)-len(ext)], sysId, locale)) + ext
}

func getDefaultLocale(locales []*Locale) string {
	hasDefault := false
	for _, loc := range locales {
		code := strings.ToLower(loc.Code)
		if loc.Default {
			return code
		}
		if code == defaultLocale {
			hasDefault = true
		}
	}
	if len(locales) > 0 && !hasDefault {
		return strings.ToLower(locales[0].Code)
	}

	return defaultLocale
}
