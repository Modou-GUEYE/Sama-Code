package media

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"samacode/backend/internal/httpx"
)

// Types acceptés, détectés sur le contenu du fichier (pas sur son nom).
type kind struct {
	mediaType string // audio | image
	mime      string
	ext       string
	maxBytes  int64
}

const (
	maxImageBytes = 5 << 20
	maxAudioBytes = 15 << 20
)

var accepted = map[string]kind{
	"image/png":       {"image", "image/png", ".png", maxImageBytes},
	"image/jpeg":      {"image", "image/jpeg", ".jpg", maxImageBytes},
	"image/webp":      {"image", "image/webp", ".webp", maxImageBytes},
	"audio/mpeg":      {"audio", "audio/mpeg", ".mp3", maxAudioBytes},
	"audio/wave":      {"audio", "audio/wav", ".wav", maxAudioBytes},
	"application/ogg": {"audio", "audio/ogg", ".ogg", maxAudioBytes},
	"video/mp4":       {"audio", "audio/mp4", ".m4a", maxAudioBytes}, // conteneur MP4 audio (.m4a)
}

var (
	errNoFile       = httpx.Validation("file", "Aucun fichier reçu (champ « file »).")
	errUnsupported  = httpx.NewError(http.StatusUnsupportedMediaType, "unsupported_media", "Format non pris en charge. Audio : MP3, M4A, OGG, WAV. Image : PNG, JPEG, WEBP.")
	errFileTooLarge = httpx.NewError(http.StatusRequestEntityTooLarge, "file_too_large", "Fichier trop volumineux (image : 5 Mo, audio : 15 Mo).")
)

type Asset struct {
	ID           string    `json:"id"`
	MediaType    string    `json:"media_type"`
	MimeType     string    `json:"mime_type"`
	SizeBytes    int64     `json:"size_bytes"`
	OriginalName string    `json:"original_name,omitempty"`
	URL          string    `json:"url"`
	CreatedAt    time.Time `json:"created_at"`
}

func fileURL(id string) string { return "/api/v1/media/" + id + "/file" }

type Handler struct {
	db      *pgxpool.Pool
	storage Storage
}

func NewHandler(db *pgxpool.Pool, storage Storage) *Handler {
	return &Handler{db: db, storage: storage}
}

// PublicRoutes : lecture des fichiers (contenus pédagogiques non confidentiels,
// identifiants non devinables ; une balise <img>/<audio> ne peut pas envoyer de jeton).
func (h *Handler) PublicRoutes(r chi.Router) {
	r.Get("/media/{id}/file", httpx.Handle(h.serveFile))
}

// AdminRoutes : envoi et liste, réservés aux rôles de contenu.
func (h *Handler) AdminRoutes(r chi.Router) {
	r.Post("/media", httpx.Handle(h.upload))
	r.Get("/media", httpx.Handle(h.list))
}

func (h *Handler) upload(w http.ResponseWriter, r *http.Request) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxAudioBytes+1<<20)
	file, header, err := r.FormFile("file")
	if err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			return errFileTooLarge
		}
		return errNoFile
	}
	defer file.Close()

	buffered := bufio.NewReader(file)
	head, _ := buffered.Peek(512)
	k, ok := accepted[http.DetectContentType(head)]
	if !ok {
		return errUnsupported
	}
	if header.Size > k.maxBytes {
		return errFileTooLarge
	}

	id := uuid.NewString()
	key := k.mediaType + "/" + id + k.ext
	hash := sha256.New()
	counter := &countingReader{r: io.TeeReader(io.LimitReader(buffered, k.maxBytes+1), hash)}
	if err := h.storage.Put(r.Context(), key, counter); err != nil {
		return err
	}
	if counter.n > k.maxBytes {
		_ = h.storage.Delete(r.Context(), key)
		return errFileTooLarge
	}

	userID := httpx.MustPrincipal(r.Context()).UserID
	asset := Asset{ID: id, MediaType: k.mediaType, MimeType: k.mime, SizeBytes: counter.n, OriginalName: header.Filename, URL: fileURL(id)}
	err = h.db.QueryRow(r.Context(),
		`INSERT INTO media_assets (id, media_type, storage_key, mime_type, size_bytes, checksum, original_name, created_by)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING created_at`,
		id, k.mediaType, key, k.mime, counter.n, hex.EncodeToString(hash.Sum(nil)), header.Filename, userID,
	).Scan(&asset.CreatedAt)
	if err != nil {
		_ = h.storage.Delete(r.Context(), key)
		return err
	}
	httpx.WriteJSON(w, http.StatusCreated, asset)
	return nil
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) error {
	mediaType := r.URL.Query().Get("type")
	if mediaType != "" && mediaType != "audio" && mediaType != "image" {
		return httpx.Validation("type", "Type invalide (audio ou image).")
	}
	page := httpx.PageParams(r)
	rows, err := h.db.Query(r.Context(),
		`SELECT id, media_type, mime_type, size_bytes, COALESCE(original_name, ''), created_at
		   FROM media_assets
		  WHERE ($1 = '' OR media_type = $1)
		  ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
		mediaType, page.Size, page.Offset())
	if err != nil {
		return err
	}
	items, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (Asset, error) {
		var a Asset
		err := row.Scan(&a.ID, &a.MediaType, &a.MimeType, &a.SizeBytes, &a.OriginalName, &a.CreatedAt)
		a.URL = fileURL(a.ID)
		return a, err
	})
	if err != nil {
		return err
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": items})
	return nil
}

func (h *Handler) serveFile(w http.ResponseWriter, r *http.Request) error {
	id, err := httpx.UUIDParam(r, "id")
	if err != nil {
		return err
	}
	var key, mime string
	err = h.db.QueryRow(r.Context(), `SELECT storage_key, mime_type FROM media_assets WHERE id = $1`, id).Scan(&key, &mime)
	if errors.Is(err, pgx.ErrNoRows) {
		return httpx.ErrNotFound
	}
	if err != nil {
		return err
	}
	f, err := h.storage.Open(r.Context(), key)
	if err != nil {
		slog.ErrorContext(r.Context(), "fichier média manquant", "id", id, "err", err)
		return httpx.ErrNotFound
	}
	defer f.Close()

	w.Header().Set("Content-Type", mime)
	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable") // un fichier ne change jamais
	w.Header().Set("X-Content-Type-Options", "nosniff")
	if seeker, ok := f.(io.ReadSeeker); ok {
		http.ServeContent(w, r, "", time.Time{}, seeker) // gère les requêtes partielles (lecture audio)
		return nil
	}
	_, err = io.Copy(w, f)
	return err
}

type countingReader struct {
	r io.Reader
	n int64
}

func (c *countingReader) Read(p []byte) (int, error) {
	n, err := c.r.Read(p)
	c.n += int64(n)
	return n, err
}
