// SPDX-License-Identifier: MIT
pragma solidity ^0.6.0;
pragma experimental ABIEncoderV2;

import "./simplevrf/VRFClient.sol";
import "./FiscoOracleClient.sol";


contract DCV is FiscoOracleClient,VRFClient{
    uint256 seed = 0;
    struct Domain {
        string domain;
        string PublicKey;
        string IP;
        uint256 expirydate;
        address Author;
        uint256 N1;
        uint256 N2;
        string result;
        bool Status;
    }

    mapping(string=>Domain) certInfos;
    string private applyCertificateURL = "/well-known/pki-validation)";
    string private httpSchema="plain(http://";

    /*VRFClient*/
    bytes32 internal keyHash;
    struct randomValidInfo{
        bool valid;
        string domain;
    }
    mapping(bytes32=>randomValidInfo) private validIds;
    address private vrfCoreAddress;
    /*指定处理的oracle*/
    struct validInfo {
        bool valid;
        string domain;
    }
    address private oracleCoreAddress;
	uint256 private timesAmount = 1;
    mapping(bytes32=>validInfo) private httpValidIds;

    event RandomNumberN1(string domain, uint256 N1);
    event RegisterResult(string domain, address Author, string PublicKey);
    //keyhash, hash of vrfcore public key
    constructor(address _vrfCore, bytes32 _keyHash, address oracleAddress) public {
        vrfCoreAddress = _vrfCore;
        keyHash = _keyHash;
        oracleCoreAddress = oracleAddress;
    }

    function registerDomain(string memory domain, string memory publicKey, string memory IP, uint256 expirydate) public{
        require(certInfos[domain].Status == false,"The domain has been register!!!");
        certInfos[domain].domain = domain;
        certInfos[domain].PublicKey = publicKey;
        certInfos[domain].Author = msg.sender;
        certInfos[domain].IP = IP;
        certInfos[domain].expirydate = expirydate;
        requestRandomNumber(seed, domain);
        seed += 1;
    }

    function StartValidation(string memory domain,uint256 N2) public{
        require(certInfos[domain].Author == msg.sender,"Author is not matched!!!");
        string memory url = string(abi.encodePacked(httpSchema, domain , applyCertificateURL));
        certInfos[domain].N2 = N2;
        certInfos[domain].result = bytes32ToHexString(keccak256(abi.encodePacked(certInfos[domain].N1,certInfos[domain].N2)));
        request(url,domain);
    }

    function modifyDomainConfig(string memory domain, string memory publicKey, string memory IP) public{
        require(keccak256(abi.encodePacked(certInfos[domain].domain))==keccak256(abi.encodePacked(domain)) && certInfos[domain].Author == msg.sender && certInfos[domain].Status,"The certificate is not matched!!!");
        certInfos[domain].PublicKey = publicKey;
        certInfos[domain].IP = IP;
    }

    function getDomain(string memory domain) public view returns(string memory){
        require(keccak256(abi.encodePacked(certInfos[domain].domain))==keccak256(abi.encodePacked(domain)),"The certificate is not exist");
        return certInfos[domain].PublicKey;
    }

    function revokeCertificate(string memory domain) public {
        require(keccak256(abi.encodePacked(certInfos[domain].domain))==keccak256(abi.encodePacked(domain)) && certInfos[domain].Author == msg.sender && certInfos[domain].Status,"The certificate is not matched!!!");
        certInfos[domain].Status = false;
    }
    /**
     * Requests randomness from a user-provided seed
     */
    function requestRandomNumber(uint256 userProvidedSeed,string memory domain) private returns (bytes32 ) {
        bytes32  requestId =  vrfQuery(vrfCoreAddress, keyHash, userProvidedSeed);
        validIds[requestId].valid = true;
        validIds[requestId].domain = domain;
        return requestId;
    }

    /**
     * Callback function used by VRF core
     */
    function __callbackRandomness(bytes32 requestId, uint256 randomness) internal override {
        require(validIds[requestId].valid, "id must be not used!");
        certInfos[validIds[requestId].domain].N1=randomness;
        emit RandomNumberN1(validIds[requestId].domain,randomness);
        delete validIds[requestId];
    }

    /*
    * start http request
    */
        function request(string memory url,string memory domain) private returns (bytes32)
    {
        returnType = ReturnType.STRING;
        bytes32  requestId = oracleQuery(oracleCoreAddress, url, timesAmount, returnType);
        httpValidIds[requestId] = validInfo(true,domain);
        return requestId;

    }

    /**
     * Receive the response in the form of int256
     */
    function __callback(bytes32 _requestId, bytes memory _result) internal override
    {
        require(httpValidIds[_requestId].valid, "id must be not used!") ;
        if (compareHashWithHexString(certInfos[httpValidIds[_requestId].domain].result,string(_result))){
            certInfos[httpValidIds[_requestId].domain].Status = true;
            emit RegisterResult(httpValidIds[_requestId].domain,certInfos[httpValidIds[_requestId].domain].Author,certInfos[httpValidIds[_requestId].domain].PublicKey);
            delete httpValidIds[_requestId];
        } else{
            emit RegisterResult(certInfos[httpValidIds[_requestId].domain].result,certInfos[httpValidIds[_requestId].domain].Author,string(_result));
            delete httpValidIds[_requestId];
        }
    }

    function bytes32ToHexString(bytes32 _bytes32) private pure returns (string memory) {
        bytes memory hexString = new bytes(64);
        bytes memory alphabet = "0123456789abcdef";

        for (uint i = 0; i < 32; i++) {
            hexString[i * 2] = alphabet[uint(uint8(_bytes32[i] >> 4))];
            hexString[i * 2 + 1] = alphabet[uint(uint8(_bytes32[i] & 0x0f))];
        }

        return string(hexString);
    }

    function compareHashWithHexString(string memory computedHash, string memory hexStringHash) private pure returns (bool) {
        // Compare the strings
        return keccak256(abi.encodePacked(computedHash)) == keccak256(abi.encodePacked(hexStringHash));
    }
}