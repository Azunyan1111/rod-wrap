package webview

import (
	"encoding/json"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// ChromeProfile はChromeプロファイルの情報を表す
type ChromeProfile struct {
	Name      string `json:"name"`
	Directory string `json:"directory"`
	Path      string `json:"path"`
}

// GetChromeUserDataDir はChromeのユーザーデータディレクトリのパスを返す
func GetChromeUserDataDir() string {
	switch runtime.GOOS {
	case "darwin":
		home, _ := os.UserHomeDir()
		return filepath.Join(home, "Library", "Application Support", "Google", "Chrome")
	case "windows":
		return filepath.Join(os.Getenv("LOCALAPPDATA"), "Google", "Chrome", "User Data")
	default: // linux
		home, _ := os.UserHomeDir()
		return filepath.Join(home, ".config", "google-chrome")
	}
}

// ListChromeProfiles はChromeのプロファイル一覧を取得する
func ListChromeProfiles() ([]ChromeProfile, error) {
	userDataDir := GetChromeUserDataDir()
	localStatePath := filepath.Join(userDataDir, "Local State")

	data, err := os.ReadFile(localStatePath)
	if err != nil {
		return nil, err
	}

	var localState struct {
		Profile struct {
			InfoCache map[string]struct {
				Name string `json:"name"`
			} `json:"info_cache"`
		} `json:"profile"`
	}

	if err := json.Unmarshal(data, &localState); err != nil {
		return nil, err
	}

	profiles := make([]ChromeProfile, 0, len(localState.Profile.InfoCache))
	for dir, info := range localState.Profile.InfoCache {
		profiles = append(profiles, ChromeProfile{
			Name:      info.Name,
			Directory: dir,
			Path:      filepath.Join(userDataDir, dir),
		})
	}

	return profiles, nil
}

// skipFiles はコピー時にスキップするファイル名のパターン
var skipFiles = []string{
	"SingletonLock",
	"SingletonSocket",
	"SingletonCookie",
	"lockfile",
	"LOCK",
	"LOG",
	"LOG.old",
}

// shouldSkipFile はファイルをスキップすべきかどうかを判定する
func shouldSkipFile(name string) bool {
	for _, skip := range skipFiles {
		if name == skip || strings.HasPrefix(name, skip) {
			return true
		}
	}
	return false
}

// CopyProfile は既存のプロファイルを指定したディレクトリにコピーする
// セッション、Cookie、ログイン情報などが引き継がれる
func CopyProfile(profile ChromeProfile, destDir string) error {
	return copyDir(profile.Path, destDir)
}

// copyDir はディレクトリを再帰的にコピーする
func copyDir(src, dst string) error {
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(dst, srcInfo.Mode()); err != nil {
		return err
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		// ロックファイルなどはスキップ
		if shouldSkipFile(entry.Name()) {
			continue
		}

		if entry.IsDir() {
			if err := copyDir(srcPath, dstPath); err != nil {
				// ディレクトリコピーエラーは警告のみでスキップ
				slog.Warn("Failed to copy directory", "path", srcPath, "error", err)
				continue
			}
		} else {
			if err := copyFile(srcPath, dstPath); err != nil {
				// ファイルコピーエラーは警告のみでスキップ
				slog.Warn("Failed to copy file", "path", srcPath, "error", err)
				continue
			}
		}
	}

	return nil
}

// copyFile はファイルをコピーする
func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	srcInfo, err := srcFile.Stat()
	if err != nil {
		return err
	}

	dstFile, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, srcInfo.Mode())
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	return err
}
