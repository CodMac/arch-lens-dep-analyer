package com.example.audit;

import java.io.File;
import java.nio.file.Path;
import java.util.ArrayList;
import java.util.List;
import org.springframework.http.HttpStatus;

public class FieldAccessSuite {
    public enum Color { RED, GREEN }

    // 枚举常量访问 + class literal + 外部字段访问
    public void testAll(File file, Color color) {
        Color c = Color.RED;                 // 枚举常量（同包枚举）
        HttpStatus s = HttpStatus.NOT_FOUND; // 外部枚举常量
        Class<?> clz1 = File.class;          // class literal (外部)
        Class<?> clz2 = Color.class;         // class literal (内部枚举)
        String p = Path.of("x").toString(); // 链式外部
        List<String> list = new ArrayList<>();
        list.add("item");                    // 变量接收者外部方法：list -> ArrayList.add
    }
}
