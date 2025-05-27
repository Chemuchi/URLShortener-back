package shortener

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"miniproject/internal/storage"
	"net/url"
	"strings"
)

type Service struct {
	store    storage.URLStore
	idLength int
}

func NewService(store storage.URLStore, idLength int) *Service {
	if idLength <= 0 {
		idLength = 6
	}
	return &Service{
		store:    store,
		idLength: idLength,
	}
}

func (s *Service) generateShortID() (string, error) {
	b := make([]byte, s.idLength)
	_, err := rand.Read(b)
	if err != nil {
		return "", fmt.Errorf("무작위 바이트 생성 실패: %w", err)
	}
	id := base64.RawURLEncoding.EncodeToString(b)
	return id, nil
}

func (s *Service) CreateShortURL(originalURL string) (string, error) {
	if !isValidURL(originalURL) {
		return "", fmt.Errorf("유효하지 않은 URL입니다: %s", originalURL)
	}

	var shortID string
	var err error
	maxRetries := 10

	for i := 0; i < maxRetries; i++ {
		shortID, err = s.generateShortID()
		if err != nil {
			return "", fmt.Errorf("짧은 ID 생성 실패: %w", err)
		}

		exists, err := s.store.Exists(shortID)
		if err != nil {
			return "", fmt.Errorf("ID 존재 여부 확인 중 에러: %w", err)
		}
		if !exists {
			break
		}
		if i == maxRetries-1 {
			return "", fmt.Errorf("짧은 ID 생성 재시도 한도 초과 (모든 ID가 충돌)")
		}
	}

	err = s.store.Save(shortID, originalURL)
	if err != nil {
		if _, ok := err.(*storage.ErrIDExist); ok {
			return "", fmt.Errorf("짧은 ID '%s' 저장 실패 (이미 존재함): %w", shortID, err)
		}
		return "", fmt.Errorf("URL 저장 실패: %w", err)
	}
	return shortID, nil
}

func (s *Service) GetOriginalURL(shortID string) (string, error) {
	if strings.TrimSpace(shortID) == "" {
		return "", fmt.Errorf("짧은 ID는 비어 있을 수 없습니다")
	}
	return s.store.Get(shortID)
}

func isValidURL(rawURL string) bool {
	u, err := url.ParseRequestURI(rawURL)
	if err != nil {
		return false
	}
	return u.Scheme == "http" || u.Scheme == "https"
}
