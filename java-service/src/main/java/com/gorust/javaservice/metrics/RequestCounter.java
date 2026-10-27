package com.gorust.javaservice.metrics;

import jakarta.servlet.FilterChain;
import jakarta.servlet.ServletException;
import jakarta.servlet.http.HttpServletRequest;
import jakarta.servlet.http.HttpServletResponse;
import java.io.IOException;
import java.util.concurrent.atomic.LongAdder;
import org.springframework.stereotype.Component;
import org.springframework.web.filter.OncePerRequestFilter;

/**
 * Servlet filter = middleware. Padanan Metrics.Middleware (Go) / count_requests (Rust).
 * LongAdder: counter yang dipecah per-thread lalu dijumlah saat dibaca; lebih cepat
 * daripada AtomicLong di bawah kontensi tinggi.
 */
@Component
public class RequestCounter extends OncePerRequestFilter {

    private final LongAdder requests = new LongAdder();
    private final long startedNanos = System.nanoTime();

    @Override
    protected void doFilterInternal(HttpServletRequest request, HttpServletResponse response, FilterChain chain)
            throws ServletException, IOException {
        requests.increment();
        chain.doFilter(request, response);
    }

    public long requests() {
        return requests.sum();
    }

    public double uptimeSeconds() {
        return (System.nanoTime() - startedNanos) / 1e9;
    }
}
