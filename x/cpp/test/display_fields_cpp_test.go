package test

import (
	"testing"

	"github.com/CodMac/arch-lens-dep-analyer/model"
)

// =============================================================================
// 20. 展示层字段边界（NamespaceQN / EnclosingQN / TypeQN）
// =============================================================================

// TestCollectorCppDisplayFieldsEdge 逐场景断言展示层三字段的边界行为。
func TestCollectorCppDisplayFieldsEdge(t *testing.T) {
	t.Run("namespace 全形态", func(t *testing.T) {
		gc := collectCpp(t, "display")

		// 三层嵌套 namespace
		AssertDisplayFields(t, gc, "ns_outer", "", "", "")
		AssertDisplayFields(t, gc, "ns_outer::ns_middle", "ns_outer", "ns_outer", "")
		AssertDisplayFields(t, gc, "ns_outer::ns_middle::ns_inner", "ns_outer::ns_middle", "ns_outer::ns_middle", "")
		AssertDisplayFields(t, gc, "ns_outer::ns_middle::ns_inner::DeepType",
			"ns_outer::ns_middle::ns_inner", "ns_outer::ns_middle::ns_inner",
			"ns_outer::ns_middle::ns_inner::DeepType")
		AssertDisplayFields(t, gc, "ns_outer::ns_middle::ns_inner::DeepType::d",
			"ns_outer::ns_middle::ns_inner", "ns_outer::ns_middle::ns_inner::DeepType",
			"ns_outer::ns_middle::ns_inner::DeepType")
		// C++17 嵌套写法
		AssertDisplayFields(t, gc, "ns_modern::ns_sub", "ns_modern", "", "")
		AssertDisplayFields(t, gc, "ns_modern::ns_sub::ModernType",
			"ns_modern::ns_sub", "ns_modern::ns_sub", "ns_modern::ns_sub::ModernType")
		// inline namespace
		AssertDisplayFields(t, gc, "ns_inline::v1", "ns_inline", "ns_inline", "")
		AssertDisplayFields(t, gc, "ns_inline::v1::InlineType",
			"ns_inline::v1", "ns_inline::v1", "ns_inline::v1::InlineType")
		// 匿名 namespace
		AssertDisplayFields(t, gc, "$anon", "", "", "")
		AssertDisplayFields(t, gc, "$anon::anon_var", "$anon", "$anon", "")
		AssertDisplayFields(t, gc, "$anon::AnonType", "$anon", "$anon", "$anon::AnonType")
		// 重开 namespace：QN 层为 ns_reopen$2，展示层 NamespaceQN 归并为 ns_reopen
		AssertDisplayFields(t, gc, "ns_reopen::First", "ns_reopen", "ns_reopen", "ns_reopen::First")
		AssertDisplayFields(t, gc, "ns_reopen$2", "", "", "")
		AssertDisplayFields(t, gc, "ns_reopen$2::Second", "ns_reopen", "ns_reopen", "ns_reopen::Second")
		AssertDisplayFields(t, gc, "ns_reopen$2::Second::b", "ns_reopen", "ns_reopen$2::Second", "ns_reopen::Second")
	})

	t.Run("类型全形态", func(t *testing.T) {
		gc := collectCpp(t, "display")

		// 三层嵌套 struct：TypeQN 恒为最外层 L1
		AssertDisplayFields(t, gc, "ty::L1", "ty", "ty", "ty::L1")
		AssertDisplayFields(t, gc, "ty::L1::L2", "ty", "ty::L1", "ty::L1")
		AssertDisplayFields(t, gc, "ty::L1::L2::L3", "ty", "ty::L1::L2", "ty::L1")
		AssertDisplayFields(t, gc, "ty::L1::L2::L3::deep", "ty", "ty::L1::L2::L3", "ty::L1")
		// union / enum
		AssertDisplayFields(t, gc, "ty::U", "ty", "ty", "ty::U")
		AssertDisplayFields(t, gc, "ty::U::ui", "ty", "ty::U", "ty::U")
		AssertDisplayFields(t, gc, "ty::ScopedE", "ty", "ty", "ty::ScopedE")
		AssertDisplayFields(t, gc, "ty::ScopedE::A", "ty", "ty::ScopedE", "ty::ScopedE")
		AssertDisplayFields(t, gc, "ty::PlainE::P_A", "ty", "ty::PlainE", "ty::PlainE")
		// 模板类内嵌套
		AssertDisplayFields(t, gc, "ty::Box", "ty", "ty", "ty::Box")
		AssertDisplayFields(t, gc, "ty::Box::Nested", "ty", "ty::Box", "ty::Box")
		AssertDisplayFields(t, gc, "ty::Box::Nested::n", "ty", "ty::Box::Nested", "ty::Box")
		AssertDisplayFields(t, gc, "ty::Box::get()", "ty", "ty::Box", "ty::Box")
		// 模板特化：QN 剥离实参
		AssertDisplayFields(t, gc, "ty::Box::special", "ty", "ty::Box", "ty::Box")

		// 匿名承载：typedef 名 / 变量名 / 字段名 —— 均无具名宿主类型
		AssertDisplayFields(t, gc, "AnonPoint", "", "", "")
		AssertDisplayFields(t, gc, "AnonPoint::tx", "", "AnonPoint", "")
		AssertDisplayFields(t, gc, "anon_obj", "", "", "")
		AssertDisplayFields(t, gc, "anon_obj::vx", "", "anon_obj", "")
		// 匿名 union 挂在具名 struct 字段下：宿主类型为外层 struct
		AssertDisplayFields(t, gc, "WithUnion::payload", "", "WithUnion", "WithUnion")
		AssertDisplayFields(t, gc, "WithUnion::payload::as_int", "", "WithUnion::payload", "WithUnion")
		AssertDisplayFields(t, gc, "OuterHolder::pair::pair_a", "", "OuterHolder::pair", "OuterHolder")
	})

	t.Run("局部类型", func(t *testing.T) {
		gc := collectCpp(t, "display")

		// 方法内局部类型：TypeQN=宿主类
		AssertDisplayFields(t, gc, "lc::Host::methodWithLocal()::LocalInMethod",
			"lc", "lc::Host::methodWithLocal()", "lc::Host")
		AssertDisplayFields(t, gc, "lc::Host::methodWithLocal()::LocalInMethod::x",
			"lc", "lc::Host::methodWithLocal()::LocalInMethod", "lc::Host")
		AssertDisplayFields(t, gc, "lc::Host::methodWithLocal()::LocalEnumInMethod::LE_A",
			"lc", "lc::Host::methodWithLocal()::LocalEnumInMethod", "lc::Host")
		// 自由函数内局部类型：无具名宿主类型，TypeQN 为空
		AssertDisplayFields(t, gc, "freeFnWithLocal()::LocalInFree",
			"", "freeFnWithLocal()", "")
		AssertDisplayFields(t, gc, "freeFnWithLocal()::LocalInFree::z",
			"", "freeFnWithLocal()::LocalInFree", "")
		AssertDisplayFields(t, gc, "freeFnWithLocal()::block$1::LocalInBlock",
			"", "freeFnWithLocal()::block$1", "")
		// lambda 内局部类型：无具名宿主类型
		AssertDisplayFields(t, gc, "lambdaWithLocal()::lambda$1::LocalInLambda",
			"", "lambdaWithLocal()::lambda$1", "")
		// 局部类套局部类
		AssertDisplayFields(t, gc, "localInLocal()::OuterLocal",
			"", "localInLocal()", "")
		AssertDisplayFields(t, gc, "localInLocal()::OuterLocal::deepen()::InnerLocal",
			"", "localInLocal()::OuterLocal::deepen()", "")
	})

	t.Run("跨文件类外定义", func(t *testing.T) {
		gc := collectCpp(t, "display")

		// 有 namespace 块：正常
		AssertDisplayFields(t, gc, "ool::Service::member(int)", "ool", "ool::Service", "ool::Service")
		AssertDisplayFields(t, gc, "ool::Service::Service()", "ool", "ool::Service", "ool::Service")
		AssertDisplayFields(t, gc, "ool::Service::~Service()", "ool", "ool::Service", "ool::Service")
		AssertDisplayFields(t, gc, "ool::Service::operator=(const Service&)",
			"ool", "ool::Service", "ool::Service")
		AssertDisplayFields(t, gc, "ool::Service::staticMember(int)", "ool", "ool::Service", "ool::Service")
		AssertDisplayFields(t, gc, "ool::freeFn(int)", "ool", "ool", "")
		AssertDisplayFields(t, gc, "ool::nested::Inner::innerMethod(int)",
			"ool::nested", "ool::nested::Inner", "ool::nested::Inner")

		// 无 namespace 块：从 QN 结构推导
		AssertDisplayFields(t, gc, "GlobalCls::method(int)", "", "GlobalCls", "GlobalCls")
		AssertDisplayFields(t, gc, "globalFreeFn(int)", "", "", "")
		AssertDisplayFields(t, gc, "ool_no_block::Remote::method(int)",
			"ool_no_block", "ool_no_block::Remote", "ool_no_block::Remote")
		AssertDisplayFields(t, gc, "ool_no_block::Remote::Remote()",
			"ool_no_block", "ool_no_block::Remote", "ool_no_block::Remote")
		AssertDisplayFields(t, gc, "ool_no_block::Remote::~Remote()",
			"ool_no_block", "ool_no_block::Remote", "ool_no_block::Remote")
	})

	t.Run("纯 C 文件", func(t *testing.T) {
		gc := collectCpp(t, "display")

		// C 无 namespace：NamespaceQN 恒空
		AssertDisplayFields(t, gc, "CStruct", "", "", "CStruct")
		AssertDisplayFields(t, gc, "CStruct::a", "", "CStruct", "CStruct")
		AssertDisplayFields(t, gc, "CTypedef", "", "", "")
		AssertDisplayFields(t, gc, "CTypedef::x", "", "CTypedef", "")
		AssertDisplayFields(t, gc, "CEnum::C_A", "", "CEnum", "CEnum")
		AssertDisplayFields(t, gc, "CUnion::ui", "", "CUnion", "CUnion")
		AssertDisplayFields(t, gc, "c_global", "", "", "")
		AssertDisplayFields(t, gc, "c_func()", "", "", "")
		AssertDisplayFields(t, gc, "c_func()::local", "", "c_func()", "")
		AssertDisplayFields(t, gc, "c_func()::CLocal", "", "c_func()", "")
	})

	t.Run("杂项", func(t *testing.T) {
		gc := collectCpp(t, "display")

		// 全局变量 / 多声明器：无宿主类型
		AssertDisplayFields(t, gc, "misc::g_var", "misc", "misc", "")
		AssertDisplayFields(t, gc, "misc::multi_a", "misc", "misc", "")
		AssertDisplayFields(t, gc, "misc::multi_ptr", "misc", "misc", "")
		// = default / = delete / 运算符
		AssertDisplayFields(t, gc, "misc::Defaulted::Defaulted()", "misc", "misc::Defaulted", "misc::Defaulted")
		AssertDisplayFields(t, gc, "misc::Defaulted::Defaulted(const Defaulted&)",
			"misc", "misc::Defaulted", "misc::Defaulted")
		AssertDisplayFields(t, gc, "misc::Operated::operator+=(int)", "misc", "misc::Operated", "misc::Operated")
		AssertDisplayFields(t, gc, "misc::Operated::operator bool()", "misc", "misc::Operated", "misc::Operated")
		// extern "C"
		AssertDisplayFields(t, gc, "misc::c_linkage_fn(int)", "misc", "misc", "")
		AssertDisplayFields(t, gc, "misc::c_linkage_var", "misc", "misc", "")
	})

	t.Run("File 元素三字段为空", func(t *testing.T) {
		gc := collectCpp(t, "display")
		for _, def := range gc.Definitions() {
			if def.Element.Kind != model.File {
				continue
			}
			if def.Element.NamespaceQN != "" || def.Element.EnclosingQN != "" || def.Element.TypeQN != "" {
				t.Errorf("File %s display fields should be empty, got NS=%q ENC=%q TYPE=%q",
					def.Element.QualifiedName, def.Element.NamespaceQN, def.Element.EnclosingQN, def.Element.TypeQN)
			}
		}
	})
}
