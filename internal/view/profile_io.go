package view

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"bestuav-mygo/internal/appmeta"
	"bestuav-mygo/internal/apppath"
	"bestuav-mygo/internal/logging"
	"bestuav-mygo/internal/profile"
	"bestuav-mygo/internal/settings"
	"bestuav-mygo/internal/store"
)

// 本文件把「配置与数据」的导出 / 导入接到界面上（方案 A），以及同步文件夹（方案 D）。
//
// 约定：
//   - 导入会替换 settings.json 与 data.db，必须在主线程做、且替换数据库前关掉连接；
//     所以按钮只登记 pendingImport，真正执行放到下一帧开头（见 applyPendingImport）。
//   - 导出/扫描是只读磁盘操作，放到 goroutine，完成后回主线程提示。

// profilePaths 返回导出/导入需要的三个路径。
func (a *App) profilePaths() (settingsPath, dbPath, dataDir string) {
	settingsPath, _ = settings.Path()
	dbPath, _ = apppath.DBPath()
	dataDir, _ = apppath.DataDir()
	return
}

// exportProfile 弹出保存框并导出（withData=false 时只导出设置）。
func (a *App) exportProfile(withData bool) {
	settingsPath, dbPath, _ := a.profilePaths()
	bg := a.settings.BackgroundImage
	go func() {
		defer logging.Recover("export-profile")
		name := "quantum-" + profile.DeviceName() + "-" + time.Now().Format("20060102") + profile.FileExt
		out, err := mygo.Dialog.Save(mygo.SaveDialogOptions{
			Title:       "导出配置与数据",
			DefaultPath: filepath.Join(filepath.Dir(settingsPath), name),
			ButtonLabel: "导出",
			Filters:     []mygo.FileFilter{{Name: "配置包", Extensions: []string{"zip"}}},
		})
		if err != nil || strings.TrimSpace(out) == "" {
			return
		}
		if !strings.HasSuffix(strings.ToLower(out), ".zip") {
			out += profile.FileExt
		}
		res, err := profile.Export(profile.ExportOptions{
			Out:           out,
			SettingsPath:  settingsPath,
			DBPath:        dbPath,
			Background:    bg,
			IncludeData:   withData,
			AppID:         appmeta.Identifier,
			AppVersion:    appmeta.BuildVersion(),
			SchemaVersion: store.SchemaVersion(),
		})
		mygo.RunOnMain(func() {
			if err != nil {
				logging.Errv("profile", "导出配置与数据失败", err)
				a.toast("导出失败：" + err.Error())
				return
			}
			logging.Audit("profile", "导出配置与数据", "path", res.Path, "data", res.IncludesData)
			parts := []string{"设置"}
			if res.IncludesData {
				parts = append(parts, "数据")
			}
			if res.IncludesImage {
				parts = append(parts, "背景图")
			}
			a.toast(fmt.Sprintf("已导出（%s）%s", strings.Join(parts, " + "), formatBytes(res.Bytes)))
			a.requestFrame()
		})
	}()
}

// importProfileDialog 选择配置包；真正导入由下一帧的 applyPendingImport 完成。
func (a *App) importProfileDialog() {
	go func() {
		defer logging.Recover("import-profile")
		paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{
			Title:       "导入配置与数据",
			ButtonLabel: "导入",
			Filters:     []mygo.FileFilter{{Name: "配置包", Extensions: []string{"zip"}}},
		})
		if err != nil || len(paths) == 0 {
			return
		}
		path := paths[0]
		mygo.RunOnMain(func() {
			a.pendingImport = path
			a.requestFrame()
		})
	}()
}

// applyPendingImport 在帧开头执行待处理的导入（主线程；会关掉数据库连接）。
func (a *App) applyPendingImport() {
	path := a.pendingImport
	if path == "" {
		return
	}
	a.pendingImport = ""
	settingsPath, dbPath, dataDir := a.profilePaths()
	res, err := profile.Import(profile.ImportOptions{
		In:            path,
		SettingsPath:  settingsPath,
		DBPath:        dbPath,
		DataDir:       dataDir,
		AppID:         appmeta.Identifier,
		SchemaVersion: store.SchemaVersion(),
		BeforeDB: func() error {
			if a.store == nil {
				return nil
			}
			err := a.store.Close()
			a.store = nil
			return err
		},
	})
	if err != nil {
		logging.Errv("profile", "导入配置与数据失败", err)
		a.toast("导入失败：" + err.Error())
		a.requestFrame()
		return
	}
	logging.Audit("profile", "导入配置与数据", "path", path,
		"data", res.ImportedData, "image", res.ImportedImage,
		"settingsBackup", res.SettingsBackup, "dbBackup", res.DBBackup)
	// 设置与数据库都被替换了：重启是最可靠的「重新加载」。
	if a.win != nil {
		mygo.App.Relaunch()
		return
	}
	a.toast("导入完成（重启应用后生效）")
	a.requestFrame()
}

// ---- 同步文件夹（方案 D） ----

// syncBundles 已扫描到的配置包（缓存；只在选择目录/点刷新/导出后更新）。
func (a *App) syncBundles() []profile.Bundle {
	v, _ := a.pageStat(settingsPageID).data["syncBundles"].([]profile.Bundle)
	return v
}

// refreshSyncBundles 扫描同步文件夹并缓存结果。
func (a *App) refreshSyncBundles() {
	folder := a.settings.SyncFolder
	if folder == "" {
		return
	}
	list, err := profile.List(folder)
	if err != nil {
		logging.Errv("profile", "扫描同步文件夹失败", err, "dir", folder)
		a.toast("扫描同步文件夹失败：" + err.Error())
		return
	}
	a.pageStat(settingsPageID).data["syncBundles"] = list
	if !a.syncScanned {
		a.syncScanned = true
	}
}

// chooseSyncFolder 选择同步文件夹（用户自己交给 OneDrive/iCloud/坚果云 同步的目录）。
func (a *App) chooseSyncFolder() {
	go func() {
		defer logging.Recover("choose-sync-folder")
		paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{
			Title:       "选择同步文件夹",
			ButtonLabel: "使用此文件夹",
			Directory:   true,
		})
		if err != nil || len(paths) == 0 {
			return
		}
		dir := paths[0]
		mygo.RunOnMain(func() {
			a.settings.SyncFolder = dir
			a.persist()
			a.refreshSyncBundles()
			a.syncScanned = true
			a.requestFrame()
		})
	}()
}

// exportToSyncFolder 把当前配置与数据导出到同步文件夹（文件名带设备名）。
func (a *App) exportToSyncFolder() {
	folder := a.settings.SyncFolder
	if folder == "" {
		a.toast("请先选择同步文件夹")
		return
	}
	settingsPath, dbPath, _ := a.profilePaths()
	bg := a.settings.BackgroundImage
	go func() {
		defer logging.Recover("export-sync-folder")
		out := filepath.Join(folder, profile.DeviceName()+profile.FileExt)
		res, err := profile.Export(profile.ExportOptions{
			Out:           out,
			SettingsPath:  settingsPath,
			DBPath:        dbPath,
			Background:    bg,
			IncludeData:   true,
			AppID:         appmeta.Identifier,
			AppVersion:    appmeta.BuildVersion(),
			SchemaVersion: store.SchemaVersion(),
		})
		mygo.RunOnMain(func() {
			if err != nil {
				logging.Errv("profile", "导出到同步文件夹失败", err)
				a.toast("导出失败：" + err.Error())
				return
			}
			logging.Audit("profile", "导出到同步文件夹", "path", res.Path)
			a.toast("已导出到同步文件夹：" + filepath.Base(res.Path))
			a.refreshSyncBundles()
			a.requestFrame()
		})
	}()
}

// syncFolderSection 设置页「数据与诊断」里的同步文件夹区块。
func (a *App) syncFolderSection(c *ui.Context, t *ui.Theme) {
	ui.Text(c, "同步文件夹").Bold().Margin(8, 0, 0, 0)
	ui.Text(c, "选一个已由 OneDrive / iCloud / 坚果云 / Dropbox 同步的目录："+
		"把配置包导出到这里，另一台机器再导入。不会自动覆盖对方的数据。").
		TextColor(t.TextMuted).FontSize(t.FontSize - 2)

	ui.Row(c).Gap(8).Wrap().AlignItems(ui.Center).Children(func() {
		if ui.Button(c, "选择文件夹…").Clicked() {
			a.chooseSyncFolder()
		}
		if ui.Button(c, "导出到此处").Clicked() {
			a.exportToSyncFolder()
		}
		if ui.Button(c, "刷新列表").Clicked() {
			a.refreshSyncBundles()
			a.requestFrame()
		}
	})
	folder := a.settings.SyncFolder
	if folder == "" {
		ui.Text(c, "未设置同步文件夹").FontSize(t.FontSize - 2).TextColor(t.TextMuted)
		return
	}
	ui.Text(c, folder).FontSize(t.FontSize - 2).TextColor(t.TextMuted).Selectable()

	bundles := a.syncBundles()
	if len(bundles) == 0 {
		ui.Text(c, "文件夹里还没有配置包（点「导出到此处」，或把别的机器导出的包放进这个目录）。").
			FontSize(t.FontSize - 2).TextColor(t.TextMuted)
		return
	}
	for _, b := range bundles {
		b := b
		ui.Row(c).Gap(12).AlignItems(ui.Center).Padding(4, 0).Children(func() {
			ui.Column(c).Gap(1).Grow(1).Children(func() {
				ui.Text(c, b.Manifest.Device+" · "+b.Manifest.Platform).SingleLine()
				ui.Text(c, b.Manifest.ExportedAt.Local().Format("2006-01-02 15:04")+
					" · "+includesText(b.Manifest.Includes)).
					FontSize(t.FontSize - 2).TextColor(t.TextMuted)
			})
			if ui.Button(c, "导入").Clicked() {
				a.pendingImport = b.Path
				a.requestFrame()
			}
		})
	}
	ui.Text(c, "导入会替换本机的设置与数据库；导入前会自动备份（.bak-<时间>），完成后自动重启。").
		FontSize(t.FontSize - 2).TextColor(t.TextMuted)
}

// includesText 把 manifest 的 includes 翻成一句人话。
func includesText(includes []string) string {
	has := func(k string) bool {
		for _, s := range includes {
			if s == k {
				return true
			}
		}
		return false
	}
	switch {
	case has("data") && has("background"):
		return "设置 + 数据 + 背景图"
	case has("data"):
		return "设置 + 数据"
	default:
		return "仅设置"
	}
}

// formatBytes 人类可读的大小。
func formatBytes(n int64) string {
	if n < 1024 {
		return fmt.Sprintf("%d B", n)
	}
	if n < 1024*1024 {
		return fmt.Sprintf("%.1f KB", float64(n)/1024)
	}
	return fmt.Sprintf("%.1f MB", float64(n)/(1024*1024))
}
