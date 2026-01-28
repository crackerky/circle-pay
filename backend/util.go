package main

import (
	"fmt"
	"html"
	"strings"
)

// ========== ユーティリティ関数 ==========

// sanitizeInput はサニタイズをする関数
func sanitizeInput(input string) string {
	sanitized := html.EscapeString(input)
	sanitized = strings.TrimSpace(sanitized)
	return sanitized
}

// formatAmount は金額フォーマット
func formatAmount(amount int) string {
	return fmt.Sprintf("%d", amount)
}
