# BACME: Blockchain-based Automatic Certificate Management Environment

This repository contains part of the prototype implementation of **BACME**, together with the reproduced baselines, the performance evaluation toolkit, and the experimental data used in the paper:

> L. Li, Z. Yan, C. Shi, and X. Liang, "BACME: Blockchain-based Automatic Certificate Management Environment."

BACME is a consortium blockchain-based ACME scheme. It manages domain control validation, certificate application, renewal, revocation, and verification through smart contracts, stores complete certificates in decentralized off-chain storage (IPFS), and supports both single-domain and multi-domain certificates.

> **Note on this release:** For confidentiality reasons, this repository does not contain the complete prototype. The following components are **not included**:
>
> - the BACME ACME client back end (`acmeClient/`);
> - the load generator (`performanceTest/`) and the compiled benchmark jar (`java-sdk-demo-1.0.0.jar`);
> - the Go contract bindings (`deployContract/DpkiAcme/`, `baseline1/DACME/`, `baseline2/DCV/`) and the `go.mod` / `go.sum` files;
> - the Truora and FISCO-BCOS contracts imported by the smart contracts (`VRFClient.sol`, `FiscoOracleClient.sol`, `Table.sol`);
> - the project files and static assets of the web UI (`package.json`, Vite configuration, `dist/assets/`);
> - all credentials and configuration files (`conf/`).
>
> The released code is therefore provided mainly for reference and review. Paths to the missing components that appear in scripts or in the sections below refer to the complete prototype.

## Repository Structure

| Path | Description |
|---|---|
| `contract/` | Solidity smart contracts: `DpkiAcme.sol` (BACME), `DACME.sol` (Baseline 1), `DCV.sol` (Baseline 2), the string utility library `LibString.sol`, and the VRF sample client `simplevrf/RandomNumberSampleVRF.sol` |
| `deployContract/` | Program for deploying the BACME smart contract (`main.go`) |
| `baseline1/` | Reproduced Baseline 1 [Kfoury et al., TSP 2020] (Go client: `cmd/`, `util/`) |
| `baseline2/` | Reproduced Baseline 2, B-DCV [Khoury et al., TSP 2022] (Go client: `cmd/`, `util/`) |
| `webui/` | Source code of the front-end graphical user interface of the ACME client (Vue 3 + Ant Design Vue): entry `src/main.js` and page components in `src/components/` |
| `bacme-exp/` | Performance evaluation toolkit: network delay emulation, node count adjustment, benchmark automation, result aggregation, report generation, plotting scripts, and patches to the FISCO-BCOS Java SDK benchmark program (`perf-patch/`) |
| `bacme-exp-results/` | Experimental data: network latency × number of consensus nodes (`v2/`) and the large certificate state experiment (`scale/`) |
| `bacme-start.sh`, `bacme-stop.sh` | Scripts that start and stop the complete BACME stack (FISCO BCOS → MySQL → Truora → IPFS → back end → front end) |

The implementation details of the two baselines are described in [Implementation Details of the Baselines](#implementation-details-of-the-baselines).

## Environment

The prototype was evaluated with the following software:

| Component | Version |
|---|---|
| FISCO-BCOS | 2.11.0 |
| Solidity compiler (solc) | 0.6.10 |
| Truora oracle | 1.1.1 |
| IPFS | Private IPFS cluster (4 nodes in our evaluation) |
| Go | 1.22 or later |
| FISCO-BCOS Go SDK | v1.1.1 |
| Java | Java 8 |
| FISCO-BCOS Java SDK | 2.9.0 |
| Node.js | Required for the web UI (Vite 6) |
| Python 3 + matplotlib | Result aggregation and plotting (`bacme-exp/`) |
| Linux `tc` (netem) | Network delay emulation (`bacme-exp/netem.sh`) |

In the original evaluation of the paper, all components were deployed on a single server with 32 CPU cores, 128 GB of RAM, and 1 TB of storage. The hardware, operating system, and consensus parameters of the experiments in `bacme-exp-results/` are recorded in `bacme-exp-results/v2/env.txt`.

## Configuration

Credentials are **not** included in this repository. Before running a Go component, provide the following files in its `conf/` directory (`deployContract/conf`, `baseline1/conf`, `baseline2/conf`); the benchmark program reads the same files from `dist/conf/`:

| File | Description |
|---|---|
| `ca.crt`, `sdk.crt`, `sdk.key` | SDK certificates of your FISCO-BCOS chain (generated when the chain is built) |
| `accounts/<address>.pem` | Private key of the blockchain account |
| `config.toml` | Node address, group ID, chain ID, and account key file |
| `contractAddress` | Address of the deployed smart contract (written after deployment) |

In addition, the following settings are hard-coded and must be adapted to your deployment:

| Location | Setting |
|---|---|
| `baseline1/util/baseline1.go`, `baseline2/util/baseline2.go` | Addresses of the Truora `VRFCore` and `OracleCore` contracts and the hash of the VRF public key |
| `webui/src/components/*.vue` | Back-end API address (`http://localhost:8080`) |
| `bacme-exp/run_exp.sh`, `bacme-exp/seed_scale.sh` | Path of the benchmark program (`PERF_DIR`) and of the contract address file |
| `bacme-start.sh`, `bacme-stop.sh` | Installation paths of FISCO BCOS, Truora, and the project |

The benchmark scripts read the contract addresses and test accounts from environment variables (`DPKIACME_ADDR`, `DACME_ADDR`, `DCV_ADDR`, `ACCOUNT_PEM`, `SCALE_DOMAIN`); see the header of `bacme-exp/run_exp.sh` and the source files in `bacme-exp/perf-patch/src/`.

## Usage

### 1. Prerequisites

1. Build a FISCO-BCOS 2.11.0 chain.
2. Deploy Truora 1.1.1 on the chain, and record the addresses of the `VRFCore` and `OracleCore` contracts and the hash of the VRF public key.
3. Start an IPFS node or cluster.

### 2. Compile the Smart Contracts and Generate the Go Bindings

Place `VRFClient.sol` and `FiscoOracleClient.sol` (provided by Truora) and `Table.sol` (provided by FISCO-BCOS) in `contract/`, compile the contracts with solc 0.6.10, and generate the Go bindings with the `abigen` tool of the FISCO-BCOS Go SDK, e.g.:

```bash
solc-0.6.10 --abi --bin -o build contract/DpkiAcme.sol
abigen --bin build/DpkiAcme.bin --abi build/DpkiAcme.abi \
       --pkg DpkiAcme --type DpkiAcme --out deployContract/DpkiAcme/DpkiAcme.go
```

Generate `baseline1/DACME/DACME.go` (package `DACME`) and `baseline2/DCV/DCV.go` (package `DCV`) in the same way, and initialize a Go module in each directory (`go mod init deployContract`, `go mod init baseline1`, `go mod init baseline2`, then `go mod tidy`).

### 3. Deploy the BACME Smart Contract

```bash
cd deployContract
go run . -v <VRFCore address> -h <VRF public key hash> -o <OracleCore address>
```

The program prints the address of the deployed contract.

### 4. Run the Baselines

Each baseline has a deployment function (`DeployDACMEContract` in `baseline1/util/baseline1.go` and `DeployDCVContract` in `baseline2/util/baseline2.go`) and functions for each certificate management operation. Select the operation to be measured in `cmd/main.go` and run:

```bash
cd baseline1            # or baseline2
mkdir -p output         # timing and memory results are appended to files in output/
go run ./cmd/main.go
```

During domain control validation, the client serves the validation token at `/well-known/pki-validation` through a built-in HTTP server on port 80 (`util/httpServer.go`), which requires the corresponding privileges.

### 5. Web UI

`webui/src/` contains the source code of the web UI pages: domain control validation, certificate application, certificate management (renewal), emergency operations (revocation), certificate verification, and the update log. The pages call the API of the ACME client back end, which is not included in this repository.

### 6. Throughput Evaluation

The throughput and latency of BACME and the two baselines were measured with a load generator based on the FISCO-BCOS Java SDK under identical request rates. `bacme-exp/` contains the toolkit used for the experiments:

| File | Purpose |
|---|---|
| `netem.sh` | Adds delay to inter-node P2P traffic with `tc netem` (one-way delay D → RTT ≈ 2D) |
| `exp_nodes.sh` | Shows node status and helps adjust the number of consensus nodes |
| `run_exp.sh` | Iterates over RTT levels × operations × repetitions and appends the results to a CSV file |
| `seed_scale.sh` | Pre-populates certificate records on chain for the large certificate state experiment |
| `summarize.py` | Aggregates mean ± standard deviation by (nodes, RTT, operation) |
| `collect_env.sh` | Collects hardware, OS, and FISCO consensus parameters into `env.txt` |
| `make_report.py` | Generates the Markdown report `EXPERIMENT_RESULTS.md` |
| `perf-patch/` | Source code (`src/`) and diffs of the patched benchmark classes: `PerformanceDpkiAcme`, `PerformanceCollector` (adds p50 / p95 / p99 latency), `PerformanceDomainValidation` (on-chain stages of domain control validation for BACME and the two baselines), and `PerformanceScale` (large certificate state experiment) |
| `plots/` | Plotting scripts for the figures in the paper |

To reproduce the experiments, build the FISCO-BCOS Java SDK demo (2.9.0) with the classes in `bacme-exp/perf-patch/src/`, copy the SDK certificates of the chain to `dist/conf/` of the benchmark program, and run, for each number of consensus nodes (10 → 7 → 4):

```bash
bash bacme-exp/run_exp.sh --nodes 10
bash bacme-exp/run_exp.sh --nodes 7 --rtts "0 20 50 100 200" --ops "apply update verify revoke" \
                          --count 10000 --qps 10000 --repeat 3
```

Supported operations:

| Experiment | Operations |
|---|---|
| Certificate management (BACME) | `apply`, `update`, `revoke`, `verify` |
| Domain control validation | `bacme_v1`, `bacme_v2`, `dacme_v1`, `dacme_v2`, `dcv_v1`, `dcv_v2` (`_v1`: VRF randomness request; `_v2`: start of HTTP validation) |
| Large certificate state | `scale_apply`, `scale_update`, `scale_revoke`, `scale_verify` |

Then collect the environment and generate the report:

```bash
bash bacme-exp/collect_env.sh
python3 bacme-exp/make_report.py
```

See `bacme-exp/README.md` for the detailed experiment steps and the changes to the benchmark program, and `bacme-exp/SCALE_NOTES.md` for the setup of the large certificate state experiment.

### 7. Experimental Data

| Path | Content |
|---|---|
| `bacme-exp-results/v2/EXPERIMENT_RESULTS.md` | Methodology, environment, and throughput / latency results for N = 10, 7, and 4 consensus nodes under different inter-node RTTs |
| `bacme-exp-results/v2/env.txt` | Experimental environment |
| `bacme-exp-results/scale/throughput_*.csv`, `latency_*.csv` | Results of the large certificate state experiment at three levels of on-chain certificate records (`s0`, `s1m`, `s2m`) |
| `bacme-exp-results/scale/logs*/` | Raw benchmark logs |
| `bacme-exp-results/scale/figs/` | Figure of the large certificate state experiment |

## Deployment and Compatibility

Compared with standard ACME, BACME introduces additional infrastructure. However, these components are not independently deployed by every domain owner. The blockchain nodes and ACME smart contract are maintained by participating CAs, while the oracle and IPFS cluster operate as shared consortium services. The VRF is provided by the oracle framework rather than deployed as a separate service. A domain owner therefore requires only the BACME client and access to a blockchain node, whereas the consortium participants bear the costs of consensus, oracle operation, and distributed storage.

The experimental evaluation quantifies the principal operational overhead through smart-contract execution, throughput, on-chain storage, protocol latency, and communication volume. In the permissioned deployment, gas consumption represents smart-contract execution complexity rather than a direct monetary fee. The 16-node experiment maintains more than 550 transactions per second for domain-control validation under the evaluated workload, while off-chain certificate storage reduces the growth of on-chain data. These benefits involve measurable trade-offs, including approximately 7 ms of additional certificate-verification latency and lower throughput for some revocation operations. Because the prototype components were co-located on one server, these results characterize protocol overhead rather than the monetary or administrative cost of a geographically distributed production deployment.

BACME preserves the X.509 certificate structure and HTTP-based domain-control validation, allowing existing Web-server and TLS mechanisms to be reused. Nevertheless, it is not a drop-in replacement for standard ACME: the client must submit blockchain transactions and publish certificates to off-chain storage, while the browser requires a BACME verification module to query the smart contract and certificate storage. BACME is therefore intended primarily for consortium or managed-network environments that require shared certificate-state governance, multi-domain lifecycle management, and resilient revocation. Where reliance on a single CA is acceptable and minimal deployment complexity is the primary objective, standard ACME remains the simpler option.

## Implementation Details of the Baselines

This document describes how the two baselines used in the performance evaluation of BACME (Section VI-B of the paper) were reproduced. Neither baseline provides a publicly available implementation, so both were reimplemented following the architectures and protocols described in their respective papers:

- **Baseline 1**: E. F. Kfoury, D. Khoury, A. AlSabeh, J. Gomez, J. Crichigno, and E. Bou-Harb, "A Blockchain-based Method for Decentralizing the ACME Protocol to Enhance Trust in PKI," in *Proc. 43rd International Conference on Telecommunications and Signal Processing (TSP)*, 2020, pp. 461–465.
- **Baseline 2 (B-DCV)**: D. Khoury, P. Balian, and E. Kfoury, "Implementation of Blockchain Domain Control Verification (B-DCV)," in *Proc. 45th International Conference on Telecommunications and Signal Processing (TSP)*, 2022, pp. 17–22.

### 1. Common Environment

Both schemes were originally implemented on Ethereum, and the original implementation of Baseline 2 uses the Provable oracle service. To eliminate the influence of the underlying infrastructure, both baselines were ported to the same environment as BACME:

| Component | Setting (identical for BACME, Baseline 1, and Baseline 2) |
|---|---|
| Blockchain | FISCO-BCOS 2.11.0 (the same network) |
| Smart contract language | Solidity, compiled with solc 0.6.10 |
| Oracle | Truora 1.1.1 (the same VRF and oracle contracts) for random number generation and HTTP-based domain control validation |
| Client SDK | FISCO-BCOS Go SDK v1.1.1 |
| Load generator | FISCO-BCOS Java SDK 2.9.0 with a rate limiter |

Consequently, the performance differences among the three schemes stem from their protocol designs rather than from the underlying platforms.

### 2. Baseline 1

**Source files**

| File | Description |
|---|---|
| `contract/DACME.sol` | Smart contract |
| `baseline1/DACME/DACME.go` | Go binding of the contract |
| `baseline1/util/baseline1.go` | Client logic (deployment, application, update, verification) |
| `baseline1/util/x509.go` | RSA key pair and X.509 certificate generation |
| `baseline1/cmd/main.go` | Entry point for time measurement |

**On-chain record.** The contract maintains a record (`CertInfo`) for each domain, indexed by the domain name. The record contains:

| Field | Description |
|---|---|
| `domain` | Domain name |
| `PublicKey` | RSA public key in PEM format |
| `Author` | Address of the registering account |
| `N1` | Random number obtained from the VRF |
| `N2` | Random number generated by the client |
| `result` | Expected validation token |
| `Verified` | Verification flag |

**Functions for certificate management**

| Function | Behavior |
|---|---|
| `RegisterDomain(domain, publicKey)` | Registers a domain together with its public key and requests a random number `N1` from the VRF. |
| `StartValidation(domain, N2)` | Receives the random number `N2` generated by the client, computes the expected token `keccak256(N1 ‖ N2)`, and requests the oracle to fetch the token from the HTTP path `/well-known/pki-validation` of the domain. The domain is marked as verified if the fetched content matches the expected token. |
| `UpdateCertificate(domain, publicKey)` | Allows the registering account to replace the public key of a verified domain. |
| `VerifyCertificate(domain)` | Returns the public key bound to a domain. |

Baseline 1 does not provide a revocation mechanism, so revocation is not evaluated for Baseline 1.

### 3. Baseline 2

**Source files**

| File | Description |
|---|---|
| `contract/DCV.sol` | Smart contract |
| `baseline2/DCV/DCV.go` | Go binding of the contract |
| `baseline2/util/baseline2.go` | Client logic (deployment, application, update, revocation, verification) |
| `baseline2/util/x509.go` | RSA key pair and X.509 certificate generation |
| `baseline2/cmd/main.go` | Entry point for time measurement |

**On-chain record.** Baseline 2 follows the same validation workflow as Baseline 1. In addition, it stores the IP address and expiration date of each domain, and replaces the verification flag with a status field (`Status`), which indicates whether the domain is validated and not revoked. The record (`Domain`) contains `domain`, `PublicKey`, `IP`, `expirydate`, `Author`, `N1`, `N2`, `result`, and `Status`.

**Functions for certificate management**

| Function | Behavior |
|---|---|
| `registerDomain(domain, publicKey, IP, expirydate)` | Registers a domain together with its public key, IP address, and expiration date, and requests a random number `N1` from the VRF. |
| `StartValidation(domain, N2)` | Same as in Baseline 1, but only accepts requests from the registering account. |
| `modifyDomainConfig(domain, publicKey, IP)` | Allows the registering account to update the public key and IP address of a validated domain. |
| `getDomain(domain)` | Returns the public key bound to a domain. |
| `revokeCertificate(domain)` | Allows the registering account to mark a domain as revoked by setting its status to `false`. |

### 4. Clients and Evaluation

**Clients.** The ACME clients of both baselines were implemented in Go with the same FISCO-BCOS Go SDK as BACME. Each client generates a 2048-bit RSA key pair and a self-signed X.509 certificate. Since both baselines bind a single domain to a public key, a certificate covering *n* domains is managed through *n* independent registrations.

**Certificate verification.** The verification procedure retrieves the public key bound to the common name of the presented certificate from the blockchain, and checks that it matches the public key of the certificate and that the certificate signature is valid (see `Baseline1VerifyCertificate` and `Baseline2VerifyCertificate`).

**Throughput.** The throughput of all three schemes was measured using the same load generator based on the FISCO-BCOS Java SDK under identical request rates, as described in [Throughput Evaluation](#6-throughput-evaluation).

## Notes

- Some comments and log messages in the source code are written in Chinese.
- This is a research prototype and is not intended for production use.
