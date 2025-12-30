# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## プロジェクト概要

rod-wrapは、go-rod/rodライブラリをラップしたGoパッケージです。Chromeブラウザをヘッドフルモードで操作するためのWebViewインターフェースを提供します。主な機能として、Chromeプロファイルの管理（一覧取得、コピー）、ページナビゲーション、DOM要素の値取得・設定、Cookie管理、要素リスナー（ポーリング方式）があります。

## ビルド・テストコマンド

```bash
# ビルド
go build ./...

# 全テスト実行
go test ./...

# 単一テスト実行
go test -run TestFunctionName ./...

# 詳細出力でテスト
go test -v ./...
```

## アーキテクチャ

### パッケージ構成

- `webview` パッケージ（モジュール名）として単一ファイル構成
- `WebView` インターフェースを中心とした設計
- `chromeWebView` が主要な実装（go-rod/rodを使用）

### 主要コンポーネント

**WebViewインターフェース**: Navigate, GetCurrentURL, GetValue, SetValue, SetReadOnly, Cookie操作, Listener管理, Run, Destroyメソッドを定義

**ChromeOption**: Functional Optionsパターンによる起動オプション設定
- `WithProfile`: プロファイルディレクトリ指定
- `WithUserDataDir`: ユーザーデータディレクトリ指定
- `WithChromeProfile`: ChromeProfile構造体から設定
- `WithCopiedProfile`: 既存プロファイルをコピーして使用

**要素監視**: goroutineによる500ms間隔のポーリングで要素値を監視

### 依存関係

- `github.com/go-rod/rod`: Chromeブラウザ制御
