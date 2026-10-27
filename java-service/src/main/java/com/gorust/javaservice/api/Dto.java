package com.gorust.javaservice.api;

import com.fasterxml.jackson.annotation.JsonUnwrapped;
import com.gorust.javaservice.domain.Stats;
import java.util.List;
import java.util.Map;

/** Request/response DTO sebagai record. Jackson memetakan JSON <-> record otomatis. */
public final class Dto {

    private Dto() {
    }

    public record HealthResponse(String status, String lang) {
    }

    public record HelloResponse(String message, String lang) {
    }

    public record CpuResponse(int n, long result, String lang) {
    }

    public record StatsRequest(List<Double> numbers) {
    }

    /** @JsonUnwrapped = padanan struct embedding (Go) / #[serde(flatten)] (Rust). */
    public record StatsResponse(@JsonUnwrapped Stats stats, String lang) {
    }

    /** Wrapper Double (bukan double) supaya field yang hilang terbaca null, bukan 0. */
    public record UpsertItemRequest(String name, Double price) {
    }

    public record MetricsResponse(
            String lang,
            String version,
            double uptimeSec,
            long requestsTotal,
            long rssBytes,
            double cpuSeconds,
            int threads,
            int items,
            Map<String, Object> runtime) {
    }
}
