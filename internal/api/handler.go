package api

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"miniproject/internal/shortener"
	"miniproject/internal/storage"
	"net"
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

// http.Request에서클라이언트 IP 주소를 추출합니다.
// X-Forward-For,X-Real-IP 헤더를 우선 확인 후 RemoteAddr을 사용합니다
func getClientIP(r *http.Request) string {
	xForwardedFor := r.Header.Get("X-Forwarded-For")
	// X-Forward-For는 "Client", "proxy1", "proxy2" 형태일 수 있으므로 첫번째 IP 사용
	if xForwardedFor != "" {
		ips := strings.Split(xForwardedFor, ",")
		clientIP := strings.TrimSpace(ips[0])
		if clientIP != "" {
			return clientIP
		}
	}

	//X-Real-IP 헤더 (일부 프록시 사용)
	xRealIP := r.Header.Get("X-Real-IP")
	if xRealIP != "" {
		return strings.TrimSpace(xRealIP)
	}

	// 위 헤더들이 없는 경우 RemoteAddr 사용
	// RemoteAddr은 "ip:port" 형태일 수 있으므로 IP 부분만 추출
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return ip
	}
	// SplitHostPort가 실패하면 RemoteAddr이 이미 IP이거나 다른 형태일 수 있음
	return r.RemoteAddr
}

type ShortenURLRequest struct {
	URL string `json:"url"`
}

type ShortenURLResponse struct {
	ShortURL string `json:"short_url"`
}

func (h *Handler) ShortenURL(w http.ResponseWriter, r *http.Request) {
	clientIP := getClientIP(r)
	userAgent := r.Header.Get("User-Agent")

	if r.Method != http.MethodPost {
		h.logger.Printf("[IP: %s, UA: %s] 메소드 불일치 (요청: %s, 허용: POST) 경로: %s", clientIP, userAgent, r.Method, r.URL.Path)
		h.respondWithError(w, http.StatusMethodNotAllowed, "허용되지 않은 메소드입니다.")
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.logger.Printf("[IP: %s, UA: %s] 요청 본문 읽기 실패: %v, 경로: %s", clientIP, userAgent, err, r.URL.Path)
		h.respondWithError(w, http.StatusInternalServerError, "요청 처리 중 오류가 발생했습니다.")
		return
	}
	defer r.Body.Close()

	var req ShortenURLRequest
	if err := json.Unmarshal(body, &req); err != nil {
		h.logger.Printf("[IP: %s, UA: %s] 잘못된 JSON 형식: %v, 내용: %s", clientIP, userAgent, err, string(body))
		h.respondWithError(w, http.StatusBadRequest, "잘못된 JSON 형식입니다.")
		return
	}

	if strings.TrimSpace(req.URL) == "" {
		h.logger.Printf("[IP: %s, UA: %s] URL 필드 비어있음, 경로: %s", clientIP, userAgent, r.URL.Path)
		h.respondWithError(w, http.StatusBadRequest, "URL 필드는 비어 있을 수 없습니다.")
		return
	}

	shortID, err := h.service.CreateShortURL(req.URL)
	if err != nil {
		h.logger.Printf("[IP: %s, UA: %s] URL 단축 실패 (원본 URL: '%s'): %v", clientIP, userAgent, req.URL, err)
		h.respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	response := ShortenURLResponse{ShortURL: shortID}
	h.respondWithJSON(w, http.StatusCreated, response)
	h.logger.Printf("[IP: %s, UA: %s] URL 단축 성공: '%s' -> '%s'", clientIP, userAgent, req.URL, shortID)
}

func (h *Handler) RedirectURL(w http.ResponseWriter, r *http.Request) {
	clientIP := getClientIP(r)
	userAgent := r.Header.Get("User-Agent")

	if r.Method != http.MethodGet {
		h.logger.Printf("[IP: %s, UA: %s] 메소드 불일치 (요청: %s, 허용: GET) 경로: %s", clientIP, userAgent, r.Method, r.URL.Path)
		h.respondWithError(w, http.StatusMethodNotAllowed, "허용되지 않은 메소드입니다.")
		return
	}

	shortID := strings.TrimPrefix(r.URL.Path, "/")
	if strings.TrimSpace(shortID) == "" {
		h.logger.Printf("[IP: %s, UA: %s] 짧은 ID가 경로에 없음: %s", clientIP, userAgent, r.URL.Path)
		h.respondWithError(w, http.StatusBadRequest, "짧은 ID가 경로에 없습니다.")
		return
	}

	originalURL, err := h.service.GetOriginalURL(shortID)
	if err != nil {
		var notFoundErr *storage.ErrNotFound
		if errors.As(err, &notFoundErr) {
			h.logger.Printf("[IP: %s, UA: %s] ID '%s' 찾을 수 없음: %v", clientIP, userAgent, shortID, err)
			h.respondWithError(w, http.StatusNotFound, "요청한 URL을 찾을 수 없습니다.")
		} else {
			h.logger.Printf("[IP: %s, UA: %s] 원본 URL 조회 실패 (ID: '%s'): %v", clientIP, userAgent, shortID, err)
			h.respondWithError(w, http.StatusInternalServerError, "내부 서버 오류입니다.")
		}
		return
	}

	http.Redirect(w, r, originalURL, http.StatusFound)
	h.logger.Printf("[IP: %s, UA: %s] 리다이렉션: '%s' -> '%s'", clientIP, userAgent, shortID, originalURL)
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
