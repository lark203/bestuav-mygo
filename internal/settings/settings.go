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

	"bestuav-mygo/internal/apppath"
)

// 目录名与路径规则统一由 internal/apppath 提供（日志、数据库同用一份逻辑）：
//
//	Windows: %APPDATA%\bestuav-mygo\settings.json
//	macOS:   ~/Library/Application Support/bestuav-mygo/settings.json
//	Linux:   ~/.config/bestuav-mygo/settings.json

// Settings 用户偏好。
type Settings struct {
	Theme string `json:"theme"` // "system" | "light" | "dark"

	FontSize float64 `json:"fontSize"` // 文本基准字号（px）

	CloseMode string `json:"closeMode"` // "ask" | "minimize" | "quit"

	// SidebarCollapsed 侧栏整体是否处于收缩（图标栏）状态。false=展开，true=收缩。
	SidebarCollapsed bool `json:"sidebarCollapsed"`

	// AllowMultiInstance 是否允许同时运行多个应用实例。
	// false（默认）=单实例：重复启动只会激活已在运行的窗口并退出新进程；
	// true=多开：每次启动都是独立窗口，互不干扰。
	AllowMultiInstance bool `json:"allowMultiInstance"`

	// BackgroundImage 背景图片的绝对路径；空串表示不使用背景图（主题默认外观）。
	BackgroundImage string `json:"backgroundImage"`

	// MenuOpacity 侧栏（菜单区）浮层的不透明度，0.2–1.0。仅在设置了背景图时生效：
	// 1.0=完全遮蔽背景图（与未设背景图时观感一致），数值越小背景图越透出来。
	MenuOpacity float64 `json:"menuOpacity"`

	// ContentOpacity 内容区浮层的不透明度，0.2–1.0，语义同 MenuOpacity。
	ContentOpacity float64 `json:"contentOpacity"`

	// PageIdleMinutes 页面缓存闲置多久后销毁（分钟）。
	// 0 = 不按闲置销毁（页面状态一直保留，直到被显式销毁）；默认 15。
	// 与 javafx 版的 idleTime 配置对应，详见 internal/view/pagelife.go。
	PageIdleMinutes int `json:"pageIdleMinutes"`
}

// 浮层不透明度的默认值与合法区间。
//
// 说明：这里用「不透明度（opacity）」而非「透明度（transparency）」作为存储与
// UI 语义，避免 0 同时被理解为「全透明」和「无穷不透明」两种歧义；界面上
// 滑块展示的百分比仍是用户直觉的「不透明度」。
const (
	// 默认留一点透出，让用户设置背景图后能立刻看出效果，同时保证正文可读。
	DefaultMenuOpacity    = 0.80
	DefaultContentOpacity = 0.88
	MinOpacity            = 0.20
	MaxOpacity            = 1.00

	// DefaultPageIdleMinutes 页面缓存闲置销毁的默认时长（分钟）。
	DefaultPageIdleMinutes = 15
	// MaxPageIdleMinutes 设置页可调上限（分钟）。
	MaxPageIdleMinutes = 120
)

// Default 返回首次启动时的默认偏好。
func Default() *Settings {
	return &Settings{
		Theme:           "system",
		FontSize:        14,
		CloseMode:       "ask",
		MenuOpacity:     DefaultMenuOpacity,
		ContentOpacity:  DefaultContentOpacity,
		PageIdleMinutes: DefaultPageIdleMinutes,
	}
}

var mu sync.Mutex // 保护 Save 的临时文件写与重命名（视图层与可能的并发调用）

// configDir 返回本应用的 per-user 配置目录：<用户配置目录>/<appName>。
//
// 路径规则统一收敛在 internal/apppath（日志目录、数据库路径也用同一份逻辑）。
func configDir() (string, error) { return apppath.ConfigDir() }

// path 定位配置文件路径（<配置目录>/settings.json）。
func path() (string, error) {
	dir, err := configDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "settings.json"), nil
}

// legacyPath 早期版本的配置文件路径：直接放在用户配置目录根下，没有按应用名
// 分子目录。仅在读取时用于一次性迁移，避免老用户升级后偏好（主题、侧栏状态等）
// 丢失。
func legacyPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
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
			// 老路径迁移：读到旧文件则原样解析，并尽力写入新位置。
			if legacy, lerr := legacyPath(); lerr == nil {
				if old, oerr := os.ReadFile(legacy); oerr == nil {
					s := Default()
					if json.Unmarshal(old, s) == nil {
						normalize(s)
						_ = s.Save() // 迁移失败不影响本次使用
						return s, nil
					}
				}
			}
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
	// 浮层不透明度：越界/零值（老版本文件没有这两个字段，反序列化得 0）
	// 一律回落到默认，避免出现「完全透明」的不可用界面。
	if s.MenuOpacity < MinOpacity || s.MenuOpacity > MaxOpacity {
		s.MenuOpacity = DefaultMenuOpacity
	}
	if s.ContentOpacity < MinOpacity || s.ContentOpacity > MaxOpacity {
		s.ContentOpacity = DefaultContentOpacity
	}
	// 页面缓存闲置销毁：负数/超上限一律纠正（负值在老文件里不存在，故只在
	// 显式越界时处理；缺字段得 0，按语义就是「不销毁」，这里保留用户意图）。
	if s.PageIdleMinutes < 0 || s.PageIdleMinutes > MaxPageIdleMinutes {
		s.PageIdleMinutes = DefaultPageIdleMinutes
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
