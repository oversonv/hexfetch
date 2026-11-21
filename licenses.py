#!/usr/bin/env python3
"""Rebuild the Go dependency section of THIRD_PARTY_LICENSES from the module cache.

python3 licenses.py          rewrite the file
python3 licenses.py --check  fail if the file is out of date (used by release.py)
"""
import subprocess
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parent
TARGET = ROOT / "THIRD_PARTY_LICENSES"
MARKER = "=== Go dependencies (compiled into the binary) ==="
PREFIXES = ("LICENSE", "LICENCE", "COPYING", "NOTICE")
TEMPLATE = "{{if and .Module (not .Module.Main)}}{{.Module.Path}}|{{.Module.Version}}|{{.Module.Dir}}{{end}}"


def modules():
    result = subprocess.run(
        ["go", "list", "-deps", "-f", TEMPLATE, "./cmd/hexfetch"],
        cwd=ROOT, capture_output=True, text=True,
    )
    if result.returncode:
        sys.exit("go list failed:\n" + result.stderr)
    found = {}
    for line in result.stdout.splitlines():
        if line.strip():
            path, version, directory = line.split("|")
            found[path] = (version, Path(directory))
    return sorted(found.items())


def license_files(directory):
    files = [f for f in sorted(directory.iterdir()) if f.is_file() and f.name.upper().startswith(PREFIXES)]
    return files


def build():
    text = TARGET.read_text(encoding="utf-8")
    base = text.split(MARKER)[0].rstrip() + "\n\n"
    parts = [base + MARKER + "\n"]
    rule = "-" * 70
    for path, (version, directory) in modules():
        files = license_files(directory)
        if not files:
            sys.exit("no license file found for " + path + " in " + str(directory))
        body = "\n\n".join(f.read_text(encoding="utf-8", errors="replace").strip() for f in files)
        parts.append("\n" + rule + "\n" + path + " " + version + "\n" + rule + "\n\n" + body + "\n")
    return "".join(parts)


def main():
    new = build()
    if "--check" in sys.argv:
        if TARGET.read_text(encoding="utf-8") != new:
            sys.exit("THIRD_PARTY_LICENSES is out of date: run python3 licenses.py and commit it")
        print("THIRD_PARTY_LICENSES is up to date")
        return
    TARGET.write_text(new, encoding="utf-8")
    print("THIRD_PARTY_LICENSES updated")


main()
