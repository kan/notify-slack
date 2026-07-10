package main

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/alecthomas/kingpin/v2"
	"github.com/joho/godotenv"
	"github.com/slack-go/slack"
)

var (
	token   = kingpin.Flag("token", "Slack API token").Envar("SLACK_API_TOKEN").Short('t').String()
	channel = kingpin.Flag("channel", "channel for send").Default("#general").Short('c').String()
	user    = kingpin.Flag("user", "user name").Short('u').String()
	icon    = kingpin.Flag("icon", "icon emoji").Short('i').String()
	file    = kingpin.Flag("file", "upload file name").Short('f').String()
	// env-file はフラグ解決より前（kingpin.Parse 前）に読み込む必要があるため
	// 値は loadEnv 側で os.Args を直接見て取得する。ここでは --help 表示と
	// 未知フラグ扱いの回避のために登録だけしておく。
	_ = kingpin.Flag("env-file", "load environment variables from a dotenv file (default: ./.env then /etc/notify-slack.env if present)").Short('e').String()
)

func isEmptyBody(body []byte) bool {
	return len(body) == 0
}

// envFilePathFromArgs は kingpin.Parse より前に env-file のパスを取り出す。
// -e / --env-file の各表記（"-e path" / "-epath" / "--env-file path" /
// "--env-file=path"）に対応する。見つからなければ空文字列を返す。
func envFilePathFromArgs(args []string) string {
	for i, a := range args {
		switch {
		case a == "--env-file" || a == "-e":
			if i+1 < len(args) {
				return args[i+1]
			}
		case strings.HasPrefix(a, "--env-file="):
			return strings.TrimPrefix(a, "--env-file=")
		case strings.HasPrefix(a, "-e") && a != "-e":
			return a[len("-e"):]
		}
	}
	return ""
}

// loadEnvFile は dotenv ファイルを読み込む。godotenv.Load は既に設定済みの
// 環境変数を上書きしないため、実環境変数がファイルの値より優先される。
func loadEnvFile(path string) error {
	return godotenv.Load(path)
}

// systemEnvFile は引数指定が無いときに参照するシステム全体の dotenv パス。
const systemEnvFile = "/etc/notify-slack.env"

// loadEnv は kingpin.Parse の前に呼ぶ。-e/--env-file が指定されていれば
// そのファイルを（読めなければエラーで）読み込む。指定が無ければ defaults を
// 先頭から順に、存在するものだけ読み込む。loadEnvFile は既存の環境変数を
// 上書きしないため、先に列挙したファイルの値が後のものより優先される。
func loadEnv(args []string, defaults []string) error {
	if path := envFilePathFromArgs(args); path != "" {
		return loadEnvFile(path)
	}
	for _, path := range defaults {
		if err := loadEnvFile(path); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return err
		}
	}
	return nil
}

func buildUploadParams(filePath string, content []byte, comment []byte, channel string) slack.UploadFileParameters {
	return slack.UploadFileParameters{
		Content:        string(content),
		FileSize:       len(content),
		Filename:       filepath.Base(filePath),
		Title:          filepath.Base(filePath),
		InitialComment: string(comment),
		Channel:        channel,
	}
}

func buildMsgOptions(body []byte, user, icon string) []slack.MsgOption {
	opts := []slack.MsgOption{slack.MsgOptionText(string(body), false)}
	if user != "" {
		opts = append(opts, slack.MsgOptionUsername(user))
	}
	if icon != "" {
		opts = append(opts, slack.MsgOptionIconEmoji(icon))
	}
	return opts
}

func main() {
	if err := loadEnv(os.Args[1:], []string{".env", systemEnvFile}); err != nil {
		fmt.Fprintln(os.Stderr, "notify-slack:", err)
		os.Exit(1)
	}

	kingpin.Parse()

	if err := run(os.Stdin); err != nil {
		fmt.Fprintln(os.Stderr, "notify-slack:", err)
		os.Exit(1)
	}
}

func run(stdin io.Reader) error {
	if *token == "" {
		return errors.New("missing Slack API token (set --token or SLACK_API_TOKEN)")
	}

	body, err := io.ReadAll(stdin)
	if err != nil {
		return fmt.Errorf("read stdin: %w", err)
	}
	if isEmptyBody(body) {
		return nil
	}

	api := slack.New(*token)

	if *file != "" {
		content, err := os.ReadFile(*file)
		if err != nil {
			return fmt.Errorf("read file %q: %w", *file, err)
		}
		if _, err := api.UploadFile(buildUploadParams(*file, content, body, *channel)); err != nil {
			return fmt.Errorf("upload file to %q: %w", *channel, err)
		}
		return nil
	}

	if _, _, err := api.PostMessage(*channel, buildMsgOptions(body, *user, *icon)...); err != nil {
		return fmt.Errorf("post message to %q: %w", *channel, err)
	}
	return nil
}
