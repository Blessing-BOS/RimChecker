package ai

import (
	"encoding/json"
	"strings"
	"testing"
	"unicode/utf8"

	"rimchecker/internal/logparse"
)

func TestRedact(t *testing.T) {
	text := `C:\Users\Alice\AppData\Player.log /home/bob/game/log /Users/carol/log
api_key="some-secret" Authorization: Bearer abc123 password=hunter2
steam_id=76561191234567890 username=Alice person@example.com
sk-abcdefghijklmnop custom-known-secret`
	got := Redact(text, "custom-known-secret")
	for _, secret := range []string{"Alice", "bob", "carol", "some-secret", "abc123", "hunter2", "76561191234567890", "person@example.com", "sk-abcdefghijklmnop", "custom-known-secret"} {
		if strings.Contains(got, secret) {
			t.Errorf("unredacted %q", secret)
		}
	}
	if Redact("Synthetic.Mod.Tick() C:\\Mods\\Example\\item42.xml") != "Synthetic.Mod.Tick() C:\\Mods\\Example\\item42.xml" {
		t.Fatal("non-private mod evidence removed")
	}
}

func TestBuildInputLimits(t *testing.T) {
	c := testConfig("https://example.invalid/v1")
	c.MaxEvents = 2
	c.MaxInputChars = 900
	result := &logparse.Result{Format: logparse.FormatLegacy, FilePath: "private-original-path"}
	for i := 0; i < 5; i++ {
		result.Diagnostics = append(result.Diagnostics, logparse.DiagnosticGroup{Event: logparse.DiagnosticEvent{Level: logparse.LevelError, Message: "api_key=secret123 " + strings.Repeat("合成异常", 1000), Stack: []string{strings.Repeat("Synthetic.Tick()", 1000)}, RawText: "raw-private-log"}, Count: 3, FirstStartLine: 10, FirstEndLine: 12, LastStartLine: 50, LastEndLine: 52})
	}
	text, err := BuildInput(result, c)
	if err != nil {
		t.Fatal(err)
	}
	var input analysisInput
	if err := json.Unmarshal([]byte(text), &input); err != nil {
		t.Fatal(err)
	}
	if utf8.RuneCountInString(text) > c.MaxInputChars || !input.Truncated || input.IncludedGroups > 2 || input.Errors != 15 || input.TotalGroups != 5 || input.OmittedGroups < 3 {
		t.Fatal("limits or counts incorrect")
	}
	for _, private := range []string{"secret123", "private-original-path", "raw-private-log"} {
		if strings.Contains(text, private) {
			t.Error("private data included")
		}
	}
	if len(input.Groups) > 0 && input.Groups[0].FirstStart != 10 {
		t.Fatal("line numbers lost")
	}
}

func TestBuildInputWithoutTruncation(t *testing.T) {
	c := testConfig("https://example.invalid/v1")
	r := &logparse.Result{Format: logparse.FormatWaylith, Diagnostics: []logparse.DiagnosticGroup{{Event: logparse.DiagnosticEvent{Level: logparse.LevelWarning, Message: "Synthetic warning", Stack: []string{"at Synthetic.Run()"}}, Count: 2}}}
	text, err := BuildInput(r, c)
	if err != nil {
		t.Fatal(err)
	}
	var in analysisInput
	_ = json.Unmarshal([]byte(text), &in)
	if in.Truncated || in.Warnings != 2 || in.IncludedGroups != 1 || in.Groups[0].Stack != "at Synthetic.Run()" {
		t.Fatal("unexpected input")
	}
}
