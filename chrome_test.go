package webview

import (
	"os"
	"strings"
	"testing"
	"time"
)

func TestListChromeProfiles(t *testing.T) {
	profiles, err := ListChromeProfiles()
	if err != nil {
		t.Logf("ListChromeProfiles returned error (Chrome may not be installed): %v", err)
		return
	}

	t.Logf("Found %d profiles:", len(profiles))
	for _, p := range profiles {
		t.Logf("  - Name: %s, Directory: %s, Path: %s", p.Name, p.Directory, p.Path)
	}
}

func TestGetChromeUserDataDir(t *testing.T) {
	dir := GetChromeUserDataDir()
	if dir == "" {
		t.Error("GetChromeUserDataDir returned empty string")
	}
	t.Logf("Chrome user data dir: %s", dir)
}

func TestChromeWebView_WithProfile(t *testing.T) {
	profiles, err := ListChromeProfiles()
	if err != nil {
		t.Skipf("Chrome profiles not available: %v", err)
	}

	profile := findDefaultProfile(profiles)
	if profile == nil {
		t.Skip("No profiles found")
	}

	t.Logf("Found default profile: %s (%s)", profile.Name, profile.Directory)

	// WithCopiedProfileは自動的に一時ディレクトリにコピーする
	// Destroy時に一時ディレクトリは自動削除される
	wv := NewChromeWebView(
		WithCopiedProfile(*profile),
	)
	defer wv.Destroy()

	wv.Navigate("https://example.com/")

	url := wv.GetCurrentURL()
	if !strings.HasPrefix(url, "https://example.com/") {
		t.Errorf("expected https://example.com/, got %s", url)
	}

	time.Sleep(time.Second * 5)

	t.Logf("Successfully navigated with custom profile")
}

func TestCopyProfile(t *testing.T) {
	profiles, err := ListChromeProfiles()
	if err != nil {
		t.Skipf("Chrome profiles not available: %v", err)
	}

	profile := findDefaultProfile(profiles)
	if profile == nil {
		t.Skip("No profiles found")
	}

	tmpDir := t.TempDir()

	err = CopyProfile(*profile, tmpDir)
	if err != nil {
		t.Fatalf("CopyProfile failed: %v", err)
	}

	// コピーされたファイルの確認
	entries, err := os.ReadDir(tmpDir)
	if err != nil {
		t.Fatalf("Failed to read copied directory: %v", err)
	}

	t.Logf("Copied %d entries to %s", len(entries), tmpDir)

	// 重要なファイルが存在するか確認
	expectedFiles := []string{"Cookies", "Preferences"}
	for _, name := range expectedFiles {
		path := tmpDir + "/" + name
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Logf("Note: %s not found (may not exist in source)", name)
		} else {
			t.Logf("Found: %s", name)
		}
	}
}

func TestChromeWebView_NavigateAndGetURL(t *testing.T) {
	wv := NewChromeWebView()
	defer wv.Destroy()

	wv.Navigate("https://example.com/")

	url := wv.GetCurrentURL()
	if url != "https://example.com//" {
		t.Errorf("expected https://example.com//, got %s", url)
	}
}

func TestChromeWebView_SetAndGetValue(t *testing.T) {
	wv := NewChromeWebView()
	defer wv.Destroy()

	// テスト用のシンプルなHTMLページを読み込む
	wv.Navigate("data:text/html,<html><body><input id='test-input' type='text'></body></html>")

	time.Sleep(500 * time.Millisecond)

	wv.AddListener("test-input")
	time.Sleep(500 * time.Millisecond)

	wv.SetValue("test-input", "hello world")
	time.Sleep(500 * time.Millisecond)

	value := wv.GetValue("test-input")
	if value != "hello world" {
		t.Errorf("expected 'hello world', got '%s'", value)
	}
}

func TestChromeWebView_Cookie(t *testing.T) {
	wv := NewChromeWebView()
	defer wv.Destroy()

	// Navigateせずに特定ドメインのCookieを設定・取得できることを確認
	wv.SetCookie("test-key", "test-value", "example.com")

	time.Sleep(500 * time.Millisecond)

	// ドメイン指定でCookieを取得（Navigateなし）
	value := wv.GetCookie("test-key", "example.com")
	if value != "test-value" {
		t.Errorf("expected 'test-value', got '%s'", value)
	}

	wv.ClearCookie()
	time.Sleep(500 * time.Millisecond)

	value = wv.GetCookie("test-key", "example.com")
	if value != "" {
		t.Errorf("expected empty string after clear, got '%s'", value)
	}
}

func TestChromeWebView_SetReadOnly(t *testing.T) {
	wv := NewChromeWebView()
	defer wv.Destroy()

	// テスト用のHTMLページを読み込む
	wv.Navigate("data:text/html,<html><body><input id='readonly-test' type='text' value='test'></body></html>")

	time.Sleep(500 * time.Millisecond)

	// 読み取り専用に設定
	wv.SetReadOnly("readonly-test", true)

	time.Sleep(500 * time.Millisecond)

	// 読み取り専用を解除
	wv.SetReadOnly("readonly-test", false)

	time.Sleep(500 * time.Millisecond)

	t.Logf("SetReadOnly test completed successfully")
}

func TestChromeWebView_Listener(t *testing.T) {
	wv := NewChromeWebView()
	defer wv.Destroy()

	wv.Navigate("data:text/html,<html><body><input id='listener-test' type='text' value='initial'></body></html>")

	time.Sleep(500 * time.Millisecond)

	wv.AddListener("listener-test")
	time.Sleep(1 * time.Second)

	value := wv.GetValue("listener-test")
	if value != "initial" {
		t.Errorf("expected 'initial', got '%s'", value)
	}

	wv.RemoveListener("listener-test")

	value = wv.GetValue("listener-test")
	if value != "" {
		t.Errorf("expected empty after remove listener, got '%s'", value)
	}
}

// findDefaultProfile はDefaultディレクトリのプロファイルを返すテストヘルパー
func findDefaultProfile(profiles []ChromeProfile) *ChromeProfile {
	for i, p := range profiles {
		if p.Directory == "Default" {
			return &profiles[i]
		}
	}
	if len(profiles) > 0 {
		return &profiles[0]
	}
	return nil
}
