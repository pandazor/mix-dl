package auth

import (
	"errors"
	"golang.org/x/sys/windows"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unsafe"
)

type Vault struct{ Dir string }

var keyRE = regexp.MustCompile(`^[a-zA-Z0-9_-]{1,80}$`)

func PrivateDir(path string) error {
	if e := os.MkdirAll(path, 0700); e != nil {
		return e
	}
	user, e := windows.GetCurrentProcessToken().GetTokenUser()
	if e != nil {
		return e
	}
	sd, e := windows.SecurityDescriptorFromString("D:P(A;OICI;FA;;;SY)(A;OICI;FA;;;" + user.User.Sid.String() + ")")
	if e != nil {
		return e
	}
	acl, _, e := sd.DACL()
	if e != nil {
		return e
	}
	return windows.SetNamedSecurityInfo(path, windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION|windows.PROTECTED_DACL_SECURITY_INFORMATION, nil, nil, acl, nil)
}
func New(path string) (*Vault, error) {
	if e := PrivateDir(path); e != nil {
		return nil, e
	}
	return &Vault{path}, nil
}
func blob(b []byte) windows.DataBlob {
	v := windows.DataBlob{Size: uint32(len(b))}
	if len(b) > 0 {
		v.Data = &b[0]
	}
	return v
}
func crypt(b []byte, encrypt bool) ([]byte, error) {
	in := blob(b)
	var out windows.DataBlob
	var e error
	if encrypt {
		e = windows.CryptProtectData(&in, nil, nil, 0, nil, windows.CRYPTPROTECT_UI_FORBIDDEN, &out)
	} else {
		e = windows.CryptUnprotectData(&in, nil, nil, 0, nil, windows.CRYPTPROTECT_UI_FORBIDDEN, &out)
	}
	if e != nil {
		return nil, e
	}
	defer windows.LocalFree(windows.Handle(unsafe.Pointer(out.Data)))
	return append([]byte{}, unsafe.Slice(out.Data, int(out.Size))...), nil
}
func (v *Vault) path(key string) (string, error) {
	if !keyRE.MatchString(key) {
		return "", errors.New("Invalid credential key")
	}
	return filepath.Join(v.Dir, key+".bin"), nil
}
func (v *Vault) Save(key string, value []byte) error {
	p, e := v.path(key)
	if e != nil {
		return e
	}
	b, e := crypt(value, true)
	if e != nil {
		return e
	}
	f, e := os.CreateTemp(v.Dir, "vault-*.tmp")
	if e != nil {
		return e
	}
	tmp := f.Name()
	defer os.Remove(tmp)
	if _, e = f.Write(b); e != nil {
		f.Close()
		return e
	}
	if e = f.Sync(); e != nil {
		f.Close()
		return e
	}
	if e = f.Close(); e != nil {
		return e
	}
	from, e := windows.UTF16PtrFromString(tmp)
	if e != nil {
		return e
	}
	to, e := windows.UTF16PtrFromString(p)
	if e != nil {
		return e
	}
	return windows.MoveFileEx(from, to, windows.MOVEFILE_REPLACE_EXISTING|windows.MOVEFILE_WRITE_THROUGH)
}
func (v *Vault) Load(key string) ([]byte, error) {
	p, e := v.path(key)
	if e != nil {
		return nil, e
	}
	b, e := os.ReadFile(p)
	if e != nil {
		return nil, e
	}
	return crypt(b, false)
}
func (v *Vault) Delete(key string) error {
	p, e := v.path(key)
	if e != nil {
		return e
	}
	e = os.Remove(p)
	if os.IsNotExist(e) {
		return nil
	}
	return e
}
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
