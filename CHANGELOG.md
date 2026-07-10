# Changelog

このプロジェクトの主な変更点を記録する。形式は
[Keep a Changelog](https://keepachangelog.com/ja/1.1.0/) に準拠し、
バージョンは [Semantic Versioning](https://semver.org/lang/ja/) に従う。

## [Unreleased]

## [1.0.0] - 2026-07-10

初回のタグ付きリリース。長く運用してきたベースを整備し、1.0.0 として公開する。

### Added

- 標準入力を Slack に投稿する。
- コマンド実行モード。`-- <command>` で指定したコマンドを実行し、結合出力
  (stdout+stderr) を投稿する。コマンドの終了コードを引き継ぐ。
- ファイル添付（`-f`）。標準入力の内容を添付コメントにする。
- dotenv 対応（`-e`、指定が無ければ `./.env` → `/etc/notify-slack.env`）。既存の
  環境変数は上書きしない。
- 投稿先チャンネル / ユーザー名 / アイコンの指定（`-c` / `-u` / `-i`）。

### Changed

- Go modules 化し、依存を現行版へ更新（slack-go/slack、kingpin v2、godotenv）。
- エラー時に panic せず、メッセージを標準エラーに出して終了コードを返すようにした。

### Security

- govulncheck による既知脆弱性チェックを CI に追加。
- SECURITY.md を追加し、脆弱性報告の窓口を用意。

### Infrastructure

- GitHub Actions で lint / test / govulncheck を実行する。
- GoReleaser により、タグ push で各プラットフォーム向けバイナリを配布する。
- Dependabot で go.mod と GitHub Actions を更新する。開発ツール
  （golangci-lint / goreleaser / govulncheck）は go.mod の tool ディレクティブで
  管理する。

[Unreleased]: https://github.com/kan/notify-slack/compare/v1.0.0...HEAD
[1.0.0]: https://github.com/kan/notify-slack/releases/tag/v1.0.0
