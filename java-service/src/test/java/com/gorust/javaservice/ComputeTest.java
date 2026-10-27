package com.gorust.javaservice;

import static org.assertj.core.api.Assertions.assertThat;

import com.gorust.javaservice.domain.Compute;
import com.gorust.javaservice.domain.Item;
import com.gorust.javaservice.domain.Stats;
import com.gorust.javaservice.store.ItemStore;
import java.util.List;
import java.util.concurrent.Executors;
import java.util.stream.LongStream;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.params.ParameterizedTest;
import org.junit.jupiter.params.provider.CsvSource;

class ComputeTest {

    // Parameterized test = padanan table-driven test di Go.
    @ParameterizedTest
    @CsvSource({"0,0", "1,1", "2,1", "10,55", "25,75025"})
    void fib(int n, long want) {
        assertThat(Compute.fib(n)).isEqualTo(want);
    }

    @Test
    void summarize() {
        assertThat(Compute.summarize(List.of(1.0, 2.0, 3.0, 4.5, -2.0)))
                .contains(new Stats(5, 8.5, -2.0, 4.5, 1.7));
    }

    @Test
    void summarizeEmpty() {
        assertThat(Compute.summarize(List.of())).isEmpty();
    }

    @Test
    void storeSeedAndGet() {
        ItemStore s = new ItemStore();
        assertThat(s.get(7)).contains(new Item(7, "item-7", 10.5));
        assertThat(s.get(999_999)).isEmpty();
    }

    @Test
    void storeConcurrentUpsertWithVirtualThreads() {
        ItemStore s = new ItemStore();
        // try-with-resources menunggu semua task selesai (mirip WaitGroup).
        try (var executor = Executors.newVirtualThreadPerTaskExecutor()) {
            LongStream.rangeClosed(1001, 1100).forEach(id ->
                    executor.submit(() -> s.upsert(new Item(id, "x", 1))));
        }
        assertThat(s.count()).isEqualTo(1100);
    }
}
