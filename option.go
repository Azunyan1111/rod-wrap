package rod_wrap

import (
	"log/slog"
	"os"
	"path/filepath"
)

// ChromeOption はChromeWebViewの起動オプション
type ChromeOption func(*chromeOptions)

type chromeOptions struct {
	profileDir  string
	userDataDir string
	tmpDir      string // 自動作成された一時ディレクトリ（Destroy時に削除）
	headless    bool
}

// WithProfile は指定したプロファイルディレクトリでChromeを起動する
func WithProfile(profileDir string) ChromeOption {
	return func(o *chromeOptions) {
		o.profileDir = profileDir
	}
}

// WithUserDataDir はユーザーデータディレクトリを指定する
func WithUserDataDir(userDataDir string) ChromeOption {
	return func(o *chromeOptions) {
		o.userDataDir = userDataDir
	}
}

// WithHeadless はヘッドレスモードでChromeを起動する
func WithHeadless() ChromeOption {
	return func(o *chromeOptions) {
		o.headless = true
	}
}

// WithChromeProfile はChromeProfileを使用してプロファイルを指定する
// 注意: 既存のChromeが起動中の場合、同じプロファイルは使用できない
func WithChromeProfile(profile ChromeProfile) ChromeOption {
	return func(o *chromeOptions) {
		o.userDataDir = GetChromeUserDataDir()
		o.profileDir = profile.Directory
	}
}

// WithCopiedProfile は既存のプロファイルを一時ディレクトリにコピーして使用する
// セッション、Cookie、ログイン情報などが引き継がれる
// 既存のChromeが起動中でも使用可能
// 一時ディレクトリはDestroy時に自動的に削除される
func WithCopiedProfile(profile ChromeProfile) ChromeOption {
	return func(o *chromeOptions) {
		// 一時ディレクトリを作成
		tmpDir, err := os.MkdirTemp("", "rod-wrap-profile-")
		if err != nil {
			slog.Error("Failed to create temp dir", "error", err)
			return
		}
		o.tmpDir = tmpDir

		// プロファイルをtmpDir/Defaultにコピーする
		// Chromeはデフォルトで"Default"プロファイルを使用する
		profileDestDir := filepath.Join(tmpDir, "Default")
		slog.Info("Copying profile", "from", profile.Path, "to", profileDestDir)
		if err := CopyProfile(profile, profileDestDir); err != nil {
			slog.Error("Failed to copy profile", "error", err)
		}

		// Local Stateファイルもコピーする（暗号化キーが含まれている）
		chromeDataDir := GetChromeUserDataDir()
		localStateSrc := filepath.Join(chromeDataDir, "Local State")
		localStateDst := filepath.Join(tmpDir, "Local State")
		if err := copyFile(localStateSrc, localStateDst); err != nil {
			slog.Warn("Failed to copy Local State", "error", err)
		} else {
			slog.Info("Copied Local State file")
		}

		// コピー結果を確認
		entries, err := os.ReadDir(profileDestDir)
		if err != nil {
			slog.Error("Failed to read copied profile dir", "error", err)
		} else {
			slog.Info("Copied profile contents", "count", len(entries))
		}

		o.userDataDir = tmpDir
		o.profileDir = "Default"
	}
}
