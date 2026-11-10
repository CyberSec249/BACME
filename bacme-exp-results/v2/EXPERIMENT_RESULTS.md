# BACME Performance Results: Network Latency x Number of Consensus Nodes

## 1. Methodology

- **Deployment**: FISCO BCOS 2.x; all consensus nodes and the benchmark client run on the same virtual machine.
- **Network delay emulation**: Linux `tc netem` injects a one-way delay D only on inter-node P2P port traffic, so the inter-node round-trip time is RTT ≈ 2D; client-to-node traffic is not delayed.
- **Number of nodes**: the number of consensus nodes N and the number of tolerated Byzantine nodes f satisfy N = 3f + 1. The number of nodes is adjusted with the console command `removeNode` (10 -> 7 -> 4).
- **Throughput experiment**: transactions are sent at a rate above the system capacity, and the TPS of successful transactions is measured.
- **Latency experiment**: transactions are sent at a fixed rate below the system capacity, and the confirmation latency (from submission to receipt) is measured as p50 / p95.
- **verify** is a read-only contract call (`view` function) executed locally on a single node without consensus, so its results are query throughput (QPS) and query latency.
- **The benchmark account** is randomly generated in each run and has not passed domain control validation, so apply / update / revoke transactions return early inside the contract at the authorization check; they still go through PBFT consensus and on-chain commitment.
- **Domains per certificate**: all tests use single-domain certificates (one domain per certificate).
- **Domain control validation (DCV)**: only the on-chain stages are measured. `*_v1` is stage 1 (VRF randomness request: BACME `controlValidate`, Baseline 1 `RegisterDomain`, Baseline 2 `registerDomain`); `*_v2` is stage 2 (start of HTTP validation: `startValidation` / `StartValidation`). Each transaction uses a new account and a unique domain; stage 2 first completes stage 1 (not recorded). Oracle callbacks are identical for the three schemes and are not included. The blocks column of `*_v2` rows includes the blocks of stage 1.
- Each combination is repeated several times; values are **mean ± standard deviation**.

## 2. Experimental Environment

```
Collected at: 2026-11-10 02:00:23 CST

== Hardware ==
CPU(s):                                  8
Model name:                              Intel(R) Core(TM) i5-14400F
Thread(s) per core:                      1
Core(s) per socket:                      1
Socket(s):                               8
Hypervisor vendor:                       VMware
Virtualization type:                     full
Memory: 7.7Gi
Disk: 49G total, 23G available

== OS and software ==
OS: Ubuntu 22.04.5 LTS
Kernel: 6.8.0-138-generic
FISCO BCOS: FISCO-BCOS Version : 2.11.0
Java: openjdk version "1.8.0_504"
Go: go version go1.22.12 linux/amd64

== Deployment ==
All nodes and the benchmark client are deployed on the same virtual machine; inter-node traffic goes through the loopback interface (127.0.0.1)
Number of node directories: 10
Running node processes: 10
Baseline inter-node RTT (ping 127.0.0.1, 20 times): rtt min/avg/max/mdev = 0.013/0.040/0.061/0.011 ms

== Consensus parameters (node0/conf/group.1.genesis) ==
[consensus]
    consensus_type=pbft
    max_trans_num=1000
    consensus_timeout=3
    epoch_sealer_num=10
    epoch_block_num=1000
[state]
    type=storage
[tx]
    gas_limit=300000000
[group]
    id=1
    timestamp=1794239220000
[evm]
    enable_free_storage=false

== Group runtime parameters (consensus / tx_pool / sync sections of node0/conf/group.1.ini) ==
[consensus]
    ; the ttl for broadcasting pbft message
    ttl=2
    ; min block generation time(ms)
    min_block_generation_time=500
    enable_dynamic_block_size=true
    enable_ttl_optimization=true
    enable_prepare_with_txsHash=true
    ; The following is the relevant configuration of rpbft
    ; set true to enable broadcast prepare request by tree
    broadcast_prepare_by_tree=true
    ; percent of nodes that broadcast prepare status to, must be between 25 and 100
    prepare_status_broadcast_percent=33
    ; max wait time before request missed transactions, ms, must be between 5ms and 1000ms
    max_request_missedTxs_waitTime=100
    ; maximum wait time before requesting a prepare, ms, must be between 10ms and 1000ms
    max_request_prepare_waitTime=100
[tx_pool]
    limit=150000
    ; transaction pool memory size limit, MB
    memory_limit=512
    ; number of threads responsible for transaction notification,
    ; default is 2, not recommended for more than 8
    notify_worker_num=2
    ; transaction expiration time, in seconds
    ; default is 10 minute(600s)
    txs_expiration_time=600
[sync]
    ; max memory size used for block sync, must >= 32MB
    max_block_sync_memory_size=512
    idle_wait_ms=200
    ; send block status by tree-topology, only supported when use pbft
    sync_block_by_tree=true
    ; send transaction by tree-topology, only supported when use pbft
    ; recommend to use when deploy many consensus nodes
    send_txs_by_tree=true
    ; must between 1000 to 3000
    ; only enabled when sync_by_tree is true
    gossip_interval_ms=1000
    gossip_peers_number=3
    ; max number of nodes that broadcast txs status to, recommended less than 5
    txs_max_gossip_peers_num=5

== Additional notes ==
The N=10 domain control validation (bacme_v1) experiment could not complete with 8 GB of memory due to out-of-memory (OOM) errors;
it was rerun after increasing the VM memory to 16 GB. The CPU (8 vCPUs, Intel Core i5-14400F) and all other settings were unchanged across all experiments.
All other experiments were conducted with 8 GB of memory.
```

## 3. Number of consensus nodes N = 10 (f = 3)

Target RTT -> measured TCP handshake RTT: 0 ms → 0.10 ms; 20 ms → 20.64 ms; 50 ms → 50.66 ms; 100 ms → 100.89 ms; 200 ms → 200.74 ms

### 3.1 Throughput (TPS of successful transactions)

Maximum sending rate 10000 tx/s, 30000 transactions per run.

| Operation | RTT 0 ms | RTT 20 ms | RTT 50 ms | RTT 100 ms | RTT 200 ms |
|---|---|---|---|---|---|
| bacme_v1 | 295.4 ± 12.2 | 296.1 ± 17.2 | 305.8 ± 16.6 | 296.2 ± 12.8 | 286.9 ± 19.1 |
| apply | 1738.7 ± 35.9 | 1773.8 ± 12.1 | 1636.9 ± 130.4 | 1366.6 ± 81.8 | 1081.8 ± 5.8 |
| update | 1741.1 ± 122.3 | 1735.2 ± 182.8 | 1709.6 ± 16.3 | 1438.1 ± 61.3 | 1113.9 ± 22.3 |
| verify | 5239.6 ± 111.3 | 5257.4 ± 235.9 | 5336.7 ± 261.9 | 5198.0 ± 141.2 | 5131.3 ± 111.2 |
| revoke | 1873.4 ± 42.3 | 1874.4 ± 69.5 | 1752.7 ± 5.7 | 1442.9 ± 50.6 | 1027.3 ± 79.9 |

### 3.2 Confirmation latency (ms, p50 / p95)

Fixed sending rate 200 tx/s, 2000 transactions per run.

| Operation | RTT 0 ms | RTT 20 ms | RTT 50 ms | RTT 100 ms | RTT 200 ms |
|---|---|---|---|---|---|
| bacme_v1 | 1686 / 2930 | 1656 / 2518 | 1985 / 2999 | 2299 / 3308 | 3291 / 4687 ⚠ |
| apply | 659 / 2347 ⚠ | 528 / 827 | 717 / 1062 | 1089 / 1576 | 1846 / 2652 |
| update | 422 / 1189 ⚠ | 513 / 821 | 749 / 1615 ⚠ | 1085 / 1596 | 1826 / 2560 |
| verify | 1 / 9 | 2 / 7 | 1 / 11 | 2 / 14 | 2 / 13 |
| revoke | 422 / 1190 ⚠ | 549 / 1377 ⚠ | 723 / 1095 | 1096 / 1590 | 1825 / 2600 |

Note: bacme_v1 @ RTT 200 ms: successful TPS 156 is clearly below the sending rate 200; possibly saturated, latency includes queuing; apply @ RTT 0 ms: run(s) 2,3: p95 exceeds 3x p50, suspected PBFT view change; update @ RTT 0 ms: run(s) 2: p95 exceeds 3x p50, suspected PBFT view change; update @ RTT 50 ms: run(s) 3: p95 exceeds 3x p50, suspected PBFT view change; revoke @ RTT 0 ms: run(s) 2: p95 exceeds 3x p50, suspected PBFT view change; revoke @ RTT 20 ms: run(s) 1: p95 exceeds 3x p50, suspected PBFT view change

<details><summary>N = 10 latency details (mean ± standard deviation)</summary>

| Operation | RTT (ms) | Runs | Successful TPS | Mean latency | p50 | p95 | p99 | Error rate % |
|---|---|---|---|---|---|---|---|---|
| bacme_v1 | 0 | 5 | 172.6 ± 14.7 | 1786.3 ± 379.6 | 1685.8 ± 213.7 | 2930.2 ± 1023.2 | 3170.8 ± 1126.2 | 0.00 ± 0.00 |
| bacme_v1 | 20 | 5 | 174.6 ± 5.7 | 1682.1 ± 156.8 | 1655.8 ± 145.7 | 2517.8 ± 272.7 | 2716.0 ± 293.4 | 0.00 ± 0.00 |
| bacme_v1 | 50 | 5 | 170.8 ± 5.3 | 2008.0 ± 153.6 | 1985.4 ± 148.2 | 2999.2 ± 328.4 | 3240.4 ± 418.5 | 0.00 ± 0.00 |
| bacme_v1 | 100 | 5 | 165.7 ± 4.8 | 2316.5 ± 71.7 | 2298.6 ± 74.6 | 3308.4 ± 135.1 | 3518.6 ± 144.5 | 0.00 ± 0.00 |
| bacme_v1 | 200 | 5 | 156.0 ± 3.9 | 3323.3 ± 148.8 | 3291.4 ± 133.3 | 4686.6 ± 304.1 | 5020.8 ± 374.9 | 0.00 ± 0.00 |
| apply | 0 | 5 | 193.1 ± 3.2 | 895.3 ± 518.8 | 659.0 ± 418.5 | 2346.8 ± 1545.0 | 2568.2 ± 1700.7 | 0.00 ± 0.00 |
| apply | 20 | 5 | 189.3 ± 2.4 | 528.0 ± 11.0 | 527.6 ± 13.5 | 827.2 ± 15.8 | 907.0 ± 30.4 | 0.00 ± 0.00 |
| apply | 50 | 5 | 190.1 ± 1.1 | 718.8 ± 11.0 | 716.6 ± 14.3 | 1062.4 ± 10.7 | 1156.6 ± 12.2 | 0.00 ± 0.00 |
| apply | 100 | 5 | 182.1 ± 4.3 | 1096.9 ± 35.6 | 1089.2 ± 30.5 | 1575.8 ± 69.1 | 1711.0 ± 75.0 | 0.00 ± 0.00 |
| apply | 200 | 5 | 174.6 ± 4.9 | 1864.9 ± 74.5 | 1845.8 ± 68.5 | 2652.4 ± 133.4 | 2874.0 ± 201.1 | 0.00 ± 0.00 |
| update | 0 | 5 | 192.7 ± 3.2 | 513.5 ± 260.7 | 421.6 ± 59.2 | 1189.4 ± 1191.7 | 1309.8 ± 1345.9 | 0.00 ± 0.00 |
| update | 20 | 5 | 192.7 ± 3.0 | 517.8 ± 9.4 | 513.4 ± 6.8 | 821.4 ± 23.1 | 900.2 ± 43.6 | 0.00 ± 0.00 |
| update | 50 | 5 | 185.8 ± 2.3 | 844.3 ± 285.0 | 748.6 ± 78.1 | 1615.0 ± 1228.3 | 1763.6 ± 1372.7 | 0.00 ± 0.00 |
| update | 100 | 5 | 186.9 ± 2.2 | 1095.6 ± 26.8 | 1085.0 ± 24.8 | 1596.2 ± 54.3 | 1779.4 ± 108.0 | 0.00 ± 0.00 |
| update | 200 | 5 | 175.1 ± 4.5 | 1840.9 ± 16.4 | 1825.6 ± 17.8 | 2560.0 ± 59.4 | 2784.6 ± 131.2 | 0.00 ± 0.00 |
| verify | 0 | 5 | 199.6 ± 0.4 | 2.7 ± 1.3 | 1.2 ± 0.4 | 9.0 ± 5.5 | 20.8 ± 6.7 | 0.00 ± 0.00 |
| verify | 20 | 5 | 199.7 ± 0.4 | 2.7 ± 2.3 | 1.8 ± 1.8 | 7.4 ± 8.0 | 13.0 ± 12.8 | 0.00 ± 0.00 |
| verify | 50 | 5 | 199.8 ± 0.3 | 3.2 ± 1.8 | 1.4 ± 0.5 | 11.0 ± 7.3 | 21.6 ± 11.3 | 0.00 ± 0.00 |
| verify | 100 | 5 | 199.8 ± 0.3 | 3.7 ± 1.7 | 2.0 ± 1.7 | 13.8 ± 5.4 | 23.6 ± 5.9 | 0.00 ± 0.00 |
| verify | 200 | 5 | 199.2 ± 0.0 | 3.8 ± 1.6 | 1.6 ± 0.5 | 13.4 ± 5.5 | 24.8 ± 6.5 | 0.00 ± 0.00 |
| revoke | 0 | 5 | 192.3 ± 1.7 | 514.8 ± 266.0 | 422.4 ± 59.7 | 1190.0 ± 1198.5 | 1313.4 ± 1352.6 | 0.00 ± 0.00 |
| revoke | 20 | 5 | 192.6 ± 3.2 | 646.9 ± 275.7 | 549.0 ± 67.1 | 1376.8 ± 1214.5 | 1541.2 ± 1346.5 | 0.00 ± 0.00 |
| revoke | 50 | 5 | 188.0 ± 3.3 | 727.2 ± 16.4 | 723.0 ± 16.6 | 1095.4 ± 28.4 | 1195.6 ± 49.6 | 0.00 ± 0.00 |
| revoke | 100 | 5 | 184.6 ± 2.4 | 1103.7 ± 34.4 | 1096.0 ± 35.9 | 1590.2 ± 48.2 | 1703.4 ± 50.6 | 0.00 ± 0.00 |
| revoke | 200 | 5 | 173.3 ± 2.9 | 1843.8 ± 51.1 | 1825.4 ± 42.4 | 2600.4 ± 99.6 | 2812.4 ± 89.5 | 0.00 ± 0.00 |

</details>

## 4. Number of consensus nodes N = 7 (f = 2)

Target RTT -> measured TCP handshake RTT: 0 ms → 0.09 ms; 20 ms → 20.69 ms; 50 ms → 50.80 ms; 100 ms → 100.80 ms; 200 ms → 200.91 ms

### 4.1 Throughput (TPS of successful transactions)

Maximum sending rate 10000 tx/s, 30000 transactions per run.

| Operation | RTT 0 ms | RTT 20 ms | RTT 50 ms | RTT 100 ms | RTT 200 ms |
|---|---|---|---|---|---|
| bacme_v1 | 406.1 ± 10.8 | 432.9 ± 25.5 | 418.0 ± 34.1 | 404.8 ± 14.4 | 354.8 ± 29.7 |
| apply | 2396.4 ± 12.3 | 2080.7 ± 325.3 | 2067.5 ± 48.8 | 1733.0 ± 28.5 | 1244.8 ± 6.2 |
| update | 2262.3 ± 255.6 | 2371.7 ± 66.3 | 2202.2 ± 77.0 | 1720.4 ± 62.1 | 1193.9 ± 53.9 |
| verify | 5668.8 ± 71.3 | 5165.1 ± 121.1 | 5117.2 ± 55.9 | 5331.9 ± 154.1 | 5194.9 ± 245.4 |
| revoke | 2483.0 ± 125.4 | 2561.8 ± 113.4 | 2070.4 ± 255.0 | 1770.8 ± 64.0 | 1233.5 ± 60.7 |

### 4.2 Confirmation latency (ms, p50 / p95)

Fixed sending rate 200 tx/s, 2000 transactions per run.

| Operation | RTT 0 ms | RTT 20 ms | RTT 50 ms | RTT 100 ms | RTT 200 ms |
|---|---|---|---|---|---|
| bacme_v1 | 1033 / 1609 | 1077 / 1632 | 1315 / 1896 | 1687 / 2490 | 2574 / 3630 |
| apply | 373 / 627 | 501 / 803 | 711 / 1089 | 1049 / 1532 | 1732 / 2530 |
| update | 456 / 2221 ⚠ | 525 / 1301 ⚠ | 726 / 1103 | 1085 / 1668 | 1791 / 2593 |
| verify | 2 / 9 | 1 / 5 | 1 / 5 | 1 / 7 | 1 / 6 |
| revoke | 602 / 2249 ⚠ | 504 / 814 | 750 / 1612 ⚠ | 1091 / 1610 | 1799 / 2651 |

Note: update @ RTT 0 ms: run(s) 1,2,3: p95 exceeds 3x p50, suspected PBFT view change; update @ RTT 20 ms: run(s) 2: p95 exceeds 3x p50, suspected PBFT view change; revoke @ RTT 0 ms: run(s) 1,3: p95 exceeds 3x p50, suspected PBFT view change; revoke @ RTT 50 ms: run(s) 5: p95 exceeds 3x p50, suspected PBFT view change

<details><summary>N = 7 latency details (mean ± standard deviation)</summary>

| Operation | RTT (ms) | Runs | Successful TPS | Mean latency | p50 | p95 | p99 | Error rate % |
|---|---|---|---|---|---|---|---|---|
| bacme_v1 | 0 | 5 | 185.7 ± 2.5 | 1046.6 ± 68.9 | 1033.2 ± 64.5 | 1609.0 ± 143.2 | 1776.4 ± 150.8 | 0.00 ± 0.00 |
| bacme_v1 | 20 | 5 | 184.6 ± 3.4 | 1089.6 ± 44.0 | 1076.8 ± 44.9 | 1632.0 ± 99.9 | 1821.0 ± 129.6 | 0.00 ± 0.00 |
| bacme_v1 | 50 | 5 | 179.1 ± 2.1 | 1321.1 ± 8.8 | 1314.6 ± 8.3 | 1896.4 ± 35.5 | 2014.2 ± 45.4 | 0.00 ± 0.00 |
| bacme_v1 | 100 | 5 | 178.1 ± 1.7 | 1719.8 ± 25.6 | 1687.4 ± 19.7 | 2489.6 ± 96.2 | 2712.8 ± 147.6 | 0.00 ± 0.00 |
| bacme_v1 | 200 | 5 | 162.8 ± 6.6 | 2589.3 ± 86.0 | 2574.0 ± 87.2 | 3629.6 ± 142.9 | 3954.6 ± 171.3 | 0.00 ± 0.00 |
| apply | 0 | 5 | 191.1 ± 3.5 | 374.9 ± 6.1 | 373.4 ± 6.1 | 627.0 ± 8.7 | 671.8 ± 15.8 | 0.00 ± 0.00 |
| apply | 20 | 5 | 191.5 ± 4.1 | 506.2 ± 7.1 | 501.0 ± 5.3 | 803.0 ± 17.1 | 887.2 ± 23.2 | 0.00 ± 0.00 |
| apply | 50 | 5 | 189.6 ± 3.0 | 713.8 ± 10.2 | 711.0 ± 10.3 | 1089.2 ± 21.8 | 1205.8 ± 39.7 | 0.00 ± 0.00 |
| apply | 100 | 5 | 183.8 ± 3.8 | 1055.1 ± 47.2 | 1049.0 ± 43.9 | 1531.8 ± 75.4 | 1685.8 ± 128.8 | 0.00 ± 0.00 |
| apply | 200 | 5 | 173.7 ± 5.2 | 1747.3 ± 46.4 | 1732.2 ± 43.7 | 2529.8 ± 118.6 | 2885.4 ± 260.7 | 0.00 ± 0.00 |
| update | 0 | 5 | 192.1 ± 3.2 | 726.5 ± 324.8 | 456.0 ± 80.4 | 2221.2 ± 1456.3 | 2475.6 ± 1642.5 | 0.00 ± 0.00 |
| update | 20 | 5 | 190.4 ± 3.2 | 610.4 ± 253.0 | 524.6 ± 65.7 | 1301.0 ± 1149.9 | 1446.8 ± 1291.1 | 0.00 ± 0.00 |
| update | 50 | 5 | 187.4 ± 3.0 | 726.2 ± 23.5 | 725.8 ± 22.4 | 1103.0 ± 50.2 | 1236.0 ± 89.5 | 0.00 ± 0.00 |
| update | 100 | 5 | 183.3 ± 1.9 | 1096.5 ± 20.4 | 1085.4 ± 22.3 | 1668.2 ± 72.9 | 1875.0 ± 116.5 | 0.00 ± 0.00 |
| update | 200 | 5 | 176.2 ± 4.9 | 1813.9 ± 89.8 | 1790.6 ± 93.1 | 2592.6 ± 108.0 | 2901.2 ± 46.0 | 0.00 ± 0.00 |
| verify | 0 | 5 | 199.7 ± 0.5 | 3.1 ± 2.3 | 1.8 ± 1.8 | 9.4 ± 7.9 | 17.4 ± 13.4 | 0.00 ± 0.00 |
| verify | 20 | 5 | 200.0 ± 0.0 | 1.8 ± 0.9 | 1.0 ± 0.0 | 5.4 ± 5.6 | 10.4 ± 10.6 | 0.00 ± 0.00 |
| verify | 50 | 5 | 199.8 ± 0.4 | 1.7 ± 0.5 | 1.0 ± 0.0 | 5.0 ± 3.9 | 11.8 ± 7.5 | 0.00 ± 0.00 |
| verify | 100 | 5 | 200.0 ± 0.0 | 2.1 ± 1.0 | 1.0 ± 0.0 | 7.4 ± 6.9 | 14.6 ± 11.8 | 0.00 ± 0.00 |
| verify | 200 | 5 | 199.7 ± 0.4 | 2.2 ± 0.8 | 1.4 ± 0.5 | 6.0 ± 4.2 | 14.8 ± 8.6 | 0.00 ± 0.00 |
| revoke | 0 | 5 | 191.7 ± 5.2 | 831.5 ± 479.4 | 602.2 ± 373.2 | 2249.0 ± 1485.6 | 2465.8 ± 1642.9 | 0.00 ± 0.00 |
| revoke | 20 | 5 | 191.3 ± 3.1 | 506.4 ± 7.8 | 504.0 ± 8.0 | 814.4 ± 18.9 | 900.2 ± 27.7 | 0.00 ± 0.00 |
| revoke | 50 | 5 | 187.2 ± 3.3 | 841.0 ± 285.3 | 750.0 ± 94.7 | 1612.4 ± 1185.6 | 1768.6 ± 1312.9 | 0.00 ± 0.00 |
| revoke | 100 | 5 | 184.6 ± 3.3 | 1097.9 ± 18.5 | 1091.0 ± 13.9 | 1610.2 ± 34.5 | 1773.2 ± 51.3 | 0.00 ± 0.00 |
| revoke | 200 | 5 | 173.9 ± 6.4 | 1829.0 ± 67.2 | 1799.4 ± 76.4 | 2650.6 ± 81.2 | 2981.2 ± 140.1 | 0.00 ± 0.00 |

</details>

## 5. Number of consensus nodes N = 4 (f = 1)

Target RTT -> measured TCP handshake RTT: 0 ms → 0.10 ms; 20 ms → 20.68 ms; 50 ms → 50.88 ms; 100 ms → 100.82 ms; 200 ms → 200.78 ms

### 5.1 Throughput (TPS of successful transactions)

Maximum sending rate 10000 tx/s, 30000 transactions per run.

| Operation | RTT 0 ms | RTT 20 ms | RTT 50 ms | RTT 100 ms | RTT 200 ms |
|---|---|---|---|---|---|
| bacme_v1 | 769.8 ± 58.6 | 799.4 ± 7.5 | 773.5 ± 64.0 | 690.1 ± 32.5 | 614.3 ± 7.6 |
| apply | 3024.7 ± 129.0 | 3060.4 ± 119.3 | 2615.7 ± 40.0 | 1975.7 ± 39.4 | 1305.9 ± 7.5 |
| update | 3226.0 ± 82.6 | 3116.5 ± 60.7 | 2609.7 ± 89.9 | 2060.1 ± 76.2 | 1336.0 ± 38.5 |
| verify | 5345.6 ± 185.7 | 5269.5 ± 35.6 | 5333.6 ± 86.6 | 4942.5 ± 386.0 | 5302.7 ± 105.5 |
| revoke | 3180.3 ± 310.9 | 3188.8 ± 209.3 | 2763.6 ± 112.8 | 2045.4 ± 46.2 | 1359.9 ± 41.4 |

### 5.2 Confirmation latency (ms, p50 / p95)

Fixed sending rate 200 tx/s, 2000 transactions per run.

| Operation | RTT 0 ms | RTT 20 ms | RTT 50 ms | RTT 100 ms | RTT 200 ms |
|---|---|---|---|---|---|
| bacme_v1 | 544 / 873 | 701 / 1085 | 979 / 1457 | 1343 / 1980 | 2130 / 3086 |
| apply | 373 / 1116 ⚠ | 490 / 787 | 755 / 1664 ⚠ | 986 / 1499 | 1639 / 2561 |
| update | 347 / 595 | 492 / 789 | 697 / 1080 | 1105 / 2124 ⚠ | 1639 / 2433 |
| verify | 1 / 5 | 1 / 6 | 1 / 9 | 1 / 15 | 1 / 8 |
| revoke | 349 / 593 | 489 / 792 | 691 / 1072 | 1067 / 1644 | 1628 / 2431 |

Note: apply @ RTT 0 ms: run(s) 4: p95 exceeds 3x p50, suspected PBFT view change; apply @ RTT 50 ms: run(s) 4: p95 exceeds 3x p50, suspected PBFT view change; update @ RTT 100 ms: run(s) 5: p95 exceeds 3x p50, suspected PBFT view change

<details><summary>N = 4 latency details (mean ± standard deviation)</summary>

| Operation | RTT (ms) | Runs | Successful TPS | Mean latency | p50 | p95 | p99 | Error rate % |
|---|---|---|---|---|---|---|---|---|
| bacme_v1 | 0 | 5 | 192.6 ± 3.4 | 547.8 ± 22.2 | 544.4 ± 21.7 | 872.6 ± 45.3 | 956.0 ± 65.4 | 0.00 ± 0.00 |
| bacme_v1 | 20 | 5 | 188.8 ± 3.3 | 705.2 ± 22.5 | 701.0 ± 18.9 | 1084.8 ± 52.8 | 1199.8 ± 84.4 | 0.00 ± 0.00 |
| bacme_v1 | 50 | 5 | 186.6 ± 2.0 | 985.5 ± 35.9 | 978.6 ± 29.4 | 1457.2 ± 74.2 | 1574.6 ± 113.2 | 0.00 ± 0.00 |
| bacme_v1 | 100 | 5 | 178.7 ± 2.9 | 1355.7 ± 40.0 | 1343.2 ± 41.3 | 1980.2 ± 79.1 | 2177.8 ± 108.4 | 0.00 ± 0.00 |
| bacme_v1 | 200 | 5 | 173.3 ± 5.6 | 2150.3 ± 57.2 | 2130.0 ± 51.0 | 3085.8 ± 191.9 | 3385.4 ± 187.4 | 0.00 ± 0.00 |
| apply | 0 | 5 | 195.1 ± 1.8 | 461.3 ± 253.0 | 372.6 ± 58.2 | 1116.2 ± 1167.1 | 1231.4 ± 1327.5 | 0.00 ± 0.00 |
| apply | 20 | 5 | 192.2 ± 3.4 | 492.7 ± 4.6 | 489.8 ± 4.8 | 786.8 ± 10.8 | 865.6 ± 26.2 | 0.00 ± 0.00 |
| apply | 50 | 5 | 186.3 ± 3.7 | 851.5 ± 296.4 | 754.8 ± 92.9 | 1663.6 ± 1211.4 | 1831.4 ± 1332.6 | 0.00 ± 0.00 |
| apply | 100 | 5 | 182.0 ± 5.4 | 995.8 ± 68.6 | 985.6 ± 68.2 | 1499.4 ± 95.7 | 1675.2 ± 121.7 | 0.00 ± 0.00 |
| apply | 200 | 5 | 180.5 ± 5.3 | 1676.5 ± 71.8 | 1639.0 ± 72.1 | 2560.6 ± 220.0 | 2867.8 ± 264.3 | 0.00 ± 0.00 |
| update | 0 | 5 | 192.2 ± 3.1 | 348.4 ± 6.3 | 346.8 ± 8.5 | 594.8 ± 9.1 | 645.4 ± 17.6 | 0.00 ± 0.00 |
| update | 20 | 5 | 188.9 ± 3.0 | 493.3 ± 7.3 | 492.0 ± 9.0 | 788.6 ± 15.5 | 865.8 ± 41.4 | 0.00 ± 0.00 |
| update | 50 | 5 | 187.8 ± 3.8 | 703.6 ± 7.2 | 697.4 ± 6.3 | 1080.2 ± 18.8 | 1185.0 ± 36.5 | 0.00 ± 0.00 |
| update | 100 | 5 | 181.3 ± 2.7 | 1201.8 ± 332.9 | 1104.8 ± 121.1 | 2124.0 ± 1252.4 | 2341.8 ± 1360.1 | 0.00 ± 0.00 |
| update | 200 | 5 | 170.8 ± 4.0 | 1657.5 ± 56.6 | 1639.2 ± 65.8 | 2433.4 ± 84.8 | 2668.6 ± 172.8 | 0.00 ± 0.00 |
| verify | 0 | 5 | 199.9 ± 0.3 | 2.0 ± 1.0 | 1.2 ± 0.4 | 5.4 ± 5.6 | 10.4 ± 10.6 | 0.00 ± 0.00 |
| verify | 20 | 5 | 199.7 ± 0.4 | 1.9 ± 0.4 | 1.0 ± 0.0 | 6.0 ± 2.9 | 15.4 ± 7.4 | 0.00 ± 0.00 |
| verify | 50 | 5 | 199.7 ± 0.4 | 2.7 ± 1.9 | 1.2 ± 0.4 | 9.0 ± 8.1 | 15.6 ± 12.7 | 0.00 ± 0.00 |
| verify | 100 | 5 | 199.4 ± 0.3 | 3.9 ± 1.2 | 1.4 ± 0.5 | 15.4 ± 3.6 | 27.4 ± 4.8 | 0.00 ± 0.00 |
| verify | 200 | 5 | 199.7 ± 0.4 | 2.4 ± 1.3 | 1.0 ± 0.0 | 8.4 ± 6.8 | 16.6 ± 12.8 | 0.00 ± 0.00 |
| revoke | 0 | 5 | 192.5 ± 2.2 | 349.1 ± 4.5 | 348.8 ± 6.0 | 592.8 ± 5.7 | 630.6 ± 8.5 | 0.00 ± 0.00 |
| revoke | 20 | 5 | 192.0 ± 4.3 | 492.2 ± 11.9 | 488.8 ± 11.5 | 792.0 ± 24.8 | 863.0 ± 36.8 | 0.00 ± 0.00 |
| revoke | 50 | 5 | 188.2 ± 4.8 | 696.9 ± 14.4 | 691.0 ± 17.9 | 1071.8 ± 19.8 | 1162.6 ± 27.3 | 0.00 ± 0.00 |
| revoke | 100 | 5 | 181.4 ± 4.8 | 1076.5 ± 33.5 | 1066.6 ± 25.3 | 1644.2 ± 109.7 | 1843.0 ± 169.9 | 0.00 ± 0.00 |
| revoke | 200 | 5 | 178.6 ± 4.3 | 1657.6 ± 71.7 | 1628.4 ± 66.6 | 2430.6 ± 106.2 | 2696.8 ± 129.5 | 0.00 ± 0.00 |

</details>

## Raw Data

- `throughput.csv`, `latency.csv`: raw results of every benchmark run
- `env.txt`: experimental environment and consensus parameters
- The full output of each run is in `logs/` next to the result files
