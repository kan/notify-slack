# CLAUDE.md

notify-slack をコーディングエージェントが扱うための要点をまとめる。利用者向けの
使い方は [README.md](README.md) を参照。

## 概要

標準入力やコマンドの出力、ファイルを Slack に投稿する CLI。実体は `main.go` 1 本。

## アーキテクチャ

- `main()`: env 読み込み → `kingpin.Parse()` → `run()`。
- `run(stdin) (int, error)`: 投稿本体。第 1 戻り値がプロセスの終了コード
  （コマンドモードではコマンドの終了コードを引き継ぐ。それ以外は 0）。
- 3 つの入力モード:
  - 標準入力: stdin を投稿する（空なら投稿しない）。
  - コマンド実行: 位置引数（`-- <command>`）を実行し、結合出力(stdout+stderr)を
    投稿する（`runCommandMode` / `runCommand` / `buildCommandMessage`）。
  - ファイル添付: `-f` でアップロードし、stdin を添付コメントにする。
- env 読み込み: `loadEnv` が `kingpin.Parse` の前に dotenv を反映する
  （`-e` 指定、なければ `./.env` → `/etc/notify-slack.env`）。既存の環境変数は
  上書きしない。パスは `envFilePathFromArgs` が `os.Args` から直接抽出する
  （`--` 以降は走査しない）。

## 開発コマンド

ホストで直接 `go`（go.mod のバージョン、Go 1.26 系）が使える。

```bash
go build ./...                    # ビルド
go test ./...                     # テスト
go vet ./...                      # vet
gofmt -l .                        # フォーマット確認（修正は gofmt -w .）
go tool golangci-lint run ./...   # lint
go tool govulncheck ./...         # 既知脆弱性チェック
```

## ツール管理

`golangci-lint` / `goreleaser` / `govulncheck` は go.mod の tool ディレクティブで
管理する。`go tool <name>` で実行し、バージョンは go.mod に固定する（Dependabot が
更新を追う）。個別に `go install` しない。

## テスト規約

- 実 Slack API を叩かない。純粋関数と、投稿を伴わない経路だけを検証する。
- テーブル駆動。純粋関数は `t.Parallel()` を付ける。グローバルフラグや環境変数を
  触るテストは非並列にし、`t.Setenv` か `withUnsetEnv` で後始末する。
- `MsgOption` の検証は `slack.UnsafeApplyMsgOptions`（ネットワークに出ない）。

## コミット規約

- 日本語。1 行目に要約、2 行目を空行、3 行目以降に動機（なぜ）を書く。
- コード変更時は build / vet / gofmt / test / lint を通してからコミットする。
- 丸数字（①②③）は使わない。

## CI / ワークフロー

- `.github/workflows/ci.yml`: push(main) / PR で gofmt / vet / test / golangci-lint。
- `.github/workflows/security.yml`: push / PR / 週次で govulncheck。
- `.github/workflows/release.yml`: タグ `v*` の push で GoReleaser を実行。

## リリース手順

1. main が最新かつ CI グリーンであることを確認する。
2. バージョンタグを打つ: `git tag vX.Y.Z`
3. タグを push する: `git push origin vX.Y.Z`
4. release ワークフローが `go tool goreleaser release --clean` を実行し、各
   プラットフォーム（linux / darwin / windows × amd64 / arm64）のバイナリと
   `checksums.txt` を GitHub Releases に配布する。
5. [Releases](https://github.com/kan/notify-slack/releases) で成果物を確認する。
