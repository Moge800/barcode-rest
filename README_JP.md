# barcode-rest

[English version / 英語版はこちら](README.md)

ローカル専用の帳票向けバーコード画像生成REST API。

Excelの出荷票・現品票などに貼り付けるバーコードのPNG画像を、
`127.0.0.1` 上のHTTP GETで返すだけのツールです。Excel VBAからの利用を想定しています。

## 特徴と制約

- 2D: DataMatrix / QR / Aztec / PDF417、1D: Code128 / Code39 / Code93 / Codabar / ITF / Code25 / EAN-13(JAN) / EAN-8 に対応
- 返却形式はPNGのみ（BMP / SVG / JPEGなし）
- `127.0.0.1` のみで待ち受け（`0.0.0.0` では待ち受けない）
- 外部通信は行わない
- サーバー側で画像ファイルを保存しない（PNGはメモリ上で生成して返す）
- リクエストでファイルパスを受け取らない
- `text` の内容はログに出力しない

## 起動方法

```powershell
barcode-rest.exe
```

ポート変更:

```powershell
barcode-rest.exe -port 8787
```

バージョン表示:

```powershell
barcode-rest.exe -version
```

### 自動起動（常駐）

`shell:startup` フォルダに `barcode-rest.exe` のショートカットを置くだけでよい。
ダブルクリックやスタートアップからの起動時はコンソールウィンドウを自動で非表示にする
（シェルから実行した場合は通常どおり表示される）。停止は `taskkill /im barcode-rest.exe`。

## エンドポイント

### GET /health

起動確認用。HTTP 200で以下を返す。

```json
{"ok": true, "name": "barcode-rest", "version": "v0.1.0"}
```

`version` はビルド時に埋め込まれるリリースタグ（ローカルビルドでは `dev`）。

### GET /datamatrix

DataMatrix画像をPNGで返す。

| パラメータ | 必須 | デフォルト | 内容 |
|---|---|---|---|
| `text` | 必須 | なし | 埋め込む文字列（UTF-8で最大128バイト） |
| `module` | 任意 | `10` | 1モジュールあたりのピクセル数（2〜32） |
| `quiet` | 任意 | `4` | 周囲余白のモジュール数（0〜16） |
| `size` | 任意 | なし | 出力画像の一辺のピクセル数（16〜2048）。指定時は `module` を無視し、収まる最大の整数モジュールで描画して白余白で中央寄せする |

### GET /qr

QRコード画像をPNGで返す。

| パラメータ | 必須 | デフォルト | 内容 |
|---|---|---|---|
| `text` | 必須 | なし | 埋め込む文字列（UTF-8で最大256バイト） |
| `module` | 任意 | `10` | 1モジュールあたりのピクセル数（2〜32） |
| `quiet` | 任意 | `4` | 周囲余白のモジュール数（0〜16） |
| `level` | 任意 | `M` | 誤り訂正レベル（L / M / Q / H） |
| `size` | 任意 | なし | `/datamatrix` と同じ |

### GET /aztec

Aztecコード（2D）をPNGで返す。パラメータは `/datamatrix` と同じ（`text` は最大256バイト）。誤り訂正は33%固定。

### GET /pdf417

PDF417（スタック型2D）をPNGで返す。

| パラメータ | 必須 | デフォルト | 内容 |
|---|---|---|---|
| `text` | 必須 | なし | 埋め込む文字列（UTF-8で最大256バイト） |
| `module` | 任意 | `3` | 1モジュールの幅ピクセル数（2〜32）。行の高さは自動で2モジュール分 |
| `quiet` | 任意 | `2` | 周囲余白のモジュール数（0〜16） |
| `level` | 任意 | `2` | セキュリティレベル（0〜8） |

`size` は非対応（横長のため）。

### 1Dバーコード

以下のエンドポイントは共通パラメータ。

```text
GET /code128   ASCII文字列（最大80文字）
GET /code39    大文字英数字と記号 -. $/+% （最大128バイト）
GET /code93    大文字英数字と記号 -. $/+% （最大128バイト、チェックサム付き）
GET /codabar   スタート/ストップ文字A〜D + 数字・-$:/.+（例: A12345B）
GET /itf       偶数桁の数字（Interleaved 2 of 5）
GET /code25    数字（Standard 2 of 5）
GET /ean13     12桁（チェックデジット自動計算）または13桁の数字（JANコード）
GET /ean8      7桁（チェックデジット自動計算）または8桁の数字
```

| パラメータ | 必須 | デフォルト | 内容 |
|---|---|---|---|
| `text` | 必須 | なし | 埋め込む文字列 |
| `module` | 任意 | `3` | 1モジュール（最細バー）の幅ピクセル数（2〜32） |
| `height` | 任意 | `80` | バーの高さピクセル数（20〜600） |
| `quiet` | 任意 | `10` | 左右余白のモジュール数（0〜16）。上下にも同じピクセル数の余白が付く |
| `fullascii` | 任意 | `0` | `/code39` `/code93` のみ（他のエンドポイントではHTTP 400）。`1` でExtendedモード（小文字などを `+N` 等のペアで符号化）。読み取り側スキャナも拡張モード対応が必要 |
| `label` | 任意 | `0` | `1` でバーの下にヒューマンリーダブル文字を描画（ビットマップフォントの整数拡大）。スキャナが読む値を表示するため、EANのチェックデジットも含まれる |

`size` は非対応（横長のため）。

### エラー

- パラメータ不正: HTTP 400 `{"ok": false, "error": "..."}`
- 出力画像が約16メガピクセル超（極端なmodule/height/quietの組み合わせ）: HTTP 400
- 各シンボロジーで使えない文字・桁数・チェックデジット不正: HTTP 400
- GET以外のメソッド: HTTP 405
- 未定義パス: HTTP 404

## 使用例

```text
http://127.0.0.1:8787/health
http://127.0.0.1:8787/datamatrix?text=ABC123
http://127.0.0.1:8787/qr?text=ABC123
http://127.0.0.1:8787/code128?text=ABC123
http://127.0.0.1:8787/ean13?text=490123456789
```

## Excel VBAからの利用イメージ

1. セルからバーコード化したい文字列を取得
2. `http://127.0.0.1:8787/datamatrix?text=...` をGET
3. 返却PNGをブック直下の `barcode_images` などに保存
4. `Shapes.AddPicture`（`LinkToFile:=msoFalse`, `SaveWithDocument:=msoTrue`）でセル上に貼り付け

そのままインポートできるVBA標準モジュールのサンプルは
[`example/vba_example.bas`](example/vba_example.bas) にあります。

## ビルド

```powershell
go build -o barcode-rest.exe
```

## 注意事項

- 本ツールはローカルPC上での帳票補助専用です。社内LANに公開しないでください。
- 認証・HTTPSはありません（127.0.0.1専用のため実装しない方針）。

## ライセンス

MIT License

バーコード生成に [github.com/boombuler/barcode](https://github.com/boombuler/barcode) (MIT License)、
ラベルフォントに [golang.org/x/image](https://pkg.go.dev/golang.org/x/image) (BSD-3-Clause) を使用しています。
