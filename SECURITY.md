# Security Policy

## Supported Versions

| Version | Supported          |
| ------- | ------------------ |
| 1.0.x   | :white_check_mark: |
| < 1.0   | :x:                |

サポート対象は最新のマイナー系列（現在 1.0.x）です。

## Reporting a Vulnerability

If you discover a security vulnerability in notify-slack, please report it
through GitHub's private vulnerability reporting feature:

1. Go to the [Security tab](https://github.com/kan/notify-slack/security)
2. Click "Report a vulnerability"
3. Provide details about the vulnerability

Alternatively, you can email the maintainer directly.

### What to include

- Description of the vulnerability
- Steps to reproduce
- Potential impact
- Suggested fix (if any)

### Response timeline

- **Initial response**: Within 48 hours
- **Status update**: Within 7 days
- **Fix release**: Depends on severity (critical: ASAP, high: 1-2 weeks, medium/low: next release)

### Disclosure policy

- We follow responsible disclosure practices
- Security advisories will be published after a fix is available
- Credit will be given to reporters (unless they prefer to remain anonymous)

## Security Best Practices for Users

notify-slack は Slack API トークンを扱う。次の点に注意する。

1. **最新版を使う** — 脆弱性修正を取り込むため、常に最新リリースを利用する。
2. **トークンを漏らさない** — トークンをコミットに含めない。`.env` や
   `/etc/notify-slack.env` に置く場合はリポジトリの管理外に置き、ファイル権限を
   絞る（例: `chmod 600`）。dotenv ファイルは `.gitignore` 済みか確認する。
3. **最小権限のスコープにする** — トークンには必要なスコープだけ付与する
   （メッセージ投稿なら `chat:write`、ファイル添付を使うときのみ `files:write`）。
4. **CI では Secrets を使う** — CI から実行する場合、トークンは環境変数へ直接
   書かず GitHub Actions Secrets 等のシークレット管理機構を使う。
