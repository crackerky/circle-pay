#!/bin/bash

# CirclePay リッチメニュー画像生成スクリプト
# 必要: ImageMagick

OUTPUT_FILE="${1:-richmenu.png}"
WIDTH=2500
HEIGHT=843
# 2500 = 833 + 834 + 833
SECTION_LEFT=833
SECTION_CENTER=834
SECTION_RIGHT=833

# 色設定
BG_LEFT="#4CAF50"      # 緑（支払い報告）
BG_CENTER="#2196F3"    # 青（状況確認）
BG_RIGHT="#FF9800"     # オレンジ（会計者になる）
TEXT_COLOR="#FFFFFF"   # 白

# フォント設定
FONT="Droid-Sans-Fallback"

echo "リッチメニュー画像を生成中..."

# 3つのセクションを生成して結合
convert -size ${SECTION_LEFT}x${HEIGHT} xc:"${BG_LEFT}" \
    -gravity center \
    -font "${FONT}" \
    -pointsize 80 \
    -fill "${TEXT_COLOR}" \
    -annotate +0-50 "💰" \
    -pointsize 48 \
    -annotate +0+80 "支払い報告" \
    section_left.png

convert -size ${SECTION_CENTER}x${HEIGHT} xc:"${BG_CENTER}" \
    -gravity center \
    -font "${FONT}" \
    -pointsize 80 \
    -fill "${TEXT_COLOR}" \
    -annotate +0-50 "📊" \
    -pointsize 48 \
    -annotate +0+80 "状況確認" \
    section_center.png

convert -size ${SECTION_RIGHT}x${HEIGHT} xc:"${BG_RIGHT}" \
    -gravity center \
    -font "${FONT}" \
    -pointsize 80 \
    -fill "${TEXT_COLOR}" \
    -annotate +0-50 "👤" \
    -pointsize 48 \
    -annotate +0+80 "会計者になる" \
    section_right.png

# 結合
convert section_left.png section_center.png section_right.png +append \
    -bordercolor "#FFFFFF" \
    "${OUTPUT_FILE}"

# 一時ファイル削除
rm -f section_left.png section_center.png section_right.png

# サイズ確認
if [ -f "${OUTPUT_FILE}" ]; then
    SIZE=$(identify -format "%wx%h" "${OUTPUT_FILE}")
    FILESIZE=$(du -h "${OUTPUT_FILE}" | cut -f1)
    echo "生成完了: ${OUTPUT_FILE}"
    echo "サイズ: ${SIZE}"
    echo "ファイルサイズ: ${FILESIZE}"
else
    echo "エラー: 画像生成に失敗しました"
    exit 1
fi
