package com.sample;

public class SimpleJava {
    public void process(int val) {
        if (val > 10) { // +1 branch, +1 condition
            System.out.println("High");
        } else {
            System.out.println("Low");
        }
    }
}
