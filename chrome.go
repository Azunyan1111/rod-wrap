package rod_wrap

import (
	"context"
	"log/slog"
	"os"
	"sync"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
	"github.com/go-rod/stealth"
)

type chromeWebView struct {
	browser   *rod.Browser
	page      *rod.Page
	elements  map[string]string
	listeners map[string]bool
	mu        sync.RWMutex
	ctx       context.Context
	cancel    context.CancelFunc
	stopChan  chan struct{}
	tmpDir    string // 一時ディレクトリ（Destroy時に削除）
}

// NewChromeWebView は新しいChromeWebViewを作成する
func NewChromeWebView(opts ...ChromeOption) WebView {
	options := &chromeOptions{}
	for _, opt := range opts {
		opt(options)
	}

	ctx, cancel := context.WithCancel(context.Background())

	// システムのChromeを探す
	chromePath, found := launcher.LookPath()
	if !found {
		slog.Error("Chrome not found in system")
	}

	// ブラウザを起動
	l := launcher.New().
		Bin(chromePath).
		Headless(options.headless).
		Delete("use-mock-keychain").
		Set("window-size", "1280,720")

	// ヘッドフルモードの場合はウィンドウ表示を有効化
	if !options.headless {
		l = l.Delete("no-startup-window")
	}

	// プロファイル設定
	if options.userDataDir != "" {
		l = l.UserDataDir(options.userDataDir)
	}
	if options.profileDir != "" {
		l = l.ProfileDir(options.profileDir)
	}

	slog.Info("Chrome launch args", "args", l.FormatArgs(), "userDataDir", options.userDataDir, "profileDir", options.profileDir)

	url := l.MustLaunch()

	browser := rod.New().
		ControlURL(url).
		Context(ctx).
		MustConnect()

	// stealthモードでページを作成（自動化検出を回避）
	page := stealth.MustPage(browser)

	// ウィンドウサイズを設定
	page.MustSetWindow(0, 0, 1280, 720)

	return &chromeWebView{
		browser:   browser,
		page:      page,
		elements:  make(map[string]string),
		listeners: make(map[string]bool),
		ctx:       ctx,
		cancel:    cancel,
		stopChan:  make(chan struct{}),
		tmpDir:    options.tmpDir,
	}
}

func (c *chromeWebView) Navigate(url string) {
	slog.Info("Navigate", "url", url)

	// Navigate前に既存の変数をクリア
	c.mu.Lock()
	c.elements = make(map[string]string)
	c.mu.Unlock()

	// 実際のURLに移動
	err := c.page.Navigate(url)
	if err != nil {
		slog.Error("Navigate failed", "error", err)
		return
	}

	// ページの読み込みを待機
	c.page.MustWaitLoad()

	// リスナーが設定されている場合は再設定
	c.mu.RLock()
	for elementID := range c.listeners {
		c.setupListener(elementID)
	}
	c.mu.RUnlock()
}

func (c *chromeWebView) GetCurrentURL() string {
	info, err := c.page.Info()
	if err != nil {
		slog.Error("GetCurrentURL failed", "error", err)
		return ""
	}
	return info.URL
}

func (c *chromeWebView) GetValue(elementID string) string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.elements[elementID]
}

func (c *chromeWebView) SetValue(elementID, value string) {
	c.mu.Lock()
	c.elements[elementID] = value
	c.mu.Unlock()

	// 要素を取得して値を設定
	el, err := c.page.Element("#" + elementID)
	if err != nil {
		slog.Error("SetValue: element not found", "elementID", elementID, "error", err)
		return
	}

	err = el.Input(value)
	if err != nil {
		slog.Error("SetValue: input failed", "elementID", elementID, "error", err)
	}
}

func (c *chromeWebView) SetReadOnly(elementID string, readOnly bool) {
	el, err := c.page.Element("#" + elementID)
	if err != nil {
		slog.Error("SetReadOnly: element not found", "elementID", elementID, "error", err)
		return
	}

	_, err = el.Eval(`(readOnly) => {
		if (this.tagName === 'INPUT' || this.tagName === 'TEXTAREA') {
			this.readOnly = readOnly;
			this.disabled = readOnly;
			if (readOnly) {
				this.style.backgroundColor = '#f0f0f0';
				this.style.cursor = 'not-allowed';
			} else {
				this.style.backgroundColor = '';
				this.style.cursor = '';
			}
		} else if (this.tagName === 'SELECT') {
			this.disabled = readOnly;
		} else {
			this.contentEditable = readOnly ? 'false' : 'true';
		}
	}`, readOnly)
	if err != nil {
		slog.Error("SetReadOnly failed", "elementID", elementID, "error", err)
	}
}

func (c *chromeWebView) SetCookie(key, value, domain string) {
	cookies := []*proto.NetworkCookieParam{
		{
			Name:   key,
			Value:  value,
			Domain: domain,
			Path:   "/",
		},
	}
	c.page.SetCookies(cookies)
}

func (c *chromeWebView) GetCookie(key, domain string) string {
	// 特定ドメインのCookieを取得するためにURLを指定
	var urls []string
	if domain != "" {
		urls = []string{"https://" + domain}
	}
	cookies, err := c.page.Cookies(urls)
	if err != nil {
		slog.Error("GetCookie failed", "error", err)
		return ""
	}

	for _, cookie := range cookies {
		if cookie.Name == key && (cookie.Domain == domain ||
			cookie.Domain == "."+domain ||
			domain == "" ||
			cookie.Domain == "") {
			return cookie.Value
		}
	}

	return ""
}

func (c *chromeWebView) ClearCookie() {
	c.page.SetCookies(nil)
}

func (c *chromeWebView) AddListener(elementID string) {
	c.mu.Lock()
	c.listeners[elementID] = true
	c.mu.Unlock()

	c.setupListener(elementID)
}

func (c *chromeWebView) RemoveListener(elementID string) {
	c.mu.Lock()
	delete(c.listeners, elementID)
	delete(c.elements, elementID)
	c.mu.Unlock()
}

func (c *chromeWebView) setupListener(elementID string) {
	// 初期値を取得
	el, err := c.page.Element("#" + elementID)
	if err != nil {
		slog.Error("setupListener: element not found", "elementID", elementID, "error", err)
		return
	}

	prop, err := el.Property("value")
	if err == nil {
		c.mu.Lock()
		c.elements[elementID] = prop.Str()
		c.mu.Unlock()
	}

	// ポーリングで値を監視するgoroutineを起動
	go c.pollElementValue(elementID)
}

func (c *chromeWebView) pollElementValue(elementID string) {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			c.mu.RLock()
			_, exists := c.listeners[elementID]
			c.mu.RUnlock()

			if !exists {
				return
			}

			el, err := c.page.Element("#" + elementID)
			if err != nil {
				continue
			}

			prop, err := el.Property("value")
			if err != nil {
				continue
			}

			c.mu.Lock()
			c.elements[elementID] = prop.Str()
			c.mu.Unlock()
		}
	}
}

func (c *chromeWebView) Run() {
	// ブラウザが閉じられるまで待機
	select {
	case <-c.ctx.Done():
	case <-c.stopChan:
	}
}

func (c *chromeWebView) Destroy() {
	c.cancel()
	close(c.stopChan)
	c.page.Close()
	c.browser.Close()

	// 一時ディレクトリが存在する場合は削除
	if c.tmpDir != "" {
		if err := os.RemoveAll(c.tmpDir); err != nil {
			slog.Warn("Failed to remove temp dir", "path", c.tmpDir, "error", err)
		}
	}
}
