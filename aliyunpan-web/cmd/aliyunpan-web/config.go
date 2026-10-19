package main

import (
	"crypto/rand"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	keyAccessToken  = "ALIYUNPAN_ACCESS_TOKEN"
	keyRefreshToken = "ALIYUNPAN_REFRESH_TOKEN"
	keyExpiresIn    = "ALIYUNPAN_EXPIRES_IN"
	keyExpireTime   = "ALIYUNPAN_EXPIRE_TIME"
	keyAppID        = "ALIYUNPAN_APP_ID"
	keyDeviceID     = "ALIYUNPAN_DEVICE_ID"
	keyDeviceName   = "ALIYUNPAN_DEVICE_NAME"
	keyModelName    = "ALIYUNPAN_MODEL_NAME"
	keyDriveID      = "ALIYUNPAN_DRIVE_ID"
	keyUserID       = "ALIYUNPAN_USER_ID"

	defaultAppID      = "25dzX3vbYqktVxyX"
	defaultDeviceName = "Chrome浏览器"
	defaultModelName  = "Windows网页版"

	// EnvFileEnv 自定义 .env 路径的环境变量
	envFileEnv = "ALIYUNPAN_ENV_FILE"
	// RootEnv 自定义项目根目录的环境变量
	rootEnv = "ALIYUNPAN_ROOT"
)

// canonicalOrder 决定 .env 中新增字段的顺序
var canonicalOrder = []string{
	keyAccessToken,
	keyRefreshToken,
	keyExpiresIn,
	keyExpireTime,
	keyAppID,
	keyDeviceID,
	keyDeviceName,
	keyModelName,
	keyDriveID,
	keyUserID,
}

var envAliases = map[string][]string{
	keyAccessToken:  {"ALIYUNPAN_ACCESS_TOKEN", "ALIYUNPAN_TOKEN", "access_token", "ACCESS_TOKEN"},
	keyRefreshToken: {"ALIYUNPAN_REFRESH_TOKEN", "REFRESH_TOKEN", "refresh_token"},
	keyExpiresIn:    {"ALIYUNPAN_EXPIRES_IN", "EXPIRES_IN", "expires_in"},
	keyExpireTime:   {"ALIYUNPAN_EXPIRE_TIME", "EXPIRE_TIME", "expire_time"},
	keyAppID:        {"ALIYUNPAN_APP_ID", "APP_ID", "app_id"},
	keyDeviceID:     {"ALIYUNPAN_DEVICE_ID", "DEVICE_ID", "device_id"},
	keyDeviceName:   {"ALIYUNPAN_DEVICE_NAME", "DEVICE_NAME", "device_name"},
	keyModelName:    {"ALIYUNPAN_MODEL_NAME", "MODEL_NAME", "model_name"},
	keyDriveID:      {"ALIYUNPAN_DRIVE_ID", "DRIVE_ID", "drive_id"},
	keyUserID:       {"ALIYUNPAN_USER_ID", "USER_ID", "user_id"},
}

// envFile 是极简的 .env 读写实现：保留注释与未知字段，只更新已知字段。
type envFile struct {
	path     string
	values   map[string]string
	lines    []string
	lineKeys map[int]string
}

func loadEnvFile(path string) (*envFile, error) {
	e := &envFile{
		path:     path,
		values:   map[string]string{},
		lineKeys: map[int]string{},
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return e, nil
		}
		return nil, err
	}
	text := strings.ReplaceAll(string(data), "\r\n", "\n")
	text = strings.TrimSuffix(text, "\n")
	if text == "" {
		return e, nil
	}
	e.lines = strings.Split(text, "\n")
	for i, line := range e.lines {
		k, v, ok := parseEnvLine(line)
		if !ok {
			continue
		}
		canon := canonicalKey(k)
		if canon == "" {
			continue
		}
		if _, exists := e.values[canon]; !exists {
			e.values[canon] = v
		}
		e.lineKeys[i] = canon
	}
	return e, nil
}

func parseEnvLine(line string) (key, value string, ok bool) {
	s := strings.TrimSpace(line)
	if s == "" || strings.HasPrefix(s, "#") {
		return "", "", false
	}
	s = strings.TrimPrefix(s, "export ")
	idx := strings.IndexByte(s, '=')
	if tab := strings.IndexByte(s, '\t'); tab >= 0 && (idx < 0 || tab < idx) {
		idx = tab
	}
	if idx <= 0 {
		return "", "", false
	}
	key = strings.TrimSpace(s[:idx])
	value = strings.TrimSpace(s[idx+1:])
	if len(value) >= 2 {
		if (value[0] == '\'' && value[len(value)-1] == '\'') || (value[0] == '"' && value[len(value)-1] == '"') {
			value = value[1 : len(value)-1]
		}
	}
	if key == "" {
		return "", "", false
	}
	return key, value, true
}

func canonicalKey(k string) string {
	u := strings.ToUpper(strings.TrimSpace(k))
	switch u {
	case "ALIYUNPAN_ACCESS_TOKEN", "ALIYUNPAN_TOKEN", "ACCESS_TOKEN":
		return keyAccessToken
	case "ALIYUNPAN_REFRESH_TOKEN", "REFRESH_TOKEN":
		return keyRefreshToken
	case "ALIYUNPAN_EXPIRES_IN", "EXPIRES_IN":
		return keyExpiresIn
	case "ALIYUNPAN_EXPIRE_TIME", "EXPIRE_TIME":
		return keyExpireTime
	case "ALIYUNPAN_APP_ID", "APP_ID":
		return keyAppID
	case "ALIYUNPAN_DEVICE_ID", "DEVICE_ID":
		return keyDeviceID
	case "ALIYUNPAN_DEVICE_NAME", "DEVICE_NAME":
		return keyDeviceName
	case "ALIYUNPAN_MODEL_NAME", "MODEL_NAME":
		return keyModelName
	case "ALIYUNPAN_DRIVE_ID", "DRIVE_ID":
		return keyDriveID
	case "ALIYUNPAN_USER_ID", "USER_ID":
		return keyUserID
	}
	return ""
}

// get 优先读取进程环境变量，其次读取 .env 文件
func (e *envFile) get(key string) string {
	for _, k := range envAliases[key] {
		if v := strings.TrimSpace(os.Getenv(k)); v != "" {
			return v
		}
	}
	return strings.TrimSpace(e.values[key])
}

func (e *envFile) set(key, value string) {
	e.values[key] = value
}

// save 保留原有注释和顺序，仅写回已知字段
func (e *envFile) save() error {
	written := map[string]bool{}
	out := make([]string, 0, len(e.lines)+len(canonicalOrder))
	for i, line := range e.lines {
		canon, ok := e.lineKeys[i]
		if ok {
			if written[canon] {
				continue
			}
			v, exists := e.values[canon]
			if !exists {
				continue
			}
			out = append(out, canon+"="+v)
			written[canon] = true
			continue
		}
		out = append(out, line)
	}
	for _, k := range canonicalOrder {
		if written[k] {
			continue
		}
		v, exists := e.values[k]
		if !exists || v == "" {
			continue
		}
		out = append(out, k+"="+v)
		written[k] = true
	}
	content := strings.Join(out, "\n")
	if !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	if dir := filepath.Dir(e.path); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}
	return os.WriteFile(e.path, []byte(content), 0600)
}

// defaultEnvPath 优先使用与可执行文件同级的技能目录中的 .env，
// 这样整个技能目录可以直接移动后继续使用；其次回退到项目根目录。
func defaultEnvPath() (string, error) {
	if exe, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exe)
		for _, p := range []string{
			filepath.Join(exeDir, "..", ".env"),
			filepath.Join(exeDir, ".env"),
		} {
			if _, statErr := os.Stat(p); statErr == nil {
				return filepath.Clean(p), nil
			}
		}
	}

	root, err := findRepoRoot()
	if err != nil {
		return "", err
	}
	p := filepath.Join(root, ".env")
	if _, err := os.Stat(p); err == nil {
		return p, nil
	}
	alt := filepath.Join(root, "aliyunpan-web", ".env")
	if _, err := os.Stat(alt); err == nil {
		return alt, nil
	}
	return p, nil
}

func findRepoRoot() (string, error) {
	if v := strings.TrimSpace(os.Getenv(rootEnv)); v != "" {
		if abs, err := filepath.Abs(v); err == nil {
			return abs, nil
		}
	}
	var starts []string
	if wd, err := os.Getwd(); err == nil {
		starts = append(starts, wd)
	}
	if exe, err := os.Executable(); err == nil {
		starts = append(starts, filepath.Dir(exe))
	}
	for _, start := range starts {
		dir := start
		for {
			if isRepoRoot(dir) {
				return dir, nil
			}
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
			dir = parent
		}
	}
	return "", fmt.Errorf("找不到 aliyunpan-api 项目根目录，请使用 --env 指定 .env，或设置 %s", rootEnv)
}

func isRepoRoot(dir string) bool {
	data, err := os.ReadFile(filepath.Join(dir, "go.mod"))
	if err != nil {
		return false
	}
	return strings.Contains(string(data), "module github.com/tickstep/aliyunpan-api")
}

func randomString(n int) string {
	const alphabet = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		s := "T6ZJyY7JqX6EN2cDzLCxMVYZ"
		for len(s) < n {
			s += s
		}
		return s[:n]
	}
	for i := range buf {
		buf[i] = alphabet[int(buf[i])%len(alphabet)]
	}
	return string(buf)
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
