package view

import (
	"time"

	"bestuav-mygo/internal/logging"
	"bestuav-mygo/internal/store"
)

// 本文件是「界面状态 ↔ 本地数据库」的胶水层。
//
// 原则：
//   - **数据库是持久化真源，App 里的切片只是本帧要用的缓存**：启动时从库加载，
//     写入时同时更新内存与库（桌面场景数据量小，无需做脏标记/批量刷盘）；
//   - **数据库不可用时必须优雅退化**：a.store == nil 时全部操作退化为纯内存，
//     界面照常可用（打开失败的原因会写进日志，不影响启动）。

const (
	// flagDemoSeeded 元数据键：演示任务是否已灌入数据库。
	// 用它而不是「表为空就灌」——否则用户清空任务后，下次启动又会被灌回来。
	flagDemoSeeded = "seeded_demo_tasks"

	// maxTaskRows 任务列表在内存与库中各自保留的最大条数。
	maxTaskRows = 20

	// maxLicenseRows 授权历史保留条数（与页面上的「最近 6 条」一致）。
	maxLicenseRows = 6

	// dbStatsTTL 库计数快照的有效期：避免界面每帧执行 4 条 COUNT。
	dbStatsTTL = 5 * time.Second
)

// AttachStore 注入本地数据库，并从库中恢复上次运行的数据。
func (a *App) AttachStore(s *store.Store) {
	a.store = s
	a.loadPersisted()
}

// loadPersisted 启动时恢复数据：任务列表、授权历史。
func (a *App) loadPersisted() {
	if a.store == nil {
		return
	}

	// 首次运行：把演示任务写进库，让「任务队列」不是空的（之后一律以库为准）。
	if _, ok, err := a.store.Flag(flagDemoSeeded); err == nil && !ok {
		rows := demoTaskRows()
		// 倒序插入：查询按 id DESC，倒着写才能让展示顺序与原演示顺序一致。
		for i := len(rows) - 1; i >= 0; i-- {
			r := rows[i]
			if _, err := a.store.InsertTask(r.name, r.kind, r.status, r.cost); err != nil {
				logging.Errv("store", "灌入演示任务失败", err)
				break
			}
		}
		if err := a.store.SetFlag(flagDemoSeeded, "1"); err != nil {
			logging.Errv("store", "写入初始化标记失败", err)
		}
	}

	if rows, err := a.store.RecentTasks(maxTaskRows); err != nil {
		logging.Errv("store", "读取任务记录失败", err)
	} else if len(rows) > 0 {
		a.tasks = tasksFromStore(rows)
	}

	if rows, err := a.store.RecentLicenses(maxLicenseRows); err != nil {
		logging.Errv("store", "读取授权历史失败", err)
	} else if len(rows) > 0 {
		a.lic.history = licensesFromStore(rows)
	}

	a.refreshDBStats()
}

// tasksFromStore 把持久化记录转成任务页使用的展示行。
func tasksFromStore(rows []store.Task) []taskRow {
	out := make([]taskRow, 0, len(rows))
	for _, r := range rows {
		out = append(out, taskRow{name: r.Name, kind: r.Kind, status: r.Status, cost: r.Cost})
	}
	return out
}

// licensesFromStore 把持久化记录转成「生成历史」的展示行。
func licensesFromStore(rows []store.License) []licRecord {
	out := make([]licRecord, 0, len(rows))
	for _, r := range rows {
		out = append(out, licRecord{
			subject: r.Subject,
			edition: r.Edition,
			term:    r.Term,
			token:   r.Token,
			at:      r.CreatedAt,
		})
	}
	return out
}

// persistTask 把一条新任务写入数据库（失败只记日志，不影响界面）。
func (a *App) persistTask(name, kind, status, cost string) {
	if a.store == nil {
		return
	}
	if _, err := a.store.InsertTask(name, kind, status, cost); err != nil {
		logging.Errv("store", "写入任务记录失败", err, "name", name)
		return
	}
	a.refreshDBStats()
}

// persistClearFinished 同步删除库里已完成的记录，返回是否删除成功。
func (a *App) persistClearFinished() {
	if a.store == nil {
		return
	}
	n, err := a.store.DeleteFinishedTasks()
	if err != nil {
		logging.Errv("store", "清理已完成任务失败", err)
		return
	}
	logging.Audit("task", "清空已完成任务", "count", n)
	a.refreshDBStats()
}

// persistLicense 写入一条授权凭证记录。
func (a *App) persistLicense(subject, edition, term, token string) {
	if a.store == nil {
		return
	}
	if _, err := a.store.InsertLicense(subject, edition, term, token); err != nil {
		logging.Errv("store", "写入授权记录失败", err, "subject", subject)
		return
	}
	a.refreshDBStats()
}

// refreshDBStats 刷新库计数快照（供界面展示，带 TTL）。
func (a *App) refreshDBStats() {
	if a.store == nil {
		return
	}
	st, err := a.store.Stats()
	if err != nil {
		logging.Errv("store", "统计数据库失败", err)
		return
	}
	a.dbStats, a.dbStatsAt = st, time.Now()
}

// DBStats 返回库计数快照；超过 TTL 时顺带刷新一次（供页面每帧调用）。
func (a *App) DBStats() store.Stats {
	if a.store != nil && time.Since(a.dbStatsAt) > dbStatsTTL {
		a.refreshDBStats()
	}
	return a.dbStats
}

// RecentEvents 读取最近的事件记录（数据库未启用时返回 nil）。
func (a *App) RecentEvents(n int) []store.Event {
	if a.store == nil {
		return nil
	}
	rows, err := a.store.RecentEvents(n)
	if err != nil {
		logging.Errv("store", "读取事件记录失败", err)
		return nil
	}
	return rows
}

// StorePath 返回数据库文件路径；未启用数据库时为空。
func (a *App) StorePath() string {
	if a.store == nil {
		return ""
	}
	return a.store.Path()
}
