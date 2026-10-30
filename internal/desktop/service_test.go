package desktop

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"rimchecker/internal/ai"
	"rimchecker/internal/logparse"
)

func newTestService(t *testing.T, handler http.HandlerFunc) (*Service, string, *atomic.Int32) {
	t.Helper()
	calls := new(atomic.Int32)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls.Add(1); handler(w, r) }))
	t.Cleanup(server.Close)
	path := filepath.Join(t.TempDir(), "synthetic log.log")
	if err := os.WriteFile(path, []byte("Error: 合成异常\n  at Synthetic.Run()\nWarning: synthetic warning\n"), 0600); err != nil {
		t.Fatal(err)
	}
	s, err := New(Options{Directory: t.TempDir(), SelectFile: func() (string, error) { return path, nil }, HTTPClient: server.Client()})
	if err != nil {
		t.Fatal(err)
	}
	if err := s.SaveAIConfig(Settings{server.URL + "/v1", "mock-model", 5}); err != nil {
		t.Fatal(err)
	}
	if err := s.SetAPIKey("synthetic-desktop-key"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.SelectLogFile(); err != nil {
		t.Fatal(err)
	}
	if _, err := s.ParseLog(); err != nil {
		t.Fatal(err)
	}
	return s, path, calls
}
func okAI(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, `{"choices":[{"message":{"content":"## 合成诊断\n无需修改存档。"}}]}`)
}

func TestParseMatchesSharedParser(t *testing.T) {
	s, path, _ := newTestService(t, okAI)
	report, err := s.ParseLog()
	if err != nil {
		t.Fatal(err)
	}
	core, err := logparse.Parse(path)
	if err != nil {
		t.Fatal(err)
	}
	if report.TotalLines != core.TotalLines || report.Format != core.Format || report.Errors != 1 || report.Warnings != 1 || !reflect.DeepEqual(report.Diagnostics, core.Diagnostics) {
		t.Fatal("GUI diverged from CLI/shared parser")
	}
}
func TestSettingsAndPromptPersistence(t *testing.T) {
	s, _, _ := newTestService(t, okAI)
	if err := s.SavePrompt("自定义提示词"); err != nil {
		t.Fatal(err)
	}
	other, err := New(Options{Directory: s.options.Directory})
	if err != nil {
		t.Fatal(err)
	}
	if other.GetAIConfig().Settings != s.GetAIConfig().Settings || other.GetAIConfig().HasKey {
		t.Fatal("settings reload or memory-only key failed")
	}
	if prompt, err := other.GetPrompt(); err != nil || prompt != "自定义提示词" {
		t.Fatal("prompt load failed")
	}
	if err := other.SavePrompt(other.ResetPrompt()); err != nil {
		t.Fatal(err)
	}
	if prompt, _ := other.GetPrompt(); prompt != other.ResetPrompt() {
		t.Fatal("prompt reset failed")
	}
	files, err := os.ReadDir(s.options.Directory)
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range files {
		data, err := os.ReadFile(filepath.Join(s.options.Directory, file.Name()))
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(data), "synthetic-desktop-key") {
			t.Fatal("API key persisted")
		}
	}
	before, _ := other.GetPrompt()
	if err := other.SavePrompt(""); err == nil {
		t.Fatal("empty prompt saved")
	}
	after, _ := other.GetPrompt()
	if before != after {
		t.Fatal("invalid edit changed prompt")
	}
}
func TestConfirmedSnapshotIsExactlySent(t *testing.T) {
	received := make(chan ai.Request, 1)
	s, path, calls := newTestService(t, func(w http.ResponseWriter, r *http.Request) {
		var req ai.Request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Error(err)
		}
		received <- req
		okAI(w, r)
	})
	preview, err := s.PreviewAIRequest()
	if err != nil {
		t.Fatal(err)
	}
	expected, _ := json.Marshal(preview.Request)
	if _, err := s.DiagnoseWithAI(preview.Token, false); err == nil || calls.Load() != 0 {
		t.Fatal("unconfirmed request sent")
	}
	if calls.Load() != 0 {
		t.Fatal("preview sent request")
	}
	// External file edits and mutations of the returned DTO cannot change the snapshot.
	if err := os.WriteFile(path, []byte("Error: changed after preview"), 0600); err != nil {
		t.Fatal(err)
	}
	preview.Request.Messages[0].Content = "tampered frontend payload"
	text, err := s.DiagnoseWithAI(preview.Token, true)
	if err != nil || !strings.Contains(text, "合成诊断") {
		t.Fatalf("mock diagnosis failed: %v", err)
	}
	actual, _ := json.Marshal(<-received)
	if string(expected) != string(actual) {
		t.Fatal("sent request differs from preview")
	}
	if _, err := s.DiagnoseWithAI(preview.Token, true); err == nil || calls.Load() != 1 {
		t.Fatal("snapshot reused")
	}
}
func TestChangesInvalidatePreview(t *testing.T) {
	for _, change := range []struct {
		name string
		do   func(*Service) error
	}{
		{"settings", func(s *Service) error { return s.SaveAIConfig(s.GetAIConfig().Settings) }},
		{"key", func(s *Service) error { return s.SetAPIKey("new-synthetic-key") }},
		{"prompt", func(s *Service) error { return s.SavePrompt("修改提示词") }},
		{"selection", func(s *Service) error { _, err := s.SelectLogFile(); return err }},
		{"parse", func(s *Service) error { _, err := s.ParseLog(); return err }},
		{"discard", func(s *Service) error { s.DiscardPreview(); return nil }},
	} {
		t.Run(change.name, func(t *testing.T) {
			s, _, calls := newTestService(t, okAI)
			p, err := s.PreviewAIRequest()
			if err != nil {
				t.Fatal(err)
			}
			if err := change.do(s); err != nil {
				t.Fatal(err)
			}
			if _, err := s.DiagnoseWithAI(p.Token, true); err == nil || calls.Load() != 0 {
				t.Fatal("stale preview sent")
			}
		})
	}
}
func TestCancelDiagnosis(t *testing.T) {
	started := make(chan struct{})
	s, _, _ := newTestService(t, func(w http.ResponseWriter, r *http.Request) {
		close(started)
		select {
		case <-r.Context().Done():
		case <-time.After(3 * time.Second):
		}
	})
	preview, err := s.PreviewAIRequest()
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { _, err := s.DiagnoseWithAI(preview.Token, true); done <- err }()
	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("mock request did not start")
	}
	s.CancelDiagnosis()
	select {
	case err := <-done:
		if err == nil || !strings.Contains(err.Error(), "取消") {
			t.Fatalf("cancel result: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("cancellation hung")
	}
}
func TestEmptyMissingAndNoSelection(t *testing.T) {
	s, path, _ := newTestService(t, okAI)
	if err := os.WriteFile(path, nil, 0600); err != nil {
		t.Fatal(err)
	}
	r, err := s.ParseLog()
	if err != nil || r.TotalLines != 0 {
		t.Fatal("empty file mishandled")
	}
	if _, err := s.PreviewAIRequest(); err == nil {
		t.Fatal("empty input accepted")
	}
	s.selected = filepath.Join(t.TempDir(), "missing.log")
	if _, err := s.ParseLog(); err == nil {
		t.Fatal("missing file accepted")
	}
	s.selected = ""
	if _, err := s.ParseLog(); err == nil {
		t.Fatal("unselected file accepted")
	}
}
