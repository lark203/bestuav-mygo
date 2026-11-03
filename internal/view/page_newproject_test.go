package view

import "testing"

// TestNewProjectPageRenders 向导页的关键字段都在（这个页面是新项目生成的唯一入口，
// 少一个输入框就等于少一个必填参数）。
func TestNewProjectPageRenders(t *testing.T) {
	a := newTestApp()
	a.page = "new-project"
	mustHave(t, render(a),
		"项目名", "品牌名称", "生成目录", "GitHub 仓库", "应用标识", "项目图标",
		"将生成到", "保留哪些菜单", "开始生成",
	)
}

// TestDeriveIdentifier 标识推导规则必须与 scaffold 侧一致（com.<去掉中划线的项目名>）。
func TestDeriveIdentifier(t *testing.T) {
	cases := map[string]string{
		"demo-app":    "com.demoapp",
		"bestuav-foo": "com.bestuavfoo",
		"app":         "com.app",
		"":            "",
		"   ":         "",
	}
	for in, want := range cases {
		if got := deriveIdentifier(in); got != want {
			t.Errorf("deriveIdentifier(%q) = %q, want %q", in, got, want)
		}
	}
}
