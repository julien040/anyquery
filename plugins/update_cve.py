#!/usr/bin/env python3
"""
CVE patch script for anyquery plugins.

Usage:
  python update_cve.py <package> [version] [--min-version v1.2.3]
  python update_cve.py golang.org/x/net
  python update_cve.py golang.org/x/net v0.38.0
  python update_cve.py github.com/some/lib v1.2.3 --min-version v1.1.0

Flags:
  --min-version  Skip plugins already at or above this version of the package.
  --dry-run      Scan and report without making any changes.

Environment:
  ANYQUERY_PASSWORD  - password for the store-manager CLI (prompted if absent)

The script will:
  1. Scan all plugin directories (excluding store-manager and sharedObject)
  2. If the plugin's go.mod depends on <package>:
       a. If --min-version is given and the current version >= min-version, skip.
  3. Update the dependency, bump manifest.toml patch version, build, publish.
"""

import getpass
import os
import re
import sys
import tomllib
import subprocess
import argparse
from pathlib import Path

# ── ANSI colors ──────────────────────────────────────────────────────────────

RESET  = "\033[0m"
BOLD   = "\033[1m"
DIM    = "\033[2m"
RED    = "\033[91m"
GREEN  = "\033[92m"
YELLOW = "\033[93m"
BLUE   = "\033[94m"
CYAN   = "\033[96m"
WHITE  = "\033[97m"

def c(color: str, text: str) -> str:
    return f"{color}{text}{RESET}"

def header(text: str) -> None:
    print(f"\n{BOLD}{BLUE}{'─' * 60}{RESET}")
    print(f"{BOLD}{WHITE}  {text}{RESET}")
    print(f"{BOLD}{BLUE}{'─' * 60}{RESET}")

def step(text: str) -> None:
    print(f"  {CYAN}▶{RESET} {text}")

def ok(text: str) -> None:
    print(f"  {GREEN}✔{RESET} {text}")

def warn(text: str) -> None:
    print(f"  {YELLOW}⚠{RESET}  {text}")

def err(text: str) -> None:
    print(f"  {RED}✘{RESET} {text}", file=sys.stderr)

def skip(text: str) -> None:
    print(f"  {DIM}–  {text}{RESET}")

# ── helpers ───────────────────────────────────────────────────────────────────

PLUGINS_DIR = Path(__file__).parent
STORE_MANAGER = PLUGINS_DIR / "store-manager" / "store-manager.out"
SKIP_DIRS = {"store-manager", "sharedObject"}


def parse_semver(version: str) -> tuple[int, int, int]:
    """
    Parse a Go module version like v1.2.3 or 1.2.3 into (major, minor, patch).
    Pre-release and build metadata are ignored for comparison purposes.
    Returns (0, 0, 0) on parse failure.
    """
    m = re.search(r"(\d+)\.(\d+)\.(\d+)", version)
    if not m:
        return (0, 0, 0)
    return (int(m.group(1)), int(m.group(2)), int(m.group(3)))


def version_gte(v: str, minimum: str) -> bool:
    """Return True if v >= minimum (semver comparison)."""
    return parse_semver(v) >= parse_semver(minimum)


def bump_patch(version: str) -> str:
    """Increment the patch component of a semver string like v0.1.5."""
    m = re.fullmatch(r"(v?)(\d+)\.(\d+)\.(\d+)(.*)", version)
    if not m:
        warn(f"Cannot parse version '{version}', appending '-patched' suffix")
        return version + "-patched"
    prefix, major, minor, patch, rest = m.groups()
    return f"{prefix}{major}.{minor}.{int(patch) + 1}{rest}"


def go_mod_package_version(go_mod_path: Path, package: str) -> str | None:
    """
    Return the version string of <package> in go.mod, or None if not found.
    Matches both direct and indirect entries, e.g.:
        github.com/foo/bar v1.2.3
        github.com/foo/bar v1.2.3 // indirect
    """
    content = go_mod_path.read_text()
    m = re.search(
        rf"^\s+{re.escape(package)}\s+(v\S+)",
        content,
        re.MULTILINE,
    )
    return m.group(1) if m else None


def run(
    cmd: list[str],
    cwd: Path,
    env: dict | None = None,
    stdin: int | None = None,
) -> subprocess.CompletedProcess:
    """Run a command, streaming output, raise on failure."""
    merged_env = os.environ.copy()
    if env:
        merged_env.update(env)
    return subprocess.run(
        cmd,
        cwd=cwd,
        env=merged_env,
        stdin=stdin,
        check=True,
    )


def update_go_dep(plugin_dir: Path, package: str, version: str | None) -> None:
    """Run `go get <package>[@version]` inside the plugin directory."""
    target = package if version is None else f"{package}@{version}"
    step(f"go get {target}")
    run(["go", "get", target], cwd=plugin_dir)
    step("go mod tidy")
    run(["go", "mod", "tidy"], cwd=plugin_dir)


def bump_manifest_version(manifest_path: Path) -> str:
    """Read manifest.toml, bump version, write it back, return new version."""
    with open(manifest_path, "rb") as f:
        data = tomllib.load(f)

    old_version = data.get("version", "v0.0.0")
    new_version = bump_patch(old_version)

    # Targeted text replacement so we don't rewrite the whole file
    content = manifest_path.read_text()
    content = re.sub(
        r'^(version\s*=\s*")[^"]*(")',
        lambda m: f'{m.group(1)}{new_version}{m.group(2)}',
        content,
        count=1,
        flags=re.MULTILINE,
    )
    manifest_path.write_text(content)
    return new_version


def build_plugin(plugin_dir: Path) -> None:
    step("goreleaser build --clean --snapshot")
    run(
        ["goreleaser", "build", "--clean", "--snapshot"],
        cwd=plugin_dir,
    )


def publish_plugin(plugin_dir: Path, plugin_name: str, password: str) -> None:
    manifest_path = plugin_dir / "manifest.toml"
    step(f"store-manager publish  (plugin={plugin_name})")
    run(
        [
            str(STORE_MANAGER),
            "-p", plugin_name,
            "-u", "juliencagniart40@gmail.com",
            "-c", str(manifest_path),
        ],
        cwd=plugin_dir,
        env={"ANYQUERY_PASSWORD": password},
        stdin=subprocess.DEVNULL,
    )


# ── main ──────────────────────────────────────────────────────────────────────

def main() -> None:
    parser = argparse.ArgumentParser(
        description="Patch a Go dependency across all anyquery plugins and republish."
    )
    parser.add_argument("package", help="Go module path to update (e.g. golang.org/x/net)")
    parser.add_argument(
        "version",
        nargs="?",
        default=None,
        help="Target version (e.g. v1.2.3). Omit for 'latest'.",
    )
    parser.add_argument(
        "--min-version",
        default=None,
        metavar="VERSION",
        help=(
            "Skip plugins whose current version of <package> is already >= VERSION. "
            "Useful for idempotent re-runs (e.g. --min-version v0.38.0)."
        ),
    )
    parser.add_argument(
        "--dry-run",
        action="store_true",
        help="Scan and report without making any changes.",
    )
    args = parser.parse_args()

    # If a target version is given but no explicit --min-version, use the
    # target as the minimum so plugins already at that version are skipped.
    if args.version and not args.min_version:
        args.min_version = args.version

    password = os.environ.get("ANYQUERY_PASSWORD", "").strip()
    if not password and not args.dry_run:
        try:
            password = getpass.getpass(
                f"  {CYAN}?{RESET} ANYQUERY_PASSWORD not set. Enter password: "
            ).strip()
        except (KeyboardInterrupt, EOFError):
            print()
            err("No password provided, aborting.")
            sys.exit(1)
        if not password:
            err("Password is empty, aborting.")
            sys.exit(1)

    if not STORE_MANAGER.exists() and not args.dry_run:
        err(f"store-manager binary not found at {STORE_MANAGER}")
        sys.exit(1)

    target_label = args.version if args.version else "latest"
    header_parts = [
        f"package={c(YELLOW, args.package)}",
        f"target={c(CYAN, target_label)}",
    ]
    if args.min_version:
        header_parts.append(f"min-version={c(CYAN, args.min_version)}")
    if args.dry_run:
        header_parts.append(c(YELLOW, "[DRY RUN]"))

    print(f"\n{BOLD}anyquery CVE patcher{RESET}  " + "  ".join(header_parts))

    plugin_dirs = sorted(
        d for d in PLUGINS_DIR.iterdir()
        if d.is_dir() and d.name not in SKIP_DIRS and not d.name.startswith(".")
    )

    affected:  list[str] = []
    skipped:   list[str] = []
    errors:    list[tuple[str, str]] = []

    for plugin_dir in plugin_dirs:
        go_mod = plugin_dir / "go.mod"
        manifest = plugin_dir / "manifest.toml"

        if not go_mod.exists():
            skip(f"{plugin_dir.name}: no go.mod")
            continue

        current_ver = go_mod_package_version(go_mod, args.package)
        if current_ver is None:
            skip(f"{plugin_dir.name}: does not depend on {args.package}")
            continue

        # Idempotency: skip if already at or above the minimum required version
        if args.min_version and version_gte(current_ver, args.min_version):
            skip(
                f"{plugin_dir.name}: {args.package} {c(GREEN, current_ver)} "
                f">= {args.min_version}, already up to date"
            )
            skipped.append(plugin_dir.name)
            continue

        if not manifest.exists():
            warn(f"{plugin_dir.name}: no manifest.toml, will skip publish step")

        header(
            f"Plugin: {plugin_dir.name}  "
            f"({args.package} {c(YELLOW, current_ver)} → {c(GREEN, target_label)})"
        )
        affected.append(plugin_dir.name)

        if args.dry_run:
            ok("Would update, build, and publish (dry run)")
            continue

        # Snapshot files that will be mutated so we can roll back on failure
        snapshots: dict[Path, bytes | None] = {}
        for tracked in (go_mod, plugin_dir / "go.sum", manifest):
            snapshots[tracked] = tracked.read_bytes() if tracked.exists() else None

        def rollback() -> None:
            step("Rolling back modified files…")
            for path, original in snapshots.items():
                if original is None:
                    if path.exists():
                        path.unlink()
                        ok(f"Removed {path.name} (did not exist before)")
                else:
                    path.write_bytes(original)
                    ok(f"Restored {path.name}")

        try:
            # 1. Update dependency
            update_go_dep(plugin_dir, args.package, args.version)
            ok("Dependency updated")

            # 2. Bump version in manifest
            if manifest.exists():
                new_ver = bump_manifest_version(manifest)
                ok(f"manifest.toml version → {c(GREEN, new_ver)}")
            else:
                new_ver = "unknown"

            # 3. Build
            build_plugin(plugin_dir)
            ok("Build succeeded")

            # 4. Publish
            if manifest.exists():
                with open(manifest, "rb") as f:
                    mdata = tomllib.load(f)
                plugin_name = mdata.get("name", plugin_dir.name)
                publish_plugin(plugin_dir, plugin_name, password)
                ok(f"Published {plugin_name} {new_ver}")
            else:
                warn("Skipping publish: no manifest.toml")

        except subprocess.CalledProcessError as exc:
            msg = f"command exited with code {exc.returncode}: {' '.join(str(a) for a in exc.cmd)}"
            err(msg)
            rollback()
            errors.append((plugin_dir.name, msg))
            warn("Continuing with next plugin…")
        except Exception as exc:  # noqa: BLE001
            err(str(exc))
            rollback()
            errors.append((plugin_dir.name, str(exc)))
            warn("Continuing with next plugin…")

    # ── Summary ───────────────────────────────────────────────────────────────
    print(f"\n{BOLD}{WHITE}{'═' * 60}{RESET}")
    print(f"{BOLD}  Summary{RESET}")
    print(f"{'═' * 60}")

    if not affected and not skipped:
        print(f"  {DIM}No plugin depends on {args.package}.{RESET}")
    else:
        if affected:
            action = "Would update" if args.dry_run else "Updated"
            print(f"  {GREEN}{action} ({len(affected)}):{RESET} {', '.join(affected)}")
        if skipped:
            print(f"  {DIM}Already up to date ({len(skipped)}): {', '.join(skipped)}{RESET}")

    if errors:
        print(f"\n  {RED}Errors ({len(errors)}):{RESET}")
        for name, msg in errors:
            print(f"    {RED}•{RESET} {name}: {msg}")
        sys.exit(1)
    else:
        print(f"\n  {GREEN}All done.{RESET}\n")


if __name__ == "__main__":
    main()
