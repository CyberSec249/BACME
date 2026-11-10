package org.fisco.bcos.sdk.demo.perf;

import com.google.common.util.concurrent.RateLimiter;
import java.math.BigInteger;
import java.net.URL;
import java.security.SecureRandom;
import java.util.ArrayList;
import java.util.List;
import java.util.concurrent.CountDownLatch;
import java.util.concurrent.TimeUnit;
import java.util.concurrent.atomic.AtomicInteger;
import java.util.stream.Collectors;
import java.util.stream.IntStream;
import org.fisco.bcos.sdk.BcosSDK;
import org.fisco.bcos.sdk.client.Client;
import org.fisco.bcos.sdk.crypto.keypair.CryptoKeyPair;
import org.fisco.bcos.sdk.demo.contract.DACME;
import org.fisco.bcos.sdk.demo.contract.DCV;
import org.fisco.bcos.sdk.demo.contract.DpkiAcme;
import org.fisco.bcos.sdk.demo.perf.callback.PerformanceCallback;
import org.fisco.bcos.sdk.demo.perf.collector.PerformanceCollector;
import org.fisco.bcos.sdk.model.ConstantConfig;
import org.fisco.bcos.sdk.model.TransactionReceipt;
import org.fisco.bcos.sdk.model.callback.TransactionCallback;
import org.fisco.bcos.sdk.utils.ThreadPoolService;
import java.util.Arrays;
import java.util.Collections;
import org.fisco.bcos.sdk.abi.FunctionEncoder;
import org.fisco.bcos.sdk.abi.TypeReference;
import org.fisco.bcos.sdk.abi.datatypes.Function;
import org.fisco.bcos.sdk.abi.datatypes.Type;
import org.fisco.bcos.sdk.abi.datatypes.Utf8String;
import org.fisco.bcos.sdk.abi.datatypes.generated.Uint256;
import org.fisco.bcos.sdk.transaction.manager.TransactionProcessor;
import org.fisco.bcos.sdk.transaction.manager.TransactionProcessorFactory;

/**
 * Benchmark of the on-chain stages of domain control validation (DCV): BACME and the two baselines.
 *
 * <p>DCV consists of two on-chain transactions (the VRF / HTTP callbacks in between are performed by the oracle; all three schemes use the same oracle, so they are not measured):
 *
 * <pre>
 *   Scheme      Stage 1 (randomness request)        Stage 2 (start of HTTP validation)
 *   BACME       controlValidate(ids)                startValidation(token2)
 *   Baseline 1  DACME.RegisterDomain(domain, pk)    DACME.StartValidation(domain, N2)
 *   Baseline 2  DCV.registerDomain(domain,pk,ip,t)  DCV.StartValidation(domain, N2)
 * </pre>
 *
 * <p>Operations: bacme_v1 / bacme_v2 / dacme_v1 / dacme_v2 / dcv_v1 / dcv_v2 (_v1 = stage 1, _v2 = stage 2).
 *
 * <ul>
 *   <li>Each transaction uses a newly generated account and a unique domain: the BACME contract allows one validation per account every 10 minutes,
 *       and the baseline contracts require the domain to be unregistered, so every transaction is actually executed rather than rejected.
 *   <li>Stage 2 requires stage 1 to be completed: the program first sends a stage-1 transaction for each account (not recorded),
 *       and once all are confirmed, it sends and measures stage-2 transactions only for accounts whose stage 1 succeeded.
 *   <li>The number of domains per BACME validation is set by the environment variable DV_DOMAINS (default 1); the baselines support one domain per certificate.
 * </ul>
 *
 * <p>Usage:
 *
 * <pre>
 *   java -cp 'conf/:lib/*:apps/*' org.fisco.bcos.sdk.demo.perf.PerformanceDomainValidation [op] [count] [qps] [groupId]
 *   java -cp 'conf/:lib/*:apps/*' org.fisco.bcos.sdk.demo.perf.PerformanceDomainValidation deploy [vrfCore] [keyHash] [oracleCore] [groupId]
 * </pre>
 *
 * <p>Environment variables: DPKIACME_ADDR, DACME_ADDR, DCV_ADDR (contract addresses), DV_DOMAINS (number of BACME domains).
 */
public class PerformanceDomainValidation {
    private static final AtomicInteger sent = new AtomicInteger(0);
    private static final SecureRandom RANDOM = new SecureRandom();
    // Shared globally to avoid XXX.load() per transaction: the SDK 2.x Contract constructor starts an
    // EventSubscribe thread per instance, and 30,000 loads exhaust native threads (OutOfMemoryError: unable to create new native thread)
    private static TransactionProcessor TX_PROCESSOR;
    private static FunctionEncoder ENCODER;

    private static void usage() {
        System.out.println(" Usage:");
        System.out.println(
                "  java -cp 'conf/:lib/*:apps/*' org.fisco.bcos.sdk.demo.perf.PerformanceDomainValidation"
                        + " [bacme_v1|bacme_v2|dacme_v1|dacme_v2|dcv_v1|dcv_v2] [count] [qps] [groupId]");
        System.out.println(
                "  java -cp 'conf/:lib/*:apps/*' org.fisco.bcos.sdk.demo.perf.PerformanceDomainValidation"
                        + " deploy [vrfCoreAddress] [keyHash(hex)] [oracleCoreAddress] [groupId]");
    }

    public static void main(String[] args) {
        try {
            URL configUrl =
                    PerformanceDomainValidation.class
                            .getClassLoader()
                            .getResource(ConstantConfig.CONFIG_FILE_NAME);
            if (configUrl == null) {
                System.out.println("The configFile " + ConstantConfig.CONFIG_FILE_NAME + " doesn't exist!");
                return;
            }
            if (args.length < 4) {
                usage();
                return;
            }
            if ("deploy".equalsIgnoreCase(args[0])) {
                int groupId = args.length > 4 ? Integer.parseInt(args[4]) : 1;
                deployBaselines(configUrl.getPath(), args[1], args[2], args[3], groupId);
                return;
            }

            String op = args[0].toLowerCase();
            int count = Integer.parseInt(args[1]);
            int qps = Integer.parseInt(args[2]);
            int groupId = Integer.parseInt(args[3]);
            String scheme = op.substring(0, op.lastIndexOf('_'));
            String stage = op.substring(op.lastIndexOf('_') + 1);
            if (!(scheme.equals("bacme") || scheme.equals("dacme") || scheme.equals("dcv"))
                    || !(stage.equals("v1") || stage.equals("v2"))) {
                usage();
                return;
            }
            int domains = Integer.parseInt(System.getenv().getOrDefault("DV_DOMAINS", "1"));
            if (!scheme.equals("bacme")) {
                domains = 1; // the baselines only support single-domain certificates
            }
            String address = contractAddress(scheme);
            System.out.println(
                    "====== PerformanceDomainValidation, op: " + op + ", count: " + count + ", qps: "
                            + qps + ", groupId: " + groupId + ", domains: " + domains
                            + ", contract: " + address);

            BcosSDK sdk = BcosSDK.build(configUrl.getPath());
            Client client = sdk.getClient(groupId);
            TX_PROCESSOR = TransactionProcessorFactory.createTransactionProcessor(
                    client, client.getCryptoSuite().getCryptoKeyPair());
            ENCODER = new FunctionEncoder(client.getCryptoSuite());
            ThreadPoolService pool =
                    new ThreadPoolService(
                            "PerformanceDomainValidation",
                            sdk.getConfig().getThreadPoolConfig().getMaxBlockingQueueSize());

            // 1. Generate accounts and unique domains (unique within a run; runs are distinguished by runId)
            long t0 = System.currentTimeMillis();
            String runId = Long.toString(System.currentTimeMillis(), 36) + Integer.toString(RANDOM.nextInt(1296), 36);
            // Generate key pairs in parallel; use keyPairFactory directly to avoid the concurrency issue of createKeyPair() overwriting the global default account
            final CryptoKeyPair factory = client.getCryptoSuite().getKeyPairFactory();
            List<CryptoKeyPair> accounts =
                    IntStream.range(0, count)
                            .parallel()
                            .mapToObj(k -> factory.generateKeyPair())
                            .collect(Collectors.toList());
            List<String[]> domainLists = new ArrayList<>(count);
            for (int i = 0; i < count; i++) {
                String[] ds = new String[domains];
                for (int j = 0; j < domains; j++) {
                    ds[j] = "d" + j + "-" + i + "-" + runId + ".bacme.test";
                }
                domainLists.add(ds);
            }
            System.out.println(
                    "====== Generated " + count + " accounts in " + (System.currentTimeMillis() - t0) + " ms");

            List<Integer> targets = new ArrayList<>(count);
            if (stage.equals("v2")) {
                // 2. Prerequisite of stage 2: complete stage 1 first (not recorded)
                boolean[] ok = runSetup(scheme, address, client, accounts, domainLists, qps, pool);
                for (int i = 0; i < count; i++) {
                    if (ok[i]) {
                        targets.add(i);
                    }
                }
                System.out.println(
                        "====== Setup (stage 1) finished: " + targets.size() + "/" + count + " succeeded");
                if (targets.isEmpty()) {
                    System.out.println("====== No account passed stage 1, abort.");
                    pool.stop();
                    System.exit(1);
                }
            } else {
                for (int i = 0; i < count; i++) {
                    targets.add(i);
                }
            }

            // 3. Measured benchmark
            int total = targets.size();
            PerformanceCollector collector = new PerformanceCollector();
            collector.setTotal(total);
            RateLimiter limiter = RateLimiter.create(qps);
            int area = Math.max(total / 10, 1);
            System.out.println("====== PerformanceDomainValidation start (" + op + ", " + total + " tx) ======");
            for (int idx : targets) {
                limiter.acquire();
                final int i = idx;
                pool.getThreadPool()
                        .execute(
                                () -> {
                                    PerformanceCallback callback = new PerformanceCallback();
                                    callback.setTimeout(0);
                                    callback.setCollector(collector);
                                    try {
                                        sendStage(scheme, stage, address, client, accounts.get(i),
                                                domainLists.get(i), callback);
                                    } catch (Exception e) {
                                        TransactionReceipt receipt = new TransactionReceipt();
                                        receipt.setStatus("-1");
                                        callback.onResponse(receipt);
                                    }
                                    int current = sent.incrementAndGet();
                                    if (current % area == 0) {
                                        System.out.println("Already sent: " + current + "/" + total + " transactions");
                                    }
                                });
            }
            while (!collector.getReceived().equals(total)) {
                Thread.sleep(500);
            }
            pool.stop();
            System.exit(0);
        } catch (Exception e) {
            System.out.println("====== PerformanceDomainValidation failed: " + e.getMessage());
            e.printStackTrace();
            System.exit(1);
        }
    }

    private static String contractAddress(String scheme) {
        String env = scheme.equals("bacme") ? "DPKIACME_ADDR" : scheme.equals("dacme") ? "DACME_ADDR" : "DCV_ADDR";
        String addr = System.getenv(env);
        if (addr == null || addr.trim().isEmpty()) {
            throw new IllegalArgumentException("environment variable " + env + " is not set");
        }
        return addr.trim();
    }

    /** Send a transaction of the given stage (asynchronously; the result is returned via the callback). Signed with each account's own private key; ABI encoding matches the contract wrapper. */
    private static void sendStage(
            String scheme, String stage, String address, Client client, CryptoKeyPair account,
            String[] domains, TransactionCallback callback) {
        BigInteger random256 = new BigInteger(256, RANDOM);
        String name;
        List<Type> args;
        if (scheme.equals("bacme")) {
            if (stage.equals("v1")) {
                name = "controlValidate";
                args = Arrays.<Type>asList(new Utf8String(String.join(",", domains)));
            } else {
                name = "startValidation";
                args = Arrays.<Type>asList(new Uint256(random256));
            }
        } else if (scheme.equals("dacme")) {
            if (stage.equals("v1")) {
                name = "RegisterDomain";
                args = Arrays.<Type>asList(new Utf8String(domains[0]), new Utf8String("pk-" + domains[0]));
            } else {
                name = "StartValidation";
                args = Arrays.<Type>asList(new Utf8String(domains[0]), new Uint256(random256));
            }
        } else {
            if (stage.equals("v1")) {
                BigInteger expiry = BigInteger.valueOf(System.currentTimeMillis() / 1000 + 365L * 24 * 3600);
                name = "registerDomain";
                args = Arrays.<Type>asList(new Utf8String(domains[0]), new Utf8String("pk-" + domains[0]),
                        new Utf8String("127.0.0.1"), new Uint256(expiry));
            } else {
                name = "StartValidation";
                args = Arrays.<Type>asList(new Utf8String(domains[0]), new Uint256(random256));
            }
        }
        Function function = new Function(name, args, Collections.<TypeReference<?>>emptyList());
        TX_PROCESSOR.sendTransactionAsync(address, ENCODER.encode(function), account, callback);
    }

    /** Run stage 1 for all accounts and return whether each account succeeded. */
    private static boolean[] runSetup(
            String scheme, String address, Client client, List<CryptoKeyPair> accounts,
            List<String[]> domainLists, int qps, ThreadPoolService pool) throws InterruptedException {
        int n = accounts.size();
        boolean[] ok = new boolean[n];
        CountDownLatch latch = new CountDownLatch(n);
        AtomicInteger succeeded = new AtomicInteger(0);
        RateLimiter limiter = RateLimiter.create(qps);
        System.out.println("====== Setup: sending stage-1 transactions for " + n + " accounts (not measured)");
        for (int idx = 0; idx < n; idx++) {
            limiter.acquire();
            final int i = idx;
            pool.getThreadPool()
                    .execute(
                            () -> {
                                TransactionCallback cb =
                                        new TransactionCallback() {
                                            @Override
                                            public void onResponse(TransactionReceipt receipt) {
                                                if (receipt.isStatusOK()) {
                                                    ok[i] = true;
                                                    succeeded.incrementAndGet();
                                                }
                                                latch.countDown();
                                            }
                                        };
                                try {
                                    sendStage(scheme, "v1", address, client, accounts.get(i), domainLists.get(i), cb);
                                } catch (Exception e) {
                                    latch.countDown();
                                }
                            });
        }
        if (!latch.await(30, TimeUnit.MINUTES)) {
            System.out.println("====== Setup timeout, " + latch.getCount() + " stage-1 receipts missing");
        }
        // Leave some margin for the last receipts to be recorded
        Thread.sleep(1000);
        return ok;
    }

    /** Deploy the two baseline contracts on the current chain and print their addresses. */
    private static void deployBaselines(
            String configFile, String vrfCore, String keyHashHex, String oracleCore, int groupId)
            throws Exception {
        String hex = keyHashHex.startsWith("0x") ? keyHashHex.substring(2) : keyHashHex;
        if (hex.length() != 64) {
            throw new IllegalArgumentException("keyHash must be 32 bytes (64 hex chars)");
        }
        byte[] keyHash = new byte[32];
        for (int i = 0; i < 32; i++) {
            keyHash[i] = (byte) Integer.parseInt(hex.substring(2 * i, 2 * i + 2), 16);
        }
        BcosSDK sdk = BcosSDK.build(configFile);
        Client client = sdk.getClient(groupId);
        CryptoKeyPair kp = client.getCryptoSuite().getCryptoKeyPair();
        DACME dacme = DACME.deploy(client, kp, vrfCore, keyHash, oracleCore);
        System.out.println("DACME_ADDR=" + dacme.getContractAddress());
        DCV dcv = DCV.deploy(client, kp, vrfCore, keyHash, oracleCore);
        System.out.println("DCV_ADDR=" + dcv.getContractAddress());
        System.exit(0);
    }
}
