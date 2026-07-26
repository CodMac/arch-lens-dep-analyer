package com.test;

class Builder {
    public Builder() {}
    public Builder select(String sql) { return this; }
    public String build() { return "result"; }
}