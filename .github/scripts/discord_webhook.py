#!/usr/bin/env python3
import sys

# .pyc / __pycache__ の生成を完全に抑制（CI 実行時にディレクトリを汚さないため）
sys.dont_write_bytecode = True

"""
Discord Webhook 通知送信スクリプト（依存なし版）

- urllib と標準ライブラリのみを使用（pip install 不要）
- GitHub Actions の ubuntu-latest でそのまま実行可能
- 環境変数で通知内容を指定し、埋め込みメッセージ（embed）を送信
- DISCORD_WEBHOOK_URL が未設定/空の場合は通知をスキップ（エラー終了しない）

対応モード:
  - build-start   （ビルド開始通知）
  - build-complete （ビルド完了/失敗通知）

各アーキテクチャ（amd64 / arm64）は TARGET_ARCH 環境変数を渡すことで
独立したメッセージを投稿する。
"""

import json
import os
import sys
from datetime import datetime, timezone
from urllib.error import HTTPError, URLError
from urllib.parse import parse_qsl, urlencode, urlsplit, urlunsplit
from urllib.request import Request, urlopen


def get_env(key: str, default: str = "") -> str:
    """環境変数を取得。デフォルト値を指定可能。"""
    return os.environ.get(key, default)


def with_query_param(url: str, key: str, value: str) -> str:
    parts = urlsplit(url)
    query = dict(parse_qsl(parts.query, keep_blank_values=True))
    query[key] = value
    return urlunsplit((parts.scheme, parts.netloc, parts.path, urlencode(query), parts.fragment))


def discord_message_url(webhook_url: str, message_id: str) -> str:
    base_url = webhook_url.split("?", 1)[0].rstrip("/")
    return f"{base_url}/messages/{message_id}"




def send_discord(webhook_url: str, payload: dict) -> None:
    """
    Discord Webhook に JSON ペイロードを送信する。
    送信失敗時は標準エラーに出力するが、終了コードは 0 のまま（CI 継続）。
    """
    data = json.dumps(payload, ensure_ascii=False).encode("utf-8")
    req = Request(
        webhook_url,
        data=data,
        headers={
            "Content-Type": "application/json",
            "User-Agent": "chunisupport-song-batch-ci/1.0 (urllib)",
        },
    )

    try:
        with urlopen(req, timeout=10) as resp:
            print(f"Sent Discord notification (HTTP {resp.status})")
    except HTTPError as e:
        # Discord 側で 400/429 などが返る場合も CI は止めない
        print(f"Failed to send Discord notification; continuing: {e.code} {e.reason}", file=sys.stderr)
    except URLError as e:
        print(f"Failed to send Discord notification; continuing: {e.reason}", file=sys.stderr)
    except Exception as e:
        print(f"Unexpected error while sending Discord notification; continuing: {e}", file=sys.stderr)






def send_discord_and_get_message_id(webhook_url: str, payload: dict) -> str:
    data = json.dumps(payload, ensure_ascii=False).encode("utf-8")
    req = Request(
        with_query_param(webhook_url, "wait", "true"),
        data=data,
        headers={
            "Content-Type": "application/json",
            "User-Agent": "chunisupport-song-batch-ci/1.0 (urllib)",
        },
    )

    try:
        with urlopen(req, timeout=10) as resp:
            body = resp.read().decode("utf-8")
            message = json.loads(body)
            message_id = str(message.get("id", ""))
            print(f"Sent Discord notification (HTTP {resp.status})")
            return message_id
    except HTTPError as e:
        # Discord 側で 400/429 などが返る場合も CI は止めない
        print(f"Failed to send Discord notification; continuing: {e.code} {e.reason}", file=sys.stderr)
    except URLError as e:
        print(f"Failed to send Discord notification; continuing: {e.reason}", file=sys.stderr)
    except Exception as e:
        print(f"Unexpected error while sending Discord notification; continuing: {e}", file=sys.stderr)
    return ""


def update_discord_message(webhook_url: str, message_id: str, payload: dict) -> bool:
    data = json.dumps(payload, ensure_ascii=False).encode("utf-8")
    req = Request(
        discord_message_url(webhook_url, message_id),
        data=data,
        headers={
            "Content-Type": "application/json",
            "User-Agent": "chunisupport-song-batch-ci/1.0 (urllib)",
        },
        method="PATCH",
    )

    try:
        with urlopen(req, timeout=10) as resp:
            print(f"Updated Discord notification (HTTP {resp.status})")
            return True
    except HTTPError as e:
        # Discord 側で 400/429 などが返る場合も CI は止めない
        print(f"Failed to update Discord notification; continuing: {e.code} {e.reason}", file=sys.stderr)
    except URLError as e:
        print(f"Failed to update Discord notification; continuing: {e.reason}", file=sys.stderr)
    except Exception as e:
        print(f"Unexpected error while updating Discord notification; continuing: {e}", file=sys.stderr)
    return False


def github_repo_url(env: dict) -> str:
    server_url = env.get("GITHUB_SERVER_URL", "https://github.com").rstrip("/")
    repo = env.get("REPO", "unknown")
    return f"{server_url}/{repo}"


def commit_link(env: dict, short_sha: str) -> str:
    sha = env.get("SHA", "unknown")
    return f"[{short_sha}]({github_repo_url(env)}/commit/{sha})"


def build_commit_section(env: dict) -> str:
    """「Commit: {短縮ハッシュ}(リンク)」行とその下にコミットメッセージを表示する文字列を返す。

    Discord通知の「Commit:」行の下にコミットメッセージを追加する（ユーザーリクエスト）。
    コミットメッセージが空または 'N/A' の場合はハッシュ行のみとする。
    """
    sha = env.get("SHA", "unknown")
    short_sha = sha[:7] if len(sha) >= 7 else sha
    link = commit_link(env, short_sha)
    msg = env.get("COMMIT_MESSAGE", "").strip()
    if msg and msg != "N/A":
        return f"Commit: {link}\n{msg}"
    return f"Commit: {link}"


def build_build_start_embed(env: dict) -> dict:
    """ビルド開始通知用の embed を構築。"""
    repo = env.get("REPO", "unknown")
    branch = env.get("BRANCH", "unknown")
    arch = env.get("TARGET_ARCH", "unknown")
    arch_label = env.get("TARGET_ARCH_LABEL", f"linux/{arch}")

    return {
        "author": {"name": repo, "url": github_repo_url(env)},
        "title": f"🚀 Build Started ({arch})",
        "description": f"Started {arch_label} build\n{build_commit_section(env)}",
        "color": 3447003,
        "footer": {"text": branch},
        "timestamp": datetime.now(timezone.utc).isoformat(),
    }


def build_build_start_embeds(env: dict) -> list[dict]:
    """ビルド開始用の embed リストを返す。

    TARGET_ARCH が指定されている場合はそのアーキテクチャ単独の embed のみ返す
    （amd64 と arm64 で独立したメッセージを投稿するため）。
    """
    target = env.get("TARGET_ARCH")
    if target:
        for arch, label in target_arches():
            if arch == target:
                return [build_build_start_embed(env | {"TARGET_ARCH": arch, "TARGET_ARCH_LABEL": label})]
        # 未知の値が来た場合はフォールバックして単独で出す
        return [build_build_start_embed(env | {"TARGET_ARCH_LABEL": f"linux/{target}"})]

    # TARGET_ARCH 未指定時は従来どおり両方（現在は使用されていない）
    return [
        build_build_start_embed(env | {"TARGET_ARCH": arch, "TARGET_ARCH_LABEL": label})
        for arch, label in target_arches()
    ]


def target_arches() -> tuple[tuple[str, str], ...]:
    return (("amd64", "linux/amd64"), ("arm64", "linux/arm64"))


def arch_label(arch: str) -> str:
    for target_arch, label in target_arches():
        if target_arch == arch:
            return label
    return f"linux/{arch}"


def build_build_complete_embed(env: dict) -> dict:
    """ビルド完了通知用の embed を構築（結果によりタイトル・色・文言を切り替え）。"""
    repo = env.get("REPO", "unknown")
    branch = env.get("BRANCH", "unknown")
    build_result = env.get("BUILD_RESULT", "failure")
    arch = env.get("TARGET_ARCH", "unknown")
    arch_label = env.get("TARGET_ARCH_LABEL", f"linux/{arch}")

    if build_result == "success":
        status = "✅"
        title_text = "Build Completed"
        color = 3066993
        desc = f"Completed {arch_label} build successfully"
    elif build_result == "cancelled":
        status = "⚠️"
        title_text = "Build Cancelled"
        color = 16776960
        desc = f"Cancelled {arch_label} build"
    else:
        status = "❌"
        title_text = "Build Failed"
        color = 15158332
        desc = f"Failed {arch_label} build"

    return {
        "author": {"name": repo, "url": github_repo_url(env)},
        "title": f"{status} {title_text} ({arch})",
        "description": f"{desc}\n{build_commit_section(env)}",
        "color": color,
        "footer": {"text": branch},
        "timestamp": datetime.now(timezone.utc).isoformat(),
    }


def build_build_complete_embeds(env: dict) -> list[dict]:
    """ビルド完了用の embed リストを返す。

    TARGET_ARCH が指定されている場合はそのアーキテクチャ単独の embed のみ返す
    （amd64 と arm64 で独立したメッセージを投稿するため）。
    """
    target = env.get("TARGET_ARCH")
    if target:
        for arch, label in target_arches():
            if arch == target:
                return [build_build_complete_embed(env | {"TARGET_ARCH": arch, "TARGET_ARCH_LABEL": label})]
        # 未知の値が来た場合はフォールバックして単独で出す
        return [build_build_complete_embed(env | {"TARGET_ARCH_LABEL": f"linux/{target}"})]

    # TARGET_ARCH 未指定時は従来どおり両方（現在は使用されていない）
    return [
        build_build_complete_embed(env | {"TARGET_ARCH": arch, "TARGET_ARCH_LABEL": label})
        for arch, label in target_arches()
    ]




def main() -> int:
    webhook_url = get_env("DISCORD_WEBHOOK_URL")
    if not webhook_url:
        print("DISCORD_WEBHOOK_URL is not set; skipping notification")
        return 0

    mode = get_env("DISCORD_NOTIFY_MODE", "build-start")

    # 環境変数から必要な値だけを辞書にまとめて関数に渡す（テスト容易性も考慮）
    env = {
        "REPO": get_env("REPO"),
        "BRANCH": get_env("BRANCH"),
        "GITHUB_SERVER_URL": get_env("GITHUB_SERVER_URL", "https://github.com"),
        "SHA": get_env("SHA"),
        "BUILD_RESULT": get_env("BUILD_RESULT"),
        "TARGET_ARCH": get_env("TARGET_ARCH"),
        "COMMIT_MESSAGE": get_env("COMMIT_MESSAGE"),
    }

    if mode == "build-start":
        embeds = build_build_start_embeds(env)
    elif mode == "build-complete":
        embeds = build_build_complete_embeds(env)
    else:
        print(f"Unknown DISCORD_NOTIFY_MODE: {mode}", file=sys.stderr)
        return 1

    payload = {
        "username": "Build | chunisupport-song-batch",
        "embeds": embeds,
    }

    message_id_path = get_env("DISCORD_MESSAGE_ID_PATH")
    message_id = ""
    if message_id_path and os.path.exists(message_id_path):
        with open(message_id_path, encoding="utf-8") as f:
            message_id = f.read().strip()

    if mode == "build-complete" and message_id:
        update_discord_message(webhook_url, message_id, payload)
        return 0

    if mode == "build-complete" and message_id_path:
        print("Discord message ID is not available; skipping update")
        return 0

    if not message_id_path:
        send_discord(webhook_url, payload)
        return 0

    message_id = send_discord_and_get_message_id(webhook_url, payload)
    if message_id:
        message_id_dir = os.path.dirname(message_id_path)
        if message_id_dir:
            os.makedirs(message_id_dir, exist_ok=True)
        with open(message_id_path, "w", encoding="utf-8") as f:
            f.write(message_id)

    return 0


if __name__ == "__main__":
    sys.exit(main())
