package com.example.rel.use.case10;

import java.util.List;
import java.util.stream.Collectors;

/**
 * StreamLambdaChainSuite 验证 stream 链式 filter+forEach 两级 lambda 的 Source 归属。
 *
 * 场景：list.stream().filter(e -> ...).forEach(config -> { body })
 *  filter 谓词 lambda 与 forEach 消费者 lambda 起始行在同一行（链式写法），
 *  修复前 determinePreciseSource 按行号匹配只取第一个 lambda，消费者 body
 *  内的语句被误归到 filter 谓词（BUG-1）。修复后按 AST 节点区间匹配最内层。
 *
 * 归属期望：
 *  - filter 谓词 lambda 内的动作 -> run().lambda$1（谓词）
 *  - forEach 消费者 body 内的动作 -> run().lambda$2（消费者，body 覆盖后续多行）
 */
public class StreamLambdaChainSuite {
    private List<String> names;

    public void run() {
        names.stream().filter(e -> e.startsWith("com")).forEach(config -> {
            String upper = config.toUpperCase();
            int len = upper.length();
            System.out.println(config);
        });
    }
}
