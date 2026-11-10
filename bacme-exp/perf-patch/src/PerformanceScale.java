package org.fisco.bcos.sdk.demo.perf;

import com.google.common.util.concurrent.RateLimiter;
import java.io.File;
import java.math.BigInteger;
import java.net.URL;
import java.nio.charset.StandardCharsets;
import java.nio.file.Files;
import java.security.SecureRandom;
import java.util.Arrays;
import java.util.Collections;
import java.util.concurrent.CountDownLatch;
import java.util.concurrent.Semaphore;
import java.util.concurrent.TimeUnit;
import java.util.concurrent.atomic.AtomicInteger;
import org.fisco.bcos.sdk.BcosSDK;
import org.fisco.bcos.sdk.abi.FunctionEncoder;
import org.fisco.bcos.sdk.abi.TypeReference;
import org.fisco.bcos.sdk.abi.datatypes.Address;
import org.fisco.bcos.sdk.abi.datatypes.Function;
import org.fisco.bcos.sdk.abi.datatypes.Type;
import org.fisco.bcos.sdk.abi.datatypes.Utf8String;
import org.fisco.bcos.sdk.abi.datatypes.generated.tuples.generated.Tuple3;
import org.fisco.bcos.sdk.client.Client;
import org.fisco.bcos.sdk.crypto.keypair.CryptoKeyPair;
import org.fisco.bcos.sdk.demo.contract.DpkiAcme;
import org.fisco.bcos.sdk.demo.perf.collector.PerformanceCollector;
import org.fisco.bcos.sdk.model.ConstantConfig;
import org.fisco.bcos.sdk.model.TransactionReceipt;
import org.fisco.bcos.sdk.model.callback.TransactionCallback;
import org.fisco.bcos.sdk.transaction.manager.TransactionProcessor;
import org.fisco.bcos.sdk.transaction.manager.TransactionProcessorFactory;
import org.fisco.bcos.sdk.utils.ThreadPoolService;

/**
 * Large certificate state experiment (performance with millions of certificate records already stored on chain).
 *
 * <p>Unlike PerformanceDpkiAcme, write operations here take the complete execution path:
 * a fixed account that has passed real domain control validation (ACCOUNT_PEM) is used, and each transaction carries a distinct certificate hash,
 * so applyCertificate / updateCertificate / revokeCertificate in the contract actually write the certificate metadata table;
 * transactions for which the contract returns false (authorization check failed) are counted as errors and excluded from the successful throughput.
 *
 * <pre>
 * Usage: java -cp 'conf/:lib/*:apps/*' org.fisco.bcos.sdk.demo.perf.PerformanceScale [op] [count] [qps] [groupId]
 *   op:
 *     scale_whoami   print the account address of ACCOUNT_PEM (count/qps are ignored)
 *     scale_probe    apply for one certificate and query it, to check whether the account holds the control right of SCALE_DOMAIN
 *     scale_seed     pre-populate count certificate records (not recorded), numbered after the progress in the state file
 *     scale_apply    [timed] apply for count new certificates
 *     scale_update   [timed] prepare count certificates (untimed), then renew each of them
 *     scale_revoke   [timed] prepare count certificates (untimed), then revoke each of them
 *     scale_verify   [timed] query count randomly chosen pre-populated certificates (read-only calls; all are expected to be valid)
 * Environment variables:
 *   DPKIACME_ADDR   contract address (required)
 *   ACCOUNT_PEM     private key pem file of an account that has passed domain control validation (required)
 *   SCALE_DOMAIN    domain validated by that account (required for apply/update/revoke/probe/seed)
 *   SCALE_STATE     progress file, default ./scale_state.txt (number of pre-populated records written)
 *   SCALE_INFLIGHT  maximum in-flight transactions in the preparation/pre-population phases, default 20000 (keeps the transaction pool and memory bounded)
 * </pre>
 */
public class PerformanceScale {
    private static final SecureRandom RANDOM = new SecureRandom();
    private static final String CERT_PK_HASH = "0x53239ed882e21afed028ea85329bc5a97ff90f6b";
    private static final AtomicInteger sent = new AtomicInteger(0);

    private static TransactionProcessor TX_PROCESSOR;
    private static FunctionEncoder ENCODER;
    private static CryptoKeyPair ACCOUNT;
    private static String ADDRESS;

    public static void main(String[] args) {
        try {
            URL configUrl =
                    PerformanceScale.class.getClassLoader().getResource(ConstantConfig.CONFIG_FILE_NAME);
            if (configUrl == null || args.length < 4) {
                System.out.println(
                        "Usage: java -cp 'conf/:lib/*:apps/*' org.fisco.bcos.sdk.demo.perf.PerformanceScale"
                                + " [scale_whoami|scale_probe|scale_seed|scale_apply|scale_update|scale_revoke|scale_verify]"
                                + " [count] [qps] [groupId]");
                return;
            }
            String op = args[0].toLowerCase();
            int count = Integer.parseInt(args[1]);
            int qps = Integer.parseInt(args[2]);
            int groupId = Integer.parseInt(args[3]);

            ADDRESS = requireEnv("DPKIACME_ADDR");
            String pem = requireEnv("ACCOUNT_PEM");
            String domain = System.getenv().getOrDefault("SCALE_DOMAIN", "").trim();
            File stateFile = new File(System.getenv().getOrDefault("SCALE_STATE", "scale_state.txt"));
            int inflight = Integer.parseInt(System.getenv().getOrDefault("SCALE_INFLIGHT", "20000"));

            BcosSDK sdk = BcosSDK.build(configUrl.getPath());
            Client client = sdk.getClient(groupId);
            client.getCryptoSuite().loadAccount("pem", pem, null);
            ACCOUNT = client.getCryptoSuite().getCryptoKeyPair();
            TX_PROCESSOR = TransactionProcessorFactory.createTransactionProcessor(client, ACCOUNT);
            ENCODER = new FunctionEncoder(client.getCryptoSuite());
            long seeded = readState(stateFile);
            System.out.println("====== PerformanceScale, op: " + op + ", count: " + count + ", qps: " + qps
                    + ", account: " + ACCOUNT.getAddress() + ", domain: " + domain
                    + ", seeded certificates: " + seeded + ", contract: " + ADDRESS);

            if (op.equals("scale_whoami")) {
                System.exit(0);
            }
            if (!op.equals("scale_verify") && domain.isEmpty()) {
                throw new IllegalArgumentException("environment variable SCALE_DOMAIN is not set");
            }
            ThreadPoolService pool =
                    new ThreadPoolService("PerformanceScale",
                            sdk.getConfig().getThreadPoolConfig().getMaxBlockingQueueSize());
            String runId = Long.toString(System.currentTimeMillis(), 36);

            switch (op) {
                case "scale_probe":
                    probe(client, domain, runId);
                    break;
                case "scale_seed": {
                    long t0 = System.currentTimeMillis();
                    int[] r = runUnmeasured(pool, count, qps, inflight, i ->
                            applyData(domain, cid("s", seeded + i)));
                    // Advance the progress only if all succeeded, so that every pre-populated record in [0, seeded) really exists
                    if (r[1] == 0) {
                        writeState(stateFile, seeded + count);
                    }
                    long ms = System.currentTimeMillis() - t0;
                    System.out.println("SEED_DONE ok=" + r[0] + " err=" + r[1] + " time_ms=" + ms
                            + " tps=" + String.format("%.1f", r[0] * 1000.0 / Math.max(ms, 1))
                            + " seeded_total=" + (r[1] == 0 ? seeded + count : seeded));
                    break;
                }
                case "scale_apply":
                    runMeasured(pool, count, qps, i -> applyData(domain, cid("n" + runId + "x", i)));
                    break;
                case "scale_update": {
                    int[] r = runUnmeasured(pool, count, qps, inflight, i -> applyData(domain, cid("u" + runId + "x", i)));
                    System.out.println("====== Setup finished: ok=" + r[0] + " err=" + r[1]);
                    runMeasured(pool, count, qps, i -> encode("updateCertificate",
                            new Utf8String(cid("u" + runId + "x", i)),
                            new Utf8String(cid("v" + runId + "x", i)),
                            new Address(CERT_PK_HASH)));
                    break;
                }
                case "scale_revoke": {
                    int[] r = runUnmeasured(pool, count, qps, inflight, i -> applyData(domain, cid("r" + runId + "x", i)));
                    System.out.println("====== Setup finished: ok=" + r[0] + " err=" + r[1]);
                    runMeasured(pool, count, qps, i -> encode("revokeCertificate",
                            new Utf8String(cid("r" + runId + "x", i))));
                    break;
                }
                case "scale_verify":
                    if (seeded <= 0) {
                        throw new IllegalStateException("no seeded certificates, run scale_seed first");
                    }
                    runVerify(pool, client, count, qps, seeded);
                    break;
                default:
                    throw new IllegalArgumentException("unknown op: " + op);
            }
            pool.stop();
            System.exit(0);
        } catch (Exception e) {
            System.out.println("====== PerformanceScale failed: " + e.getMessage());
            e.printStackTrace();
            System.exit(1);
        }
    }

    // ------------------------------------------------------------------ Transaction construction

    private interface DataFn {
        String data(long i);
    }

    private static String encode(String name, Type... args) {
        Function f = new Function(name, Arrays.<Type>asList(args), Collections.<TypeReference<?>>emptyList());
        return ENCODER.encode(f);
    }

    private static String applyData(String domain, String certHash) {
        return encode("applyCertificate", new Utf8String(domain), new Utf8String(certHash), new Address(CERT_PK_HASH));
    }

    /** Generate a certificate hash with the same length as a real IPFS CIDv1 (59 characters), so storage entries have realistic size. */
    static String cid(String prefix, long i) {
        String body = prefix + Long.toString(i, 36);
        StringBuilder sb = new StringBuilder("bafkrei");
        for (int k = body.length(); k < 52; k++) {
            sb.append('a');
        }
        return sb.append(body).toString();
    }

    /** The contract functions return bool: the output is 32 bytes, and a value of 1 means the call really succeeded. */
    private static boolean returnedTrue(TransactionReceipt receipt) {
        String out = receipt.getOutput();
        if (out == null || out.length() < 3) {
            return false;
        }
        try {
            return new BigInteger(out.startsWith("0x") ? out.substring(2) : out, 16).equals(BigInteger.ONE);
        } catch (NumberFormatException e) {
            return false;
        }
    }

    // ------------------------------------------------------------------ Timed benchmark

    /** Timed: results go to PerformanceCollector, which prints RESULT_CSV (compatible with run_exp.sh). A false return from the contract counts as an error. */
    private static void runMeasured(ThreadPoolService pool, int count, int qps, DataFn fn) throws Exception {
        PerformanceCollector collector = new PerformanceCollector();
        collector.setTotal(count);
        RateLimiter limiter = RateLimiter.create(qps);
        int area = Math.max(count / 10, 1);
        sent.set(0);
        System.out.println("====== PerformanceScale start (" + count + " tx) ======");
        for (int idx = 0; idx < count; idx++) {
            limiter.acquire();
            final long i = idx;
            pool.getThreadPool().execute(() -> {
                final long start = System.currentTimeMillis();
                TransactionCallback cb = new TransactionCallback() {
                    @Override
                    public void onResponse(TransactionReceipt receipt) {
                        long cost = System.currentTimeMillis() - start;
                        if (receipt.isStatusOK() && !returnedTrue(receipt)) {
                            receipt.setStatus("0x1a"); // the contract returned false: mark as failed
                            receipt.setMessage("contract returned false");
                        }
                        collector.onMessage(receipt, cost);
                    }
                };
                cb.setTimeout(0);
                try {
                    TX_PROCESSOR.sendTransactionAsync(ADDRESS, fn.data(i), ACCOUNT, cb);
                } catch (Exception e) {
                    TransactionReceipt r = new TransactionReceipt();
                    r.setStatus("-1");
                    cb.onResponse(r);
                }
                int current = sent.incrementAndGet();
                if (current % area == 0) {
                    System.out.println("Already sent: " + current + "/" + count + " transactions");
                }
            });
        }
        while (!collector.getReceived().equals(count)) {
            Thread.sleep(200);
        }
    }

    /** Read-only queries over randomly chosen pre-populated certificates; all are expected to be valid (invalid ones count as errors). */
    private static void runVerify(ThreadPoolService pool, Client client, int count, int qps, long seeded)
            throws Exception {
        DpkiAcme dpkiAcme = DpkiAcme.load(ADDRESS, client, ACCOUNT); // load only once
        PerformanceCollector collector = new PerformanceCollector();
        collector.setTotal(count);
        RateLimiter limiter = RateLimiter.create(qps);
        int area = Math.max(count / 10, 1);
        System.out.println("====== PerformanceScale start (verify, " + count + " queries over " + seeded
                + " seeded certificates) ======");
        for (int idx = 0; idx < count; idx++) {
            limiter.acquire();
            final long target = (RANDOM.nextLong() & Long.MAX_VALUE) % seeded;
            final int n = idx + 1;
            pool.getThreadPool().execute(() -> {
                long start = System.currentTimeMillis();
                TransactionReceipt r = new TransactionReceipt();
                try {
                    Tuple3<String, String, Boolean> res = dpkiAcme.verifyCertificate(cid("s", target));
                    r.setStatus(Boolean.TRUE.equals(res.getValue3()) ? "0x0" : "0x1b");
                } catch (Exception e) {
                    r.setStatus("-1");
                }
                collector.onMessage(r, System.currentTimeMillis() - start);
                if (n % area == 0) {
                    System.out.println("Already queried: " + n + "/" + count);
                }
            });
        }
        while (!collector.getReceived().equals(count)) {
            Thread.sleep(200);
        }
    }

    // ------------------------------------------------------------------ Untimed (pre-population / preparation)

    /** Returns {ok, err}. A semaphore bounds the number of in-flight transactions to keep the transaction pool and memory bounded. */
    private static int[] runUnmeasured(ThreadPoolService pool, int count, int qps, int inflight, DataFn fn)
            throws Exception {
        CountDownLatch latch = new CountDownLatch(count);
        AtomicInteger ok = new AtomicInteger(0);
        AtomicInteger err = new AtomicInteger(0);
        AtomicInteger firstErrLogged = new AtomicInteger(0);
        Semaphore window = new Semaphore(inflight);
        RateLimiter limiter = RateLimiter.create(qps);
        int area = Math.max(count / 10, 1);
        for (int idx = 0; idx < count; idx++) {
            limiter.acquire();
            window.acquire();
            final long i = idx;
            pool.getThreadPool().execute(() -> {
                TransactionCallback cb = new TransactionCallback() {
                    @Override
                    public void onResponse(TransactionReceipt receipt) {
                        if (receipt.isStatusOK() && returnedTrue(receipt)) {
                            ok.incrementAndGet();
                        } else {
                            err.incrementAndGet();
                            if (firstErrLogged.getAndIncrement() < 3) {
                                System.out.println("  [error] status=" + receipt.getStatus() + " output="
                                        + receipt.getOutput() + " msg=" + receipt.getMessage());
                            }
                        }
                        window.release();
                        long done = latch.getCount();
                        latch.countDown();
                        if ((count - done + 1) % area == 0) {
                            System.out.println("  progress: " + (count - done + 1) + "/" + count
                                    + " (ok=" + ok.get() + ", err=" + err.get() + ")");
                        }
                    }
                };
                cb.setTimeout(0);
                try {
                    TX_PROCESSOR.sendTransactionAsync(ADDRESS, fn.data(i), ACCOUNT, cb);
                } catch (Exception e) {
                    TransactionReceipt r = new TransactionReceipt();
                    r.setStatus("-1");
                    r.setMessage(e.getMessage());
                    cb.onResponse(r);
                }
            });
        }
        if (!latch.await(2, TimeUnit.HOURS)) {
            System.out.println("  [warn] timeout waiting for receipts");
        }
        return new int[] {ok.get(), err.get()};
    }

    /** Apply for one certificate and query it, to determine whether the account holds the control right of SCALE_DOMAIN. */
    private static void probe(Client client, String domain, String runId) throws Exception {
        String hash = cid("p" + runId + "x", 0);
        TransactionReceipt receipt = TX_PROCESSOR.sendTransactionAndGetReceipt(ADDRESS, applyData(domain, hash), ACCOUNT);
        boolean applied = receipt.isStatusOK() && returnedTrue(receipt);
        Tuple3<String, String, Boolean> res = DpkiAcme.load(ADDRESS, client, ACCOUNT).verifyCertificate(hash);
        System.out.println("applyCertificate status=" + receipt.getStatus() + " returned=" + applied);
        System.out.println("verifyCertificate -> ids=" + res.getValue1() + " valid=" + res.getValue3());
        if (applied && Boolean.TRUE.equals(res.getValue3())) {
            System.out.println("PROBE_OK: account " + ACCOUNT.getAddress() + " owns " + domain);
        } else {
            System.out.println("PROBE_FAIL: account " + ACCOUNT.getAddress() + " has no valid control right for "
                    + domain + " (complete domain control validation with this account first; it is valid for 30 days)");
        }
    }

    // ------------------------------------------------------------------ Utilities

    private static String requireEnv(String name) {
        String v = System.getenv(name);
        if (v == null || v.trim().isEmpty()) {
            throw new IllegalArgumentException("environment variable " + name + " is not set");
        }
        return v.trim();
    }

    private static long readState(File f) throws Exception {
        if (!f.exists()) {
            return 0;
        }
        String s = new String(Files.readAllBytes(f.toPath()), StandardCharsets.UTF_8).trim();
        return s.isEmpty() ? 0 : Long.parseLong(s);
    }

    private static void writeState(File f, long v) throws Exception {
        Files.write(f.toPath(), (v + "\n").getBytes(StandardCharsets.UTF_8));
    }
}
