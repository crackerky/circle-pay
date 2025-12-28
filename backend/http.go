package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
)

// ========== HTTP処理の共通化 ==========

// APIContext は認証済みリクエストのコンテキストを保持
type APIContext struct {
	UserID      string
	DisplayName string
	Writer      http.ResponseWriter
	Request     *http.Request
}

// AuthenticatedHandler は認証が必要なハンドラーの型
type AuthenticatedHandler func(ctx *APIContext)

// WithAuth は認証を行うミドルウェア
func WithAuth(handler AuthenticatedHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, displayName, err := authenticateRequest(r)
		if err != nil {
			log.Printf("認証エラー: %v", err)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		ctx := &APIContext{
			UserID:      userID,
			DisplayName: displayName,
			Writer:      w,
			Request:     r,
		}

		handler(ctx)
	}
}

// ========== リクエスト処理ヘルパー ==========

// DecodeJSON はリクエストボディをJSONとしてデコードする
func (ctx *APIContext) DecodeJSON(v interface{}) bool {
	if err := json.NewDecoder(ctx.Request.Body).Decode(v); err != nil {
		log.Printf("JSONデコードエラー: %v", err)
		http.Error(ctx.Writer, "Invalid JSON", http.StatusBadRequest)
		return false
	}
	return true
}

// ========== レスポンス処理ヘルパー ==========

// JSON はJSONレスポンスを返す
func (ctx *APIContext) JSON(data interface{}) {
	ctx.Writer.Header().Set("Content-Type", "application/json")
	json.NewEncoder(ctx.Writer).Encode(data)
}

// Success は成功レスポンスを返す
func (ctx *APIContext) Success(data map[string]interface{}) {
	if data == nil {
		data = make(map[string]interface{})
	}
	data["status"] = "ok"
	ctx.JSON(data)
}

// Error はエラーレスポンスを返す
func (ctx *APIContext) Error(message string, statusCode int) {
	http.Error(ctx.Writer, message, statusCode)
}

// ========== LIFF認証関数 ==========

// authenticateRequest はHTTPリクエストからLIFFトークンを検証し、userIDを取得
func authenticateRequest(r *http.Request) (string, string, error) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return "", "", errMissingAuth
	}

	token := strings.TrimPrefix(authHeader, "Bearer ")
	if token == authHeader {
		return "", "", errInvalidAuth
	}

	return verifyLIFFToken(token)
}

// エラー定義
var (
	errMissingAuth = &authError{"authorization header is missing"}
	errInvalidAuth = &authError{"invalid authorization header format"}
)

type authError struct {
	message string
}

func (e *authError) Error() string {
	return e.message
}
