import argparse
import hashlib
import json
import os
import platform
import re
import shutil
import struct
import subprocess
import sys
import time
import tomllib
from concurrent.futures import ThreadPoolExecutor, as_completed
from datetime import date
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
META = ROOT / "src" / "cmd" / "hexfetch" / "hexfetch.toml"
TEMPLATE = ROOT / "scripts" / "release_template.md"
OUT = Path.home() / "hexfetch" / "releases"
HOST = "github.com"
REPO = "oversonv/hexfetch"
TAG = re.compile(r"v(\d+)\.(\d+)\.(\d+)")
VERSION = re.compile(r'(?m)^(version = ")(\d+\.\d+\.\d+)(")([ \t]*#.*)?\r?$')
SEMVER = re.compile(r"\d+\.\d+\.\d+")
NOISE = re.compile(r"^(release:|Update \d{4}-\d{2}-\d{2})")
TARGETS = {
    "linux-amd64": ("linux", "amd64", "", 62),
    "linux-arm64": ("linux", "arm64", "", 183),
    "linux-armv7": ("linux", "arm", "7", 40),
    "linux-386": ("linux", "386", "", 3),
    "linux-riscv64": ("linux", "riscv64", "", 243),
    "android-arm64": ("android", "arm64", "", 183),
}


class Fail(Exception):
    pass


def log(message):
    print(f"[*] {message}\n", end="", flush=True)


def sh(*args, env=None, check=True, binary=False):
    command = [str(item) for item in args]
    result = subprocess.run(command, cwd=ROOT, env=env, capture_output=True, text=not binary)
    if check and result.returncode:
        detail = result.stderr.decode(errors="replace") if binary else result.stdout + result.stderr
        raise Fail(f"{' '.join(command)}\n{detail.strip()}")
    return result


def git(*args):
    return sh("git", *args).stdout.strip()


def sha256(path):
    digest = hashlib.sha256()
    with path.open("rb") as handle:
        for chunk in iter(lambda: handle.read(1 << 20), b""):
            digest.update(chunk)
    return digest.hexdigest()


def semver(tag):
    return tuple(map(int, TAG.fullmatch(tag).groups()))


def latest(names, exclude=None):
    found = [name for name in names if TAG.fullmatch(name) and name != exclude]
    return max(found, key=semver, default=None)


def read_version():
    try:
        version = tomllib.loads(META.read_text(encoding="utf-8"))["package"]["version"]
    except (OSError, tomllib.TOMLDecodeError, KeyError, TypeError) as error:
        raise Fail(f"cannot read [package].version from {META}: {error}") from error
    if not isinstance(version, str) or not SEMVER.fullmatch(version):
        raise Fail(f"{META}: version must look like X.Y.Z, got {version!r}")
    return version


def write_version(version):
    original = META.read_text(encoding="utf-8")
    updated, count = VERSION.subn(rf"\g<1>{version}\g<3>\g<4>", original, count=1)
    if count != 1:
        raise Fail(f"no 'version = \"X.Y.Z\"' line in {META}")
    META.write_text(updated, encoding="utf-8")
    if read_version() != version:
        META.write_text(original, encoding="utf-8")
        raise Fail(f"{META}: version was not updated to {version}")


def bump(current, names):
    top = latest(names)
    numbers = [tuple(map(int, current.split(".")))]
    if top:
        numbers.append(semver(top))
    major, minor, patch = max(numbers)
    return f"{major}.{minor}.{patch + 1}"


def preflight():
    for tool in ("git", "go", "gh"):
        if not shutil.which(tool):
            raise Fail(f"{tool} is not installed")
    remote = git("remote", "get-url", "origin")
    match = re.search(rf"{re.escape(HOST)}[:/](.+?)(?:\.git)?$", remote)
    if not match:
        raise Fail(f"origin must point to {HOST}: {remote}")
    project = match.group(1)
    if project != REPO:
        raise Fail(f"origin must be {REPO}, got {project}")
    if git("status", "--porcelain", "--untracked-files=all"):
        raise Fail("working tree is not clean")
    branch = git("branch", "--show-current")
    if not branch or git("rev-parse", "--abbrev-ref", "@{upstream}") != f"origin/{branch}":
        raise Fail("the current branch must track origin/<branch>")
    sh("gh", "auth", "status")
    sh("git", "fetch", "--prune", "origin")
    ahead, behind = map(int, git("rev-list", "--left-right", "--count", "HEAD...@{upstream}").split())
    if behind:
        raise Fail(f"branch is {behind} commit(s) behind origin")
    if ahead and (ahead > 1 or "Release-Tag: " not in git("log", "-1", "--format=%B")):
        raise Fail("push local commits before releasing")
    return branch, project


def remote_tags():
    names = set(git("tag", "--list").split())
    for line in git("ls-remote", "--tags", "origin").splitlines():
        names.add(line.split()[1].removeprefix("refs/tags/").removesuffix("^{}"))
    return names


def build(name, version, dest):
    goos, goarch, goarm, machine = TARGETS[name]
    env = {key: value for key, value in os.environ.items() if key not in ("GOARM", "GOAMD64", "GOARM64", "CC")}
    env.update(GOOS=goos, GOARCH=goarch, CGO_ENABLED="0")
    if goarm:
        env["GOARM"] = goarm
    path = dest / f"hexfetch_{version}_{name}"
    started = time.monotonic()
    sh("go", "build", "-trimpath", "-buildvcs=false", "-ldflags=-s -w", "-o", path, "./src/cmd/hexfetch", env=env)
    with path.open("rb") as handle:
        header = handle.read(20)
    order = "<" if header[5:6] == b"\x01" else ">"
    if len(header) < 20 or header[:4] != b"\x7fELF" or struct.unpack(f"{order}H", header[18:20])[0] != machine:
        raise Fail(f"{name}: unexpected binary format")
    path.chmod(0o755)
    log(f"{name}: {path.stat().st_size / 1048576:.1f} MiB in {time.monotonic() - started:.0f}s")
    return path


def build_all(names, version, dest, workers):
    workers = max(1, min(workers, len(names)))
    log(f"building {len(names)} targets with {workers} workers")
    paths, errors = {}, []
    with ThreadPoolExecutor(max_workers=workers) as pool:
        futures = {pool.submit(build, name, version, dest): name for name in names}
        for future in as_completed(futures):
            try:
                paths[futures[future]] = future.result()
            except Exception as error:
                errors.append(f"{futures[future]}: {error}")
    if errors:
        raise Fail("build failed\n" + "\n".join(errors))
    return [paths[name] for name in names]


def verify_binaries(paths, version):
    needle = f'version = "{version}"'.encode()
    host = {"x86_64": "amd64", "amd64": "amd64", "aarch64": "arm64", "arm64": "arm64"}.get(platform.machine().lower())
    for path in paths:
        if needle not in path.read_bytes():
            raise Fail(f"{path.name}: embedded hexfetch.toml does not contain {needle.decode()}")
        target = path.name.removeprefix(f"hexfetch_{version}_")
        if target not in TARGETS or TARGETS[target][1] != host:
            continue
        try:
            result = subprocess.run([str(path), "--version"], capture_output=True, text=True, timeout=20)
        except OSError:
            log(f"{path.name}: cannot be executed on this host, embedded version checked only")
            continue
        if result.stdout.strip() != f"hexfetch {version}":
            raise Fail(f"{path.name}: --version printed {result.stdout.strip()!r}, want 'hexfetch {version}'")
        log(f"{path.name}: --version ok")


def changes(previous):
    span = ["log", f"{previous}..HEAD"] if previous else ["log", "-20"]
    lines = git(*span, "--format=%s").splitlines()
    return "\n".join(f"- {line}" for line in lines if not NOISE.match(line)) or "- No notable changes."


def render(values):
    try:
        return re.sub(r"\{\{(\w+)\}\}", lambda match: values[match.group(1)], TEMPLATE.read_text())
    except KeyError as error:
        raise Fail(f"unknown template field {error}") from error


def table(files):
    rows = [f"| `{path.name}` | {path.stat().st_size / 1048576:.1f} MiB | `{sha256(path)}` |" for path in files]
    return "\n".join(["| File | Size | SHA256 |", "| --- | --- | --- |", *rows])


LICENSES = ROOT / "THIRD_PARTY_LICENSES"
LICENSES_MARKER = "=== Go dependencies (compiled into the binary) ==="
EMBEDDED_LICENSES = (ROOT / "src" / "legal" / "LICENSE", ROOT / "src" / "legal" / "THIRD_PARTY_LICENSES")
LICENSE_PREFIXES = ("LICENSE", "LICENCE", "COPYING", "NOTICE")
MODULE_FORMAT = "{{if and .Module (not .Module.Main)}}{{.Module.Path}}|{{.Module.Version}}|{{.Module.Dir}}{{end}}"


def license_text():
    out = sh("go", "list", "-deps", "-f", MODULE_FORMAT, "./src/cmd/hexfetch").stdout
    found = {}
    for line in out.splitlines():
        if line.strip():
            path, version, directory = line.split("|")
            found[path] = (version, Path(directory))
    text = LICENSES.read_text(encoding="utf-8")
    parts = [text.split(LICENSES_MARKER)[0].rstrip() + "\n\n" + LICENSES_MARKER + "\n"]
    rule = "-" * 70
    for path, (version, directory) in sorted(found.items()):
        files = [f for f in sorted(directory.iterdir()) if f.is_file() and f.name.upper().startswith(LICENSE_PREFIXES)]
        if not files:
            raise Fail(f"no license file found for {path} in {directory}")
        body = "\n\n".join(f.read_text(encoding="utf-8", errors="replace").strip() for f in files)
        parts.append(f"\n{rule}\n{path} {version}\n{rule}\n\n{body}\n")
    return "".join(parts)


def update_licenses():
    """Regenerate the Go dependency notices. True if the file changed."""
    new = license_text()
    if LICENSES.read_text(encoding="utf-8") == new:
        return False
    LICENSES.write_text(new, encoding="utf-8")
    log("THIRD_PARTY_LICENSES updated")
    return True


def sync_embedded_licenses():
    changed = False
    for source, target in zip((ROOT / "LICENSE", LICENSES), EMBEDDED_LICENSES):
        content = source.read_bytes()
        if not target.is_file() or target.read_bytes() != content:
            target.write_bytes(content)
            changed = True
    if changed:
        log("embedded license files synchronized")
    return changed


def commit(version, tag):
    sh("git", "add", "--", META.relative_to(ROOT), LICENSES.relative_to(ROOT), *(path.relative_to(ROOT) for path in EMBEDDED_LICENSES))
    sh("git", "commit", "-m", f"release: {tag}\n\nRelease-Version: {version}\nRelease-Tag: {tag}")


def tag_and_push(branch, tag, version):
    head = git("rev-parse", "HEAD")
    existing = sh("git", "rev-parse", "--verify", "-q", f"refs/tags/{tag}^{{commit}}", check=False).stdout.strip()
    if not existing:
        sh("git", "tag", "-a", tag, "-m", f"Hexfetch {version}")
    elif existing != head:
        raise Fail(f"tag {tag} points to another commit")
    sh("git", "push", "origin", f"HEAD:refs/heads/{branch}")
    sh("git", "push", "origin", f"refs/tags/{tag}")
    log(f"pushed {branch} and {tag}")


class GitHub:
    def __init__(self, project):
        self.project = project

    def release_exists(self, tag):
        result = sh("gh", "release", "view", tag, "--repo", self.project, check=False)
        return result.returncode == 0

    def asset_names(self, tag):
        result = sh("gh", "release", "view", tag, "--repo", self.project, "--json", "assets", check=False)
        if result.returncode:
            return set()
        data = json.loads(result.stdout)
        return {item["name"] for item in data.get("assets", [])}

    def create(self, tag, title, body):
        if self.release_exists(tag):
            log(f"release {tag} already exists")
            return
        notes_file = OUT / tag / "RELEASE_NOTES.md"
        sh(
            "gh", "release", "create", tag,
            "--repo", self.project,
            "--title", title,
            "--notes-file", str(notes_file),
        )
        log(f"release {tag} created")

    def upload(self, tag, files):
        existing = self.asset_names(tag)
        for path in files:
            if path.name in existing:
                log(f"{path.name}: already uploaded")
                continue
            sh("gh", "release", "upload", tag, str(path), "--repo", self.project, "--clobber")
            log(f"{path.name}: uploaded")

    def verify(self, tag, files):
        existing = self.asset_names(tag)
        missing = [path.name for path in files if path.name not in existing]
        if missing:
            raise Fail(f"missing assets on release: {', '.join(missing)}")
        log(f"verified {len(files)} remote assets")


def parse():
    parser = argparse.ArgumentParser(description="Build every target and publish a hexfetch release to GitHub.")
    parser.add_argument("summary", nargs="?", help="short description of what changed, inserted into the release notes")
    parser.add_argument("--version", help="X.Y.Z, default: next patch version")
    parser.add_argument("--targets", default="all", help="comma-separated targets, default: all")
    parser.add_argument("--workers", type=int, default=2, help="parallel builds, default: 2")
    parser.add_argument("--dry-run", action="store_true", help="build and render notes without committing or publishing")
    return parser.parse_args()


def main():
    args = parse()
    if not TEMPLATE.is_file():
        raise Fail(f"missing template: {TEMPLATE}")
    try:
        os.nice(19)
    except OSError:
        pass
    names = list(TARGETS) if args.targets == "all" else args.targets.split(",")
    unknown = [name for name in names if name not in TARGETS]
    if unknown:
        raise Fail(f"unknown targets: {', '.join(unknown)}")
    branch, project = preflight()
    licenses_changed = update_licenses()
    licenses_changed = sync_embedded_licenses() or licenses_changed
    known = remote_tags()
    current = read_version()
    version = args.version or (current if f"v{current}" not in known else bump(current, known))
    if not re.fullmatch(r"\d+\.\d+\.\d+", version):
        raise Fail("version must look like X.Y.Z")
    tag = f"v{version}"
    changed = version != current
    if changed and tag in known:
        raise Fail(f"tag {tag} already exists")
    previous = latest(set(git("tag", "--list").split()), tag)
    dest, stage = OUT / tag, OUT / f".{tag}.tmp"
    log(f"release {tag}, {len(names)} targets, output {dest}")
    shutil.rmtree(stage, ignore_errors=True)
    stage.mkdir(parents=True)
    committed = False
    try:
        if changed:
            write_version(version)
        binaries = build_all(names, version, stage, args.workers)
        verify_binaries(binaries, version)
        (stage / "SHA256SUMS").write_text("".join(f"{sha256(path)}  {path.name}\n" for path in binaries))
        for name in ("LICENSE", "THIRD_PARTY_LICENSES"):
            shutil.copy2(ROOT / name, stage / name)
        if previous:
            compare = f"[{previous}...{tag}](https://github.com/{project}/compare/{previous}...{tag})"
        else:
            compare = f"[{tag}](https://github.com/{project}/commits/{tag})"
        body = render({
            "version": version,
            "tag": tag,
            "date": date.today().isoformat(),
            "summary": args.summary or f"Hexfetch {version} release.",
            "assets": table(binaries),
            "changes": changes(previous),
            "compare": compare,
        })
        (stage / "RELEASE_NOTES.md").write_text(body)
        shutil.rmtree(dest, ignore_errors=True)
        stage.rename(dest)
        assets = [dest / path.name for path in binaries] + [dest / "SHA256SUMS", dest / "LICENSE", dest / "THIRD_PARTY_LICENSES"]
        log(f"built {len(binaries)} binaries in {dest}")
        if args.dry_run:
            log("dry run: nothing was committed or published")
            return
        if changed or licenses_changed:
            commit(version, tag)
            committed = True
        tag_and_push(branch, tag, version)
        github = GitHub(project)
        github.create(tag, f"Hexfetch {version}", body)
        github.upload(tag, assets)
        github.verify(tag, assets)
        log(f"done: https://github.com/{project}/releases/tag/{tag}")
    finally:
        shutil.rmtree(stage, ignore_errors=True)
        if not committed:
            if changed:
                sh("git", "checkout", "--", META.relative_to(ROOT), check=False)
            if licenses_changed:
                sh("git", "checkout", "--", LICENSES.relative_to(ROOT), *(path.relative_to(ROOT) for path in EMBEDDED_LICENSES), check=False)


if __name__ == "__main__":
    try:
        main()
    except Fail as error:
        print(f"[*] error: {error}", file=sys.stderr)
        sys.exit(1)
    except KeyboardInterrupt:
        sys.exit(130)
