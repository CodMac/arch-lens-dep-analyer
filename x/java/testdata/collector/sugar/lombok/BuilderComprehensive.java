package com.example.lombok;

import lombok.Builder;
import lombok.AllArgsConstructor;
import lombok.NoArgsConstructor;

@Builder
@NoArgsConstructor
@AllArgsConstructor
public class BuilderComprehensive {
    private String host;
    private int port;
    private boolean secure;
    private String username;
    private String password;

    // 嵌套Builder模式
    @Builder
    public static class Config {
        private String database;
        private int maxConnections;
        private long timeout;
    }
}

public class BuilderUsageTest {
    public void testBuilderPattern() {
        // 链式调用构建对象
        BuilderComprehensive config = BuilderComprehensive.builder()
            .host("localhost")
            .port(8080)
            .secure(false)
            .username("admin")
            .password("secret")
            .build();

        // 使用默认构造器
        BuilderComprehensive defaultConfig = new BuilderComprehensive();

        // 使用全参构造器
        BuilderComprehensive fullConfig = new BuilderComprehensive(
            "localhost", 8080, true, "admin", "secret"
        );

        // 嵌套Builder
        BuilderComprehensive.Config dbConfig = BuilderComprehensive.Config.builder()
            .database("mysql")
            .maxConnections(100)
            .timeout(5000L)
            .build();
    }
}