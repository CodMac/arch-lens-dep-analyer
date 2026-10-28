package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	FileStartMarker = "=== BEGIN FILE: "
	FileEndMarker   = "=== END FILE ==="
)

func main() {
	mode := flag.String("mode", "", "操作模式: pack (打包) 或 unpack (还原)")
	src := flag.String("src", "", "源文件夹路径 (打包时为源码目录，还原时为打包txt所在目录)")
	out := flag.String("out", "", "输出文件夹路径 (打包时为txt输出目录，还原时为文件还原目录)")
	exts := flag.String("ext", "", "文件后缀列表，用逗号分隔 (例如: .go,.txt,.md)")
	maxLines := flag.Int("maxlines", 2000, "每个输出txt文件的最大行数")

	flag.Parse()

	if *mode == "" || *src == "" || *out == "" {
		printUsage()
		os.Exit(1)
	}

	switch *mode {
	case "pack":
		if *exts == "" {
			fmt.Println("错误: 打包模式下必须指定 -ext 参数")
			os.Exit(1)
		}
		extList := parseExtensions(*exts)
		err := packFiles(*src, *out, extList, *maxLines)
		if err != nil {
			fmt.Printf("打包失败: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("打包成功！")

	case "unpack":
		err := unpackFiles(*src, *out)
		if err != nil {
			fmt.Printf("还原失败: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("还原成功！")

	default:
		fmt.Printf("未知模式: %s，支持的模式为 pack 或 unpack\n", *mode)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("用法示例:")
	fmt.Println("  打包: go run main.go -mode=pack -src=./my_project -out=./output_txts -ext=.go,.md -maxlines=1500")
	fmt.Println("  还原: go run main.go -mode=unpack -src=./output_txts -out=./restored_project")
	fmt.Println("\n参数说明:")
	fmt.Println("  -mode     操作模式 (pack 或 unpack)")
	fmt.Println("  -src      源路径")
	fmt.Println("  -out      输出路径")
	fmt.Println("  -ext      文件后缀 (仅 pack 模式需要)")
	fmt.Println("  -maxlines 单个txt最大行数 (仅 pack 模式需要，默认 2000)")
}

func parseExtensions(extStr string) []string {
	parts := strings.Split(extStr, ",")
	var list []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			if !strings.HasPrefix(p, ".") {
				p = "." + p
			}
			list = append(list, strings.ToLower(p))
		}
	}
	return list
}

// -------------------- 功能 1：打包 --------------------

func packFiles(srcDir, outDir string, extList []string, maxLines int) error {
	// 1. 收集所有符合条件的文件路径（使用相对路径以保证跨平台和还原）
	var filePaths []string
	absSrc, err := filepath.Abs(srcDir)
	if err != nil {
		return err
	}

	err = filepath.WalkDir(absSrc, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			// 跳过隐藏文件夹，如 .git
			if strings.HasPrefix(d.Name(), ".") && path != absSrc {
				return filepath.SkipDir
			}
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		match := false
		for _, e := range extList {
			if ext == e {
				match = true
				break
			}
		}

		if match {
			rel, err := filepath.Rel(absSrc, path)
			if err != nil {
				return err
			}
			filePaths = append(filePaths, rel)
		}
		return nil
	})

	if err != nil {
		return err
	}

	// 排序保证顺序一致
	sort.Strings(filePaths)

	if len(filePaths) == 0 {
		return fmt.Errorf("没有找到符合条件的文件")
	}

	// 2. 创建输出目录
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return err
	}

	fileIndex := 1
	currentLineCount := 0
	var currentWriter *bufio.Writer
	var currentFile *os.File

	openNewFile := func() error {
		if currentFile != nil {
			currentWriter.Flush()
			currentFile.Close()
		}
		outFilePath := filepath.Join(outDir, fmt.Sprintf("part_%03d.txt", fileIndex))
		f, err := os.Create(outFilePath)
		if err != nil {
			return err
		}
		currentFile = f
		currentWriter = bufio.NewWriter(f)
		fileIndex++
		currentLineCount = 0
		return nil
	}

	if err := openNewFile(); err != nil {
		return err
	}
	defer func() {
		if currentFile != nil {
			currentWriter.Flush()
			currentFile.Close()
		}
	}()

	// 3. 逐个文件读取并写入
	for _, relPath := range filePaths {
		fullPath := filepath.Join(absSrc, relPath)
		f, err := os.Open(fullPath)
		if err != nil {
			return err
		}

		// 计算当前文件需要占据的行数 (头 + 内容 + 尾)
		// 为了精准控制，我们先读取文件内容
		var fileLines []string
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			fileLines = append(fileLines, scanner.Text())
		}
		f.Close()

		// 块头尾占 2 行
		neededLines := len(fileLines) + 2

		// 如果当前文件放不下，且当前文件已经写了一些内容，则切换新文件
		if currentLineCount+neededLines > maxLines && currentLineCount > 0 {
			if err := openNewFile(); err != nil {
				return err
			}
		}

		// 写入文件头
		header := fmt.Sprintf("%s%s ===\n", FileStartMarker, relPath)
		currentWriter.WriteString(header)
		currentLineCount++

		// 写入文件内容
		for _, line := range fileLines {
			currentWriter.WriteString(line + "\n")
			currentLineCount++
		}

		// 写入文件尾
		currentWriter.WriteString(FileEndMarker + "\n")
		currentLineCount++
	}

	return nil
}

// -------------------- 功能 2：还原 --------------------

func unpackFiles(srcDir, outDir string) error {
	absSrc, err := filepath.Abs(srcDir)
	if err != nil {
		return err
	}

	// 1. 查找源目录下所有的 txt 文件
	entries, err := os.ReadDir(absSrc)
	if err != nil {
		return err
	}

	var txtFiles []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(strings.ToLower(e.Name()), ".txt") {
			txtFiles = append(txtFiles, filepath.Join(absSrc, e.Name()))
		}
	}

	sort.Strings(txtFiles)
	if len(txtFiles) == 0 {
		return fmt.Errorf("在 %s 下没有找到任何 .txt 文件", srcDir)
	}

	if err := os.MkdirAll(outDir, 0755); err != nil {
		return err
	}

	// 2. 依次读取每个 txt 文件并还原
	var currentOutFile *os.File
	var currentWriter *bufio.Writer

	defer func() {
		if currentOutFile != nil {
			currentWriter.Flush()
			currentOutFile.Close()
		}
	}()

	for _, txtPath := range txtFiles {
		f, err := os.Open(txtPath)
		if err != nil {
			return err
		}

		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			line := scanner.Text()

			if strings.HasPrefix(line, FileStartMarker) {
				// 提取相对路径
				relPath := strings.TrimSuffix(strings.TrimPrefix(line, FileStartMarker), " ===")
				targetFilePath := filepath.Join(outDir, relPath)

				// 创建父目录
				if err := os.MkdirAll(filepath.Dir(targetFilePath), 0755); err != nil {
					f.Close()
					return err
				}

				// 关闭上一个文件
				if currentOutFile != nil {
					currentWriter.Flush()
					currentOutFile.Close()
				}

				// 创建新的目标文件
				outF, err := os.Create(targetFilePath)
				if err != nil {
					f.Close()
					return err
				}
				currentOutFile = outF
				currentWriter = bufio.NewWriter(outF)

			} else if line == FileEndMarker {
				if currentOutFile != nil {
					currentWriter.Flush()
					currentOutFile.Close()
					currentOutFile = nil
					currentWriter = nil
				}
			} else {
				if currentWriter != nil {
					currentWriter.WriteString(line + "\n")
				}
			}
		}
		f.Close()
	}

	return nil
}
