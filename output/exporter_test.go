package output

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/CodMac/arch-lens-dep-analyer/core"
	"github.com/CodMac/arch-lens-dep-analyer/model"
	"github.com/stretchr/testify/assert"
)

func TestExporter_ExportJsonL(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "exporter_test")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	exporter := NewExporter(tempDir, JsonL)

	gCtx := &core.GlobalContext{
		Definitions: []*core.DefinitionEntry{
			{
				Element: &model.CodeElement{
					QualifiedName: "pkg.Class",
					Name:          "Class",
					Kind:          model.Class,
				},
			},
		},
	}

	rels := []*model.DependencyRelation{
		{
			Type:   model.Call,
			Source: &model.CodeElement{QualifiedName: "src"},
			Target: &model.CodeElement{QualifiedName: "tgt"},
		},
		{
			Type:   model.Import,
			Source: &model.CodeElement{QualifiedName: "src"},
			Target: &model.CodeElement{QualifiedName: "tgt"},
		},
	}

	ec, rc, err := exporter.ExportJsonL(gCtx, rels)
	assert.NoError(t, err)
	assert.Equal(t, 1, ec)
	assert.Equal(t, 2, rc)

	// Check files
	assert.FileExists(t, filepath.Join(tempDir, "element.jsonl"))
	assert.FileExists(t, filepath.Join(tempDir, "relation_call.jsonl"))
	assert.FileExists(t, filepath.Join(tempDir, "relation_import.jsonl"))
	assert.NoFileExists(t, filepath.Join(tempDir, "relation.jsonl"))
}
