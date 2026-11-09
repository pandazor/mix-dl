package model

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"net/url"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

type Filters struct {
	Images     bool     `json:"images"`
	Videos     bool     `json:"videos"`
	Extensions []string `json:"extensions"`
}
type SiteSettings struct {
	Destination string   `json:"destination"`
	Layout      string   `json:"layout"`
	Filename    string   `json:"filename"`
	ProfileID   string   `json:"profileId"`
	Filters     *Filters `json:"filters,omitempty"`
}
type Settings struct {
	CustomSites map[string]CustomSite   `json:"customSites,omitempty"`
	Destination string                  `json:"destination"`
	Layout      string                  `json:"layout"`
	Filename    string                  `json:"filename"`
	Filters     Filters                 `json:"filters"`
	Concurrency int                     `json:"concurrency"`
	QuotaBytes  int64                   `json:"quotaBytes"`
	RateKiB     int64                   `json:"rateKiB"`
	GalleryPath string                  `json:"galleryPath"`
	YTDLPPath   string                  `json:"ytdlpPath"`
	FFmpegPath  string                  `json:"ffmpegPath"`
	Sites       map[string]SiteSettings `json:"sites"`
}
type Job struct {
	DestinationMode string  `json:"destinationMode,omitempty"`
	ProfileMode     string  `json:"profileMode"`
	ID              string  `json:"id"`
	URL             string  `json:"url"`
	Site            string  `json:"site"`
	Destination     string  `json:"destination"`
	Layout          string  `json:"layout"`
	Filename        string  `json:"filename"`
	Filters         Filters `json:"filters"`
	ProfileID       string  `json:"profileId"`
	Status          string  `json:"status"`
	Added           string  `json:"added"`
	Started         string  `json:"started"`
	Ended           string  `json:"ended"`
	Error           string  `json:"error"`
	ExitCode        int     `json:"exitCode"`
	Bytes           int64   `json:"bytes"`
	Files           int     `json:"files"`
	Skipped         int     `json:"skipped"`
	Current         string  `json:"current"`
	Order           int64   `json:"order"`
}
type Profile struct {
	ID     string `json:"id"`
	Site   string `json:"site"`
	Name   string `json:"name"`
	Method string `json:"method"`
	State  string `json:"state"`
}
type Snapshot struct {
	Settings  Settings  `json:"settings"`
	Jobs      []Job     `json:"jobs"`
	Profiles  []Profile `json:"profiles"`
	UsedBytes int64     `json:"usedBytes"`
	Running   bool      `json:"running"`
	Active    int       `json:"active"`
	LastError string    `json:"lastError"`
}
type Event struct {
	Kind     string
	Message  string
	Bytes    int64
	ExitCode int
}

// ProfileSelection handles pre-0.2 jobs, which did not record inheritance separately.
func (j Job) ProfileSelection() string {
	if j.ProfileMode != "" {
		return j.ProfileMode
	}
	if j.ProfileID != "" {
		return "explicit"
	}
	return "inherit"
}
func ResolveProfile(j Job, s Snapshot) (string, error) {
	id := j.ProfileID
	switch j.ProfileSelection() {
	case "inherit":
		id = s.Settings.Sites[j.Site].ProfileID
	case "none":
		return "", nil
	case "explicit":
		if id == "" {
			return "", errors.New("Choisissez un compte")
		}
	default:
		return "", errors.New("Sélection de compte invalide")
	}
	if id == "" {
		return "", nil
	}
	for _, p := range s.Profiles {
		if p.ID == id && (p.Site == j.Site || p.Site == "generic") {
			return id, nil
		}
	}
	return "", errors.New("Compte introuvable ou incompatible : choisissez un compte dans les détails de la tâche")
}

func ID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)
}
func Now() string { return time.Now().UTC().Format(time.RFC3339) }
func DefaultFilters() Filters {
	return Filters{true, true, []string{"jpg", "jpeg", "png", "webp", "gif", "mp4", "webm", "mov", "avi", "mkv"}}
}
func DefaultSettings() Settings {
	return Settings{Layout: "creator", Filename: "default", Filters: DefaultFilters(), Concurrency: 1, Sites: map[string]SiteSettings{}}
}

var extRE = regexp.MustCompile(`^[a-z0-9]{1,12}$`)

func (f Filters) Validate() error {
	if !f.Images && !f.Videos {
		return errors.New("Sélectionnez Images ou Vidéos")
	}
	if len(f.Extensions) == 0 || len(f.Extensions) > 40 {
		return errors.New("Choisissez au moins une extension (40 maximum)")
	}
	for _, e := range f.Extensions {
		if !extRE.MatchString(e) {
			return errors.New("Extension invalide : lettres minuscules et chiffres uniquement")
		}
	}
	return nil
}
func (f Filters) Allowed() []string {
	videos := map[string]bool{"mp4": true, "webm": true, "mov": true, "avi": true, "mkv": true, "m4v": true, "ts": true, "flv": true}
	out := []string{}
	for _, e := range f.Extensions {
		if (videos[e] && f.Videos) || (!videos[e] && f.Images) {
			out = append(out, e)
		}
	}
	return out
}
func ValidLayout(v string) bool   { return v == "" || v == "flat" || v == "site" || v == "creator" }
func ValidFilename(v string) bool { return v == "" || v == "default" || v == "id-title" }
func (s Settings) Validate() error {
	if e := ValidateCustomSites(s.CustomSites); e != nil {
		return e
	}
	if s.Concurrency < 1 || s.Concurrency > 3 {
		return errors.New("La concurrence doit être comprise entre 1 et 3")
	}
	if s.QuotaBytes < 0 || s.RateKiB < 0 {
		return errors.New("Les limites ne peuvent pas être négatives")
	}
	if !ValidLayout(s.Layout) || !ValidFilename(s.Filename) {
		return errors.New("Organisation de dossiers ou nommage invalide")
	}
	if err := s.Filters.Validate(); err != nil {
		return err
	}
	if s.Destination != "" && !filepath.IsAbs(s.Destination) {
		return errors.New("Le dossier doit être un chemin absolu")
	}
	for id, v := range s.Sites {
		if strings.HasPrefix(id, "custom_") {
			if _, ok := s.CustomSites[id]; !ok {
				return errors.New("Réglages de site personnalisé orphelins")
			}
		}
		if !ValidLayout(v.Layout) || !ValidFilename(v.Filename) {
			return errors.New("Réglage de site invalide")
		}
		if v.Destination != "" && !filepath.IsAbs(v.Destination) {
			return errors.New("Le dossier du site doit être absolu")
		}
		if v.Filters != nil {
			if e := v.Filters.Validate(); e != nil {
				return e
			}
		}
	}
	return nil
}
func ParseURL(raw string) (string, error) {
	if len(raw) > 8192 || strings.ContainsAny(raw, "\r\n\t\x00") {
		return "", errors.New("URL invalide")
	}
	u, e := url.Parse(raw)
	if e != nil || u.Hostname() == "" || (u.Scheme != "https" && u.Scheme != "http") || u.User != nil {
		return "", errors.New("Une adresse HTTP(S) sans identifiants intégrés est nécessaire")
	}
	h := strings.ToLower(u.Hostname())
	sites := map[string]string{"youtube.com": "youtube", "youtu.be": "youtube", "instagram.com": "instagram", "tiktok.com": "tiktok", "tumblr.com": "tumblr", "patreon.com": "patreon", "artstation.com": "artstation", "rule34.xxx": "rule34", "deviantart.com": "deviantart", "pixiv.net": "pixiv"}
	for domain, site := range sites {
		if h == domain || strings.HasSuffix(h, "."+domain) {
			return site, nil
		}
	}
	return "generic", nil
}

var forbidden = regexp.MustCompile(`[<>:"/\\|?*\x00-\x1f]`)
var reserved = regexp.MustCompile(`(?i)^(CON|PRN|AUX|NUL|COM[1-9]|LPT[1-9])(?:\.|$)`)

func SafeSegment(s string) string {
	s = forbidden.ReplaceAllString(s, "_")
	s = strings.Trim(s, ". ")
	if s == "" {
		s = "_"
	}
	if reserved.MatchString(s) {
		s = "_" + s
	}
	r := []rune(s)
	if len(r) > 100 {
		s = string(r[:100])
	}
	return s
}
func Terminal(s string) bool {
	return s == "completed" || s == "failed" || s == "cancelled" || s == "authentication_required"
}
