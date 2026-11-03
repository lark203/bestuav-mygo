#!/usr/bin/env python3
"""把本仓库的源码快照到 internal/scaffold/testdata/template。

为什么需要快照：
    「新建项目」生成器优先读本地源码目录（永远最新），但把这个 exe 拿到没有源码
    的机器上时，就只能用编译进二进制的这份快照。快照过期会让生成出来的项目是旧版
    代码，所以**每次发版前必须跑一次本脚本**。

为什么放在 testdata 下：
    go 工具链会忽略 testdata 目录，快照里的 .go 文件不会被当作本模块的包参与编译；
    而 go:embed 仍然能读到它们。放在别处（例如 internal/scaffold/template）会导致
    同一份代码被编译两遍，import 路径还会互相冲突。

用法：
    python scripts/snapshot-template.py          # 刷新快照
    python scripts/snapshot-template.py --check  # 只检查是否与源码一致（CI 用）

白名单必须与 Go 侧 internal/scaffold/template.go 的 keepExact / keepPrefix /
dropPrefix / dropExact 保持一致，改一边要同步另一边。
"""

import argparse
import filecmp
import os
import shutil
import sys

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
DEST = os.path.join(ROOT, "internal", "scaffold", "testdata", "template")

KEEP_EXACT = {
    "go.mod",
    "go.sum",
    "mygo.json",
    "main.go",
    ".gitignore",
    ".gitattributes",
}
KEEP_PREFIX = ("internal/", "scripts/", "resources/", ".github/", ".agents/")
DROP_PREFIX = ("internal/scaffold/", "assets/")
DROP_EXACT = {
    # 生成器自己的菜单页（依赖 internal/scaffold）
    "internal/view/page_newproject.go",
    "internal/view/page_newproject_test.go",
    # 快照刷新脚本只对本仓库有意义
    "scripts/snapshot-template.py",
}

# go:embed 拒绝嵌入「含 go.mod 的目录」（会被当成另一个 module），所以快照里的
# 这两个文件改名存放，复制时由 Go 侧改回原名。
RENAME_IN_SNAPSHOT = {
    "go.mod": "go.mod.snapshot",
    "go.sum": "go.sum.snapshot",
}


def snapshot_name(rel: str) -> str:
    return RENAME_IN_SNAPSHOT.get(rel, rel)


def wanted(rel: str) -> bool:
    rel = rel.replace(os.sep, "/")
    if rel in DROP_EXACT:
        return False
    if any(rel.startswith(p) for p in DROP_PREFIX):
        return False
    if rel in KEEP_EXACT:
        return True
    return any(rel.startswith(p) for p in KEEP_PREFIX)


def collect():
    """返回仓库内应当进入快照的相对路径列表。"""
    out = []
    for dirpath, dirnames, filenames in os.walk(ROOT):
        rel_dir = os.path.relpath(dirpath, ROOT)
        if rel_dir == ".":
            rel_dir = ""
        # 跳过 VCS 与构建产物目录，避免把 .git 拷进去。
        dirnames[:] = [
            d
            for d in dirnames
            if d not in (".git", "build", ".idea", "node_modules", "__pycache__")
        ]
        for name in filenames:
            rel = f"{rel_dir}/{name}" if rel_dir else name
            rel = rel.replace(os.sep, "/")
            if wanted(rel):
                out.append(rel)
    return sorted(out)


def main() -> int:
    ap = argparse.ArgumentParser(description="Refresh the embedded project template snapshot.")
    ap.add_argument("--check", action="store_true", help="only verify the snapshot is up to date")
    args = ap.parse_args()

    files = collect()
    if args.check:
        problems = []
        for rel in files:
            src = os.path.join(ROOT, rel)
            dst = os.path.join(DEST, snapshot_name(rel))
            if not os.path.exists(dst):
                problems.append(f"missing: {rel}")
            elif not filecmp.cmp(src, dst, shallow=False):
                problems.append(f"stale:   {rel}")
        if problems:
            print("snapshot out of date:")
            for p in problems:
                print("  " + p)
            print("run: python scripts/snapshot-template.py")
            return 1
        print(f"snapshot up to date ({len(files)} files)")
        return 0

    if os.path.isdir(DEST):
        shutil.rmtree(DEST)
    for rel in files:
        dst = os.path.join(DEST, snapshot_name(rel))
        os.makedirs(os.path.dirname(dst), exist_ok=True)
        shutil.copy2(os.path.join(ROOT, rel), dst)
    print(f"snapshot refreshed: {len(files)} files -> {os.path.relpath(DEST, ROOT)}")
    return 0


if __name__ == "__main__":
    sys.exit(main())
