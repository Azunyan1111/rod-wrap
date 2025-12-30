package webview

// WebView はブラウザ操作のインターフェース
type WebView interface {
	// Navigate は指定したURLに移動する
	Navigate(url string)
	// GetCurrentURL は現在のURLを取得する
	GetCurrentURL() string
	// GetValue は指定した要素の値を取得する
	GetValue(elementID string) string
	// SetValue は指定した要素に値を設定する
	SetValue(elementID, value string)
	// SetReadOnly は指定した要素の読み取り専用状態を設定する
	SetReadOnly(elementID string, readOnly bool)
	// SetCookie はCookieを設定する（Chromeプロファイルに直接設定）
	SetCookie(key, value, domain string)
	// GetCookie は指定したCookieの値を取得する（Chromeプロファイルから直接取得）
	GetCookie(key, domain string) string
	// ClearCookie は全てのCookieをクリアする
	ClearCookie()
	// AddListener は指定した要素の値変更を監視する
	AddListener(elementID string)
	// RemoveListener は指定した要素の監視を解除する
	RemoveListener(elementID string)
	// Run はブラウザが閉じられるまで待機する
	Run()
	// Destroy はブラウザを終了する
	Destroy()
}
