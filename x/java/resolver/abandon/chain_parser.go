package abandon

import (
	"strings"

	"github.com/CodMac/arch-lens-dep-analyer/model"

	"github.com/CodMac/arch-lens-dep-analyer/core"
	sitter "github.com/tree-sitter/go-tree-sitter"
)

// =============================================================================
// types
// =============================================================================

// ReceiverType receiver的类型枚举
type ReceiverType int

const (
	ReceiverNone ReceiverType = iota
	ReceiverThis
	ReceiverSuper
	ReceiverClassName
	ReceiverVariable
	ReceiverField
	ReceiverChained
)

// ChainStep 链式调用中的一个步骤
type ChainStep struct {
	Name    string       // 标识符名称
	IsCall  bool         // 是否为方法调用
	IsNew   bool         // 是否为对象创建表达式
	IsField bool         // 是否为字段访问
	IsArray bool         // 是否为数组访问
	ASTNode *sitter.Node // 对应的AST节点
	RawText string       // 原始文本
}

// ChainedContext 链式调用的上下文信息
type ChainedContext struct {
	Steps       []ChainStep        // 链式调用的各个步骤
	CurrentType *model.CodeElement // 当前解析到的代码元素
	Resolved    bool               // 是否已解析完成
	Error       error              // 解析过程中的错误信息
}

// Receiver 统一的接收者处理结构体
type Receiver struct {
	Type            ReceiverType       // receiver类型
	RawText         string             // 原始文本
	ResolvedElement *model.CodeElement // 解析后的代码元素
	IsStatic        bool               // 是否为静态上下文
	Chained         *ChainedContext    // 链式调用上下文(如果适用)
	Node            *sitter.Node       // 原始AST节点
}

// =============================================================================
// ChainParser
// =============================================================================

// ChainParser 链式调用解析器
type ChainParser struct {
	gCtx *core.GlobalContext
	fCtx *core.FileContext
	src  *[]byte
}

// NewChainParser 创建链式调用解析器
func NewChainParser(gCtx *core.GlobalContext, fCtx *core.FileContext) *ChainParser {
	return &ChainParser{
		gCtx: gCtx,
		fCtx: fCtx,
		src:  fCtx.SourceBytes,
	}
}

// ParseReceiverFromNode 从AST节点解析Receiver
func (cp *ChainParser) ParseReceiverFromNode(node *sitter.Node) *Receiver {
	if node == nil {
		return &Receiver{Type: ReceiverNone}
	}

	// 获取节点的原始文本
	rawText := node.Utf8Text(*cp.src)
	rawText = strings.TrimSpace(rawText)

	if rawText == "" {
		return &Receiver{Type: ReceiverNone}
	}

	// 检查是否是链式调用（包含多个"."或复杂表达式）
	if cp.isChainedExpression(node) {
		return cp.parseChainedReceiver(node, rawText)
	}

	// 简单Expression解析
	return cp.parseSimpleReceiver(node, rawText)
}

// isChainedExpression 判断是否为链式调用表达式
func (cp *ChainParser) isChainedExpression(node *sitter.Node) bool {
	// 直接检查节点类型，所有 field_access 和 method_invocation 都需要作为链式调用处理
	kind := node.Kind()

	if kind == "field_access" || kind == "method_invocation" {
		return true
	}

	return false
}

// parseChainedReceiver 解析链式调用receiver
func (cp *ChainParser) parseChainedReceiver(chainedNode *sitter.Node, rawText string) *Receiver {
	var steps []ChainStep
	steps = cp.recursiveExtractSteps(chainedNode, &steps)

	chainedCtx := &ChainedContext{
		Steps:    steps,
		Resolved: false,
	}

	return &Receiver{
		Type:    ReceiverChained,
		RawText: rawText,
		Chained: chainedCtx,
		Node:    chainedNode,
	}
}

// parseSimpleReceiver 解析简单receiver
func (cp *ChainParser) parseSimpleReceiver(node *sitter.Node, rawText string) *Receiver {
	kind := node.Kind()

	switch kind {
	case "identifier":
		switch rawText {
		case "this":
			return &Receiver{Type: ReceiverThis, RawText: rawText, Node: node}
		case "super":
			return &Receiver{Type: ReceiverSuper, RawText: rawText, Node: node}
		default:
			return &Receiver{Type: ReceiverVariable, RawText: rawText, Node: node}
		}

	case "object_creation_expression":
		return &Receiver{Type: ReceiverClassName, RawText: rawText, Node: node}

	case "field_access", "method_invocation":
		return cp.parseChainedReceiver(node, rawText)

	default:
		return &Receiver{Type: ReceiverVariable, RawText: rawText, Node: node}
	}
}

// recursiveExtractSteps 递归提取链式调用步骤
func (cp *ChainParser) recursiveExtractSteps(node *sitter.Node, steps *[]ChainStep) []ChainStep {
	switch node.Kind() {
	case "field_access":
		cp.processObjectNode(node, steps, cp.extractFieldStep)

	case "method_invocation":
		cp.processObjectNode(node, steps, cp.extractMethodStep)

	case "array_access":
		cp.processObjectNode(node, steps, cp.extractArrayAccessStep)

	case "parenthesized_expression":
		child := node.NamedChild(0)
		if child != nil {
			return cp.recursiveExtractSteps(child, steps)
		}

	default:
		if step := cp.extractBaseStep(node); step != nil {
			*steps = append(*steps, *step)
		}
	}
	return *steps
}

// processObjectNode 处理包含object字段的节点，统一处理object提取和当前节点步骤提取
func (cp *ChainParser) processObjectNode(node *sitter.Node, steps *[]ChainStep, stepExtractor func(*sitter.Node) *ChainStep) {
	obj := node.ChildByFieldName("object")
	if obj != nil {
		if cp.shouldExtract(obj) {
			*steps = cp.recursiveExtractSteps(obj, steps)
		} else {
			if step := cp.extractBaseStep(obj); step != nil {
				*steps = append(*steps, *step)
			}
		}
	}
	if step := stepExtractor(node); step != nil {
		*steps = append(*steps, *step)
	}
}

// shouldExtract 判断节点是否需要继续提取
func (cp *ChainParser) shouldExtract(node *sitter.Node) bool {
	kind := node.Kind()
	return kind == "field_access" || kind == "method_invocation" || kind == "array_access" || kind == "parenthesized_expression"
}

// extractMethodStep 提取方法调用步骤
func (cp *ChainParser) extractMethodStep(node *sitter.Node) *ChainStep {
	// 获取方法名
	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		return nil
	}

	methodName := nameNode.Utf8Text(*cp.src)
	return &ChainStep{
		Name:    methodName,
		IsCall:  true,
		ASTNode: node,
		RawText: node.Utf8Text(*cp.src),
	}
}

// extractFieldStep 提取字段访问步骤
func (cp *ChainParser) extractFieldStep(node *sitter.Node) *ChainStep {
	fieldNode := node.ChildByFieldName("field")
	if fieldNode == nil {
		return nil
	}

	fieldName := fieldNode.Utf8Text(*cp.src)

	return &ChainStep{
		Name:    fieldName,
		IsCall:  false,
		IsField: true,
		ASTNode: node,
		RawText: node.Utf8Text(*cp.src),
	}
}

// extractBaseStep 提取最底层的表达式步骤
func (cp *ChainParser) extractBaseStep(node *sitter.Node) *ChainStep {
	rawText := node.Utf8Text(*cp.src)
	rawText = strings.TrimSpace(rawText)

	switch node.Kind() {
	case "identifier":
		return &ChainStep{
			Name:    rawText,
			IsCall:  false,
			ASTNode: node,
			RawText: rawText,
		}
	case "this":
		return &ChainStep{
			Name:    "this",
			IsCall:  false,
			ASTNode: node,
			RawText: rawText,
		}
	case "super":
		return &ChainStep{
			Name:    "super",
			IsCall:  false,
			ASTNode: node,
			RawText: rawText,
		}
	case "object_creation_expression":
		return &ChainStep{
			Name:    rawText,
			IsCall:  false,
			IsNew:   true,
			ASTNode: node,
			RawText: rawText,
		}
	case "string_literal", "decimal_integer_literal", "decimal_floating_point_literal":
		return &ChainStep{
			Name:    rawText,
			IsCall:  false,
			ASTNode: node,
			RawText: rawText,
		}
	}

	return &ChainStep{
		Name:    rawText,
		IsCall:  false,
		ASTNode: node,
		RawText: rawText,
	}
}

// extractArrayAccessStep 提取数组访问步骤
func (cp *ChainParser) extractArrayAccessStep(node *sitter.Node) *ChainStep {
	rawText := node.Utf8Text(*cp.src)
	return &ChainStep{
		Name:    rawText,
		IsCall:  false,
		IsArray: true,
		ASTNode: node,
		RawText: rawText,
	}
}
