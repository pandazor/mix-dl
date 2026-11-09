package model

import (
	"errors"
	"golang.org/x/net/publicsuffix"
	"net"
	"net/url"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

type CustomSite struct {
	Name   string `json:"name"`
	Domain string `json:"domain"`
}

var customID = regexp.MustCompile(`^custom_[0-9a-f]{32}$`)
var domainLabel = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]*[a-z0-9])?$`)

func IsCustomSite(id string) bool { return customID.MatchString(id) }
func DomainMatches(host, domain string) bool {
	host = strings.ToLower(strings.TrimSuffix(host, "."))
	return host == domain || strings.HasSuffix(host, "."+domain)
}
func NormalizeDomain(raw string) (string, error) {
	d := strings.ToLower(strings.TrimSuffix(strings.TrimSpace(raw), "."))
	bad := errors.New("Indiquez un domaine de site valide, sans https://, chemin ni port")
	if len(d) > 253 || net.ParseIP(d) != nil {
		return "", bad
	}
	labels := strings.Split(d, ".")
	if len(labels) < 2 {
		return "", bad
	}
	for _, label := range labels {
		if len(label) > 63 || !domainLabel.MatchString(label) {
			return "", bad
		}
	}
	if suffix, _ := publicsuffix.PublicSuffix(d); suffix == d {
		return "", errors.New("Choisissez le domaine d'un site, pas un suffixe public")
	}
	if builtin, _ := ParseURL("https://" + d); builtin != "generic" {
		return "", errors.New("Ce domaine dispose déjà d'une fiche intégrée")
	}
	return d, nil
}
func ValidateCustomSites(sites map[string]CustomSite) error {
	if len(sites) > 100 {
		return errors.New("100 sites personnalisés maximum")
	}
	seen := map[string]bool{}
	for id, c := range sites {
		if !IsCustomSite(id) || strings.TrimSpace(c.Name) == "" || strings.TrimSpace(c.Name) != c.Name || utf8.RuneCountInString(c.Name) > 100 {
			return errors.New("Nom ou identifiant de site personnalisé invalide")
		}
		for _, r := range c.Name {
			if unicode.IsControl(r) {
				return errors.New("Nom de site invalide")
			}
		}
		d, e := NormalizeDomain(c.Domain)
		if e != nil {
			return e
		}
		if d != c.Domain {
			return errors.New("Domaine non normalisé")
		}
		if seen[d] {
			return errors.New("Un site utilise déjà ce domaine")
		}
		seen[d] = true
	}
	return nil
}
func (s Settings) ResolveSite(raw string) (string, error) {
	builtin, e := ParseURL(raw)
	if e != nil || builtin != "generic" {
		return builtin, e
	}
	u, _ := url.Parse(raw)
	best, found := "", "generic"
	for id, c := range s.CustomSites {
		if DomainMatches(u.Hostname(), c.Domain) && len(c.Domain) > len(best) {
			best, found = c.Domain, id
		}
	}
	return found, nil
}
