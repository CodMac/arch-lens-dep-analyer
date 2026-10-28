// Package output 提供分析结果的 JSONL 序列化导出。
package output

import (
	"encoding/json"
	"io"
)

// JSONLWriter 逐行写 JSON 对象（每行一个）。
type JSONLWriter struct {
	encoder *json.Encoder
}

// NewJSONLWriter 创建包装 w 的 JSONL 写入器。
func NewJSONLWriter(w io.Writer) *JSONLWriter {
	return &JSONLWriter{encoder: json.NewEncoder(w)}
}

// Write 将 v 编码为一行 JSON 写入。
func (w *JSONLWriter) Write(v interface{}) error { return w.encoder.Encode(v) }
