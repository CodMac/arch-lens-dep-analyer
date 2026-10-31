package com.example.base.enum;

import lombok.Getter;

import java.util.HashMap;
import java.util.Map;

/**
 * EnumLombok - 验证 @Getter 作用于枚举时生成 getter 方法（枚举内部访问自身 getter）。
 */
@Getter
public enum EnumLombok {
    /**
     * 项.
     */
    FIRST("first", 1),
    /**
     * 项.
     */
    SECOND("second", 2);

    private final String key;
    private final int code;

    EnumLombok(String key, int code) {
        this.key = key;
        this.code = code;
    }

    private static final Map<String, EnumLombok> LOOKUP = new HashMap<>();

    static {
        for (EnumLombok item : values()) {
            LOOKUP.put(item.getKey(), item);
        }
    }

    /**
     * 根据 key 查找.
     *
     * @param key key
     * @return 枚举项
     */
    public static EnumLombok fromKey(String key) {
        return LOOKUP.get(key);
    }
}
