package auth

import (
	"dl-manager/internal/model"
	"errors"
	"regexp"
	"strings"
)

var keyRE = regexp.MustCompile(`^[a-zA-Z0-9_-]{1,80}$`)

func ValidateCookies(text, site string) error {
	if len(text) > 10*1024*1024 {
		return errors.New("Fichier de cookies trop volumineux")
	}
	domains := map[string]string{"instagram": "instagram.com", "youtube": "youtube.com", "tiktok": "tiktok.com", "patreon": "patreon.com", "tumblr": "tumblr.com", "artstation": "artstation.com", "pixiv": "pixiv.net", "deviantart": "deviantart.com", "rule34": "rule34.xxx"}
	domain := domains[site]
	count := 0
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		line = strings.TrimPrefix(line, "#HttpOnly_")
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.Split(line, "\t")
		if len(parts) != 7 {
			return errors.New("Format Netscape attendu : sept colonnes séparées par des tabulations")
		}
		host := strings.TrimPrefix(parts[0], ".")
		if domain == "" || host == domain || strings.HasSuffix(host, "."+domain) {
			count++
		}
	}
	if count == 0 {
		return errors.New("Aucun cookie du site choisi dans ce fichier Netscape")
	}
	return nil
}

func ValidateCookiesForDomain(text, domain string) error {
	if e := ValidateCookies(text, "generic"); e != nil {
		return e
	}
	count := 0
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimPrefix(strings.TrimSpace(line), "#HttpOnly_")
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.Split(line, "\t")
		host := strings.ToLower(strings.TrimPrefix(parts[0], "."))
		if !model.DomainMatches(host, domain) {
			return errors.New("Le fichier contient des cookies d'un autre domaine ; exportez uniquement ceux du site choisi")
		}
		count++
	}
	if count == 0 {
		return errors.New("Aucun cookie du domaine choisi")
	}
	return nil
}
