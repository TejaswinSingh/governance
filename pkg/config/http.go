package config

import "strings"

type HttpMode string

func (mode HttpMode) IsHttps() bool {
	return strings.EqualFold(string(mode), "HTTPS")
}

const (
	Http  HttpMode = "HTTP"
	Https HttpMode = "HTTPS"
)

func ValidateHttpMode(mode HttpMode) HttpMode {
	switch mode {
	case Http, Https:
		return mode
	default:
		return Http
	}
}
