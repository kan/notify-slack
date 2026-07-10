package main

import (
	"testing"

	"github.com/slack-go/slack"
)

func TestIsEmptyBody(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		body []byte
		want bool
	}{
		{name: "nil body", body: nil, want: true},
		{name: "empty slice", body: []byte{}, want: true},
		{name: "non-empty body", body: []byte("hello"), want: false},
		{name: "whitespace only body is not empty", body: []byte(" "), want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := isEmptyBody(tt.body); got != tt.want {
				t.Errorf("isEmptyBody(%q) = %v, want %v", tt.body, got, tt.want)
			}
		})
	}
}

func TestBuildUploadParams(t *testing.T) {
	t.Parallel()

	params := buildUploadParams("/tmp/some/dir/report.csv", []byte("col1,col2\n1,2\n"), []byte("please check"), "#general")

	if got, want := params.Filename, "report.csv"; got != want {
		t.Errorf("Filename = %q, want %q", got, want)
	}
	if got, want := params.Title, "report.csv"; got != want {
		t.Errorf("Title = %q, want %q", got, want)
	}
	if got, want := params.Content, "col1,col2\n1,2\n"; got != want {
		t.Errorf("Content = %q, want %q", got, want)
	}
	if got, want := params.FileSize, len("col1,col2\n1,2\n"); got != want {
		t.Errorf("FileSize = %d, want %d", got, want)
	}
	if got, want := params.InitialComment, "please check"; got != want {
		t.Errorf("InitialComment = %q, want %q", got, want)
	}
	if got, want := params.Channel, "#general"; got != want {
		t.Errorf("Channel = %q, want %q", got, want)
	}
}

func TestBuildUploadParamsFileNameWithoutDir(t *testing.T) {
	t.Parallel()

	params := buildUploadParams("report.csv", []byte("x"), []byte(""), "#random")

	if got, want := params.Filename, "report.csv"; got != want {
		t.Errorf("Filename = %q, want %q", got, want)
	}
	if got, want := params.InitialComment, ""; got != want {
		t.Errorf("InitialComment = %q, want %q", got, want)
	}
}

func TestBuildMsgOptions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		body         []byte
		user         string
		icon         string
		wantUsername string
		wantIconSet  bool
		wantIcon     string
	}{
		{
			name: "text only, no user or icon",
			body: []byte("hello world"),
		},
		{
			name:         "with user",
			body:         []byte("hello"),
			user:         "notify-bot",
			wantUsername: "notify-bot",
		},
		{
			name:        "with icon",
			body:        []byte("hello"),
			icon:        ":ghost:",
			wantIconSet: true,
			wantIcon:    ":ghost:",
		},
		{
			name:         "with user and icon",
			body:         []byte("hello"),
			user:         "notify-bot",
			icon:         ":ghost:",
			wantUsername: "notify-bot",
			wantIconSet:  true,
			wantIcon:     ":ghost:",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			opts := buildMsgOptions(tt.body, tt.user, tt.icon)

			// UnsafeApplyMsgOptions はネットワークアクセスせず MsgOption を
			// url.Values に適用するだけのテスト用ユーティリティ。
			_, values, err := slack.UnsafeApplyMsgOptions("xoxb-dummy", "#general", "", opts...)
			if err != nil {
				t.Fatalf("UnsafeApplyMsgOptions returned error: %v", err)
			}

			if got, want := values.Get("text"), string(tt.body); got != want {
				t.Errorf("text = %q, want %q", got, want)
			}

			if tt.user == "" {
				if values.Has("username") {
					t.Errorf("username should not be set, got %q", values.Get("username"))
				}
			} else if got := values.Get("username"); got != tt.wantUsername {
				t.Errorf("username = %q, want %q", got, tt.wantUsername)
			}

			if !tt.wantIconSet {
				if values.Has("icon_emoji") {
					t.Errorf("icon_emoji should not be set, got %q", values.Get("icon_emoji"))
				}
			} else if got := values.Get("icon_emoji"); got != tt.wantIcon {
				t.Errorf("icon_emoji = %q, want %q", got, tt.wantIcon)
			}
		})
	}
}
