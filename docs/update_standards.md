# Industry Standards for CLI Tool Updates

When implementing an `--update` or `self-update` command for a CLI tool, there are 3 main standard approaches, ranked by reliability and user experience.

## 1. Package Managers (The Gold Standard)
The most robust "standard" is to rely on system package managers.
- **MacOS**: Homebrew (`brew upgrade aseity`)
- **Linux**: apt, yum, snap, etc.
- **Windows**: Winget, Scoop, Chocolatey.

**Pros**: Handles dependencies, permissions, signing, path management, and rollbacks.
**Cons**: Requires maintaining packages for each system. Code is not updated instantly (release lag).

## 2. Binary Replacement (Self-Updater)
This is the most common approach for standalone binaries (like `kubectl`, `deno`, or specialized dev tools).
**How it works**:
1. The tool queries an API (e.g., GitHub Releases) to find the latest version.
2. Downloads the pre-compiled binary for the current OS/Arch.
3. Verifies checksum/signature.
4. Replaces the current executable on disk.

**Libraries**:
- Go: [`minio/selfupdate`](https://github.com/minio/selfupdate), [`rhysd/go-github-selfupdate`](https://github.com/rhysd/go-github-selfupdate)
- Rust: `self_update`

**Pros**: Fast, works everywhere, user gets exactly what you released.
**Cons**: Requires setting up a release pipeline (CI/CD) to build binaries.

## 3. Source-Based Update (Current Implementation)
The tool acts as a wrapper around `git pull` and `go build`.
**How it works**:
1. Checks if the source is a git repo.
2. Runs `git pull`.
3. Runs `go build` and overwrites the binary.

**Pros**: simple to implement for open-source tools intended for developers.
**Cons**: Fails if user installed via Homebrew or downloaded a binary. Fails if user doesn't have Go installed. Fragile (as seen here).

## Recommendation for Aseity
Since Aseity seems to be a developer tool distributed as source (or valid local build), the current approach is acceptable **IF** strictly for development.
However, for widespread distribution, **Binary Replacement (Method 2)** combined with **Homebrew (Method 1)** is the professional standard.
