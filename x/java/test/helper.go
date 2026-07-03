package test

import (
	"fmt"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/CodMac/arch-lens-dep-analyer/x/java/constants"

	"github.com/CodMac/arch-lens-dep-analyer/core"
	"github.com/CodMac/arch-lens-dep-analyer/model"
	"github.com/CodMac/arch-lens-dep-analyer/parser"
)

func GetTestFilePath(name string) string {
	currentDir, _ := filepath.Abs(filepath.Dir("./../../"))
	return filepath.Join(currentDir, "testdata", name)
}

func GetJavaParser(t *testing.T) parser.Parser {
	javaParser, err := parser.NewParser(core.LangJava)
	if err != nil {
		t.Fatalf("Failed to create Java parser: %v", err)
	}

	return javaParser
}

const outputAst = false
const formatAst = true

func RunPhase1Collection(t *testing.T, files []string) *core.GlobalContext {
	resolver, err := core.GetSymbolResolver(core.LangJava)
	if err != nil {
		t.Fatalf("Failed to create resolver: %v", err)
	}

	builder, err := core.GetSymbolBuilder(core.LangJava)
	if err != nil {
		t.Fatalf("Failed to create symbol builder: %v", err)
	}

	gc := core.NewGlobalContext(resolver, builder)
	javaParser, err := parser.NewParser(core.LangJava)
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	col, err := core.GetCollector(core.LangJava)
	if err != nil {
		t.Fatalf("Failed to create collector: %v", err)
	}

	for _, file := range files {
		rootNode, sourceBytes, err := javaParser.ParseFile(file, outputAst, formatAst)
		if err != nil {
			t.Fatalf("Failed to parse file %s: %v", file, err)
		}

		fCtx, err := col.CollectDefinitions(rootNode, file, sourceBytes)
		if err != nil {
			t.Fatalf("Failed to collect definitions for %s: %v", file, err)
		}
		gc.RegisterFileContext(fCtx)
	}

	binder, err := core.GetBinder(core.LangJava)
	if err != nil {
		t.Fatalf("Failed to create binder: %v", err)
	}
	binder.BindSymbols(gc)

	return gc
}

const printEle = true

func PrintCodeElements(fCtx *core.FileContext) {
	if !printEle {
		return
	}

	fmt.Printf("Package: %s\n", fCtx.PackageName)
	for _, def := range fCtx.Definitions {
		fmt.Printf("Short: %s -> Kind: %s, QN: %s\n", def.Element.Name, def.Element.Kind, def.Element.QualifiedName)
		fmt.Printf("      -> Extra: %v\n", def.Element.Extra.Mores)
	}
}

const printRel = true

func PrintRelations(relations []*model.DependencyRelation) {
	if !printRel {
		return
	}
	fmt.Printf("\n--- Found %d relations ---\n", len(relations))
	for _, rel := range relations {
		startLine := "-"
		if rel.Location != nil {
			startLine = strconv.FormatInt(int64(rel.Location.StartLine), 10)
		}

		fmt.Printf("[%s] %s (%s) --> %s (%s)\n	Line: %s\n",
			rel.Type,
			rel.Source.QualifiedName, rel.Source.Kind,
			rel.Target.QualifiedName, rel.Target.Kind,
			startLine)
		if len(rel.Mores) > 0 {
			for k, v := range rel.Mores {
				if k == constants.TmpExpressNode || k == constants.TmpNode {
					continue
				}
				fmt.Printf("    Mores[%v] -> %v\n", k, v)
			}
		}
	}
}

func PrintRelationsOnKinds(relations []*model.DependencyRelation, kinds []model.DependencyType) {
	if !printRel {
		return
	}
	fmt.Printf("\n--- Found %d relations ---\n", len(relations))
	for _, rel := range relations {
		for _, kind := range kinds {
			if kind == rel.Type {
				startLine := "-"
				if rel.Location != nil {
					startLine = strconv.FormatInt(int64(rel.Location.StartLine), 10)
				}

				fmt.Printf("[%s] %s (%s) --> %s (%s)\n	Line: %s\n",
					rel.Type,
					rel.Source.QualifiedName, rel.Source.Kind,
					rel.Target.QualifiedName, rel.Target.Kind,
					startLine)
				if len(rel.Mores) > 0 {
					for k, v := range rel.Mores {
						if k == constants.TmpExpressNode || k == constants.TmpNode {
							continue
						}
						fmt.Printf("    Mores[%v] -> %v\n", k, v)
					}
				}
			}
		}
	}
}

// FindDefinitionsByQN 根据 QN 在 fCtx 中查找定义
func FindDefinitionsByQN(fCtx *core.FileContext, targetQN string) []*core.DefinitionEntry {
	var result []*core.DefinitionEntry
	for _, entry := range fCtx.Definitions {
		if entry.Element.QualifiedName == targetQN {
			result = append(result, entry)
		}
	}

	return result
}

// Contains 判断 slice 是否包含 string
func Contains(s []string, str string) bool {
	for _, v := range s {
		if v == str {
			return true
		}
	}

	return false
}
