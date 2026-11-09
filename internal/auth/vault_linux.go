package auth

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const vaultFormat = "dl-manager-vault-1\x00"

var errKeyring = errors.New("Coffre Secret Service indisponible. Installez secret-tool (libsecret-tools), puis ouvrez et déverrouillez le trousseau de votre session Linux")

type Vault struct {
	Dir     string
	mu      sync.Mutex
	keyring func(op, id string, input []byte) ([]byte, error)
}

func PrivateDir(path string) error {
	if err := os.MkdirAll(path, 0700); err != nil {
		return err
	}
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("Dossier privé invalide : %s", path)
	}
	return os.Chmod(path, 0700)
}

func New(path string) (*Vault, error) {
	if err := PrivateDir(path); err != nil {
		return nil, err
	}
	return &Vault{Dir: path, keyring: secretTool}, nil
}

func secretTool(op, id string, input []byte) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	args := []string{op}
	if op == "store" {
		args = append(args, "--label=dl-manager")
	}
	args = append(args, "application", "dl-manager", "vault-id", id)
	cmd := exec.CommandContext(ctx, "secret-tool", args...)
	if op == "store" {
		cmd.Stdin = bytes.NewReader(input)
	}
	// Discard diagnostics from the subprocess. They need not be safe to expose.
	return cmd.Output()
}

func (v *Vault) path(key string) (string, error) {
	if !keyRE.MatchString(key) {
		return "", errors.New("Invalid credential key")
	}
	return filepath.Join(v.Dir, key+".bin"), nil
}

func (v *Vault) aead(create bool) (cipher.AEAD, error) {
	idPath := filepath.Join(v.Dir, "vault.id")
	id, err := os.ReadFile(idPath)
	var key []byte
	if errors.Is(err, os.ErrNotExist) && create {
		entries, err := os.ReadDir(v.Dir)
		if err != nil {
			return nil, err
		}
		for _, entry := range entries {
			if strings.HasSuffix(entry.Name(), ".bin") {
				return nil, errors.New("Identifiant du coffre manquant. Restaurez le fichier vault.id ou créez un dossier de données distinct pour Linux")
			}
		}
		rawID := make([]byte, 16)
		rand.Read(rawID)
		id = []byte(hex.EncodeToString(rawID))
		key = make([]byte, 32)
		rand.Read(key)
		encoded := make([]byte, base64.RawStdEncoding.EncodedLen(len(key)))
		base64.RawStdEncoding.Encode(encoded, key)
		_, err = v.keyring("store", string(id), encoded)
		clear(encoded)
		if err != nil {
			clear(key)
			return nil, errKeyring
		}
		if err = writeVaultFile(v.Dir, idPath, id); err != nil {
			clear(key)
			return nil, err
		}
	} else {
		if err != nil {
			return nil, fmt.Errorf("Lecture de l’identifiant du coffre impossible : %v", err)
		}
		if decoded, err := hex.DecodeString(string(id)); err != nil || len(decoded) != 16 {
			return nil, errors.New("Identifiant du coffre Linux invalide")
		}
		encoded, err := v.keyring("lookup", string(id), nil)
		if err != nil {
			return nil, errKeyring
		}
		key = make([]byte, base64.RawStdEncoding.DecodedLen(len(encoded)))
		n, err := base64.RawStdEncoding.Decode(key, bytes.TrimSpace(encoded))
		clear(encoded)
		if err != nil || n != 32 {
			clear(key)
			return nil, errors.New("Clé du coffre Linux invalide dans Secret Service")
		}
		key = key[:n]
	}
	defer clear(key)
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	return cipher.NewGCMWithRandomNonce(block)
}

func writeVaultFile(dir, path string, data []byte) error {
	f, err := os.CreateTemp(dir, "vault-*.tmp")
	if err != nil {
		return err
	}
	tmp := f.Name()
	defer os.Remove(tmp)
	defer f.Close()
	if _, err := f.Write(data); err != nil {
		return err
	}
	if err := f.Sync(); err != nil {
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func (v *Vault) Save(key string, value []byte) error {
	v.mu.Lock()
	defer v.mu.Unlock()
	path, err := v.path(key)
	if err != nil {
		return err
	}
	aead, err := v.aead(true)
	if err != nil {
		return err
	}
	ciphertext := aead.Seal(nil, nil, value, []byte(key))
	return writeVaultFile(v.Dir, path, append([]byte(vaultFormat), ciphertext...))
}

func (v *Vault) Load(key string) ([]byte, error) {
	v.mu.Lock()
	defer v.mu.Unlock()
	path, err := v.path(key)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	ciphertext, ok := bytes.CutPrefix(data, []byte(vaultFormat))
	if !ok {
		return nil, errors.New("Format de coffre incompatible. Les comptes Windows doivent être configurés à nouveau sous Linux")
	}
	aead, err := v.aead(false)
	if err != nil {
		return nil, err
	}
	value, err := aead.Open(nil, nil, ciphertext, []byte(key))
	if err != nil {
		return nil, errors.New("Impossible de déchiffrer les données du compte Linux")
	}
	return value, nil
}

func (v *Vault) Delete(key string) error {
	v.mu.Lock()
	defer v.mu.Unlock()
	path, err := v.path(key)
	if err != nil {
		return err
	}
	err = os.Remove(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}
