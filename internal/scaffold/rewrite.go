package scaffold

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// rewriteModule 把 Go module 路径从旧名改成新名。
//
// 为什么要全量扫：module 名在 import 里出现近百次（`"bestuav-mygo/internal/..."`），
// 漏掉任何一处都会编译失败，而且报错位置在别处、很难反查。
//
// 只替换带引号的形式，避免误伤注释里提到项目名的普通文字（那部分由 rewriteTextRefs
// 单独处理，改了也不影响编译）。
func rewriteModule(root, oldModule, newModule string) (int, error) {
	changed := 0
	oldImport := `"` + oldModule + `/`
	newImport := `"` + newModule + `/`
	oldExact := `"` + oldModule + `"`
	newExact := `"` + newModule + `"`

	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(p, ".go") {
			return nil
		}
		data, rerr := os.ReadFile(p)
		if rerr != nil {
			return rerr
		}
		text := string(data)
		if !strings.Contains(text, oldImport) && !strings.Contains(text, oldExact) {
			return nil
		}
		text = strings.ReplaceAll(text, oldImport, newImport)
		text = strings.ReplaceAll(text, oldExact, newExact)
		if wErr := os.WriteFile(p, []byte(text), 0o644); wErr != nil {
			return wErr
		}
		changed++
		return nil
	})
	if err != nil {
		return changed, err
	}

	// go.mod 的 module 行。
	gomod := filepath.Join(root, "go.mod")
	data, err := os.ReadFile(gomod)
	if err != nil {
		return changed, fmt.Errorf("读取 go.mod: %w", err)
	}
	re := regexp.MustCompile(`(?m)^module\s+` + regexp.QuoteMeta(oldModule) + `\s*$`)
	if !re.Match(data) {
		return changed, fmt.Errorf("go.mod 里没有找到 module %s 声明", oldModule)
	}
	out := re.ReplaceAll(data, []byte("module "+newModule))
	if err := os.WriteFile(gomod, out, 0o644); err != nil {
		return changed, err
	}
	return changed, nil
}

// rewriteAppmeta 改写产品标识常量，并同步该文件注释里的旧名。
func rewriteAppmeta(root, oldModule, project, brand, identifier string) error {
	p := filepath.Join(root, "internal", "appmeta", "appmeta.go")
	data, err := os.ReadFile(p)
	if err != nil {
		return fmt.Errorf("读取 appmeta.go: %w", err)
	}
	text := string(data)
	for _, sub := range []struct {
		key string
		val string
	}{
		{"Name", project},
		{"Title", brand},
		{"Identifier", identifier},
	} {
		re := regexp.MustCompile(`(?m)^const ` + sub.key + ` = ".*"$`)
		if !re.MatchString(text) {
			return fmt.Errorf("appmeta.go 里没有找到 const %s", sub.key)
		}
		text = re.ReplaceAllString(text, `const `+sub.key+` = "`+escapeQuotes(sub.val)+`"`)
	}
	// 注释里残留的旧项目名会让新项目看起来像没改干净。
	text = strings.ReplaceAll(text, oldModule, project)
	return os.WriteFile(p, []byte(text), 0o644)
}

// rewriteMygoJSON 改写 mygo.json 的产品字段与更新配置。
//
// 不做 JSON 反序列化再序列化：那样会丢掉字段顺序，也要为将来新增的字段反复改
// 结构体。这里按键做定点替换，替换完再用 json.Unmarshal 兜底校验一次合法性。
func rewriteMygoJSON(root string, project, brand, identifier, repo, publicKey string) error {
	p := filepath.Join(root, "mygo.json")
	data, err := os.ReadFile(p)
	if err != nil {
		return fmt.Errorf("读取 mygo.json: %w", err)
	}
	text := string(data)
	fields := []struct{ key, val string }{
		{"name", brand},
		{"slug", project},
		{"identifier", identifier},
		{"version", "0.1.0"},
		{"github", repo},
		{"publicKey", publicKey},
	}
	for _, f := range fields {
		text, err = setJSONString(text, f.key, f.val)
		if err != nil {
			return fmt.Errorf("mygo.json 的 %s 字段: %w", f.key, err)
		}
	}
	if err := os.WriteFile(p, []byte(text), 0o644); err != nil {
		return err
	}
	if err := validateJSONFile(p); err != nil {
		return fmt.Errorf("改写后的 mygo.json 不是合法 JSON: %w", err)
	}
	return nil
}

// setJSONString 把 `"key": "..."` 的字符串值替换成 val。
//
// 只匹配第一个出现；值里允许有转义字符（`"[^"\\]*(?:\\.[^"\\]*)*"`）。
func setJSONString(src, key, val string) (string, error) {
	re := regexp.MustCompile(`(:\s*")((?:[^"\\]|\\.)*)(")`)
	// 先定位 key 的位置，再从它后面找第一个字符串值，避免误改别的字段。
	keyRe := regexp.MustCompile(`"` + regexp.QuoteMeta(key) + `"`)
	loc := keyRe.FindStringIndex(src)
	if loc == nil {
		return src, fmt.Errorf("没有找到字段 %q", key)
	}
	rest := src[loc[1]:]
	valLoc := re.FindStringSubmatchIndex(rest)
	if valLoc == nil {
		return src, fmt.Errorf("字段 %q 后面不是字符串值", key)
	}
	// 正则有三个捕获组：1=`:\s*"`、2=值、3=收尾引号；值对应 valLoc[4:5]。
	replaced := rest[:valLoc[4]] + escapeJSON(val) + rest[valLoc[5]:]
	return src[:loc[1]] + replaced, nil
}

// rewriteScripts 让构建脚本认新项目。
//
// build-windows.ps1 里有两处项目相关的路径：输出文件名提示与「签名私钥默认位置」。
// 私钥必须按项目分目录——若两个项目共用同一个默认 key 文件，后生成的项目会覆盖前
// 一个的私钥，导致前一个项目再也签不出可被已安装客户端接受的更新包。
func rewriteScripts(root, oldModule, project string) (warnings []string, err error) {
	dir := filepath.Join(root, "scripts")
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		p := filepath.Join(dir, e.Name())
		data, rerr := os.ReadFile(p)
		if rerr != nil {
			return warnings, rerr
		}
		text := string(data)
		orig := text

		// 私钥路径：%APPDATA%\mygo\update-keys\<项目名>\mygo-update.key
		oldKeyLine := `Join-Path $env:APPDATA "mygo\update-keys\mygo-update.key"`
		newKeyLine := `Join-Path $env:APPDATA "mygo\update-keys\` + project + `\mygo-update.key"`
		if strings.Contains(text, oldKeyLine) {
			text = strings.ReplaceAll(text, oldKeyLine, newKeyLine)
		}
		text = strings.ReplaceAll(text, oldModule, project)

		if text != orig {
			if wErr := os.WriteFile(p, []byte(text), 0o644); wErr != nil {
				return warnings, wErr
			}
		}
	}
	return warnings, nil
}

// rewriteTextRefs 把剩余文本文件里提到的旧项目名换成新名（注释、文档、CI 注释）。
func rewriteTextRefs(root, oldModule, project string) error {
	return filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == ".git" {
				return fs.SkipDir
			}
			return nil
		}
		rel, rerr := filepath.Rel(root, p)
		if rerr != nil {
			return rerr
		}
		if !isRewritableSource(filepath.ToSlash(rel)) {
			return nil
		}
		data, rerr := os.ReadFile(p)
		if rerr != nil {
			return rerr
		}
		text := string(data)
		if !strings.Contains(text, oldModule) {
			return nil
		}
		text = strings.ReplaceAll(text, oldModule, project)
		return os.WriteFile(p, []byte(text), 0o644)
	})
}

// writeGitignore 在模板自带的 .gitignore 基础上补充「更新签名私钥绝不入库」。
func writeGitignore(root string) error {
	p := filepath.Join(root, ".gitignore")
	data, err := os.ReadFile(p)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	text := string(data)
	if strings.Contains(text, "mygo-update.key") {
		return nil
	}
	if text != "" && !strings.HasSuffix(text, "\n") {
		text += "\n"
	}
	text += "\n# 更新签名私钥：一旦入库等于把发版权限交出去\nmygo-update.key\nmygo-update.pub\n"
	return os.WriteFile(p, []byte(text), 0o644)
}

// escapeQuotes 转义 Go 字符串字面量里的双引号与反斜杠。
func escapeQuotes(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	return strings.ReplaceAll(s, `"`, `\"`)
}

// escapeJSON 转义 JSON 字符串里的特殊字符。
func escapeJSON(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	s = strings.ReplaceAll(s, "\n", `\n`)
	return s
}

// validateJSONFile 校验文件是合法 JSON（改写后兜底，避免交出一个坏配置）。
func validateJSONFile(p string) error {
	data, err := os.ReadFile(p)
	if err != nil {
		return err
	}
	var v any
	return json.Unmarshal(data, &v)
}
