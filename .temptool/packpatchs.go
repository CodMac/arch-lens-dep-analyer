package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

func main() {
	mode := flag.String("mode", "", "操作模式: pack (打包) 或 unpack (还原)")
	baseCommit := flag.String("base", "", "基准 commit / tag / branch (仅 pack 模式需要)")
	patchDir := flag.String("dir", ".patchs", "patch 文件存放目录")

	flag.Parse()

	if *mode == "" {
		printPackPatchsUsage()
		os.Exit(1)
	}

	switch *mode {
	case "pack":
		if *baseCommit == "" {
			fmt.Println("错误: pack 模式下必须指定 -base 参数")
			os.Exit(1)
		}
		if err := packPatches(*baseCommit, *patchDir); err != nil {
			fmt.Printf("打包失败: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("打包成功！")

	case "unpack":
		if err := unpackPatches(*patchDir); err != nil {
			fmt.Printf("还原失败: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("还原成功！")

	default:
		fmt.Printf("未知模式: %s，支持的模式为 pack 或 unpack\n", *mode)
		printPackPatchsUsage()
		os.Exit(1)
	}
}

func printPackPatchsUsage() {
	fmt.Println("用法示例:")
	fmt.Println("  打包: go run main.go -- -mode=pack -base=v1.0.0 -dir=.patchs")
	fmt.Println("  还原: go run main.go -- -mode=unpack -dir=.patchs")
	fmt.Println("\n参数说明:")
	fmt.Println("  -mode     操作模式 (pack 或 unpack)")
	fmt.Println("  -base     基准 commit (pack 模式必填)")
	fmt.Println("  -dir      patch 文件目录 (默认 .patchs)")
}

func runGit(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	output, err := cmd.CombinedOutput()
	return string(output), err
}

// -------------------- 功能 1：打包 --------------------

func packPatches(baseCommit, patchDir string) error {
	// 1. 获取从 baseCommit 到 HEAD 的所有 commit hash (按时间正序)
	out, err := runGit("log", "--reverse", "--format=%H", fmt.Sprintf("%s..HEAD", baseCommit))
	if err != nil {
		return fmt.Errorf("获取 commit 列表失败: %v, 输出: %s", err, out)
	}

	lines := strings.Split(strings.TrimSpace(out), "\n")
	var commits []string
	for _, l := range lines {
		l = strings.TrimSpace(l)
		if l != "" {
			commits = append(commits, l)
		}
	}

	if len(commits) == 0 {
		return fmt.Errorf("没有找到从 %s 到 HEAD 的任何新 commit", baseCommit)
	}

	// 2. 创建 patch 目录
	if err := os.MkdirAll(patchDir, 0755); err != nil {
		return err
	}

	// 3. 逐个 commit 生成 diff patch
	prev := baseCommit
	for i, commit := range commits {
		patchName := fmt.Sprintf("%03d_%s.patch", i+1, commit[:8])
		patchPath := filepath.Join(patchDir, patchName)

		// 生成 git diff
		diffOut, err := runGit("diff", prev, commit)
		if err != nil {
			return fmt.Errorf("生成 commit %s 的 patch 失败: %v, 输出: %s", commit[:8], err, diffOut)
		}

		if err := os.WriteFile(patchPath, []byte(diffOut), 0644); err != nil {
			return err
		}
		fmt.Printf("已生成: %s\n", patchName)
		prev = commit
	}

	return nil
}

// -------------------- 功能 2：解包/还原 --------------------

func unpackPatches(patchDir string) error {
	absDir, err := filepath.Abs(patchDir)
	if err != nil {
		return err
	}

	entries, err := os.ReadDir(absDir)
	if err != nil {
		return fmt.Errorf("读取 patch 目录失败: %v", err)
	}

	var patchFiles []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(strings.ToLower(e.Name()), ".patch") {
			patchFiles = append(patchFiles, filepath.Join(absDir, e.Name()))
		}
	}

	// 按文件名排序（保证 001, 002... 的先后顺序）
	sort.Strings(patchFiles)
	if len(patchFiles) == 0 {
		return fmt.Errorf("在 %s 下没有找到任何 .patch 文件", patchDir)
	}

	// 逐个应用 patch 并自动提交
	for _, patchPath := range patchFiles {
		fileName := filepath.Base(patchPath)
		fmt.Printf("正在应用: %s\n", fileName)

		// 1. git apply
		if out, err := runGit("apply", patchPath); err != nil {
			return fmt.Errorf("应用 patch %s 失败:\n%s", fileName, out)
		}

		// 2. git add
		if out, err := runGit("add", "-A"); err != nil {
			return fmt.Errorf("git add 失败:\n%s", out)
		}

		// 3. git commit
		commitMsg := fmt.Sprintf("Apply patch: %s", fileName)
		if out, err := runGit("commit", "-m", commitMsg); err != nil {
			return fmt.Errorf("git commit 失败:\n%s", out)
		}
	}

	return nil
}
