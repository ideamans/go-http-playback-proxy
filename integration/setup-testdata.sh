#!/bin/bash

# テストデータ作成スクリプト
# ImageMagickが必要: brew install imagemagick

set -e

TESTDATA_DIR="$(cd "$(dirname "$0")" && pwd)/testdata"
IMAGE_DIR="$TESTDATA_DIR/images"
HTML_DIR="$TESTDATA_DIR/html"
CSS_DIR="$TESTDATA_DIR/css"
JS_DIR="$TESTDATA_DIR/js"
API_DIR="$TESTDATA_DIR/api"

echo "Creating test data directories..."
mkdir -p "$IMAGE_DIR" "$HTML_DIR" "$CSS_DIR" "$JS_DIR" "$API_DIR"

# ImageMagickのチェック
if ! command -v magick &> /dev/null; then
    echo "Error: ImageMagick is required. Please install with: brew install imagemagick"
    exit 1
fi

echo "Generating test images..."

# 小さい画像 (1KB未満)
magick -size 50x50 xc:red "$IMAGE_DIR/small.jpg"
magick -size 50x50 xc:blue "$IMAGE_DIR/small.png"
# SVGは別の方法で生成
cat > "$IMAGE_DIR/small.svg" << 'EOF'
<svg width="50" height="50" xmlns="http://www.w3.org/2000/svg">
  <rect width="50" height="50" fill="green"/>
</svg>
EOF

# 中サイズ画像 (1KB-100KB)
magick -size 300x200 plasma: "$IMAGE_DIR/medium.jpg"
magick -size 300x200 plasma: "$IMAGE_DIR/medium.png"
magick -size 300x200 plasma: "$IMAGE_DIR/medium.webp"

# 大きい画像 (100KB-10MB)
magick -size 1920x1080 plasma: -quality 90 "$IMAGE_DIR/large.jpg"
magick -size 1920x1080 plasma: "$IMAGE_DIR/large.png"

# 日本語ファイル名の画像
magick -size 100x100 xc:yellow "$IMAGE_DIR/日本語ファイル名.jpg"
magick -size 100x100 xc:magenta "$IMAGE_DIR/特殊文字-テスト_画像.png"

echo "Generating HTML test files..."

# UTF-8 HTML
cat > "$HTML_DIR/utf8.html" << 'EOF'
<!DOCTYPE html>
<html lang="ja">
<head>
    <meta charset="UTF-8">
    <title>UTF-8テストページ</title>
</head>
<body>
    <h1>日本語テストページ</h1>
    <p>これはUTF-8エンコーディングのテストです。</p>
    <p>特殊文字: ♠♣♥♦ ©®™ αβγδε</p>
</body>
</html>
EOF

# Shift_JIS HTML (iconvで変換)
iconv -f UTF-8 -t SHIFT_JIS << 'EOF' > "$HTML_DIR/shift_jis.html"
<!DOCTYPE html>
<html lang="ja">
<head>
    <meta charset="Shift_JIS">
    <title>Shift_JISテストページ</title>
</head>
<body>
    <h1>Shift_JIS日本語テスト</h1>
    <p>これはShift_JISエンコーディングです。</p>
</body>
</html>
EOF

# EUC-JP HTML
iconv -f UTF-8 -t EUC-JP << 'EOF' > "$HTML_DIR/euc_jp.html"
<!DOCTYPE html>
<html lang="ja">
<head>
    <meta charset="EUC-JP">
    <title>EUC-JPテストページ</title>
</head>
<body>
    <h1>EUC-JP日本語テスト</h1>
    <p>これはEUC-JPエンコーディングです。</p>
</body>
</html>
EOF

# ISO-8859-1 HTML
cat > "$HTML_DIR/iso8859.html" << 'EOF'
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="ISO-8859-1">
    <title>ISO-8859-1 Test Page</title>
</head>
<body>
    <h1>European Characters Test</h1>
    <p>Café, naïve, résumé, piñata</p>
    <p>àáâãäåæçèéêëìíîïñòóôõöøùúûüý</p>
</body>
</html>
EOF

echo "Generating CSS test files..."

# UTF-8 CSS
cat > "$CSS_DIR/utf8.css" << 'EOF'
@charset "UTF-8";
/* UTF-8 CSSファイル */
.japanese-text {
    font-family: "ヒラギノ角ゴ Pro", sans-serif;
    /* 日本語コメント: スタイル定義 */
}
.special-chars::before {
    content: "★☆♪♫";
}
EOF

# Shift_JIS CSS
iconv -f UTF-8 -t SHIFT_JIS << 'EOF' > "$CSS_DIR/shift_jis.css"
@charset "Shift_JIS";
/* Shift_JIS CSSファイル */
.japanese-font {
    font-family: "ＭＳ ゴシック", monospace;
}
EOF

echo "Generating JavaScript test files..."

# UTF-8 JavaScript
cat > "$JS_DIR/utf8.js" << 'EOF'
// UTF-8 JavaScript
const message = "こんにちは、世界！";
const specialChars = "★☆♪♫©®™";
console.log(message, specialChars);

function greet(name) {
    return `こんにちは、${name}さん！`;
}
EOF

# Shift_JIS JavaScript
iconv -f UTF-8 -t SHIFT_JIS << 'EOF' > "$JS_DIR/shift_jis.js"
// Shift_JIS JavaScript
var message = "こんにちは";
console.log(message);
EOF

echo "Generating API test data..."

# JSON API レスポンス
cat > "$API_DIR/users.json" << 'EOF'
{
    "users": [
        {"id": 1, "name": "田中太郎", "email": "tanaka@example.com"},
        {"id": 2, "name": "佐藤花子", "email": "sato@example.com"},
        {"id": 3, "name": "山田次郎", "email": "yamada@example.com"}
    ],
    "meta": {
        "total": 3,
        "page": 1,
        "timestamp": "2024-01-01T00:00:00Z"
    }
}
EOF

# 大きなJSONファイル (性能テスト用)
cat > "$API_DIR/large_data.json" << 'EOF'
{
    "data": [
EOF

# 10000個のエントリを生成
for i in {1..10000}; do
    if [ $i -eq 10000 ]; then
        echo "        {\"id\": $i, \"name\": \"ユーザー$i\", \"description\": \"これは大きなデータセットのテストエントリです。パフォーマンステストに使用されます。\"}" >> "$API_DIR/large_data.json"
    else
        echo "        {\"id\": $i, \"name\": \"ユーザー$i\", \"description\": \"これは大きなデータセットのテストエントリです。パフォーマンステストに使用されます。\"}," >> "$API_DIR/large_data.json"
    fi
done

cat >> "$API_DIR/large_data.json" << 'EOF'
    ],
    "meta": {
        "count": 10000,
        "size": "large"
    }
}
EOF

# XMLデータ
cat > "$API_DIR/sample.xml" << 'EOF'
<?xml version="1.0" encoding="UTF-8"?>
<root>
    <users>
        <user id="1">
            <name>田中太郎</name>
            <email>tanaka@example.com</email>
        </user>
        <user id="2">
            <name>佐藤花子</name>
            <email>sato@example.com</email>
        </user>
    </users>
</root>
EOF

echo "Generating binary test files..."

# 小さなPDFファイル (ダミー)
echo "%PDF-1.4
1 0 obj
<<
/Type /Catalog
/Pages 2 0 R
>>
endobj
2 0 obj
<<
/Type /Pages
/Kids [3 0 R]
/Count 1
>>
endobj
3 0 obj
<<
/Type /Page
/Parent 2 0 R
/MediaBox [0 0 612 792]
>>
endobj
xref
0 4
0000000000 65535 f 
0000000010 00000 n 
0000000079 00000 n 
0000000173 00000 n 
trailer
<<
/Size 4
/Root 1 0 R
>>
startxref
274
%%EOF" > "$TESTDATA_DIR/sample.pdf"

# ZIPファイル
cd "$TESTDATA_DIR"
zip -q sample.zip html/utf8.html css/utf8.css
cd - > /dev/null

echo "Test data generation completed!"
echo "Generated files:"
ls -la "$TESTDATA_DIR"
echo ""
echo "Images:"
ls -la "$IMAGE_DIR"
echo ""
echo "HTML files:"
ls -la "$HTML_DIR"
echo ""
echo "File sizes:"
du -h "$TESTDATA_DIR"/*