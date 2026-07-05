# barcode-rest

[日本語版はこちら / Japanese version](README_JP.md)

Local-only barcode image REST API for spreadsheets and printed forms.

Returns DataMatrix / QR / 1D barcode PNG images over HTTP GET on `127.0.0.1`,
intended to be called from Excel VBA when printing shipping labels, item tags
and similar forms.

## Features and constraints

- 2D: DataMatrix / QR / Aztec / PDF417 — 1D: Code128 / Code39 / Code93 / Codabar / ITF / Code25 / EAN-13(JAN) / EAN-8
- PNG output only (no BMP / SVG / JPEG)
- Listens on `127.0.0.1` only (never `0.0.0.0`)
- No outbound network access
- Never writes image files server-side (PNGs are generated in memory)
- Never accepts file paths in requests
- Never logs `text` contents

## Usage

```powershell
barcode-rest.exe
```

Change port:

```powershell
barcode-rest.exe -port 9999
```

Print version:

```powershell
barcode-rest.exe -version
```

### One-shot CLI generation

Generate a PNG directly, without starting the server:

```powershell
barcode-rest.exe generate datamatrix --text ABC123 --size 256 --output dm.png
barcode-rest.exe generate code128 --text ABC123 --label --output c128.png
```

- Same symbologies and parameters as the HTTP endpoints
  (`--module`, `--quiet`, `--size`, `--height`, `--level`, `--fullascii`, `--label`)
- `--output -` writes the PNG to stdout for pipelines
  (note: PowerShell's `>` is not binary-safe — use a file path there)
- The HTTP API still never accepts file paths; only the CLI writes files,
  and only where you explicitly specify

### Auto-start (resident)

Just put a shortcut to `barcode-rest.exe` in the `shell:startup` folder.
When launched by double-click or startup, the console window hides itself
automatically (it stays visible when run from a shell).
Stop with `taskkill /im barcode-rest.exe`.

## Endpoints

### GET /health

Liveness check. Returns HTTP 200 with:

```json
{"ok": true, "name": "barcode-rest", "version": "v0.1.0"}
```

`version` is the release tag embedded at build time (`dev` for local builds).

### GET /datamatrix

Returns a DataMatrix PNG.

| Param | Required | Default | Description |
|---|---|---|---|
| `text` | yes | — | String to encode (max 128 bytes UTF-8) |
| `module` | no | `10` | Pixels per module (2–32) |
| `quiet` | no | `4` | Quiet-zone modules around the code (0–16) |
| `size` | no | — | Output edge length in px (16–2048). Overrides `module`: draws at the largest integer module that fits, centered with white padding. The minimum usable size depends on the encoded data, symbol type, and quiet zone — HTTP 400 is returned if the symbol cannot fit |

### GET /qr

Returns a QR code PNG.

| Param | Required | Default | Description |
|---|---|---|---|
| `text` | yes | — | String to encode (max 256 bytes UTF-8) |
| `module` | no | `10` | Pixels per module (2–32) |
| `quiet` | no | `4` | Quiet-zone modules around the code (0–16) |
| `level` | no | `M` | Error correction level (L / M / Q / H) |
| `size` | no | — | Same as `/datamatrix` |

### GET /aztec

Returns an Aztec code PNG. Same parameters as `/datamatrix`
(`text` max 256 bytes). Error correction is fixed at 33%.

### GET /pdf417

Returns a PDF417 (stacked 2D) PNG.

| Param | Required | Default | Description |
|---|---|---|---|
| `text` | yes | — | String to encode (max 256 bytes UTF-8) |
| `module` | no | `3` | Module width in px (2–32). Row height is automatically 2 modules |
| `quiet` | no | `2` | Quiet-zone modules (0–16) |
| `level` | no | `2` | Security level (0–8) |

`size` is not supported (the symbol is not square).

### 1D barcodes

These endpoints share common parameters.

```text
GET /code128   any ASCII string (max 80 chars)
GET /code39    uppercase letters, digits and -. $/+% (max 128 bytes)
GET /code93    uppercase letters, digits and -. $/+% (max 128 bytes, with checksum)
GET /codabar   start/stop chars A-D with digits or -$:/.+ between (e.g. A12345B, max 128 bytes)
GET /itf       even number of digits (Interleaved 2 of 5, max 128 bytes)
GET /code25    digits (Standard 2 of 5, max 128 bytes)
GET /ean13     12 digits (check digit computed) or 13 digits (JAN code)
GET /ean8      7 digits (check digit computed) or 8 digits
```

| Param | Required | Default | Description |
|---|---|---|---|
| `text` | yes | — | String to encode |
| `module` | no | `3` | Narrow-bar width in px (2–32) |
| `height` | no | `80` | Bar height in px (20–600) |
| `quiet` | no | `10` | Quiet-zone modules left/right (0–16); the same pixel margin is added top/bottom |
| `fullascii` | no | `0` | `/code39` and `/code93` only (HTTP 400 elsewhere). `1` enables Extended mode (encodes lowercase etc. as `+N` pairs). The scanner must also support extended mode |
| `label` | no | `0` | `1` draws the human-readable content below the bars (bitmap font, integer-scaled). Shows what a scanner reads — e.g. EAN check digits are included |

`size` is not supported (the symbols are not square).

### Errors

- Invalid parameters: HTTP 400 `{"ok": false, "error": "..."}`
- Output image over ~16 megapixels (extreme module/height/quiet combinations or a large label): HTTP 400
- Characters/length/check-digit not valid for the symbology: HTTP 400
- Non-GET methods: HTTP 405
- Unknown paths: HTTP 404

## Examples

```text
http://127.0.0.1:8787/health
http://127.0.0.1:8787/datamatrix?text=ABC123
http://127.0.0.1:8787/qr?text=ABC123
http://127.0.0.1:8787/code128?text=ABC123
http://127.0.0.1:8787/ean13?text=490123456789
```

## Using from Excel VBA

1. Read the string to encode from a cell
2. GET `http://127.0.0.1:8787/datamatrix?text=...`
3. Save the returned PNG to a temporary file under `%TEMP%`
4. Embed it over the target cell with `Shapes.AddPicture`
   (`LinkToFile:=msoFalse`, `SaveWithDocument:=msoTrue`)
5. Delete the temporary PNG after embedding it

Ready-to-run examples:

- [Excel VBA standard module](example/vba_example.bas) (Windows Excel only — uses MSXML2.XMLHTTP and ADODB.Stream)
- [Python client](example/python_example.py) (standard library only; run with `uv run example/python_example.py`)
- [HTML client](example/html_example.html) (open directly in a browser)

If barcode-rest is started with a different port, update `API_BASE` in the example.

## Build

Requires Go 1.26 or later.

```powershell
go build -o barcode-rest.exe
```

## Notes

- This tool is for local form-printing assistance only. Do not expose it on a LAN.
- No authentication and no HTTPS (by design — `127.0.0.1` only).

## License

MIT License

Barcode encoding by [github.com/boombuler/barcode](https://github.com/boombuler/barcode) (MIT License).
Label font from [golang.org/x/image](https://pkg.go.dev/golang.org/x/image) (BSD-3-Clause).
