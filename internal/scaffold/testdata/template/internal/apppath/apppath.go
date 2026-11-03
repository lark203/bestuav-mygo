// Package apppath 统一计算应用的 per-user 目录与文件路径。
//
// 为什么单独一个包：配置（settings.json）、日志（logs/）、数据库（data.db）都放在
// 同一个应用目录下，但分别由 internal/settings、internal/logging、internal/store
// 使用。若各自写一遍 os.UserConfigDir() + appmeta.Name 的拼接，一旦目录规则调整
// （比如改成 UserCacheDir）就要改三处，且很容易改漏。收敛到这里，只改一处。
//
// 目录布局（Windows 为例，macOS/Linux 见各自平台惯例）：
//
//	%APPDATA%\bestuav-mygo\
//	├── settings.json   用户偏好（internal/settings）
//	├── data.db         SQLite 数据库（internal/store）
//	└── logs\           按天轮转的日志（internal/logging）
package apppath

import (
	"fmt"
	"os"
	"path/filepath"

	"bestuav-mygo/internal/appmeta"
)

// ConfigDir 返回本应用的 per-user 根目录：<用户配置目录>/<产品名>。
func ConfigDir() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		home, herr := os.UserHomeDir()
		if herr != nil {
			return "", fmt.Errorf("无法定位配置目录: %w", err)
		}
		dir = home
	}
	return filepath.Join(dir, appmeta.Name), nil
}

// LogDir 返回日志目录：<配置目录>/logs。
func LogDir() (string, error) {
	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "logs"), nil
}

// DBPath 返回 SQLite 数据库文件路径：<配置目录>/data.db。
func DBPath() (string, error) {
	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "data.db"), nil
}

// EnsureDir 创建目录（含父级），已存在时返回 nil。
func EnsureDir(dir string) error { return os.MkdirAll(dir, 0o755) }
