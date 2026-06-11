package com.example.lombok;

import lombok.Data;
import lombok.extern.slf4j.Slf4j;

@Data
@Slf4j
public class MultipleAnnotationsExample {
    private String id;
    private String name;
    private int value;

    public void processData() {
        // 使用Lombok @Data生成的getter/setter
        this.setName("Test");
        this.setValue(100);

        String result = this.getName() + ":" + this.getValue();

        // 使用Lombok @Slf4j生成的log字段
        log.info("Processed data: {}", result);

        // 使用Lombok @Data生成的toString
        String str = this.toString();
        log.debug("Object as string: {}", str);
    }
}