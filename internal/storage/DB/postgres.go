package DB

import (
	"database/sql"
	"errors"
	"fmt"
	"github.com/lib/pq" // PostgreSQL 드라이버의 에러 코드를 확인하기 위함
	"miniproject/internal/storage"
)

type PostgresStore struct {
	db *sql.DB
}

func NewPostgresStore(dataSourceName string) (*PostgresStore, error) {
	// DB 연결
	db, err := sql.Open("postgres", dataSourceName)
	if err != nil {
		return nil, fmt.Errorf("DB 연결에 실패하였습니다. : %w", err)
	}

	store := &PostgresStore{db: db}

	//스키마 초기화
	if err = store.initSchema(); err != nil {
		db.Close()
		return nil, fmt.Errorf("DB 스키마 초기화에 실패했습니다. : %w", err)
	}

	return store, nil
}

// 'urls' 테이블이 존재하지 않으면 생성
func (s *PostgresStore) initSchema() error {
	query := `
	CREATE TABLE IF NOT EXISTS urls (
	    short_id VARCHAR(255 ) PRIMARY KEY,
	    original_url TEXT NOT NULL,
	    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
	);
	CREATE INDEX IF NOT EXISTS idx_urls_created_at ON urls(created_at);
	`
	_, err := s.db.Exec(query)
	if err != nil {
		return fmt.Errorf("'urls' 테이블 생성에 실패하였습니다. : %w", err)
	}
	return nil
}

// Save 주어진 짧은 ID 와 원본 URL을 DB에 저장
// 이미 ID가 존재한다면 storage.ErrIDExists 에러를 반환
func (s *PostgresStore) Save(id string, originalURL string) error {
	query := "INSERT INTO urls (short_id, original_url) VALUES ($1, $2)"
	_, err := s.db.Exec(query, id, originalURL)
	if err != nil {
		// PostgresSQL 에러인지 확인, unique_violation (23505) 인지 검사
		if pqErr, ok := err.(*pq.Error); ok {
			if pqErr.Code == "23505" {
				return &storage.ErrIDExist{ID: id}
			}
		}
		return fmt.Errorf("URL 저장 중에 DB 에러가 발생했습니다. (id: %s): %w", id, err)
	}
	return nil
}

// Get 주어진 짧은 ID에 매핑된 URL를 DB에서 조회해 반환합니다.
// ID가 존재하지 않으면 storage.ErrNotFound 에러를 반환합니다.
func (s *PostgresStore) Get(id string) (string, error) {
	var originalURL string
	query := "SELECT original_url FROM urls WHERE short_id = $1"
	err := s.db.QueryRow(query, id).Scan(&originalURL)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", &storage.ErrNotFound{ID: id}
		}
		return "", fmt.Errorf("원본 URL 조회 중 DB 에러가 발생했습니다. (id: %s): %w", id, err)
	}
	return originalURL, nil
}

// Exists 주어진 짧은 ID가 DB에 존재하는지 확인합니다.
func (s *PostgresStore) Exists(id string) (bool, error) {
	var placeholder int // 스캔 결과를 저장할 변수, 실제 값은 중요 X
	query := "SELECT 1 FROM urls WHERE short_id = $1 LIMIT 1"
	err := s.db.QueryRow(query, id).Scan(&placeholder)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil // ID가 존재하지 않음
		}
		return false, fmt.Errorf("ID 존재 여부 확인중 DB 에러가 발생했습니다. (id: %s): %w", id, err)
	}
	return true, nil // ID가 존재 함
}

// Close DB 연결을 종료합니다. 애플리케이션 종료 시 호출 될 수 있습니다.
func (s *PostgresStore) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}
