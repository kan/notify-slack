package main

import (
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

	api := slack.New(*token)

	body, err := io.ReadAll(os.Stdin)
	if err != nil {
		panic(err)
	}
	if isEmptyBody(body) {
		return
	}

	if *file != "" {
		content, err := os.ReadFile(*file)
		if err != nil {
			panic(err)
		}
		params := buildUploadParams(*file, content, body, *channel)
		if _, err := api.UploadFile(params); err != nil {
			panic(err)
		}
		return
	}

	opts := buildMsgOptions(body, *user, *icon)
	if _, _, err := api.PostMessage(*channel, opts...); err != nil {
		panic(err)
	}
}
