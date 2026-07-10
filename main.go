package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/alecthomas/kingpin/v2"
	"github.com/slack-go/slack"
)

var (
	token   = kingpin.Flag("token", "Slack API token").Envar("SLACK_API_TOKEN").Short('t').String()
	channel = kingpin.Flag("channel", "channel for send").Default("#general").Short('c').String()
	user    = kingpin.Flag("user", "user name").Short('u').String()
	icon    = kingpin.Flag("icon", "icon emoji").Short('i').String()
	file    = kingpin.Flag("file", "upload file name").Short('f').String()
)

func isEmptyBody(body []byte) bool {
	return len(body) == 0
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
