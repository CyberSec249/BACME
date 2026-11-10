// SPDX-License-Identifier: MIT
pragma solidity ^0.6.0;
pragma experimental ABIEncoderV2;

import "./simplevrf/VRFClient.sol";
import "./FiscoOracleClient.sol";
import "./Table.sol";
import {LibString} from "./LibString.sol";

contract DpkiAcme is FiscoOracleClient,VRFClient {

    /*==============================================
        Data structure definitions of the smart contract
    ==============================================*/
    KVTableFactory tableFactory;
    string constant TABLE_NAME1 = "id_control_right";
    string constant TABLE_NAME2 = "cert_meta_data";
    string constant TABLE_NAME3 = "random_valid_info";
    string constant TABLE_NAME4 = "http_valid_info";
    string constant TABLE_NAME5 = "id_control_validation";

    // Seed for VRF requests
    uint256 private seed = 0;

    // Fixed validation URL used for ACME domain control validation
    string private constant idValidationURL = "/acme/id-validation)";
    string private constant httpSchema="plain(http://";

    // VRF client & HTTP oracle client
    bytes32 internal keyHash;
    address private vrfCoreAddress;
    address private oracleCoreAddress;

    // Contract events
    event RandomToken(address managementAccount, uint256 randomToken);
    event ValidationResult(address managementAccount, string id, bool result);

    /*=========================================================
        Contract initialization (constructor)
        Parameters:
            _keyhash: hash of the vrfCore public key
            _vrfCore: address of the vrfCore contract
            oracleAddress: address of the HTTP oracle contract
    ===========================================================*/
    constructor(address _vrfCore, bytes32 _keyHash, address oracleAddress) public {
        require(_vrfCore!=address(0) && oracleAddress!=address(0), "E1");
        vrfCoreAddress = _vrfCore;
        keyHash = _keyHash;
        oracleCoreAddress = oracleAddress;
        tableFactory = KVTableFactory(0x1010);
        // Data structure of an ID control right
        tableFactory.createTable(TABLE_NAME1, "id", "owner_wallet,validate_time");
        // Data structure of certificate metadata
        tableFactory.createTable(TABLE_NAME2, "cert_hash", "ids,cert_publicKey_hash,management_account,valid");
        // Oracle request structure for VRF
        tableFactory.createTable(TABLE_NAME3, "request_id","valid,management_account");
        // Oracle request structure for control validation
        tableFactory.createTable(TABLE_NAME4,"request_id","valid,id,management_account");
        // Order structure for control validation
        tableFactory.createTable(TABLE_NAME5,"wallet_address","ids,token1,token,time");
    }

    /*=========================================================
        Contract functions for full certificate lifecycle management
        Functions:
            controlValidate() --- domain control validation
            certificateApply() --- certificate application
            certificateVerify() --- certificate verification
            certificateUpdate() --- certificate renewal (account-based)
            certificateRevoke() --- certificate revocation (account-based)
            certificateRevokeWithSK() --- certificate revocation (private key proof)
            certificateRevokeWithDC() --- certificate revocation (domain control validation)
    =========================================================*/

    /*=========================================================
        Domain control validation interface.
        Parameters:
            Ids: list of IDs whose control right is to be validated
    =========================================================*/
    function controlValidate(string memory Ids) public {
        KVTable idControlValidationTable = tableFactory.openTable(TABLE_NAME5);
        string memory walletAddress = LibString.addressToString(msg.sender);
        Entry idControlValidationEntry;
        bool ok;
        (ok, idControlValidationEntry) = idControlValidationTable.get(walletAddress);
        if (ok){
            // If this account has requested validation before, check that more than 10 minutes have passed since the last request, to prevent DoS through frequent validation.
            require((block.timestamp-idControlValidationEntry.getUInt("time")) >= 600000, "E2");
        }
        // If no validation order exists or 10 minutes have passed since the last one, create and initialize a validation order
        idControlValidationEntry = idControlValidationTable.newEntry();
        idControlValidationEntry.set("wallet_address", walletAddress);
        idControlValidationEntry.set("ids",Ids);
        idControlValidationEntry.set("time", block.timestamp);
        idControlValidationEntry.set("token1",uint256(0));
        idControlValidationEntry.set("token","");
        idControlValidationTable.set(walletAddress, idControlValidationEntry);

        // Request a random token
        requestRandomNumber(seed, msg.sender);
        seed += 1;
    }

    /*=========================================================
        Request a VRF random number.
        Parameters:
            userProvidedSeed: seed
            managementAccount: account requesting the control validation
    =========================================================*/
    function requestRandomNumber(uint256 userProvidedSeed, address managementAccount) internal {
        bytes32  requestId = vrfQuery(vrfCoreAddress, keyHash, userProvidedSeed);
        KVTable randomNumberTable = tableFactory.openTable(TABLE_NAME3);
        // Create the oracle randomness order table
        Entry randomNumberEntry = randomNumberTable.newEntry();
        string memory requestIdString = LibString.bytes32ToString(requestId);
        randomNumberEntry.set("request_id", requestIdString);
        randomNumberEntry.set("valid",uint256(1));
        randomNumberEntry.set("management_account", managementAccount);
        randomNumberTable.set(requestIdString, randomNumberEntry);
    }

    /*=========================================================
        Callback from VRFCore that writes back the random number
        Parameters:
            requestId: request ID of the VRF randomness request
            randomness: VRF random number
    =========================================================*/
    function __callbackRandomness(bytes32 requestId, uint256 randomness) internal override {
        // Check whether the requestID is valid
        string memory requestIdString = LibString.bytes32ToString(requestId);
        KVTable randomNumberTable = tableFactory.openTable(TABLE_NAME3);
        Entry randomNumberEntry;
        bool ok;
        (ok, randomNumberEntry) = randomNumberTable.get(requestIdString);
        // No record of the requestId: invalid
        // A record of the requestId exists but is inactive: also invalid
        require(randomNumberEntry.getUInt("valid") == 1 && ok , "E3");

        // Record the random token
        address managementAccount = randomNumberEntry.getAddress("management_account");
        KVTable idControlValidationTable = tableFactory.openTable(TABLE_NAME5);
        Entry idControlValidationEntry = idControlValidationTable.newEntry();
        idControlValidationEntry.set("token1", randomness);
        idControlValidationTable.set(LibString.addressToString(managementAccount),idControlValidationEntry);
        emit RandomToken(managementAccount,randomness);

        // Complete the randomness order by setting its valid flag to 0
        randomNumberEntry.set("valid",uint256(0));
        randomNumberTable.set(requestIdString,randomNumberEntry);
    }

    /*=========================================================
        Start ID control validation
        Parameters:
            Token2: Token2 generated by the ACME client
    =========================================================*/
    function startValidation(uint256 token2) public {
        // Mitigate DoS: if the token is already set (i.e., an HTTP validation has already been started), do not continue
        string memory walletAddress = LibString.addressToString(msg.sender);
        bool ok;
        Entry idControlValidationEntry;
        KVTable idControlValidationTable = tableFactory.openTable(TABLE_NAME5);
        // Abort if the control validation order does not exist
        (ok, idControlValidationEntry) = idControlValidationTable.get(walletAddress);
        // If the token has been set, startValidation has already been called; repeated calls are not allowed
        require(ok && LibString.equals(idControlValidationEntry.getString("token"),""), "E4");

        // Get the ID list
        string[] memory ids_list = LibString.split(idControlValidationEntry.getString("ids"),",");

        // Compute the token and record it in the control validation order
        idControlValidationEntry.set("token", LibString.bytes32ToHexString(sha256(abi.encodePacked(idControlValidationEntry.getUInt("token1"), token2))));
        idControlValidationTable.set(walletAddress,idControlValidationEntry);

        // Perform HTTP validation for each ID
        for (uint256 i = 0; i < ids_list.length; i++){
            string memory url = string(abi.encodePacked(httpSchema, ids_list[i], idValidationURL));
            request(url, ids_list[i], msg.sender);
        }
    }

    /*=========================================================
        HTTP validation request (calls the oracle)
     =========================================================*/
    function request(string memory url, string memory id, address managementAccount) internal
    {
        returnType = ReturnType.STRING;
        bytes32  requestId = oracleQuery(oracleCoreAddress, url, 0, returnType);
        KVTable httpValidTable = tableFactory.openTable(TABLE_NAME4);
        Entry httpValidEntry = httpValidTable.newEntry();
        string memory requestIdString = LibString.bytes32ToString(requestId);
        httpValidEntry.set("request_id", requestIdString);
        httpValidEntry.set("valid", uint256(1));
        httpValidEntry.set("id", id);
        httpValidEntry.set("management_account", managementAccount);
        httpValidTable.set(requestIdString, httpValidEntry);
    }

    /*=========================================================
        Receives the response from the oracle
    =========================================================*/
    function __callback(bytes32 _requestId, bytes memory _result) internal override
    {
        // Check whether the requestID is valid
        string memory requestIdString = LibString.bytes32ToString(_requestId);
        bool ok;
        Entry httpValidEntry;
        KVTable httpValidTable = tableFactory.openTable(TABLE_NAME4);
        (ok, httpValidEntry) = httpValidTable.get(requestIdString);
        require(httpValidEntry.getUInt("valid") == 1 && ok, "E5");
        string memory id = httpValidEntry.getString("id");
        address managementAccount = httpValidEntry.getAddress("management_account");

        // Open the control validation order and read its token
        Entry idControlValidationEntry;
        (ok, idControlValidationEntry) = tableFactory.openTable(TABLE_NAME5).get(LibString.addressToString(managementAccount));
        require(ok, "E6");
        string memory token = idControlValidationEntry.getString("token");

        // Open the ID control right table
        KVTable idControlRightTable = tableFactory.openTable(TABLE_NAME1);
        Entry idControlRightEntry = idControlRightTable.newEntry();

        // Check the control validation
        if(LibString.compareHashWithHexString(token, string(_result))){
            // Add the control right record
            idControlRightEntry.set("id", id);
            idControlRightEntry.set("validate_time", block.timestamp);
            idControlRightEntry.set("owner_wallet", managementAccount);
            idControlRightTable.set(id,idControlRightEntry);

            // Clean up the HTTP request order
            httpValidEntry.set("valid", uint256(0));
            httpValidTable.set(requestIdString,httpValidEntry);
            emit ValidationResult(managementAccount,  id,true);
        }
        else{
            // On failure, report that HTTP validation failed
            emit ValidationResult(managementAccount, id, false);
        }
    }

    /*=========================================================
        Certificate application interface
        Parameters:
            Ids: all domains to be protected
            CertHash: certificate hash
        Returns: false (failure, control right not held), true (success, certificate applied)
    =========================================================*/
    function applyCertificate(string memory Ids, string memory CertHash, address CertPKHash) public returns(bool) {
        // Split ids into a list
        string[] memory idList = LibString.split(Ids,",");
        bool ok;
        Entry idControlRightEntry;
        // The control rights of all IDs must belong to the applicant account and be within the 30-day validity period
        for(uint256 i=0; i<idList.length; i++){
            (ok, idControlRightEntry) = tableFactory.openTable(TABLE_NAME1).get(idList[i]);
            // If the record exists, check that it is within the validity period and belongs to the applicant. If it does not exist, fail.
            if (ok) {
                address ownerWallet = idControlRightEntry.getAddress("owner_wallet");
                uint passTime = block.timestamp-idControlRightEntry.getUInt("validate_time");
                if(ownerWallet!=msg.sender || passTime >= 2592000000){
                    return false;
                }
            }
            else{
                return false;
            }
        }

        // Create the certificate metadata
        KVTable certMetaDataTable = tableFactory.openTable(TABLE_NAME2);
        Entry certMetaDataEntry = certMetaDataTable.newEntry();
        certMetaDataEntry.set("cert_hash", CertHash);
        certMetaDataEntry.set("ids", Ids);
        certMetaDataEntry.set("cert_publicKey_hash", CertPKHash);
        certMetaDataEntry.set("management_account", msg.sender);
        certMetaDataEntry.set("valid",uint256(1));
        certMetaDataTable.set(CertHash, certMetaDataEntry);
        return true;
    }

    /*
        Check whether the target certificate is valid and belongs to the requesting account
        Parameters:
            oldCertHash: hash of the target certificate
            certMetaDataTable: certificate metadata table
        Returns:
            (validity status, all IDs)
    */
    function isValidCert(string memory oldCertHash, KVTable certMetaDataTable) private view returns(bool, string memory){
        bool ok;
        Entry certMetaDataEntry;
        (ok, certMetaDataEntry) = certMetaDataTable.get(oldCertHash);
        if (ok){
            // If the metadata is found, the management account must be the requester and the certificate must be valid
            if (certMetaDataEntry.getAddress("management_account")==msg.sender && certMetaDataEntry.getUInt("valid") == 1){
                return (true,certMetaDataEntry.getString("ids"));
            }
            else{
                return (false,"");
            }
        }
        else{
            // If the metadata is not found, the certificate is invalid
            return (false,"");
        }

    }


    /*=========================================================
        Certificate renewal interface
        Parameters:
            oldCertHash: hash of the old certificate
            CertHash: hash of the new certificate
            CertPKHash: hash of the new certificate's public key
        Returns: false (failure, management account mismatch), true (success, certificate renewed)
    =========================================================*/
    function updateCertificate(string memory oldCertHash, string memory CertHash, address CertPKHash) public returns(bool){
        bool ok;
        string memory ids;
        KVTable certMetaDataTable = tableFactory.openTable(TABLE_NAME2);
        (ok,ids) = isValidCert(oldCertHash,certMetaDataTable);
        // If the certificate is valid and its management address is the requester, update it directly.
        if (ok){
            // Create the new certificate metadata
            Entry certMetaDataEntry = certMetaDataTable.newEntry();
            certMetaDataEntry.set("cert_hash", CertHash);
            certMetaDataEntry.set("ids", ids);
            certMetaDataEntry.set("cert_publicKey_hash", CertPKHash);
            certMetaDataEntry.set("management_account", msg.sender);
            certMetaDataEntry.set("valid", uint256(1));
            certMetaDataTable.set(CertHash, certMetaDataEntry);

            // Mark the old certificate metadata as invalid
            certMetaDataEntry = certMetaDataTable.newEntry();
            certMetaDataEntry.set("valid", uint256(0));
            certMetaDataTable.set(oldCertHash,certMetaDataEntry);
            return true;
        }
        else{
            return false;
        }
    }

    /*=========================================================
        Certificate revocation interface
        Parameters:
            oldCertHash: hash of the certificate to revoke
        Returns: false (failure, management account mismatch), true (success, certificate renewed)
    =========================================================*/
    function revokeCertificate(string memory oldCertHash) public returns(bool){
        // If the certificate belongs to the requesting account and is valid, revoke it in the certificate metadata table
        bool ok;
        string memory ids;
        KVTable certMetaDataTable = tableFactory.openTable(TABLE_NAME2);
        (ok,ids) = isValidCert(oldCertHash,certMetaDataTable);
        // Mark the old certificate as invalid
        if (ok){
            Entry certMetaDataEntry = certMetaDataTable.newEntry();
            certMetaDataEntry.set("valid",uint256(0));
            certMetaDataTable.set(oldCertHash,certMetaDataEntry);
            return true;
        }
        else{
            return false;
        }
    }

    /*=========================================================
        Revoke a certificate with a private key proof
        Parameters:
            oldCertHash: hash of the certificate to revoke
            v, r, s: signature
    =========================================================*/
    function revokeCertificateWithPrivateKey(string memory oldCertHash, uint8 v, bytes32 r, bytes32 s) public returns(bool){
        // Verify the signature
        bytes32 messageHash = keccak256(abi.encodePacked(oldCertHash));
        address recoveredAddress = ecrecover(messageHash, v, r, s);
        Entry certMetaDataEntry;
        bool ok;
        KVTable certMetaDataTable = tableFactory.openTable(TABLE_NAME2);
        (ok, certMetaDataEntry) = certMetaDataTable.get(oldCertHash);
        uint256 valid = certMetaDataEntry.getUInt("valid");
        require(ok && valid == 1, "E7");
        // If the signature is valid, revoke the certificate
        if (recoveredAddress == certMetaDataEntry.getAddress("cert_publicKey_hash")){
            certMetaDataEntry.set("valid", uint256(0));
            certMetaDataTable.set(oldCertHash, certMetaDataEntry);
            return true;
        }
        else{
            return false;
        }
    }

    /*=========================================================
        Revoke a certificate through domain control validation
    =========================================================*/
    function revokeCertificateWithDV(string memory oldCertHash) public returns(bool){
        bool ok;
        Entry idControlRightEntry;
        Entry certMetaDataEntry;
        KVTable certMetaDataTable = tableFactory.openTable(TABLE_NAME2);
        // If the certificate does not exist or is already invalid, there is nothing to revoke
        (ok, certMetaDataEntry) = certMetaDataTable.get(oldCertHash);
        require(ok && certMetaDataEntry.getUInt("valid") == 1, "E8");
        // Get the ids from the certificate metadata and split them into an array
        string[] memory idList = LibString.split(certMetaDataEntry.getString("ids"),",");
        for(uint256 i = 0; i < idList.length; i++){
            (ok, idControlRightEntry) = tableFactory.openTable(TABLE_NAME1).get(idList[i]);
            // If the record exists, check that it is compliant. If it does not exist, fail.
            if (ok) {
                uint passTime = block.timestamp-idControlRightEntry.getUInt("validate_time");
                if(idControlRightEntry.getAddress("owner_wallet")!=msg.sender || passTime >= 2592000000){
                    return false;
                }
            }
            else{
                return false;
            }
        }
        // All ID control right checks passed; revoke the certificate
        certMetaDataEntry.set("valid", uint256(0));
        certMetaDataTable.set(oldCertHash, certMetaDataEntry);
        return true;
    }

    /*=========================================================
        Certificate verification
    =========================================================*/
    function verifyCertificate(string memory CertHash) view public returns(string memory, address, bool) {
        Entry certMetaDataEntry;
        bool ok;
        (ok, certMetaDataEntry) = tableFactory.openTable(TABLE_NAME2).get(CertHash);
        if (ok) {
            if (certMetaDataEntry.getUInt("valid") == 1){
                return (certMetaDataEntry.getString("ids"), certMetaDataEntry.getAddress("cert_publicKey_hash"), true);
            }
            else{
                return ("", address(0), false);
            }
        }
        else{
            return ("", address(0), false);
        }
    }
}