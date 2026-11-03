package scaffold

import (
	"fmt"
	"image/png"
	"io"
	"os"
	"path/filepath"
)

// maxIconBytes 图标文件上限。1024×1024 的 PNG 通常 100KB~2MB，
// 放到 8MB 是为了挡住「选了一张 30MB 的壁纸」这类误操作。
const maxIconBytes = 8 << 20

// applyProjectIcon 用用户挑选的 PNG 替换生成项目里的 resources/icon.png。
//
// 为什么必须校验内容而不只看扩展名：图标会被 go:embed 编进二进制、并在
// `go tool mygo build` 时转成 .ico/.icns。文件不是真 PNG 时，构建可能成功但
// 产物没有图标，或者在某些平台上直接失败——这种「改完没效果」最难查。
func applyProjectIcon(root, src string) error {
	if src == "" {
		return nil
	}
	info, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("读取图标失败: %w", err)
	}
	if info.Size() > maxIconBytes {
		return fmt.Errorf("图标过大（%.1f MB），上限 %d MB", float64(info.Size())/(1<<20), maxIconBytes>>20)
	}

	f, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("打开图标失败: %w", err)
	}
	defer func() { _ = f.Close() }()

	// DecodeConfig 只读文件头，不展开整幅像素，够快也不吃内存。
	cfg, err := png.DecodeConfig(f)
	if err != nil {
		return fmt.Errorf("不是合法的 PNG 图片: %w", err)
	}
	if cfg.Width < 64 || cfg.Height < 64 {
		return fmt.Errorf("图标尺寸过小（%d×%d），至少 64×64，建议 1024×1024", cfg.Width, cfg.Height)
	}
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return err
	}

	dst := filepath.Join(root, "resources", "icon.png")
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, f); err != nil {
		_ = out.Close()
		return err
	}
	return out.Close()
}
