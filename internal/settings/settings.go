// Package settings 负责应用用户偏好的持久化（纯 Go，无 UI 依赖）。
//
// 设计约定：
//   - 本包只放业务逻辑与可独立单测的函数，禁止 import
//     github.com/egoist/mygo/ui 或 mygo 的视图层，以保证脱离窗口也能测试；
//   - UI 层（internal/view）通过本包读写设置，不直接操作配置文件。
//
// 存储选型（调研结论）：
//
//	桌面应用的「用户偏好」属于单用户、单机、小体量键值配置。业界惯例
//	（Electron 的 electron-store 写入 userData、.NET 的 user.config、
//	Wails/原生 Go 写入 os.UserConfigDir）一致采用 per-user 目录下的 JSON
//	文件；SQLite/数据库只在需要结构化关系数据或跨设备同步时才引入，
//	对当前场景属于过度设计。故此处使用 JSON 文件方案。
package settings

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// appName 配置目录名（也是产品标识），最终路径形如：
//
//	Windows: %APPDATA%\bestuav-mygo\settings.json
//	macOS:   ~/Library/Application Support/bestuav-mygo/settings.json
//	Linux:   ~/.config/bestuav-mygo/settings.json
const appName = "bestuav-mygo"

// Settings 用户偏好。
type Settings struct {
	Theme string `json:"theme"` // "system" | "light" | "dark"

	FontSize float64 `json:"fontSize"` // 文本基准字号（px）

	CloseMode string `json:"closeMode"` // "ask" | "minimize" | "quit"

	// SidebarCollapsed 侧栏整体是否处于收缩（图标栏）状态。false=展开，true=收缩。
	SidebarCollapsed bool `json:"sidebarCollapsed"`
}

// Default 返回首次启动时的默认偏好。
func Default() *Settings {
	return &Settings{
		Theme:     "system",
		FontSize:  14,
		CloseMode: "ask",
	}
}

var mu sync.Mutex // 保护 Save 的临时文件写与重命名（视图层与可能的并发调用）

// path 定位配置文件路径；若标准用户配置目录不可用，退路到用户主目录下的
// 隐藏目录，避免崩溃。
func path() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		if home, herr := os.UserHomeDir(); herr == nil {
			dir = filepath.Join(home, "."+appName)
		} else {
			return "", fmt.Errorf("无法定位配置目录: %w", err)
		}
	}
	return filepath.Join(dir, "settings.json"), nil
}

// Path 暴露配置文件路径，供 UI 层「打开配置目录」等场景展示。
func Path() (string, error) { return path() }

// Load 读取设置；文件不存在或损坏时返回默认偏好（不写盘、不报错中断）。
func Load() (*Settings, error) {
	p, err := path()
	if err != nil {
		return Default(), err
	}
	data, err := os.ReadFile(p)
	if err != nil {
		if os.IsNotExist(err) {
			return Default(), nil
		}
		return Default(), fmt.Errorf("读取设置失败，使用默认: %w", err)
	}
	s := Default()
	if err := json.Unmarshal(data, s); err != nil {
		return Default(), fmt.Errorf("解析设置失败，使用默认: %w", err)
	}
	normalize(s)
	return s, nil
}

// MustLoad 同 Load，但忽略错误返回默认兜底，便于调用方一行初始化。
func MustLoad() *Settings {
	s, _ := Load()
	return s
}

// normalize 把脏数据/越界值纠正为合法枚举，避免外部手动改坏文件导致崩溃。
func normalize(s *Settings) {
	switch s.Theme {
	case "light", "dark", "system":
	default:
		s.Theme = "system"
	}
	if s.FontSize < 11 || s.FontSize > 22 {
		s.FontSize = 14
	}
	switch s.CloseMode {
	case "ask", "minimize", "quit":
	default:
		s.CloseMode = "ask"
	}
}

// Save 以「临时文件 + 重命名」方式原子写入，避免写入过程中断电/崩溃留下
// 半截配置导致下次启动解析失败。
func (s *Settings) Save() error {
	mu.Lock()
	defer mu.Unlock()

	p, err := path()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	tmp := p + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, p)
}
