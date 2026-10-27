package com.gorust.javaservice;

import org.springframework.boot.SpringApplication;
import org.springframework.boot.autoconfigure.SpringBootApplication;

/**
 * Entry point. Berbeda dengan Go/Rust yang merakit dependency secara manual di main(),
 * Spring memindai package ini, membuat bean (@RestController, @Repository, @Component),
 * lalu menyuntikkan dependency lewat constructor (dependency injection otomatis).
 */
@SpringBootApplication
public class JavaServiceApplication {

    public static void main(String[] args) {
        SpringApplication.run(JavaServiceApplication.class, args);
    }
}
