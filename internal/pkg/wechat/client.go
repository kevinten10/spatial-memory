package wechat

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/spatial-memory/spatial-memory/internal/config"
)

// UserInfo is the user profile from WeChat OAuth.
type UserInfo struct {
	OpenID   string `json:"openid"`
	Nickname string `json:"nickname"`
	Avatar   string `json:"headimgurl"`
}

// Client is the interface for WeChat OAuth operations.
type Client interface {
	ExchangeCode(ctx context.Context, code string) (*UserInfo, error)
}

type client struct {
	cfg        config.WeChatConfig
	httpClient *http.Client
}

func NewClient(cfg config.WeChatConfig) Client {
	return &client{
		cfg: cfg,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

type tokenResponse struct {
	AccessToken string `json:"access_token"`
	OpenID      string `json:"openid"`
	ErrCode     int    `json:"errcode"`
	ErrMsg      string `json:"errmsg"`
}

func (c *client) ExchangeCode(ctx context.Context, code string) (*UserInfo, error) {
	// Step 1: Exchange code for access token
	tokenURL := fmt.Sprintf(
		"https://api.weixin.qq.com/sns/oauth2/access_token?appid=%s&secret=%s&code=%s&grant_type=authorization_code",
		c.cfg.AppID, c.cfg.AppSecret, code,
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, tokenURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create token request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("exchange code: %w", err)
	}
	defer resp.Body.Close()

	var tokenResp tokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("decode token response: %w", err)
	}
	if tokenResp.ErrCode != 0 {
		return nil, fmt.Errorf("wechat token error: %d %s", tokenResp.ErrCode, tokenResp.ErrMsg)
	}

	// Step 2: Get user info
	infoURL := fmt.Sprintf(
		"https://api.weixin.qq.com/sns/userinfo?access_token=%s&openid=%s",
		tokenResp.AccessToken, tokenResp.OpenID,
	)

	req, err = http.NewRequestWithContext(ctx, http.MethodGet, infoURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create userinfo request: %w", err)
	}

	resp, err = c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("get user info: %w", err)
	}
	defer resp.Body.Close()

	var info UserInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, fmt.Errorf("decode user info: %w", err)
	}

	if info.OpenID == "" {
		return nil, fmt.Errorf("empty openid in wechat response")
	}

	return &info, nil
}
