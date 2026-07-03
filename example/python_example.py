"""Fetch a barcode PNG from barcode-rest using only the Python standard library."""

from argparse import ArgumentParser
from pathlib import Path
from urllib.error import HTTPError, URLError
from urllib.parse import urlencode
from urllib.request import urlopen


API_BASE = "http://127.0.0.1:8787"


def main() -> None:
    parser = ArgumentParser(description="Save a barcode-rest PNG.")
    parser.add_argument("text", nargs="?", default="ABC123")
    parser.add_argument(
        "--type",
        choices=("datamatrix", "qr", "code128"),
        default="qr",
        dest="barcode_type",
    )
    parser.add_argument("--output", default="barcode.png")
    args = parser.parse_args()

    params: dict[str, str | int] = {"text": args.text}
    if args.barcode_type == "code128":
        params.update(module=3, height=80, label=1)
    else:
        params["size"] = 512
        if args.barcode_type == "qr":
            params["level"] = "M"

    url = f"{API_BASE}/{args.barcode_type}?{urlencode(params)}"

    try:
        with urlopen(url, timeout=10) as response:
            png = response.read()
    except HTTPError as error:
        detail = error.read().decode("utf-8", errors="replace")
        raise SystemExit(f"barcode-rest returned HTTP {error.code}: {detail}") from error
    except URLError as error:
        raise SystemExit(f"Could not connect to barcode-rest: {error.reason}") from error

    output = Path(args.output)
    output.write_bytes(png)
    print(f"Saved {output.resolve()}")


if __name__ == "__main__":
    main()
