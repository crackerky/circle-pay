package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
)

// ========== リッチメニュー作成 ==========

// CreateCirclePayRichMenu はCirclePay用のリッチメニューを作成
func CreateCirclePayRichMenu() (*RichMenuResponse, error) {
	liffURL := os.Getenv("LIFF_URL")
	if liffURL == "" {
		liffURL = "https://liff.line.me/2008577348-GDBXaBEr"
	}

	// 3分割レイアウト（横並び）- サイズ 2500x843
	richMenu := RichMenu{
		Size: RichMenuSize{
			Width:  2500,
			Height: 843,
		},
		Selected:    true,
		Name:        "CirclePay メインメニュー",
		ChatBarText: "メニューを開く",
		Areas: []RichMenuArea{
			// 左: 支払い報告
			{
				Bounds: RichMenuBounds{
					X:      0,
					Y:      0,
					Width:  833,
					Height: 843,
				},
				Action: RichMenuAction{
					Type:  "message",
					Label: "支払い報告",
					Text:  "💰 支払いました",
				},
			},
			// 中央: 状況確認
			{
				Bounds: RichMenuBounds{
					X:      833,
					Y:      0,
					Width:  834,
					Height: 843,
				},
				Action: RichMenuAction{
					Type:  "message",
					Label: "状況確認",
					Text:  "📊 状況確認",
				},
			},
			// 右: 会計者になる（LIFFへ）
			{
				Bounds: RichMenuBounds{
					X:      1667,
					Y:      0,
					Width:  833,
					Height: 843,
				},
				Action: RichMenuAction{
					Type:  "uri",
					Label: "会計者になる",
					URI:   liffURL,
				},
			},
		},
	}

	return createRichMenu(richMenu)
}

// createRichMenu はLINE APIでリッチメニューを作成
func createRichMenu(menu RichMenu) (*RichMenuResponse, error) {
	token := os.Getenv("LINE_CHANNEL_ACCESS_TOKEN")
	if token == "" {
		return nil, fmt.Errorf("LINE_CHANNEL_ACCESS_TOKEN is not set")
	}

	body, err := json.Marshal(menu)
	if err != nil {
		return nil, fmt.Errorf("JSON marshal error: %v", err)
	}

	req, err := http.NewRequest("POST", "https://api.line.me/v2/bot/richmenu", bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("request creation error: %v", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("API request error: %v", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	var result RichMenuResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("response parse error: %v", err)
	}

	log.Printf("[リッチメニュー] 作成成功: %s", result.RichMenuID)
	return &result, nil
}

// ========== リッチメニュー画像アップロード ==========

// UploadRichMenuImage はリッチメニューに画像をアップロード（ファイルパスから）
func UploadRichMenuImage(richMenuID string, imagePath string) error {
	imageData, err := os.ReadFile(imagePath)
	if err != nil {
		return fmt.Errorf("image file read error: %v", err)
	}

	return UploadRichMenuImageData(richMenuID, imageData, "image/png")
}

// UploadRichMenuImageData はバイト配列から画像をアップロード
func UploadRichMenuImageData(richMenuID string, imageData []byte, contentType string) error {
	token := os.Getenv("LINE_CHANNEL_ACCESS_TOKEN")
	if token == "" {
		return fmt.Errorf("LINE_CHANNEL_ACCESS_TOKEN is not set")
	}

	url := fmt.Sprintf("https://api-data.line.me/v2/bot/richmenu/%s/content", richMenuID)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(imageData))
	if err != nil {
		return fmt.Errorf("request creation error: %v", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", contentType)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("API request error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	log.Printf("[リッチメニュー] 画像アップロード成功: %s", richMenuID)
	return nil
}

// ========== デフォルトリッチメニュー設定 ==========

// SetDefaultRichMenu はデフォルトリッチメニューを設定
func SetDefaultRichMenu(richMenuID string) error {
	token := os.Getenv("LINE_CHANNEL_ACCESS_TOKEN")
	if token == "" {
		return fmt.Errorf("LINE_CHANNEL_ACCESS_TOKEN is not set")
	}

	url := fmt.Sprintf("https://api.line.me/v2/bot/user/all/richmenu/%s", richMenuID)
	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return fmt.Errorf("request creation error: %v", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("API request error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	log.Printf("[リッチメニュー] デフォルト設定成功: %s", richMenuID)
	return nil
}

// ========== リッチメニュー一覧取得 ==========

// GetRichMenuList はリッチメニュー一覧を取得
func GetRichMenuList() (*RichMenuListResponse, error) {
	token := os.Getenv("LINE_CHANNEL_ACCESS_TOKEN")
	if token == "" {
		return nil, fmt.Errorf("LINE_CHANNEL_ACCESS_TOKEN is not set")
	}

	req, err := http.NewRequest("GET", "https://api.line.me/v2/bot/richmenu/list", nil)
	if err != nil {
		return nil, fmt.Errorf("request creation error: %v", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("API request error: %v", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	var result RichMenuListResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("response parse error: %v", err)
	}

	return &result, nil
}

// ========== リッチメニュー削除 ==========

// DeleteRichMenu はリッチメニューを削除
func DeleteRichMenu(richMenuID string) error {
	token := os.Getenv("LINE_CHANNEL_ACCESS_TOKEN")
	if token == "" {
		return fmt.Errorf("LINE_CHANNEL_ACCESS_TOKEN is not set")
	}

	url := fmt.Sprintf("https://api.line.me/v2/bot/richmenu/%s", richMenuID)
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("request creation error: %v", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("API request error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	log.Printf("[リッチメニュー] 削除成功: %s", richMenuID)
	return nil
}
