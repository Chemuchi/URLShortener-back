package api

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"miniproject/internal/shortener"
	"miniproject/internal/storage"
	"net/http"
	"strings"
)

type Handler struct {
	service *shortener.Service
	logger  *log.Logger
}

func NewHandler(service *shortener.Service, logger *log.Logger) *Handler {
	return &Handler{
		service: service,
		logger:  logger,
	}
}

type ShortenURLRequest struct {
	URL string `json:"url"`
}

type ShortenURLResponse struct {
	ShortURL string `json:"short_url"`
}

func (h *Handler) ShortenURL(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.respondWithError(w, http.StatusMethodNotAllowed, "허용되지 않은 메소드입니다.")
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.logger.Printf("요청 본문 읽기 실패: %v", err)
		h.respondWithError(w, http.StatusInternalServerError, "요청 처리 중 오류가 발생했습니다.")
		return
	}
	defer r.Body.Close()

	var req ShortenURLRequest
	if err := json.Unmarshal(body, &req); err != nil {
		h.respondWithError(w, http.StatusBadRequest, "잘못된 JSON 형식입니다.")
		return
	}

	if strings.TrimSpace(req.URL) == "" {
		h.respondWithError(w, http.StatusBadRequest, "URL 필드는 비어 있을 수 없습니다.")
		return
	}

	shortID, err := h.service.CreateShortURL(req.URL)
	if err != nil {
		h.logger.Printf("URL 단축 실패 ('%s'): %v", req.URL, err)
		h.respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}
	response := ShortenURLResponse{ShortURL: shortID} // 실제로는 http://yourdomain/shortID 형태여야 함
	h.respondWithJSON(w, http.StatusCreated, response)
	h.logger.Printf("URL 단축 성공: '%s' -> '%s'", req.URL, shortID)
}

func (h *Handler) RedirectURL(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.respondWithError(w, http.StatusMethodNotAllowed, "허용되지 않은 메소드입니다.")
		return
	}

	shortID := strings.TrimPrefix(r.URL.Path, "/")
	if strings.TrimSpace(shortID) == "" {
		h.respondWithError(w, http.StatusBadRequest, "짧은 ID가 경로에 없습니다.")
		return
	}

	originalURL, err := h.service.GetOriginalURL(shortID)
	if err != nil {
		var notFoundErr *storage.ErrNotFound
		if errors.As(err, &notFoundErr) {
			h.logger.Printf("ID '%s' 찾을 수 없음: %v", shortID, err)
			h.respondWithError(w, http.StatusNotFound, "요청한 URL을 찾을 수 없습니다.")
		} else {
			h.logger.Printf("원본 URL 조회 실패 ('%s'): %v", shortID, err)
			h.respondWithError(w, http.StatusInternalServerError, "내부 서버 오류입니다.")
		}
		return
	}
	http.Redirect(w, r, originalURL, http.StatusFound)
	h.logger.Printf("리다이렉션: '%s' -> '%s'", shortID, originalURL)
}

func (h *Handler) respondWithError(w http.ResponseWriter, code int, message string) {
	h.respondWithJSON(w, code, map[string]string{"error": message})
}

func (h *Handler) respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	response, err := json.Marshal(payload)
	if err != nil {
		h.logger.Printf("JSON 마샬링 실패: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "응답 생성 중 오류가 발생했습니다."}`))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(response)
}
