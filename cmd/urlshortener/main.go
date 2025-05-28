package main

import (
	"fmt"
	"github.com/rs/cors"
	"log"
	"miniproject/internal/api"
	"miniproject/internal/shortener"
	"miniproject/internal/storage/DB"
	"net/http"
	"os"
	"time"
)

func main() {
	// 로거 설정
	logger := log.New(os.Stdout, "urlshortener: ", log.LstdFlags|log.Lshortfile)

	// 1. 스토리지 계층 초기화 (PostgreSQL 사용)
	dbHost := os.Getenv("DB_HOST")
	dbPort := os.Getenv("DB_PORT")
	dbUser := os.Getenv("DB_USER")
	dbPassword := os.Getenv("DB_PASSWORD")
	dbName := os.Getenv("DB_NAME")

	// 데이터베이스 연결 문자열 (DSN) 생성
	// 실제 프로덕션에서는 sslmode=require 또는 verify-full 등을 사용하는 것이 좋습니다.
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPassword, dbName)

	// PostgreSQL 스토어 초기화
	pgStore, err := DB.NewPostgresStore(dsn)
	if err != nil {
		logger.Fatalf("PostgreSQL 스토리지 초기화 실패: %v", err)
	}
	logger.Println("PostgreSQL 스토리지 초기화 완료")
	// store := memory.NewMemoryStore() // 기존 인메모리 스토어 주석 처리
	store := pgStore // PostgreSQL 스토어를 사용

	// 2. 서비스 계층 (핵심 로직) 초기화
	shortenerService := shortener.NewService(store, 6) // ID 길이 6
	logger.Println("Shortener 서비스 초기화 완료")

	// 3. API 핸들러 계층 초기화
	apiHandler := api.NewHandler(shortenerService, logger)
	logger.Println("API 핸들러 초기화 완료")

	// 4. HTTP 라우터 설정
	mux := http.NewServeMux()
	mux.HandleFunc("POST /shorten", apiHandler.ShortenURL)
	mux.HandleFunc("GET /{shortID}", apiHandler.RedirectURL)

	// --- CORS 미들웨어 설정 시작 ---
	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"https://your-frontend-domain.com", "http://localhost:3000"}, // 실제 프론트엔드 도메인으로 변경
		AllowedMethods:   []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodOptions},
		AllowedHeaders:   []string{"Content-Type", "Authorization", "X-Requested-With"}, // 필요한 헤더 추가
		AllowCredentials: true,
		// Debug: true, // 개발 중 디버깅 로그 활성화
	})
	// 기존 핸들러에 CORS 미들웨어 적용
	handlerWithCORS := c.Handler(mux)
	// --- CORS 미들웨어 설정 끝 ---

	// 서버 주소 및 포트 설정
	port := os.Getenv("PORT")
	if port == "" {
		port = "3000" // 기본 포트
	}
	serverAddr := ":" + port

	// HTTP 서버 설정
	server := &http.Server{
		Addr:         serverAddr,
		Handler:      handlerWithCORS, // CORS 적용된 핸들러 사용
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Fatalf("서버 시작 실패: %v", err)
	}
}
