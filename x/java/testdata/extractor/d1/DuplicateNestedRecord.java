package com.example.rel.d1;

// D-1 同文件重名嵌套类型解析错配用例。
// 复现 spring DataBufferUtils 的 ReadCompletionHandler.Attachment / WriteCompletionHandler.Attachment
// 双同名嵌套 record：Read 版先定义，Write 版后定义。
// 修复前：WriteCompletionHandler 内裸名 Attachment 引用全部按文件收集顺序错配到
// ReadCompletionHandler.Attachment（CREATE/CALL/PARAMETER）。
// 修复后：Write 作用域内引用 -> WriteCompletionHandler.Attachment；Read 作用域内 -> Read 版。

class DuplicateNestedRecord {

    static class ReadCompletionHandler {
        record Attachment(DataBuffer dataBuffer, ByteBufferIterator iterator) {
        }

        void read() {
            DataBuffer buffer = new DataBuffer();
            ByteBufferIterator it = new ByteBufferIterator();
            Attachment attachment = new Attachment(buffer, it);
            attachment.dataBuffer().use();
        }
    }

    static class WriteCompletionHandler {
        record Attachment(ByteBuffer byteBuffer, DataBuffer dataBuffer, ByteBufferIterator iterator) {
        }

        void hookOnNext(DataBuffer dataBuffer) {
            ByteBufferIterator iterator = new ByteBufferIterator();
            ByteBuffer byteBuffer = new ByteBuffer();
            Attachment attachment = new Attachment(byteBuffer, dataBuffer, iterator);
            attachment.byteBuffer().flip();
            attachment.dataBuffer().use();
        }

        void completed(Attachment attachment) {
            attachment.iterator().close();
        }
    }
}
