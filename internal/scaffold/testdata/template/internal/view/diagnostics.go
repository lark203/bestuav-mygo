package view

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/egoist/mygo"

	"bestuav-mygo/internal/apppath"
	"bestuav-mygo/internal/logging"
	"bestuav-mygo/internal/settings"
)

// 本文件是「支持与诊断」：把用户环境、配置、日志与数据库快照打成一个 zip。
//
// 为什么需要：用户报障时最常听到的是「它自己坏了」。让他去 %APPDATA% 里找
// 三个不同位置的文件并分辨哪个该发，基本等于让他放弃。这里一键打包，
// 支持人员拿到一个文件就能复现与定位（版本、系统、路径、错误计数、最近日志、
// 库里的任务/授权记录）。

// diagLogDays 诊断包里只带最近几天的日志（日志本身按天轮转、保留 7 天）。
const diagLogDays = 3

// openConfigDir 打开 per-user 配置目录（settings.json 所在处）。
func (a *App) openConfigDir() {
	dir, err := apppath.ConfigDir()
	if err != nil {
		a.toast("定位配置目录失败：" + err.Error())
		return
	}
	if err := mygo.Shell.OpenPath(dir); err != nil {
		a.toast("打开配置目录失败：" + err.Error())
	}
}

// openLogDir 打开日志目录。
func (a *App) openLogDir() {
	dir, err := apppath.LogDir()
	if err != nil {
		a.toast("定位日志目录失败：" + err.Error())
		return
	}
	if err := mygo.Shell.OpenPath(dir); err != nil {
		a.toast("打开日志目录失败：" + err.Error())
	}
}

// exportDiagnostics 导出诊断包，成功后打开所在目录并提示完整路径。
func (a *App) exportDiagnostics() {
	path, err := a.writeDiagnosticsArchive()
	if err != nil {
		logging.Errv("diag", "导出诊断包失败", err)
		a.toast("导出失败：" + err.Error())
		return
	}
	logging.Audit("diag", "导出诊断包", "path", path)
	// 打开所在目录：用户接下来要做的就是把这个文件拖给支持人员。
	if err := mygo.Shell.OpenPath(filepath.Dir(path)); err != nil {
		a.toast("诊断包已导出：" + path)
		return
	}
	a.toast("诊断包已导出：" + path)
}

// writeDiagnosticsArchive 生成诊断包，返回 zip 的完整路径。
func (a *App) writeDiagnosticsArchive() (string, error) {
	name := fmt.Sprintf("bestuav-mygo-diag-%s.zip", time.Now().Format("20060102-150405"))
	zipPath := filepath.Join(os.TempDir(), name)

	f, err := os.Create(zipPath)
	if err != nil {
		return "", err
	}
	defer f.Close()
	zw := zip.NewWriter(f)

	// 1) 用户偏好。
	if p, err := settings.Path(); err == nil {
		if err := addFileToZip(zw, "settings.json", p); err != nil && !os.IsNotExist(err) {
			return "", err
		}
	}

	// 2) 最近的日志文件。
	if dir, err := apppath.LogDir(); err == nil {
		if err := addRecentLogs(zw, dir); err != nil {
			return "", err
		}
	}

	// 3) 数据库一致性快照（库未启用时跳过）。
	if a.store != nil {
		tmp := filepath.Join(os.TempDir(), fmt.Sprintf("bestuav-mygo-diag-%d.db", time.Now().UnixNano()))
		if err := a.store.BackupTo(tmp); err != nil {
			return "", err
		}
		err := addFileToZip(zw, "data.snapshot.db", tmp)
		_ = os.Remove(tmp)
		if err != nil {
			return "", err
		}
	}

	// 4) 环境与版本信息（纯文本，方便直接贴进 issue）。
	if err := addBytesToZip(zw, "env.txt", []byte(a.basicInfoText())); err != nil {
		return "", err
	}

	// 5) 说明文件。
	if err := addBytesToZip(zw, "README.txt", []byte(diagReadme)); err != nil {
		return "", err
	}

	if err := zw.Close(); err != nil {
		return "", err
	}
	return zipPath, nil
}

const diagReadme = `bestuav-mygo 诊断包

包含：
- settings.json          用户偏好（主题、字号、关闭行为、背景图路径等）
- logs/                  最近三天的运行日志（JSON，每行一条）
- data.snapshot.db       数据库一致性快照（任务 / 授权 / 事件记录）
- env.txt                版本、系统、运行时、路径与错误计数

注意：settings.json 与日志里可能包含本机文件路径等个人信息，
请只发送给你信任的支持人员。
`

// addRecentLogs 把 dir 下最近 diagLogDays 天的日志写入 zip 的 logs/ 目录。
func addRecentLogs(zw *zip.Writer, dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	cutoff := time.Now().AddDate(0, 0, -diagLogDays)
	// 目录项按名字排序；日志名 app-YYYY-MM-DD.log，名字序即日期序。
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
	for _, e := range entries {
		if e.IsDir() || !strings.HasPrefix(e.Name(), "app-") || !strings.HasSuffix(e.Name(), ".log") {
			continue
		}
		info, err := e.Info()
		if err != nil || info.ModTime().Before(cutoff) {
			continue
		}
		if err := addFileToZip(zw, "logs/"+e.Name(), filepath.Join(dir, e.Name())); err != nil {
			return err
		}
	}
	return nil
}

// addFileToZip 把一个磁盘文件写入 zip（保持原内容）。
func addFileToZip(zw *zip.Writer, name, src string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	w, err := zw.Create(name)
	if err != nil {
		return err
	}
	_, err = io.Copy(w, in)
	return err
}

// addBytesToZip 把内存里的内容写入 zip。
func addBytesToZip(zw *zip.Writer, name string, data []byte) error {
	w, err := zw.Create(name)
	if err != nil {
		return err
	}
	_, err = w.Write(data)
	return err
}
