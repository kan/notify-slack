package main

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/slack-go/slack"
)

// errReader is an io.Reader that always fails, used to exercise the
// "read stdin" error-wrapping path in run() without touching real stdin.
type errReader struct{}

func (errReader) Read(_ []byte) (int, error) {
	return 0, errors.New("boom")
}

func TestRunStdinReadError(t *testing.T) {
	saved := *token
	defer func() { *token = saved }()

	*token = "xoxb-dummy"
	_, err := run(errReader{})
	if err == nil {
		t.Fatal("run() should return an error when stdin read fails")
	}
	if !strings.Contains(err.Error(), "read stdin") {
		t.Errorf("error should mention read stdin, got %q", err.Error())
	}
	if !strings.Contains(err.Error(), "boom") {
		t.Errorf("error should wrap the underlying read error, got %q", err.Error())
	}
}

func TestRunFileReadError(t *testing.T) {
	savedToken := *token
	savedFile := *file
	defer func() {
		*token = savedToken
		*file = savedFile
	}()

	*token = "xoxb-dummy"
	*file = filepath.Join(t.TempDir(), "does-not-exist.csv")

	_, err := run(strings.NewReader("hello"))
	if err == nil {
		t.Fatal("run() should return an error when the upload file cannot be read")
	}
	if !strings.Contains(err.Error(), "read file") {
		t.Errorf("error should mention read file, got %q", err.Error())
	}
	var pathErr *fs.PathError
	if !errors.As(err, &pathErr) {
		t.Errorf("error should wrap a *fs.PathError, got %v (%T)", err, err)
	}
}

func TestRunRequiresToken(t *testing.T) {
	saved := *token
	defer func() { *token = saved }()

	*token = ""
	_, err := run(strings.NewReader("hello"))
	if err == nil {
		t.Fatal("run() with empty token should return an error")
	}
	if !strings.Contains(err.Error(), "token") {
		t.Errorf("error should mention token, got %q", err.Error())
	}
}

func TestRunEmptyStdinIsNoop(t *testing.T) {
	savedToken := *token
	defer func() { *token = savedToken }()

	*token = "xoxb-dummy"
	if _, err := run(strings.NewReader("")); err != nil {
		t.Errorf("run() with empty stdin should be a no-op, got %v", err)
	}
}

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

func TestRunCommand(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		cmdName       string
		args          []string
		wantExitCode  int
		wantOutput    string
		wantOutputEq  bool
		wantOutputHas []string
	}{
		{
			name:         "success with output",
			cmdName:      "sh",
			args:         []string{"-c", "printf out"},
			wantExitCode: 0,
			wantOutput:   "out",
			wantOutputEq: true,
		},
		{
			name:          "non-zero exit captures stderr",
			cmdName:       "sh",
			args:          []string{"-c", "echo e >&2; exit 3"},
			wantExitCode:  3,
			wantOutputHas: []string{"e"},
		},
		{
			name:         "no output",
			cmdName:      "sh",
			args:         []string{"-c", "exit 5"},
			wantExitCode: 5,
			wantOutput:   "",
			wantOutputEq: true,
		},
		{
			name:          "stdout and stderr are combined",
			cmdName:       "sh",
			args:          []string{"-c", "printf out1; echo out2 >&2"},
			wantExitCode:  0,
			wantOutputHas: []string{"out1", "out2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			output, code := runCommand(tt.cmdName, tt.args)

			if code != tt.wantExitCode {
				t.Errorf("exit code = %d, want %d", code, tt.wantExitCode)
			}
			if tt.wantOutputEq {
				if got := string(output); got != tt.wantOutput {
					t.Errorf("output = %q, want %q", got, tt.wantOutput)
				}
			}
			for _, want := range tt.wantOutputHas {
				if !strings.Contains(string(output), want) {
					t.Errorf("output = %q, want it to contain %q", output, want)
				}
			}
		})
	}
}

func TestRunCommandStartupFailure(t *testing.T) {
	t.Parallel()

	output, code := runCommand("notify-slack-definitely-not-a-real-command-xyz", nil)

	if code != 127 {
		t.Errorf("exit code = %d, want 127", code)
	}
	if len(output) == 0 {
		t.Error("output should contain the launch error text, got empty output")
	}
}

func TestBuildCommandMessage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		cmdName  string
		args     []string
		output   []byte
		exitCode int
		want     string
	}{
		{
			name:     "with args",
			cmdName:  "sh",
			args:     []string{"-c", "true"},
			output:   []byte("output\n"),
			exitCode: 0,
			want:     "$ sh -c true (exit 0)\noutput\n",
		},
		{
			name:     "without args",
			cmdName:  "ls",
			args:     nil,
			output:   []byte(""),
			exitCode: 2,
			want:     "$ ls (exit 2)\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := buildCommandMessage(tt.cmdName, tt.args, tt.output, tt.exitCode); got != tt.want {
				t.Errorf("buildCommandMessage() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestRunCommandModeEmptyOutputDoesNotPost exercises run()'s command mode with
// a command that produces no output. runCommandMode must skip the Slack post
// (no network access) and simply propagate the command's exit code.
func TestRunCommandModeEmptyOutputDoesNotPost(t *testing.T) {
	savedToken := *token
	savedCommand := *command
	defer func() {
		*token = savedToken
		*command = savedCommand
	}()

	*token = "xoxb-dummy"
	*command = []string{"sh", "-c", "exit 7"}

	code, err := run(strings.NewReader(""))
	if err != nil {
		t.Fatalf("run() returned unexpected error: %v", err)
	}
	if code != 7 {
		t.Errorf("run() code = %d, want 7", code)
	}
}

// withUnsetEnv ensures key is unset for the duration of the test, restoring
// its previous value (or leaving it unset) afterwards. Tests using it mutate
// global process environment state, so they must not run in parallel.
func withUnsetEnv(t *testing.T, key string) {
	t.Helper()
	if old, ok := os.LookupEnv(key); ok {
		t.Cleanup(func() { _ = os.Setenv(key, old) })
	} else {
		t.Cleanup(func() { _ = os.Unsetenv(key) })
	}
	_ = os.Unsetenv(key)
}

func TestEnvFilePathFromArgs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "no args", args: nil, want: ""},
		{name: "no env-file flag present", args: []string{"--token", "xoxb-x", "-c", "#general"}, want: ""},
		{name: "-e with space-separated path", args: []string{"-e", "/path/to/.env"}, want: "/path/to/.env"},
		{name: "-e attached path", args: []string{"-e/path/to/.env"}, want: "/path/to/.env"},
		{name: "--env-file with space-separated path", args: []string{"--env-file", "/path/to/.env"}, want: "/path/to/.env"},
		{name: "--env-file= attached path", args: []string{"--env-file=/path/to/.env"}, want: "/path/to/.env"},
		{name: "-e alone with no following value", args: []string{"-e"}, want: ""},
		{name: "--env-file alone with no following value", args: []string{"--env-file"}, want: ""},
		{name: "-e mixed among other args", args: []string{"--token", "xoxb-x", "-e", "/tmp/x.env", "-c", "#general"}, want: "/tmp/x.env"},
		{name: "-e after -- is part of the command and must not be detected", args: []string{"--token", "xoxb-x", "--", "sh", "-c", "-e $HOME"}, want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := envFilePathFromArgs(tt.args); got != tt.want {
				t.Errorf("envFilePathFromArgs(%v) = %q, want %q", tt.args, got, tt.want)
			}
		})
	}
}

func TestLoadEnvFileSetsUnsetVars(t *testing.T) {
	const key = "NOTIFY_SLACK_TEST_LOADENVFILE_NEW"
	withUnsetEnv(t, key)

	path := filepath.Join(t.TempDir(), ".env")
	if err := os.WriteFile(path, []byte(key+"=file-value\n"), 0o600); err != nil {
		t.Fatalf("write env file: %v", err)
	}

	if err := loadEnvFile(path); err != nil {
		t.Fatalf("loadEnvFile() returned error: %v", err)
	}

	if got, want := os.Getenv(key), "file-value"; got != want {
		t.Errorf("%s = %q, want %q", key, got, want)
	}
}

func TestLoadEnvFileDoesNotOverrideExistingVar(t *testing.T) {
	const key = "NOTIFY_SLACK_TEST_LOADENVFILE_OVERRIDE"
	t.Setenv(key, "existing-value")

	path := filepath.Join(t.TempDir(), ".env")
	if err := os.WriteFile(path, []byte(key+"=file-value\n"), 0o600); err != nil {
		t.Fatalf("write env file: %v", err)
	}

	if err := loadEnvFile(path); err != nil {
		t.Fatalf("loadEnvFile() returned error: %v", err)
	}

	if got, want := os.Getenv(key), "existing-value"; got != want {
		t.Errorf("%s = %q, want %q (existing env var must not be overridden)", key, got, want)
	}
}

func TestLoadEnvFileMissingPath(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "does-not-exist.env")

	err := loadEnvFile(path)
	if err == nil {
		t.Fatal("loadEnvFile() should return an error for a missing file")
	}
	if !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("error should wrap fs.ErrNotExist, got %v", err)
	}
}

func TestLoadEnvFileParseError(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), ".env")
	if err := os.WriteFile(path, []byte("NOEQUALSSIGN\n"), 0o600); err != nil {
		t.Fatalf("write env file: %v", err)
	}

	if err := loadEnvFile(path); err == nil {
		t.Fatal("loadEnvFile() should return an error for an invalid env file")
	}
}

func TestLoadEnvDashESelectsExplicitFile(t *testing.T) {
	const key = "NOTIFY_SLACK_TEST_LOADENV_EXPLICIT"
	withUnsetEnv(t, key)

	path := filepath.Join(t.TempDir(), "custom.env")
	if err := os.WriteFile(path, []byte(key+"=explicit-value\n"), 0o600); err != nil {
		t.Fatalf("write env file: %v", err)
	}

	if err := loadEnv([]string{"-e", path}, nil); err != nil {
		t.Fatalf("loadEnv() returned error: %v", err)
	}

	if got, want := os.Getenv(key), "explicit-value"; got != want {
		t.Errorf("%s = %q, want %q", key, got, want)
	}
}

func TestLoadEnvDashEMissingFileReturnsError(t *testing.T) {
	t.Parallel()

	missing := filepath.Join(t.TempDir(), "missing.env")

	err := loadEnv([]string{"-e", missing}, nil)
	if err == nil {
		t.Fatal("loadEnv() with -e pointing to a missing file should return an error")
	}
	if !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("error should wrap fs.ErrNotExist, got %v", err)
	}
}

func TestLoadEnvDashETakesPrecedenceOverDefaults(t *testing.T) {
	const key = "NOTIFY_SLACK_TEST_LOADENV_PRECEDENCE"
	withUnsetEnv(t, key)

	dir := t.TempDir()
	explicitPath := filepath.Join(dir, "explicit.env")
	defaultPath := filepath.Join(dir, "default.env")

	if err := os.WriteFile(explicitPath, []byte(key+"=explicit-value\n"), 0o600); err != nil {
		t.Fatalf("write explicit env file: %v", err)
	}
	if err := os.WriteFile(defaultPath, []byte(key+"=default-value\n"), 0o600); err != nil {
		t.Fatalf("write default env file: %v", err)
	}

	if err := loadEnv([]string{"--env-file=" + explicitPath}, []string{defaultPath}); err != nil {
		t.Fatalf("loadEnv() returned error: %v", err)
	}

	if got, want := os.Getenv(key), "explicit-value"; got != want {
		t.Errorf("%s = %q, want %q (-e should take precedence over defaults)", key, got, want)
	}
}

func TestLoadEnvDefaultsPriorityAndMissingIgnored(t *testing.T) {
	const key = "NOTIFY_SLACK_TEST_LOADENV_DEFAULTS_PRIORITY"
	withUnsetEnv(t, key)

	dir := t.TempDir()
	localPath := filepath.Join(dir, "local.env")
	systemPath := filepath.Join(dir, "system.env")
	missingPath := filepath.Join(dir, "does-not-exist.env")

	if err := os.WriteFile(localPath, []byte(key+"=local-value\n"), 0o600); err != nil {
		t.Fatalf("write local env file: %v", err)
	}
	if err := os.WriteFile(systemPath, []byte(key+"=system-value\n"), 0o600); err != nil {
		t.Fatalf("write system env file: %v", err)
	}

	if err := loadEnv(nil, []string{localPath, systemPath, missingPath}); err != nil {
		t.Fatalf("loadEnv() returned error: %v", err)
	}

	if got, want := os.Getenv(key), "local-value"; got != want {
		t.Errorf("%s = %q, want %q (local default should win over system default)", key, got, want)
	}
}

func TestLoadEnvDoesNotOverrideExistingVar(t *testing.T) {
	const key = "NOTIFY_SLACK_TEST_LOADENV_OVERRIDE"
	t.Setenv(key, "existing-value")

	path := filepath.Join(t.TempDir(), ".env")
	if err := os.WriteFile(path, []byte(key+"=dotenv-value\n"), 0o600); err != nil {
		t.Fatalf("write env file: %v", err)
	}

	if err := loadEnv([]string{"-e", path}, nil); err != nil {
		t.Fatalf("loadEnv() returned error: %v", err)
	}

	if got, want := os.Getenv(key), "existing-value"; got != want {
		t.Errorf("%s = %q, want %q (existing env var must not be overridden)", key, got, want)
	}
}

func TestLoadEnvDefaultsParseErrorPropagates(t *testing.T) {
	const key = "NOTIFY_SLACK_TEST_LOADENV_DEFAULTS_PARSE_ERROR"
	withUnsetEnv(t, key)

	dir := t.TempDir()
	badPath := filepath.Join(dir, "bad.env")
	laterPath := filepath.Join(dir, "later.env")

	if err := os.WriteFile(badPath, []byte("NOEQUALSSIGN\n"), 0o600); err != nil {
		t.Fatalf("write bad env file: %v", err)
	}
	if err := os.WriteFile(laterPath, []byte(key+"=later-value\n"), 0o600); err != nil {
		t.Fatalf("write later env file: %v", err)
	}

	err := loadEnv(nil, []string{badPath, laterPath})
	if err == nil {
		t.Fatal("loadEnv() should return an error when a default file (other than the missing one) fails to parse")
	}
	if errors.Is(err, fs.ErrNotExist) {
		t.Errorf("error should not be fs.ErrNotExist, got %v", err)
	}

	if got, want := os.Getenv(key), ""; got != want {
		t.Errorf("%s = %q, want %q (loadEnv should stop at the first non-missing error and not read later defaults)", key, got, want)
	}
}

func TestLoadEnvNoArgsNoDefaultsIsNoop(t *testing.T) {
	t.Parallel()

	if err := loadEnv(nil, nil); err != nil {
		t.Errorf("loadEnv() with no args and no defaults should be a no-op, got %v", err)
	}
}
