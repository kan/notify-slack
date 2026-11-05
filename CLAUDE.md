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

ホストで直接 `go`（go.mod の go ディレクティブを満たすバージョン）が使える。

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
- **main へ直接 push しない。** ブランチを切って PR を作る。tagpr はマージされた PR から
  リリース PR を組み立てるので、直接 push した変更はリリースの確認から漏れる。
- 利用者に届く変更を入れる PR では、`CHANGELOG.md` の `## [Unreleased]` に項目を足す
  （下の「リリース手順」）。

## scripts/tagpr

[kan/tagpr-scripts](https://github.com/kan/tagpr-scripts) を git subtree で取り込んだもの。
使い方は同梱の README.md と AGENTS.md にある。

- **`scripts/tagpr/` の中は編集しない。** 直すときは kan/tagpr-scripts を直し、こちらへは
  次で取り込む（subtree は `commit.gpgsign` が効かないので `-S` を付ける）。

  ```bash
  git subtree pull -S --prefix=scripts/tagpr https://github.com/kan/tagpr-scripts.git main --squash
  ```

- **取り込む PR はマージコミットでマージする。** squash すると subtree が前回の取り込みを
  見つけられず、次の `git subtree pull` が壊れる。
- 取り込みは CHANGELOG に書かない（利用者に届く変更ではない）。

## CI / ワークフロー

- `.github/workflows/ci.yml`: push(main) / PR で gofmt / vet / test / golangci-lint。
- `.github/workflows/security.yml`: push / PR / 週次で govulncheck。
- `.github/workflows/tagpr.yml`: push(main) でリリース PR を作り直す。リリース PR の
  マージでタグを打ち、`release.yml` をそのタグで `workflow_dispatch` で起動する
  （GITHUB_TOKEN で push したタグでは `push: tags` が起動しないため）。
- `.github/workflows/release.yml`: タグで起動する。`build` ジョブ（`contents: read`）が
  CHANGELOG の節の有無を確かめてから、`goreleaser release --skip=publish` で
  アーカイブを作る。`publish` ジョブ（`contents: write`）は、CHANGELOG の節を本文に
  して Release を作る。書き込み権限を持つジョブではビルドしない。
- `.github/workflows/toolchain.yml`: 週次で go.mod の go ディレクティブが最新の
  安定版 Go かを確かめる（Dependabot は go ディレクティブを上げない）。
- `.github/workflows/verify-lockfile.yml`: push(main) / PR で `actions.lock` が
  ワークフローと一致しているかを `gh actions-lock --verify` で確かめる。
- Actions は `.github/workflows/actions.lock`（workflow lockfile）でコミットを
  固定する。`uses:` を足したり変えたりしたら `gh actions-lock` を再実行し、
  lockfile も一緒にコミットする。lockfile は手で編集しない。拡張は
  verify-lockfile.yml と同じ版を入れる
  （`gh extension install github/gh-actions-lock --pin v0.4.1`）。

## リリース手順

**リリースは tagpr のリリース PR をマージして始める**（v1.1.3 の次の版から）。
`CHANGELOG.md` の節の書き換えとタグは、リリース PR とワークフローが行う。
**「CHANGELOG に vX.Y.Z を追加」のコミットを手で作らない。タグも手で打たない。**
設定は `.tagpr`、CHANGELOG の処理は `scripts/tagpr/`（上の「scripts/tagpr」）。

1. **変更を入れる PR で CHANGELOG を書く**

   利用者に届く変更は、その PR で `## [Unreleased]` に足す。**この節がそのまま
   GitHub のリリースノートになる。** 項目の中のリンクはインラインで書く（参照形式の
   リンクの定義は節の外にあるので、リリースノートではリンクにならない）。末尾の
   リンク定義はリリース PR が版ごとに足すので、手で足さない。

2. **リリース PR を確かめる**

   ```bash
   gh pr list --label tagpr --state open --json number,title,headRefName
   gh run list --workflow tagpr.yml --branch main -L 3
   ```

   - **最新の tagpr の run が success であること。** 失敗していたら
     `release-prepare.mjs` の出力を読む。多いのは `## [Unreleased]` が空のとき。
     CHANGELOG を書く PR を main にマージすると作り直される
   - **版（PR のタイトル）が正しいこと。** 既定は patch。前回のリリース以降に
     マージした PR が `enhancement` ラベルを持っていれば minor になる
     （`.tagpr` の `minorLabels`）。手で変えるときはリリース PR にラベルを付けて作り直す

     ```bash
     gh pr edit <番号> --add-label tagpr:minor
     gh workflow run tagpr.yml --ref main
     ```

   - **PR のタイトルの版と CHANGELOG の見出しの版が一致していること。** タグの版は
     マージの時点のラベルで決まり、見出しは最後に作り直した時点の版のまま。
     食い違ったままマージすると、タグは打たれるが `release.yml` が節を見つけられずに
     止まる
   - Files changed が `CHANGELOG.md`（見出しとリンク定義）だけであること
   - 節を直すときは、リリース PR のブランチではなく **main の `## [Unreleased]` を
     直す**（ブランチは tagpr が作り直すたびに main から作られる）
   - 見出しの日付は、tagpr が最後にリリース PR を作り直した日（日本時間）。マージする
     日と違えば `gh workflow run tagpr.yml --ref main` で作り直す
   - minor 以上の版では、`SECURITY.md` のサポート対象（最新のマイナー系列）を
     先に main で更新しておく
   - リリース PR の CI は承認待ち（`action_required`）で止まることがある
     （GITHUB_TOKEN が作った PR のため）。そのときは今の head の run を承認する
     （`gh api -X POST repos/kan/notify-slack/actions/runs/<id>/approve`）

3. **リリース PR をマージする**

   ```bash
   gh pr merge <番号> --squash --delete-branch
   git switch main && git pull --ff-only
   gh run list --workflow tagpr.yml --commit "$(git rev-parse HEAD)" -L 1
   gh run list --workflow release.yml -L 1
   ```

   マージの push で走る tagpr の run がマージコミットにタグを打ち、`release` ジョブが
   `release.yml` をそのタグで起動する。**タグが打たれるまで main へほかの PR を
   マージしない。** tagpr の run は前の run が終わるまで待ち、待っている run は新しい
   push の run に置き換えられる（cancelled になる）。置き換えられたら、マージコミットの
   run を `gh run rerun <run-id>` で再実行するとタグが打たれる。

   リリースの直後は `## [Unreleased]` が空なので、次に CHANGELOG を書くまで main への
   push ごとに tagpr の run が失敗する。これは想定どおりで、その間のリリース PR は
   マージしない。

4. **Release の成果物と本文を確かめる**

   `release.yml` が各プラットフォーム（linux / darwin / windows × amd64 / arm64）の
   アーカイブと `checksums.txt` を Release に載せる。
   [Releases](https://github.com/kan/notify-slack/releases) で成果物と本文を確かめる。
   Release の本文は `node scripts/tagpr/release-notes.mjs vX.Y.Z` の出力と同じ。
