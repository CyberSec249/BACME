package org.fisco.bcos.sdk.demo.perf;

import com.google.common.util.concurrent.RateLimiter;
import java.math.BigInteger;
import java.net.URL;
import java.util.Random;
import java.util.concurrent.atomic.AtomicInteger;
import org.fisco.bcos.sdk.BcosSDK;
import org.fisco.bcos.sdk.BcosSDKException;
import org.fisco.bcos.sdk.client.Client;
import org.fisco.bcos.sdk.crypto.keypair.CryptoKeyPair;
import org.fisco.bcos.sdk.demo.contract.DpkiAcme;
import org.fisco.bcos.sdk.demo.perf.callback.PerformanceCallback;
import org.fisco.bcos.sdk.demo.perf.collector.PerformanceCollector;
import org.fisco.bcos.sdk.model.ConstantConfig;
import org.fisco.bcos.sdk.model.TransactionReceipt;
import org.fisco.bcos.sdk.utils.ThreadPoolService;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

public class PerformanceDpkiAcme {
    // Logger
    private static Logger logger = LoggerFactory.getLogger(PerformanceDpkiAcme.class);
    // Atomic counter of sent transactions
    private static AtomicInteger sendedTransactions = new AtomicInteger(0);

    // Print usage
    private static void Usage() {
        System.out.println(" Usage:");
        System.out.println("===== PerformanceDpkiAcme test===========");
        System.out.println(
                " \t java -cp 'conf/:lib/*:apps/*' org.fisco.bcos.sdk.demo.perf.PerformanceDpkiAcme [operation] [count] [tps] [groupId].");
    }

    // Main function
    public static void main(String[] args) {
        try {
            // Configuration file name
            String configFileName = ConstantConfig.CONFIG_FILE_NAME;
            // Get the URL of the configuration file
            URL configUrl = PerformanceDpkiAcme.class.getClassLoader().getResource(configFileName);
            if (configUrl == null) {
                System.out.println("The configFile " + configFileName + " doesn't exist!");
                return;
            }
            // Check the number of arguments
            if (args.length < 3) {
                Usage();
                return;
            }
            // Parse command-line arguments
            String operation = args[0];
            Integer count = Integer.valueOf(args[1]); // total number of transactions
            Integer qps = Integer.valueOf(args[2]); // transactions per second (QPS)
            Integer groupId = Integer.valueOf(args[3]); // group ID
            System.out.println(
                    "====== PerformanceDpkiAcme, count: "
                            + count
                            + ", qps:"
                            + qps
                            + ", groupId: "
                            + groupId);

            // Get the configuration file path
            String configFile = configUrl.getPath();
            // Initialize BcosSDK
            BcosSDK sdk = BcosSDK.build(configFile);

            // Get the client instance
            Client client = sdk.getClient(groupId);
            // Get the key pair
            CryptoKeyPair cryptoKeyPair = client.getCryptoSuite().getCryptoKeyPair();

            // Deploy the DpkiAcme contract
            System.out.println("====== Deploy DpkiAcme ====== ");
            // [Modified] the contract address can be set via the environment variable DPKIACME_ADDR; the original default is used otherwise
            String contractAddress =
                    System.getenv()
                            .getOrDefault(
                                    "DPKIACME_ADDR", "0x55fe377410e00423a097e17c2e10b71bc4276aa0");
            DpkiAcme dpkiAcme = DpkiAcme.load(contractAddress, client, cryptoKeyPair);
            System.out.println(
                    "====== Deploy DpkiAcme success, address: " + contractAddress + " ====== ");
            // Initialize the performance collector
            PerformanceCollector collector = new PerformanceCollector();
            collector.setTotal(count); // set the total number of transactions
            // Create a rate limiter to control the QPS
            RateLimiter limiter = RateLimiter.create(qps);
            Integer area = count / 10; // print progress every 10%
            final Integer total = count;

            System.out.println("====== PerformanceDpkiAcme start ======");
            // Create the thread pool service
            ThreadPoolService threadPoolService =
                    new ThreadPoolService(
                            "PerformanceDpkiAcme",
                            sdk.getConfig().getThreadPoolConfig().getMaxBlockingQueueSize());
            // Send transactions in a loop
            for (Integer i = 0; i < count; ++i) {
                limiter.acquire(); // rate limiting, controls the sending rate
                threadPoolService
                        .getThreadPool()
                        .execute(
                                new Runnable() {
                                    @Override
                                    public void run() {
                                        // Invoke the contract method
                                        executeOperation(operation, dpkiAcme, collector);
                                        // Update the number of sent transactions
                                        int current = sendedTransactions.incrementAndGet();
                                        // Print progress every 10%
                                        if (current >= area && ((current % area) == 0)) {
                                            System.out.println(
                                                    "Already sent: "
                                                            + current
                                                            + "/"
                                                            + total
                                                            + " transactions");
                                        }
                                    }
                                });
            }
            // Wait for all transactions to complete
            while (!collector.getReceived().equals(count)) {
                Thread.sleep(1000);
            }
            // Stop the thread pool
            threadPoolService.stop();
            System.exit(0);
        } catch (BcosSDKException | InterruptedException e) {
            // Catch exceptions and print the error
            System.out.println(
                    "====== PerformanceDpkiAcme test failed, error message: " + e.getMessage());
            System.exit(0);
        }
    }

    private static void executeOperation(
            String operation, DpkiAcme dpkiAcme, PerformanceCollector collector) {
        if (operation.compareToIgnoreCase("validation") == 0) {
            controlValidtion(dpkiAcme, collector);
        } else if (operation.compareToIgnoreCase("apply") == 0) {
            apply(dpkiAcme, collector);
        } else if (operation.compareToIgnoreCase("update") == 0) {
            update(dpkiAcme, collector);
        } else if (operation.compareToIgnoreCase("revoke") == 0) {
            revoke(dpkiAcme, collector);
        } else if (operation.compareToIgnoreCase("revokeDV") == 0) {
            revokeDV(dpkiAcme, collector);
        } else if (operation.compareToIgnoreCase("revokeP") == 0) {
            revokeP(dpkiAcme, collector);
        } else if (operation.compareToIgnoreCase("verify") == 0) {
            verify(dpkiAcme, collector);
        }
    }

    private static void controlValidtion(DpkiAcme dpkiAcme, PerformanceCollector collector) {
        // Create the callback
        PerformanceCallback callback = createCallback(collector);
        try {
            dpkiAcme.controlValidate("www.t0004.com", callback);
        } catch (Exception e) {
            // Handle exceptions
            sendTransactionException(e, "controlValidation", callback);
        }
    }

    private static void apply(DpkiAcme dpkiAcme, PerformanceCollector collector) {
        // Create the callback
        PerformanceCallback callback = createCallback(collector);
        try {
            dpkiAcme.applyCertificate(
                    "www.t0004.com",
                    "bafkreihaqhsp26xr4cn2sexf3hp5sobpmj7pvm2kpxyhozcjg7badyygzy",
                    "0x53239ed882e21afed028ea85329bc5a97ff90f6b",
                    callback);
        } catch (Exception e) {
            // Handle exceptions
            sendTransactionException(e, "applyCertificate", callback);
        }
    }

    private static void update(DpkiAcme dpkiAcme, PerformanceCollector collector) {
        PerformanceCallback callback = createCallback(collector);
        try {
            dpkiAcme.updateCertificate(
                    "bafkreihaqhsp26xr4cn2sexf3hp5sobpmj7pvm2kpxyhozcjg7badyygzy",
                    "bafkreihaqhsp26xr4cn2sexf3hp5sobpmj7pvm2kpxyhozcjg7badyygzy",
                    "0x53239ed882e21afed028ea85329bc5a97ff90f6b",
                    callback);
        } catch (Exception e) {
            sendTransactionException(e, "updateCertificate", callback);
        }
    }

    private static void revoke(DpkiAcme dpkiAcme, PerformanceCollector collector) {
        PerformanceCallback callback = createCallback(collector);
        try {
            dpkiAcme.revokeCertificate(
                    "bafkreihaqhsp26xr4cn2sexf3hp5sobpmj7pvm2kpxyhozcjg7badyygzy", callback);
        } catch (Exception e) {
            sendTransactionException(e, "revokeCertificate", callback);
        }
    }

    private static void revokeDV(DpkiAcme dpkiAcme, PerformanceCollector collector) {
        PerformanceCallback callback = createCallback(collector);
        try {
            dpkiAcme.revokeCertificateWithDV(
                    "bafkreihaqhsp26xr4cn2sexf3hp5sobpmj7pvm2kpxyhozcjg7badyygzy", callback);
        } catch (Exception e) {
            sendTransactionException(e, "revokeCertificateWithDV", callback);
        }
    }

    private static void revokeP(DpkiAcme dpkiAcme, PerformanceCollector collector) {
        PerformanceCallback callback = createCallback(collector);
        try {
            dpkiAcme.revokeCertificateWithPrivateKey(
                    "bafkreifwhxbz4bhspaumbjkcxkj4lcbwscmks2tafqgcotibmi4edowv3m",
                    BigInteger.valueOf(27),
                    "0x49b82ffa9999e78dbb8b777050688b545497694339618849f34764e1e573865f".getBytes(),
                    "0xff4cfe5c57655de4c331bf07a5f828704da23b997621d7e8441951a2aa8e651".getBytes(),
                    callback);
        } catch (Exception e) {
            sendTransactionException(e, "revokeCertificateWithPrivatekey", callback);
        }
    }

    private static void verify(DpkiAcme dpkiAcme, PerformanceCollector collector) {
        PerformanceCallback callback = createCallback(collector);
        try {
            Long time_before = System.currentTimeMillis();
            Random r = new Random();
            long l1 = r.nextLong();
            dpkiAcme.verifyCertificate(
                    "bafkreihaqhsp26xr4cn2sexf3hp5sobpmj7pvm2kpxyhozcjg7badyygzy");
            Long time_after = System.currentTimeMillis();
            TransactionReceipt receipt = new TransactionReceipt();
            receipt.setStatus("0x0");
            collector.onMessage(receipt, time_after - time_before);
        } catch (Exception e) {
            sendTransactionException(e, "verifyCertificate", callback);
        }
    }

    // Create the performance callback
    private static PerformanceCallback createCallback(PerformanceCollector collector) {
        PerformanceCallback callback = new PerformanceCallback();
        callback.setTimeout(0); // set the timeout
        callback.setCollector(collector); // set the performance collector
        return callback;
    }

    // Handle transaction exceptions
    private static void sendTransactionException(
            Throwable e, String command, PerformanceCallback callback) {
        TransactionReceipt receipt = new TransactionReceipt();
        receipt.setStatus("-1"); // mark the transaction as failed
        callback.onResponse(receipt); // invoke the callback
        logger.info("call command {} failed, error info: {}", command, e.getMessage()); // log it
    }
}
