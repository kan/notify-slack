# notify-slack

[![CI](https://github.com/kan/notify-slack/actions/workflows/ci.yml/badge.svg)](https://github.com/kan/notify-slack/actions/workflows/ci.yml)
[![Security Scan](https://github.com/kan/notify-slack/actions/workflows/security.yml/badge.svg)](https://github.com/kan/notify-slack/actions/workflows/security.yml)
[![Go Version](https://img.shields.io/github/go-mod/go-version/kan/notify-slack)](go.mod)
[![Release](https://img.shields.io/github/v/release/kan/notify-slack?sort=semver)](https://github.com/kan/notify-slack/releases)

標準入力やコマンドの出力を Slack に投稿する小さな CLI ツール。

`echo message | notify-slack -c '#alerts'` のようにパイプで使うほか、コマンドを
ラップして実行結果を通知する使い方もできる。

## インストール

### リリースバイナリ

[Releases](https://github.com/kan/notify-slack/releases) から OS / アーキテクチャ
に合ったアーカイブ（linux / macOS / Windows、amd64 / arm64）を取得する。

### go install

```bash
go install github.com/kan/notify-slack@latest
```

### ソースからビルド

```bash
git clone https://github.com/kan/notify-slack.git
cd notify-slack
go build -o notify-slack .
```

## 事前準備（Slack トークン）

Slack App のトークンを用意する。必要なスコープは用途に応じて次のとおり。

- メッセージ投稿：`chat:write`
- ファイル添付（`-f`）：`files:write`

トークンは `--token` で渡すか、環境変数 `SLACK_API_TOKEN` に設定する。dotenv
ファイル（後述）にも置ける。

## 使い方

### 標準入力を投稿する

```bash
echo "デプロイが完了しました" | notify-slack -c '#deploy'
```

標準入力が空のときは投稿しない。

### コマンドを実行して出力を投稿する

`command 2>&1 | notify-slack` の置き換えとして、コマンドを直接ラップできる。
`--` の後ろに実行するコマンドを書く。

```bash
notify-slack -c '#cron' -- ./backup.sh --full
```

- コマンドの標準出力と標準エラーを結合して投稿する。
- 出力があるときだけ、先頭に `$ <コマンド> (exit N)` のヘッダを付けて投稿する。
  出力が空なら投稿しない。
- コマンドの終了コードを `notify-slack` の終了コードとして引き継ぐ（cron や CI
  でラッパーとして使える）。
- 自身のフラグを持つコマンドは必ず `--` の後ろに置く。

### ファイルを添付する

`-f` でファイルを添付する。このとき標準入力の内容が添付のコメントになる。

```bash
echo "本日の集計結果です" | notify-slack -c '#report' -f report.csv
```

## オプション

| フラグ | 短縮 | 説明 | デフォルト |
|---|---|---|---|
| `--token` | `-t` | Slack API トークン（環境変数 `SLACK_API_TOKEN` でも指定可） | （なし） |
| `--channel` | `-c` | 投稿先チャンネル | `#general` |
| `--user` | `-u` | 表示ユーザー名 | （なし） |
| `--icon` | `-i` | アイコン絵文字（例 `:ghost:`） | （なし） |
| `--file` | `-f` | 添付するファイル | （なし） |
| `--env-file` | `-e` | 読み込む dotenv ファイル | 下記参照 |

## 環境変数ファイル（dotenv）

`SLACK_API_TOKEN` などを dotenv ファイルから読み込める。トークンを毎回渡さずに
済む。

- `-e <path>` を指定すると、そのファイルを読み込む（読めなければエラー）。
- 未指定のときは、存在すれば `./.env` を、続いて `/etc/notify-slack.env` を
  読み込む。
- 優先順位は「実際の環境変数 ＞ `./.env` ＞ `/etc/notify-slack.env`」。既に
  設定済みの環境変数は上書きしない。

```dotenv
# .env
SLACK_API_TOKEN=xoxb-xxxxxxxx
```

## 開発

開発者向けの詳細（アーキテクチャ、ビルドやテスト、lint、ツール管理、リリース手順）は
[CLAUDE.md](CLAUDE.md) にまとめている。

```bash
go test ./...                     # テスト
go tool golangci-lint run ./...   # lint
```

lint / test / 脆弱性チェックは GitHub Actions で push / PR ごとに実行する。`v` から
始まるタグを push すると各プラットフォーム向けバイナリが GitHub Releases に配布される
（手順は [CLAUDE.md](CLAUDE.md) を参照）。
