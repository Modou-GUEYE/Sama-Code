// Package media gère les fichiers audio et images : envoi par l'administration,
// métadonnées en base (media_assets) et lecture par les applications.
package media

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Storage stocke les fichiers. En développement : disque local ; en production :
// stockage objet compatible S3 (à brancher avec le choix d'hébergement, Q8).
type Storage interface {
	Put(ctx context.Context, key string, r io.Reader) error
	Open(ctx context.Context, key string) (io.ReadCloser, error)
	Delete(ctx context.Context, key string) error
}

type LocalStorage struct {
	dir string
}

func NewLocalStorage(dir string) (*LocalStorage, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(abs, 0o750); err != nil {
		return nil, err
	}
	return &LocalStorage{dir: abs}, nil
}

var errBadKey = errors.New("clé de stockage invalide")

// path refuse toute clé qui sortirait du dossier de stockage.
func (s *LocalStorage) path(key string) (string, error) {
	p := filepath.Join(s.dir, filepath.FromSlash(key))
	if !strings.HasPrefix(p, s.dir+string(filepath.Separator)) {
		return "", errBadKey
	}
	return p, nil
}

func (s *LocalStorage) Put(_ context.Context, key string, r io.Reader) error {
	p, err := s.path(key)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o750); err != nil {
		return err
	}
	f, err := os.OpenFile(p, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o640)
	if err != nil {
		return err
	}
	if _, err := io.Copy(f, r); err != nil {
		f.Close()
		os.Remove(p)
		return err
	}
	return f.Close()
}

func (s *LocalStorage) Open(_ context.Context, key string) (io.ReadCloser, error) {
	p, err := s.path(key)
	if err != nil {
		return nil, err
	}
	return os.Open(p)
}

func (s *LocalStorage) Delete(_ context.Context, key string) error {
	p, err := s.path(key)
	if err != nil {
		return err
	}
	return os.Remove(p)
}
