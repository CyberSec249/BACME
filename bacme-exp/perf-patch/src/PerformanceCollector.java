/**
 * Copyright 2014-2020 [fisco-dev]
 *
 * <p>Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
 * except in compliance with the License. You may obtain a copy of the License at
 *
 * <p>http://www.apache.org/licenses/LICENSE-2.0
 *
 * <p>Unless required by applicable law or agreed to in writing, software distributed under the
 * License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either
 * express or implied. See the License for the specific language governing permissions and
 * limitations under the License.
 */
package org.fisco.bcos.sdk.demo.perf.collector;

import java.util.ArrayList;
import java.util.Collections;
import java.util.List;
import java.util.concurrent.ConcurrentLinkedQueue;
import java.util.concurrent.atomic.AtomicInteger;
import java.util.concurrent.atomic.AtomicLong;
import org.fisco.bcos.sdk.model.JsonRpcResponse;
import org.fisco.bcos.sdk.model.TransactionReceipt;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

public class PerformanceCollector {
    private static Logger logger = LoggerFactory.getLogger(PerformanceCollector.class);
    private AtomicLong less50 = new AtomicLong(0);
    private AtomicLong less100 = new AtomicLong(0);
    private AtomicLong less200 = new AtomicLong(0);
    private AtomicLong less400 = new AtomicLong(0);
    private AtomicLong less1000 = new AtomicLong(0);
    private AtomicLong less2000 = new AtomicLong(0);
    private AtomicLong timeout2000 = new AtomicLong(0);
    private AtomicLong totalCost = new AtomicLong(0);

    private Integer total = 0;
    private AtomicInteger received = new AtomicInteger(0);
    private AtomicInteger error = new AtomicInteger(0);
    private Long startTimestamp = System.currentTimeMillis();
    // [Added] record the latency (ms) of every successful request, used to compute p50/p95/p99
    private ConcurrentLinkedQueue<Long> successCosts = new ConcurrentLinkedQueue<>();
    // [Added] number of requests accounted for. The `received` counter the main thread waits on counts the last request only after the results are printed,
    // so that the main thread does not call System.exit before the results are printed
    private AtomicInteger counted = new AtomicInteger(0);

    public Integer getTotal() {
        return total;
    }

    public void setTotal(Integer total) {
        this.total = total;
    }

    public Integer getReceived() {
        return received.get();
    }

    public void setReceived(Integer received) {
        this.received.getAndSet(received);
    }

    public void onRpcMessage(JsonRpcResponse response, Long cost) {
        try {
            boolean errorMessage = false;
            if (response.getError() != null && response.getError().getCode() != 0) {
                logger.warn("receive error jsonRpcResponse: {}", response.toString());
                errorMessage = true;
            }
            stat(errorMessage, cost);
        } catch (Exception e) {
            logger.error("onRpcMessage exception: {}", e.getMessage());
        }
    }

    public void onMessage(TransactionReceipt receipt, Long cost) {
        try {
            boolean errorMessage = false;
            if (!receipt.isStatusOK()) {
                logger.error(
                        "error receipt, status: {}, output: {}, message: {}",
                        receipt.getStatus(),
                        receipt.getOutput(),
                        receipt.getMessage());
                errorMessage = true;
            }
            stat(errorMessage, cost);
        } catch (Exception e) {
            logger.error("error:", e);
        }
    }

    public void stat(boolean errorMessage, Long cost) {
        if (errorMessage) {
            error.addAndGet(1);
        } else {
            successCosts.add(cost); // [Added]
        }

        if (cost < 50) {
            less50.incrementAndGet();
        } else if (cost < 100) {
            less100.incrementAndGet();
        } else if (cost < 200) {
            less200.incrementAndGet();
        } else if (cost < 400) {
            less400.incrementAndGet();
        } else if (cost < 1000) {
            less1000.incrementAndGet();
        } else if (cost < 2000) {
            less2000.incrementAndGet();
        } else {
            timeout2000.incrementAndGet();
        }

        totalCost.addAndGet(cost);

        // [Modified] count only after all statistics are updated, so that all other requests are included when the last one prints
        int current = counted.incrementAndGet(); // [Modified] originally received.get() + 1
        if (total >= 10 && current % (total / 10) == 0) {
            System.out.println(
                    "                                                       |received:"
                            + String.valueOf(current * 100 / total)
                            + "%");
        }

        if (current < total) {
            received.incrementAndGet();
            return;
        }
        // [Modified] last request: print the results first, then count it in received (see finally below)
        try {
            System.out.println("total");

            Long totalTime = System.currentTimeMillis() - startTimestamp;

            System.out.println(
                    "===================================================================");

            System.out.println("Total transactions:  " + String.valueOf(total));
            System.out.println("Total time: " + String.valueOf(totalTime) + "ms");
            System.out.println(
                    "TPS(include error requests): "
                            + String.valueOf(total / ((double) totalTime / 1000)));
            System.out.println(
                    "TPS(exclude error requests): "
                            + String.valueOf(
                                    (double) (total - error.get()) / ((double) totalTime / 1000)));
            System.out.println("Avg time cost: " + String.valueOf(totalCost.get() / total) + "ms");
            System.out.println(
                    "Error rate: "
                            + String.valueOf((error.get() / (double) total) * 100)
                            + "%");

            System.out.println("Time area:");
            System.out.println(
                    "0    < time <  50ms   : "
                            + String.valueOf(less50)
                            + "  : "
                            + String.valueOf((double) less50.get() / total * 100)
                            + "%");
            System.out.println(
                    "50   < time <  100ms  : "
                            + String.valueOf(less100)
                            + "  : "
                            + String.valueOf((double) less100.get() / total * 100)
                            + "%");
            System.out.println(
                    "100  < time <  200ms  : "
                            + String.valueOf(less200)
                            + "  : "
                            + String.valueOf((double) less200.get() / total * 100)
                            + "%");
            System.out.println(
                    "200  < time <  400ms  : "
                            + String.valueOf(less400)
                            + "  : "
                            + String.valueOf((double) less400.get() / total * 100)
                            + "%");
            System.out.println(
                    "400  < time <  1000ms : "
                            + String.valueOf(less1000)
                            + "  : "
                            + String.valueOf((double) less1000.get() / total * 100)
                            + "%");
            System.out.println(
                    "1000 < time <  2000ms : "
                            + String.valueOf(less2000)
                            + "  : "
                            + String.valueOf((double) less2000.get() / total * 100)
                            + "%");
            System.out.println(
                    "2000 < time           : "
                            + String.valueOf(timeout2000)
                            + "  : "
                            + String.valueOf((double) timeout2000.get() / total * 100)
                            + "%");
            printPercentiles(totalTime); // [Added]
        } finally {
            received.incrementAndGet();
        }
    }

    // [Added] compute and print latency percentiles of successful requests, plus a CSV line for scripts to parse
    private void printPercentiles(Long totalTime) {
        List<Long> costs = new ArrayList<>(successCosts);
        Collections.sort(costs);
        long p50 = percentile(costs, 50);
        long p95 = percentile(costs, 95);
        long p99 = percentile(costs, 99);
        long max = costs.isEmpty() ? -1 : costs.get(costs.size() - 1);
        double seconds = (double) totalTime / 1000;
        double tpsAll = total / seconds;
        double tpsOk = (total - error.get()) / seconds;
        double avg = (double) totalCost.get() / total;
        double errRate = (double) error.get() / total * 100;
        System.out.println("Latency of successful requests (ms):");
        System.out.println("  p50 = " + p50 + ", p95 = " + p95 + ", p99 = " + p99 + ", max = " + max);
        // Fields: total,ok,err,time_ms,tps_all(incl. failures),tps_ok(successful only),avg_ms,p50,p95,p99,max,err_rate%
        System.out.println(
                String.format(
                        "RESULT_CSV,%d,%d,%d,%d,%.2f,%.2f,%.2f,%d,%d,%d,%d,%.4f",
                        total,
                        total - error.get(),
                        error.get(),
                        totalTime,
                        tpsAll,
                        tpsOk,
                        avg,
                        p50,
                        p95,
                        p99,
                        max,
                        errRate));
        System.out.flush();
    }

    // Nearest-rank percentile; returns -1 if there are no successful samples
    private static long percentile(List<Long> sorted, int pct) {
        if (sorted.isEmpty()) {
            return -1;
        }
        int rank = (int) Math.ceil(pct / 100.0 * sorted.size());
        return sorted.get(Math.max(rank, 1) - 1);
    }

    public void setStartTimestamp(Long startTimestamp) {
        this.startTimestamp = startTimestamp;
    }

    public Long getStartTimestamp() {
        return startTimestamp;
    }
}
