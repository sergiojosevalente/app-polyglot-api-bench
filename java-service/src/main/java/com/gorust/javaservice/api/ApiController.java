package com.gorust.javaservice.api;

import com.gorust.javaservice.api.Dto.CpuResponse;
import com.gorust.javaservice.api.Dto.HealthResponse;
import com.gorust.javaservice.api.Dto.HelloResponse;
import com.gorust.javaservice.api.Dto.MetricsResponse;
import com.gorust.javaservice.api.Dto.StatsRequest;
import com.gorust.javaservice.api.Dto.StatsResponse;
import com.gorust.javaservice.api.Dto.UpsertItemRequest;
import com.gorust.javaservice.domain.Compute;
import com.gorust.javaservice.domain.Item;
import com.gorust.javaservice.metrics.ProcStats;
import com.gorust.javaservice.metrics.RequestCounter;
import com.gorust.javaservice.store.ItemStore;
import java.lang.management.ManagementFactory;
import java.util.ArrayList;
import java.util.LinkedHashMap;
import java.util.List;
import java.util.Map;
import org.springframework.boot.SpringBootVersion;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.PutMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RequestParam;
import org.springframework.web.bind.annotation.RestController;

/**
 * Lapisan HTTP. Padanannya: handler.go, handlers.rs, Handler.php, handlers.ts.
 * Routing deklaratif lewat anotasi; parameter method diisi Spring dari request
 * (mirip extractor axum di Rust).
 */
@RestController
public class ApiController {

    private static final String LANG = "java";
    private static final int DEFAULT_FIB_N = 25;

    private final ItemStore store;
    private final RequestCounter counter;

    // Constructor injection: Spring memberikan bean ItemStore & RequestCounter.
    public ApiController(ItemStore store, RequestCounter counter) {
        this.store = store;
        this.counter = counter;
    }

    @GetMapping("/health")
    HealthResponse health() {
        return new HealthResponse("ok", LANG);
    }

    @GetMapping("/api/hello")
    HelloResponse hello(@RequestParam(required = false) String name) {
        String n = (name == null || name.isEmpty()) ? "World" : name;
        return new HelloResponse("Hello, " + n + "!", LANG);
    }

    @GetMapping("/api/cpu")
    CpuResponse cpu(@RequestParam(required = false) Integer n) {
        int value = n == null ? DEFAULT_FIB_N : n;
        if (value < 0) {
            throw ApiException.badRequest("n must be a non-negative integer");
        }
        if (value > Compute.MAX_FIB) {
            throw ApiException.badRequest("n must be <= " + Compute.MAX_FIB);
        }
        // Virtual thread yang sibuk CPU tetap memegang carrier thread-nya (tidak di-preempt
        // seperti goroutine), tetapi kode blocking I/O otomatis melepasnya.
        return new CpuResponse(value, Compute.fib(value), LANG);
    }

    @PostMapping("/api/json")
    StatsResponse stats(@RequestBody StatsRequest req) {
        List<Double> numbers = req.numbers() == null ? List.of() : req.numbers();
        if (numbers.contains(null)) {
            throw ApiException.badRequest("invalid JSON: numbers must be an array of numbers");
        }
        return Compute.summarize(numbers)
                .map(s -> new StatsResponse(s, LANG))
                .orElseThrow(() -> ApiException.badRequest("numbers must not be empty"));
    }

    @GetMapping("/api/items/{id}")
    Item getItem(@PathVariable long id) {
        requireValidId(id);
        return store.get(id).orElseThrow(() -> ApiException.notFound("item " + id + " not found"));
    }

    @PutMapping("/api/items/{id}")
    Item putItem(@PathVariable long id, @RequestBody UpsertItemRequest req) {
        requireValidId(id);
        if (req.name() == null || req.name().isBlank()) {
            throw ApiException.badRequest("name must not be empty");
        }
        if (req.price() == null) {
            throw ApiException.badRequest("price is required");
        }
        if (req.price() < 0) {
            throw ApiException.badRequest("price must be >= 0");
        }
        return store.upsert(new Item(id, req.name(), req.price()));
    }

    @GetMapping("/metrics")
    MetricsResponse metrics() {
        ProcStats ps = ProcStats.read();
        var heap = ManagementFactory.getMemoryMXBean().getHeapMemoryUsage();

        long gcCount = 0;
        long gcTimeMs = 0;
        List<String> gcNames = new ArrayList<>();
        for (var gc : ManagementFactory.getGarbageCollectorMXBeans()) {
            gcCount += Math.max(0, gc.getCollectionCount());
            gcTimeMs += Math.max(0, gc.getCollectionTime());
            gcNames.add(gc.getName());
        }

        Map<String, Object> runtime = new LinkedHashMap<>();
        runtime.put("heap_used_bytes", heap.getUsed());
        runtime.put("heap_max_bytes", heap.getMax());
        runtime.put("gc", String.join(", ", gcNames));
        runtime.put("gc_count", gcCount);
        runtime.put("gc_time_ms", gcTimeMs);
        runtime.put("jvm_platform_threads", ManagementFactory.getThreadMXBean().getThreadCount());
        runtime.put("available_processors", Runtime.getRuntime().availableProcessors());
        runtime.put("virtual_threads", true);

        return new MetricsResponse(
                LANG,
                "Java " + Runtime.version().feature() + " (" + Runtime.version() + ") · Spring Boot " + SpringBootVersion.getVersion(),
                counter.uptimeSeconds(),
                counter.requests(),
                ps.rssBytes(),
                ps.cpuSeconds(),
                ps.threads(),
                store.count(),
                runtime);
    }

    private static void requireValidId(long id) {
        if (id < 0) {
            throw ApiException.badRequest("id must be a non-negative integer");
        }
    }
}
