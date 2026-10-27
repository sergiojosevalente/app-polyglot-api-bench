package com.gorust.javaservice.domain;

/** Record (Java 16+): class data immutable; constructor, getter, equals, hashCode dibuat otomatis. */
public record Stats(int count, double sum, double min, double max, double mean) {
}
