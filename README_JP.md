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

## ダウンロード

Windows / Linux 向けのビルド済みバイナリは、各GitHub Releaseに添付されています。

**https://github.com/moge800/barcode-rest/releases/latest**

各リリースには公開バイナリの `checksums.txt`（SHA-256）が含まれます。
ソースからビルドする場合は [ビルド](#ビルド) を参照してください。

## 起動方法

```powershell
barcode-rest.exe
```

ポート変更:

```powershell
barcode-rest.exe -port 9999
```

バージョン表示:

```powershell
barcode-rest.exe -version
```

exit token を固定する（未指定なら起動時にランダム生成して表示）— [POST /exit](#post-exit) を参照:

```powershell
barcode-rest.exe -exit-token abc123
```

### CLIで単発生成

サーバーを起動せずにPNGを直接生成できる。

```powershell
barcode-rest.exe generate datamatrix --text ABC123 --size 256 --output dm.png
barcode-rest.exe generate code128 --text ABC123 --label --output c128.png
```

- シンボロジーとパラメータはHTTPエンドポイントと同じ
  （`--module`, `--quiet`, `--size`, `--height`, `--level`, `--fullascii`, `--label`）
- `--output -` でstdoutにPNGを出力（パイプ用。PowerShellの `>` はバイナリを壊すので、その場合はファイルパス指定を使う）
- HTTP API側は今後もファイルパスを受け取らない。ファイルを書くのはCLIだけで、書き先は明示指定した場所のみ

### 自動起動（常駐）

`shell:startup` フォルダに `barcode-rest.exe` のショートカットを置くだけでよい。
ダブルクリックやスタートアップからの起動時はコンソールウィンドウを自動で非表示にする
（シェルから実行した場合は通常どおり表示される）。停止は `POST /exit?token=...`（下記参照）、
Ctrl-C、または `taskkill /im barcode-rest.exe`。

スタートアップ起動したインスタンスを `POST /exit` で止めたい場合は、ショートカット側に
固定tokenを指定しておく（`barcode-rest.exe -exit-token <token>`）。未指定時のランダムtokenは
コンソール出力にのみ表示されるが、ダブルクリック／スタートアップ起動ではコンソールが非表示になるため。

## エンドポイント

### GET /health

起動確認用。HTTP 200で以下を返す。

```json
{"ok": true, "name": "barcode-rest", "version": "v0.1.0"}
```

`version` はビルド時に埋め込まれるリリースタグ（ローカルビルドでは `dev`）。

### POST /exit

**`barcode-rest` プロセス**を正常終了する（WindowsやPC自体をシャットダウンするものではなく、
このプログラムを終了するだけ）。HTTP 200で以下を返し、

```json
{"ok": true}
```

その後、新規接続の受け付けを止め、処理中のリクエストを捌ききってからプロセスを終了する。
`barcode-rest` を常駐ヘルパーとして起動する呼び出し側（例: `barcodekit`）が、
用済み時にきれいに停止させる用途を想定している。

token が必須:

| パラメータ | 必須 | 内容 |
|---|---|---|
| `token` | 必須 | サーバーの exit token と一致する必要がある。起動時に `-exit-token <token>` で固定値を渡すか、起動時に表示される `exit token:` 行のランダム値を使う。なし・不一致は HTTP 403 |

閲覧中のWebページからサーバーをうっかりまたはCSRF的に落とされないよう、2段構えで守る:
`POST` のみ対応（`GET /exit` は HTTP 405 なので、ブラウザでのアクセス・リンクプレビュー・
誤クリックでは何も起きない）。加えて、外部サイトの `fetch`／フォームPOSTが `127.0.0.1` に
到達しても token を推測できないため HTTP 403 になる。Ctrl-C / SIGTERM でも同じ正常終了を行う。

```powershell
# 呼び出し側が token を指定する場合
barcode-rest.exe -exit-token abc123
curl -X POST "http://127.0.0.1:8787/exit?token=abc123"
```

### GET /datamatrix

DataMatrix画像をPNGで返す。

| パラメータ | 必須 | デフォルト | 内容 |
|---|---|---|---|
| `text` | 必須 | なし | 埋め込む文字列（UTF-8で最大3116バイト） |
| `module` | 任意 | `10` | 1モジュールあたりのピクセル数（2〜32） |
| `quiet` | 任意 | `4` | 周囲余白のモジュール数（0〜16） |
| `size` | 任意 | なし | 出力画像の一辺のピクセル数（16〜2048）。指定時は `module` を無視し、収まる最大の整数モジュールで描画して白余白で中央寄せする。実際に使用できる最小サイズはデータ量・シンボル形式・quietの値によって異なり、バーコードが収まらない場合はHTTP 400を返す |

### GET /qr

QRコード画像をPNGで返す。

| パラメータ | 必須 | デフォルト | 内容 |
|---|---|---|---|
| `text` | 必須 | なし | 埋め込む文字列（UTF-8で最大7089バイト） |
| `module` | 任意 | `10` | 1モジュールあたりのピクセル数（2〜32） |
| `quiet` | 任意 | `4` | 周囲余白のモジュール数（0〜16） |
| `level` | 任意 | `M` | 誤り訂正レベル（L / M / Q / H） |
| `size` | 任意 | なし | `/datamatrix` と同じ |

### GET /aztec

Aztecコード（2D）をPNGで返す。パラメータは `/datamatrix` と同じ（`text` は最大3748バイト）。誤り訂正は33%固定。

### GET /pdf417

PDF417（スタック型2D）をPNGで返す。

| パラメータ | 必須 | デフォルト | 内容 |
|---|---|---|---|
| `text` | 必須 | なし | 埋め込む文字列（UTF-8で最大2610バイト） |
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
GET /codabar   スタート/ストップ文字A〜D + 数字・-$:/.+（例: A12345B、最大128バイト）
GET /itf       偶数桁の数字（Interleaved 2 of 5、最大128バイト）
GET /code25    数字（Standard 2 of 5、最大128バイト）
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
- テキストが長すぎる: HTTP 400。上記の2Dの `text` バイト上限は各規格の最も密なモード（数字）での最大収容量です。
  英字・バイナリ・QRの誤り訂正レベルが高い場合はより少ない文字数しか入らず、バイト上限に収まっていても
  実際のシンボルに入り切らないテキストは 400 で返します（500 にはしません）
- 出力画像が約16メガピクセル超（極端なmodule/height/quietの組み合わせや大きなラベル）: HTTP 400
- 各シンボロジーで使えない文字・桁数・チェックデジット不正: HTTP 400
- エンドポイントに対する不正なメソッド（生成エンドポイントへのGET以外、`/exit` へのPOST以外）: HTTP 405
- `POST /exit` で `token` がない・一致しない: HTTP 403
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
3. 返却PNGを `%TEMP%` 配下の一時ファイルとして保存
4. `Shapes.AddPicture`（`LinkToFile:=msoFalse`, `SaveWithDocument:=msoTrue`）でセル上に埋め込み
5. 埋め込み後に一時PNGを削除

すぐに試せるサンプル:

- [Excel VBA標準モジュール](example/vba_example.bas)（MSXML2.XMLHTTPとADODB.Streamを使用するためWindows版Excel向け）
- [Pythonクライアント](example/python_example.py)（標準ライブラリのみ。`uv run example/python_example.py` で実行）
- [HTMLクライアント](example/html_example.html)（ブラウザで直接開く）
- [CLIスクリプト](example/cli_example.ps1)（PowerShell。サーバーなしで `generate` によりPNG生成。Linuxバイナリでも同じフラグが使える）

barcode-restを別ポートで起動する場合は、各サンプルの `API_BASE` も変更する。

## ビルド

Go 1.26以降が必要。

```powershell
go build -o barcode-rest.exe
```

## 注意事項

- 本ツールはローカルPC上での帳票補助専用です。社内LANに公開しないでください。
- 認証・HTTPSはありません（127.0.0.1専用のため実装しない方針）。

## ライセンス

Apache License 2.0 — [LICENSE](LICENSE) および [NOTICE](NOTICE) を参照。

v0.2.x 以前のリリースは MIT License で配布していました。v0.3.0 以降は Apache-2.0 です。

サードパーティコンポーネント（変更なし）:

- バーコード生成: [github.com/boombuler/barcode](https://github.com/boombuler/barcode) (MIT License)
- ラベルフォント: [golang.org/x/image](https://pkg.go.dev/golang.org/x/image) (BSD-3-Clause)
- Go 標準ライブラリ / ランタイム (BSD-3-Clause)
