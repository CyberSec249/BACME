# BACME: Blockchain-based Automatic Certificate Management Environment

This repository contains the prototype implementation of **BACME**, together with the reproduced baselines and the performance evaluation tools used in the paper:

> L. Li, Z. Yan, C. Shi, and X. Liang, "BACME: Blockchain-based Automatic Certificate Management Environment."

BACME is a consortium blockchain-based ACME scheme. It manages domain control validation, certificate application, renewal, revocation, and verification through smart contracts, stores complete certificates in decentralized off-chain storage (IPFS), and supports both single-domain and multi-domain certificates.

## Environment

The prototype was evaluated with the following software:

| Component | Version |
|---|---|
| FISCO-BCOS | 2.11.0 |
| Solidity compiler (solc) | 0.6.10 |
| Truora oracle | 1.1.1 |
| IPFS | Private IPFS cluster (4 nodes in our evaluation) |
| Go | 1.22 or later (as required by the `go.mod` files) |
| FISCO-BCOS Go SDK | v1.1.1 |
| Echo framework | 4.13.3 |
| Java / Gradle | Java 8 / Gradle 6.3 (Gradle wrapper included) |
| FISCO-BCOS Java SDK | 2.9.0 |
| Node.js | Required for the web UI (Vite 6) |

In our evaluation, all components were deployed on a single server with 32 CPU cores, 128 GB of RAM, and 1 TB of storage.

## Configuration

Credentials are **not** included in this repository. Before running any component, provide the following files in the corresponding `conf/` directory (`acmeClient/conf`, `deployContract/conf`, `baseline1/conf`, `baseline2/conf`, and `performanceTest/java-sdk-demo/dist/conf`):

| File | Description |
|---|---|
| `ca.crt`, `sdk.crt`, `sdk.key` | SDK certificates of your FISCO-BCOS chain (generated when the chain is built) |
| `accounts/<address>.pem` | Private key of the blockchain account (can be generated with `conf/get_account.sh`) |
| `config.toml` | Node address, group ID, chain ID, and account key file |
| `contractAddress` | Address of the deployed smart contract (written after deployment) |

In addition, the following addresses are hard-coded and must be adapted to your deployment (due to confidentiality requirements, the acmeClient and webui directories are not publicly available at this time):

| Location | Setting |
|---|---|
| `acmeClient/util/CertificateManagement.go` | IPFS API endpoint (`http://192.168.136.128:9001`) |
| `acmeClient/util/fiscoClient.go`, `baseline1/util/baseline1.go`, `baseline2/util/baseline2.go` | Addresses of the Truora `VRFCore` and `OracleCore` contracts and the hash of the VRF public key |
| `webui/src/components/*.vue` | Back-end API address (`http://localhost:8080`) |

## Usage

### 1. Prerequisites

1. Build a FISCO-BCOS 2.11.0 chain.
2. Deploy Truora 1.1.1 on the chain, and record the addresses of the `VRFCore` and `OracleCore` contracts and the hash of the VRF public key.
3. Start an IPFS node or cluster.

### 2. Deploy the BACME Smart Contract

```bash
cd deployContract
go run . -v <VRFCore address> -h <VRF public key hash> -o <OracleCore address>
```

Write the printed contract address to `acmeClient/conf/contractAddress`.

### 3. Run the ACME Client

```bash
cd acmeClient
go run ./cmd/main.go      # starts the back-end API server on port 8080
```

The client serves the HTTP validation token through a built-in file server on port 80 (`./validation/acme/id-validation`) or through a local Nginx web root (`/var/www/html/acme/id-validation`).

`acmeClient/cmd/main2.go` runs the complete certificate lifecycle without the web UI: contract deployment, domain control validation, certificate application, renewal, the three revocation methods, and certificate verification.

### 4. Run the Web UI

```bash
cd webui
npm install
npm run dev
```

### 5. Run the Baselines

Each baseline has a deployment function (`DeployDACMEContract` in `baseline1/util/baseline1.go` and `DeployDCVContract` in `baseline2/util/baseline2.go`) and functions for each certificate management operation. Select the operation to be measured in `cmd/main.go` and run:

```bash
cd baseline1            # or baseline2
go run ./cmd/main.go
```

### 6. Throughput Evaluation

```bash
cd performanceTest/java-sdk-demo
./gradlew build
cd dist
bash dpkiacme.sh        # BACME
bash dacme.sh           # Baseline 1
bash dcv.sh             # Baseline 2
```

Each script invokes the load generator as follows:

```bash
java -cp 'conf/:lib/*:apps/*' org.fisco.bcos.sdk.demo.perf.<Class> <operation> <count> <tps> <groupId>
```

The arguments are the operation, the total number of transactions, the target rate (transactions per second), and the group ID. In our evaluation, each script runs every operation five times with 10,000 transactions at a target rate of 10,000 transactions per second, e.g.:

```bash
java -cp 'conf/:lib/*:apps/*' org.fisco.bcos.sdk.demo.perf.PerformanceDCV apply 10000 10000 1
```

| Scheme | Class | Operations |
|---|---|---|
| BACME | `PerformanceDpkiAcme` | `validation`, `apply`, `update`, `revoke`, `revokeP`, `revokeDV`, `verify` |
| Baseline 1 | `PerformanceDACME` | `apply`, `update`, `verify` |
| Baseline 2 | `PerformanceDCV` | `apply`, `update`, `revoke`, `verify` |

If the build reports a Google Java Format error, run `./gradlew googleJavaFormat` first.

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
