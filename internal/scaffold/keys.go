package scaffold

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// keyPair 更新签名密钥对。
type keyPair struct {
	PublicKey  string // ed25519 公钥（base64 单行），写进 mygo.json
	PrivateKey string // 私钥（base64 单行），只交给调用方，不落进仓库
	Dir        string // 私钥所在目录
	KeyPath    string // 私钥文件绝对路径
}

// generateKeys 调 `go tool mygo keygen` 生成密钥对。
//
// 目录按项目分：<用户配置目录>/mygo/update-keys/<项目名>。
//
// 为什么必须分目录：mygo 的 keygen 默认写到 update-keys/ 根下，两个项目共用同一个
// 私钥文件时，后生成的项目会覆盖前一个。私钥一换，前一个项目已安装的客户端就再也
// 验不过新包（公钥固化在旧客户端里），属于不可逆故障。
func generateKeys(root, project, keyDir string) (keyPair, error) {
	if keyDir == "" {
		base, err := os.UserConfigDir()
		if err != nil {
			return keyPair{}, fmt.Errorf("无法定位用户配置目录: %w", err)
		}
		keyDir = filepath.Join(base, "mygo", "update-keys", project)
	}
	if err := os.MkdirAll(keyDir, 0o755); err != nil {
		return keyPair{}, fmt.Errorf("创建密钥目录失败: %w", err)
	}

	cmd := exec.Command("go", "tool", "mygo", "keygen", "-force", "-o", keyDir)
	cmd.Dir = root
	if out, err := cmd.CombinedOutput(); err != nil {
		return keyPair{}, fmt.Errorf("生成更新密钥失败: %v\n%s", err, truncate(string(out), 1200))
	}

	priv, err := os.ReadFile(filepath.Join(keyDir, "mygo-update.key"))
	if err != nil {
		return keyPair{}, fmt.Errorf("读取私钥失败: %w", err)
	}
	pub, err := os.ReadFile(filepath.Join(keyDir, "mygo-update.pub"))
	if err != nil {
		return keyPair{}, fmt.Errorf("读取公钥失败: %w", err)
	}
	return keyPair{
		PublicKey:  strings.TrimSpace(string(pub)),
		PrivateKey: strings.TrimSpace(string(priv)),
		Dir:        keyDir,
		KeyPath:    filepath.Join(keyDir, "mygo-update.key"),
	}, nil
}

// privateKeyPathHint 供 UI 与手册展示的私钥路径（按项目分目录的默认位置）。
func privateKeyPathHint(project string) string {
	base, err := os.UserConfigDir()
	if err != nil {
		return "<用户配置目录>/mygo/update-keys/" + project + "/mygo-update.key"
	}
	return filepath.Join(base, "mygo", "update-keys", project, "mygo-update.key")
}
