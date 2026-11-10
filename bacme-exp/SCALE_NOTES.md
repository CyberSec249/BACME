# Large Certificate State Experiment: Points That Must Be Stated in the Paper and the Response Letter

## 1. Pre-populated certificate records
- Pre-populated records are written to the contract through **exactly the same `applyCertificate` transaction** as a real certificate application (original DpkiAcme contract, unmodified).
- Each record contains the same five metadata fields written by a real application: certificate hash (IPFS CID), covered domain, certificate public key hash, management account, and validity status;
  **its format and size are identical to those of a record written by a real application** (the certificate hash has 59 characters, the same length as a CIDv1).
- The only differences: the certificate hashes are generated from sequence numbers, and the corresponding X.509 certificate files are neither generated nor uploaded to IPFS; all records belong to the same account, which has passed real domain control validation, and the same domain.
- The off-chain IPFS storage is not involved in the on-chain processing evaluated in this experiment, so for the contract and the blockchain, pre-populated records are indistinguishable from really issued certificates.
- Use "pre-populated certificate records" in the paper, not "seed certificates".

Suggested wording:
> To emulate a large-scale deployment, we pre-populate the ACME smart contract with certificate metadata records through regular certificate application transactions before the measurement. Each record has the same structure and size as that of a real certificate (i.e., the CID of the certificate, the covered domain, the public key hash, the management account, and the validity status), while the corresponding certificate files are not uploaded to IPFS, since the off-chain storage does not affect the on-chain processing evaluated in this experiment.

## 2. Complete execution path
- All write operations in this experiment take the complete execution path (validated account, a distinct certificate hash per transaction); transactions for which the contract returns `false` are counted as errors.
- This differs from the network latency experiment (PerformanceDpkiAcme, random account, early return at the authorization check), so the absolute write throughput of the two experiments cannot be compared directly.

## 3. Number of certificate records per level
- Levels are reported by the number of on-chain certificate records during the measurement: < 0.4 M, about 1.4 M (1.41 M -> 1.81 M), and about 2.8 M (third level).
- The record count includes valid certificates as well as records invalidated by renewal or revocation.
- The measurement of each level itself adds about 0.4 million records.

## 4. Data processing
- The first-level verify latency (5 runs) was measured while pre-populated records were being written in the background and is excluded; the first-level verify throughput (3 runs) finished before the writing started (02:22:57) and is kept.
- The second level was measured immediately after the pre-population finished and may be affected by RocksDB background compaction; the third level was measured after an idle period of 30 minutes.
- During the pre-population of the third level, the VM was paused once because the host's E: drive ran out of space; after recovery, writing resumed from the last checkpoint (writing is idempotent: the same sequence number yields the same certificate hash, so nothing is counted twice). The measurement took place after all writing had finished and an idle period of 30 minutes, so it is not affected.
- Runs with view changes are excluded with the same rules as in the network latency experiment; the block count of renewal/revocation includes the untimed preparation phase, so the block rule is not applied to them.
- Certificate verification queries randomly selected pre-populated records (random reads), which cannot be compared directly with the network latency experiment, where the same certificate is queried repeatedly (cache hits).

## 5. Summary of results (three levels, N = 10, RTT ≈ 0)
| Operation | < 0.4 M | ~ 1.4 M (measured right after writing) | ~ 2.8 M (measured after 30 min idle) |
|---|---|---|---|
| Application throughput | 1059 ± 150 tx/s | 1098 ± 94 | 1192 ± 56 |
| Renewal throughput | 1067 ± 17 | 908 ± 144 | 1005 ± 51 |
| Revocation throughput | 1440 ± 95 | 1018 ± 16 | 1315 ± 121 |
| Verification throughput | 4598 ± 99 queries/s | 3481 ± 411 | 4131 ± 195 |
| Application p50/p95 | 464 / 745 ms | 484 / 792 | 464 / 753 |
| Renewal p50/p95 | 479 / 768 | 510 / 849 | 494 / 801 |
| Revocation p50/p95 | 428 / 698 | 441 / 717 | 442 / 721 |
| Verification p50/p95 | (excluded) | 4 / 19 | 6 / 22 |

- The third level starts at about 2.8 M records and ends at about 3.2 M; the block data of node0 is 4.7 GB.
- The third level (more records, but measured after 30 minutes of idle time) performs better than the second level, indicating that the decrease at the second level mainly came from temporary RocksDB background compaction after bulk insertion.
- From < 0.4 M to ~2.8 M records: write p50 changes by at most 3.3% and p95 by at most 4.3%; throughput changes by +12.5% (application), −5.8% (renewal) and −8.7% (revocation); verification throughput changes by −10.2%.
- 7 runs were excluded in total (2 throughput runs, 5 latency runs), all due to PBFT view changes.
