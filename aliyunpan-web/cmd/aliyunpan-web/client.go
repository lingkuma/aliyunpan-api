package main

import (
	"encoding/base64"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/tickstep/aliyunpan-api/aliyunpan"
	"github.com/tickstep/aliyunpan-api/aliyunpan_web"
)

type app struct {
	opts *options
	cfg  *envFile

	tok     *aliyunpan_web.WebLoginToken
	wc      *aliyunpan_web.WebPanClient
	user    *aliyunpan.UserInfo
	driveID string
}

func newApp(opts *options) (*app, error) {
	path := strings.TrimSpace(opts.envPath)
	if path == "" {
		if v := strings.TrimSpace(os.Getenv(envFileEnv)); v != "" {
			path = v
		} else {
			p, err := defaultEnvPath()
			if err != nil {
				return nil, err
			}
			path = p
		}
	}
	cfg, err := loadEnvFile(path)
	if err != nil {
		return nil, fmt.Errorf("读取 %s 失败: %v", path, err)
	}
	return &app{opts: opts, cfg: cfg}, nil
}

// token 返回可用的访问令牌，必要时使用 refresh_token 自动刷新并写回 .env
func (a *app) token(force bool) (*aliyunpan_web.WebLoginToken, error) {
	if a.tok != nil && !force {
		return a.tok, nil
	}

	at := a.cfg.get(keyAccessToken)
	rt := a.cfg.get(keyRefreshToken)
	exp, hasExp := parseExpireTime(a.cfg.get(keyExpireTime))

	usable := at != "" && looksLikeAccessToken(at)
	if usable && hasExp && time.Until(exp) < 2*time.Minute {
		usable = false
	}
	if force {
		usable = false
	}

	if !usable {
		tok, err := a.refreshToken()
		if err != nil {
			if at != "" && looksLikeAccessToken(at) {
				fmt.Fprintf(os.Stderr, "警告: %v，继续使用现有 access_token\n", err)
				usable = true
			} else {
				return nil, err
			}
		} else {
			a.tok = tok
			return tok, nil
		}
	}

	tok := &aliyunpan_web.WebLoginToken{
		AccessTokenType: "Bearer",
		AccessToken:     at,
		RefreshToken:    rt,
	}
	if hasExp {
		tok.ExpireTime = exp.Format("2006-01-02 15:04:05")
		tok.ExpiresIn = int(time.Until(exp).Seconds())
	}
	a.tok = tok
	return tok, nil
}

func (a *app) refreshToken() (*aliyunpan_web.WebLoginToken, error) {
	rt := a.cfg.get(keyRefreshToken)
	if rt == "" {
		return nil, fmt.Errorf("缺少 refresh_token，请在 %s 中配置 %s", a.cfg.path, keyRefreshToken)
	}
	tok, apiErr := aliyunpan_web.GetAccessTokenFromRefreshToken(rt)
	if apiErr != nil {
		return nil, fmt.Errorf("刷新 access_token 失败: %s", apiErr.Error())
	}
	if tok.AccessTokenType == "" {
		tok.AccessTokenType = "Bearer"
	}
	a.cfg.set(keyAccessToken, tok.AccessToken)
	if tok.RefreshToken != "" {
		a.cfg.set(keyRefreshToken, tok.RefreshToken)
	}
	a.cfg.set(keyExpiresIn, strconv.Itoa(tok.ExpiresIn))
	a.cfg.set(keyExpireTime, tok.ExpireTime)
	if err := a.cfg.save(); err != nil {
		fmt.Fprintf(os.Stderr, "警告: 更新 %s 失败: %v\n", a.cfg.path, err)
	} else {
		fmt.Fprintf(os.Stderr, "已刷新 access_token 并写入 %s\n", a.cfg.path)
	}
	return tok, nil
}

func (a *app) client() (*aliyunpan_web.WebPanClient, error) {
	if a.wc != nil {
		return a.wc, nil
	}
	tok, err := a.token(false)
	if err != nil {
		return nil, err
	}

	appID := firstNonEmpty(a.cfg.get(keyAppID), defaultAppID)
	deviceID := a.cfg.get(keyDeviceID)
	if deviceID == "" {
		deviceID = randomString(32)
		a.cfg.set(keyDeviceID, deviceID)
		if e := a.cfg.save(); e != nil {
			fmt.Fprintf(os.Stderr, "警告: 写入设备ID到 %s 失败: %v\n", a.cfg.path, e)
		}
	}
	appConfig := aliyunpan_web.AppConfig{
		AppId:    appID,
		DeviceId: deviceID,
		UserId:   a.cfg.get(keyUserID),
	}
	sessionConfig := aliyunpan_web.SessionConfig{
		DeviceName: firstNonEmpty(a.cfg.get(keyDeviceName), defaultDeviceName),
		ModelName:  firstNonEmpty(a.cfg.get(keyModelName), defaultModelName),
	}
	wc := aliyunpan_web.NewWebPanClient(*tok, aliyunpan_web.AppLoginToken{}, appConfig, sessionConfig)
	if a.opts.timeout > 0 {
		wc.SetTimeout(a.opts.timeout)
	}
	if _, apiErr := wc.CreateSession(nil); apiErr != nil {
		fmt.Fprintf(os.Stderr, "警告: 创建设备会话失败: %s（部分接口可能不可用）\n", apiErr.Error())
	} else if a.opts.verbose {
		fmt.Fprintln(os.Stderr, "设备会话已创建")
	}
	a.wc = wc
	return wc, nil
}

func (a *app) userInfo() (*aliyunpan.UserInfo, error) {
	if a.user != nil {
		return a.user, nil
	}
	wc, err := a.client()
	if err != nil {
		return nil, err
	}
	ui, apiErr := wc.GetUserInfo()
	if apiErr != nil {
		return nil, fmt.Errorf("获取用户信息失败: %s", apiErr.Error())
	}
	a.user = ui
	wc.UpdateUserId(ui.UserId)
	return ui, nil
}

// drive 解析当前使用的网盘ID：--drive > .env 中的 ALIYUNPAN_DRIVE_ID > 默认备份盘
func (a *app) drive() (string, error) {
	if a.driveID != "" {
		return a.driveID, nil
	}
	arg := strings.TrimSpace(a.opts.drive)
	if arg != "" && !isDriveAlias(arg) {
		a.driveID = arg
		return arg, nil
	}
	if arg == "" {
		if v := a.cfg.get(keyDriveID); v != "" {
			a.driveID = v
			return v, nil
		}
	}
	ui, err := a.userInfo()
	if err != nil {
		return "", err
	}
	var id string
	switch strings.ToLower(arg) {
	case "", "file", "backup":
		id = ui.FileDriveId
	case "resource":
		id = ui.ResourceDriveId
	case "album":
		id = ui.AlbumDriveId
	case "safe", "safebox":
		id = ui.SafeBoxDriveId
	default:
		id = arg
	}
	if id == "" {
		return "", fmt.Errorf("无法确定网盘ID（--drive %s），请执行 whoami 查看可用网盘ID", arg)
	}
	a.driveID = id
	return id, nil
}

func isDriveAlias(s string) bool {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "file", "backup", "resource", "album", "safe", "safebox":
		return true
	}
	return false
}

func parseExpireTime(s string) (time.Time, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, false
	}
	layouts := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02T15:04:05Z0700",
		"2006-01-02 15:04:05",
		"2006-01-02",
	}
	for _, layout := range layouts {
		if t, err := time.ParseInLocation(layout, s, time.Local); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

// looksLikeAccessToken 过滤聊天记录里被截断的 token
func looksLikeAccessToken(s string) bool {
	if s == "" || strings.Contains(s, "…") || strings.Contains(s, "...") {
		return false
	}
	parts := strings.Split(s, ".")
	if len(parts) != 3 {
		return false
	}
	for _, p := range parts {
		if p == "" {
			return false
		}
		if _, err := base64.RawURLEncoding.DecodeString(p); err != nil {
			return false
		}
	}
	return true
}
