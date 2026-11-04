package com.example.rel.implement.case1.local;

/**
 * SelfQualifiedImpl 验证同文件 self 限定嵌套接口 implements（C-10 修复变体）：
 *   implements SelfQualifiedImpl.NestedContract
 *
 * 此形态走 PreciseResolve 包内拼 QN 路径（fc.PackageName + symbol）即可命中
 * （...local.SelfQualifiedImpl.NestedContract 恰为正确 QN）。
 */
class SelfQualifiedImpl implements SelfQualifiedImpl.NestedContract {

    interface NestedContract {
        void doWork();
    }

    @Override
    public void doWork() {}
}
