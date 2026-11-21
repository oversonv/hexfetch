import argparse
import hashlib
import json
import os
import re
import shutil
import struct
import subprocess
import sys
import time
from concurrent.futures import ThreadPoolExecutor, as_completed
from datetime import date
from pathlib import Path
from urllib.parse import quote

ROOT = Path(__file__).resolve().parent
MAIN = ROOT / "cmd" / "hexfetch" / "main.go"
TEMPLATE = ROOT / "release_template.md"
OUT = Path.home() / "hexfetch" / "releases"
HOST = "gitlab.com"
TAG = re.compile(r"v(\d+)\.(\d+)\.(\d+)")
VERSION = re.compile(r'(?m)^(const version = ")(\d+\.\d+\.\d+)(")$')
NOISE = re.compile(r"^(release:|Update \d{4}-\d{2}-\d{2})")
TARGETS = {
    "linux-amd64": ("linux", "amd64", "", 62),
    "linux-arm64": ("linux", "arm64", "", 183),
    "linux-armv7": ("linux", "arm", "7", 40),
    "linux-386": ("linux", "386", "", 3),
    "linux-riscv64": ("linux", "riscv64", "", 243),
    "android-arm64": ("android", "arm64", "", 183),
}
GLAB_ENV = {
    **os.environ,
    "GITLAB_HOST": HOST,
    "REMOTE_ALIAS": "origin",
    "GITLAB_RELEASE_ASSETS_USE_PACKAGE_REGISTRY": "false",
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
    match = VERSION.search(MAIN.read_text())
    if not match:
        raise Fail(f"no 'const version' declaration in {MAIN}")
    return match.group(2)


def write_version(version):
    MAIN.write_text(VERSION.sub(rf"\g<1>{version}\g<3>", MAIN.read_text()))


def bump(current, names):
    top = latest(names)
    numbers = [tuple(map(int, current.split(".")))]
    if top:
        numbers.append(semver(top))
    major, minor, patch = max(numbers)
    return f"{major}.{minor}.{patch + 1}"


def preflight():
    for tool in ("git", "go", "glab"):
        if not shutil.which(tool):
            raise Fail(f"{tool} is not installed")
    remote = git("remote", "get-url", "origin")
    match = re.search(rf"{re.escape(HOST)}[:/](.+?)(?:\.git)?$", remote)
    if not match:
        raise Fail(f"origin must point to {HOST}: {remote}")
    if git("status", "--porcelain", "--untracked-files=all"):
        raise Fail("working tree is not clean")
    branch = git("branch", "--show-current")
    if not branch or git("rev-parse", "--abbrev-ref", "@{upstream}") != f"origin/{branch}":
        raise Fail("the current branch must track origin/<branch>")
    sh("glab", "auth", "status", "--hostname", HOST, env=GLAB_ENV)
    sh("git", "fetch", "--prune", "origin")
    ahead, behind = map(int, git("rev-list", "--left-right", "--count", "HEAD...@{upstream}").split())
    if behind:
        raise Fail(f"branch is {behind} commit(s) behind origin")
    if ahead and (ahead > 1 or "Release-Tag: " not in git("log", "-1", "--format=%B")):
        raise Fail("push local commits before releasing")
    return branch, match.group(1)


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
    sh("go", "build", "-trimpath", "-buildvcs=false", "-ldflags=-s -w", "-o", path, "./cmd/hexfetch", env=env)
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


def commit(version, tag):
    sh("git", "add", "--", MAIN.relative_to(ROOT))
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


class GitLab:
    def __init__(self, project):
        self.project = project
        self.id = quote(project, safe="")

    def call(self, path, *flags, check=True, binary=False):
        return sh("glab", "api", f"projects/{self.id}/{path}", *flags, env=GLAB_ENV, check=check, binary=binary)

    def release(self, tag):
        result = self.call(f"releases/{quote(tag, safe='')}", check=False)
        if result.returncode == 0:
            return json.loads(result.stdout)
        if "404" in result.stdout + result.stderr:
            return None
        raise Fail(f"release lookup failed\n{result.stdout}{result.stderr}".strip())

    def assets(self, tag):
        links = json.loads(self.call(f"releases/{quote(tag, safe='')}/assets/links").stdout)
        return {item["name"]: item["url"] for item in links}

    def create(self, tag, title, body):
        if self.release(tag):
            log(f"release {tag} already exists")
            return
        try:
            self.call("releases", "--method", "POST", "--raw-field", f"tag_name={tag}", "--raw-field", f"name={title}", "--raw-field", f"description={body}")
        except Fail:
            if not self.release(tag):
                raise
        log(f"release {tag} created")

    def upload(self, tag, files):
        for path in files:
            if path.name in self.assets(tag):
                log(f"{path.name}: already uploaded")
                continue
            result = sh("glab", "release", "upload", tag, path, "--repo", self.project, env=GLAB_ENV, check=False)
            if path.name not in self.assets(tag):
                raise Fail(f"upload failed: {path.name}\n{result.stdout}{result.stderr}".strip())
            log(f"{path.name}: uploaded")

    def verify(self, tag, files):
        assets = self.assets(tag)

        def check(path):
            url = assets.get(path.name, "")
            if "/uploads/" not in url:
                raise Fail(f"missing upload link: {path.name}")
            data = self.call(f"uploads/{url.split('/uploads/', 1)[1]}", binary=True).stdout
            if hashlib.sha256(data).hexdigest() != sha256(path):
                raise Fail(f"checksum mismatch: {path.name}")

        with ThreadPoolExecutor(max_workers=len(files)) as pool:
            list(pool.map(check, files))
        log(f"verified {len(files)} remote assets")


def parse():
    parser = argparse.ArgumentParser(description="Build every target and publish a hexfetch release to GitLab.")
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
        (stage / "SHA256SUMS").write_text("".join(f"{sha256(path)}  {path.name}\n" for path in binaries))
        compare = f"[{previous}...{tag}](https://{HOST}/{project}/-/compare/{previous}...{tag})" if previous else f"[{tag}](https://{HOST}/{project}/-/commits/{tag})"
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
        assets = [dest / path.name for path in binaries] + [dest / "SHA256SUMS"]
        log(f"built {len(binaries)} binaries in {dest}")
        if args.dry_run:
            log("dry run: nothing was committed or published")
            return
        if changed:
            commit(version, tag)
            committed = True
        tag_and_push(branch, tag, version)
        gitlab = GitLab(project)
        gitlab.create(tag, f"Hexfetch {version}", body)
        gitlab.upload(tag, assets)
        gitlab.verify(tag, assets)
        log(f"done: https://{HOST}/{project}/-/releases/{tag}")
    finally:
        shutil.rmtree(stage, ignore_errors=True)
        if changed and not committed:
            sh("git", "checkout", "--", MAIN.relative_to(ROOT), check=False)


if __name__ == "__main__":
    try:
        main()
    except Fail as error:
        print(f"[*] error: {error}", file=sys.stderr)
        sys.exit(1)
    except KeyboardInterrupt:
        sys.exit(130)

