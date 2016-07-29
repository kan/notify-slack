package main

import (
	"io/ioutil"
	"os"

	"github.com/alecthomas/kingpin"
	"github.com/nlopes/slack"
)

var (
	token   = kingpin.Flag("token", "Slack API token").OverrideDefaultFromEnvar("SLACK_API_TOKEN").Short('t').String()
	channel = kingpin.Flag("channel", "channel for send").Default("#general").Short('c').String()
	user    = kingpin.Flag("user", "user name").Short('u').String()
	icon    = kingpin.Flag("icon", "icon emoji").Short('i').String()
	file    = kingpin.Flag("file", "upload file name").Short('f').String()
)

func main() {
	kingpin.Parse()

	api := slack.New(*token)

	body, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		panic(err)
	}

	if file != nil {
		content, err := ioutil.ReadFile(*file)
		if err != nil {
			panic(err)
		}
		params := slack.FileUploadParameters{
			File: *file, Filetype: "auto", Content: string(content),
			Title: "file upload", InitialComment: string(body),
			Channels: []string{*channel},
		}
		_, err = api.UploadFile(params)
		if err != nil {
			panic(err)
		}
		return
	}

	params := slack.NewPostMessageParameters()
	if user != nil {
		params.Username = *user
	}
	if icon != nil {
		params.IconEmoji = *icon
	}

	_, _, err = api.PostMessage(*channel, string(body), params)
	if err != nil {
		panic(err)
	}
}
