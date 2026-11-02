// SPDX-License-Identifier: MIT
pragma solidity ^0.6.0;
pragma experimental ABIEncoderV2;

import "./simplevrf/VRFClient.sol";
import "./FiscoOracleClient.sol";
import "./Table.sol";
import {LibString} from "./LibString.sol";

contract DpkiAcme is FiscoOracleClient,VRFClient {

    /*==============================================
        智能合约数据结构定义区域
    ==============================================*/
    KVTableFactory tableFactory;
    string constant TABLE_NAME1 = "id_control_right";
    string constant TABLE_NAME2 = "cert_meta_data";
    string constant TABLE_NAME3 = "random_valid_info";
    string constant TABLE_NAME4 = "http_valid_info";
    string constant TABLE_NAME5 = "id_control_validation";

    //VRF请求时的种子
    uint256 private seed = 0;

    // ACME控制权验证时使用的固定验证URL
    string private constant idValidationURL = "/acme/id-validation)";
    string private constant httpSchema="plain(http://";

    // VRF客户端&HTTP Oracle客户端
    bytes32 internal keyHash;
    address private vrfCoreAddress;
    address private oracleCoreAddress;

    // 合约事件
    event RandomToken(address managementAccount, uint256 randomToken);
    event ValidationResult(address managementAccount, string id, bool result);

    /*=========================================================
        合约初始化函数（构造函数）
        参数：
            _keyhash：vrfCore公钥的hash值
            _vrfCore：vrfCore合约地址
            oracleAddress：HTTP Oracle合约地址
    ===========================================================*/
    constructor(address _vrfCore, bytes32 _keyHash, address oracleAddress) public {
        require(_vrfCore!=address(0) && oracleAddress!=address(0), "E1");
        vrfCoreAddress = _vrfCore;
        keyHash = _keyHash;
        oracleCoreAddress = oracleAddress;
        tableFactory = KVTableFactory(0x1010);
        // ID控制权数据结构
        tableFactory.createTable(TABLE_NAME1, "id", "owner_wallet,validate_time");
        // 证书元数据的数据结构
        tableFactory.createTable(TABLE_NAME2, "cert_hash", "ids,cert_publicKey_hash,management_account,valid");
        // VRF的预言机订单结构
        tableFactory.createTable(TABLE_NAME3, "request_id","valid,management_account");
        // 控制权验证的预言机订单结构
        tableFactory.createTable(TABLE_NAME4,"request_id","valid,id,management_account");
        // 控制权验证的订单结构
        tableFactory.createTable(TABLE_NAME5,"wallet_address","ids,token1,token,time");
    }

    /*=========================================================
        具体的证书全生命周期管理合约函数
        函数名：
            controlValidate() --- 控制权验证
            certificateApply() --- 证书申请
            certificateVerify() --- 证书验证
            certificateUpdate() --- 证书更新-基于账号
            certificateRevoke() --- 证书撤销-基于账号
            certificateRevokeWithSK() --- 证书撤销-基于私钥验证
            certificateRevokeWithDC() --- 证书撤销-基于控制权验证
    =========================================================*/

    /*=========================================================
        控制权验证接口。
        参数：
            Ids：需要验证控制权的ID列表
    =========================================================*/
    function controlValidate(string memory Ids) public {
        KVTable idControlValidationTable = tableFactory.openTable(TABLE_NAME5);
        string memory walletAddress = LibString.addressToString(msg.sender);
        Entry idControlValidationEntry;
        bool ok;
        (ok, idControlValidationEntry) = idControlValidationTable.get(walletAddress);
        if (ok){
            // 如果该账号曾经进行过控制权验证，则判断上次验证请求是否已经超过10分钟，避免频繁验证DoS攻击。
            require((block.timestamp-idControlValidationEntry.getUInt("time")) >= 600000, "E2");
        }
        // 没有创建过控制权验证订单或上次验证已经过去10分钟，则创建和初始化控制权验证订单
        idControlValidationEntry = idControlValidationTable.newEntry();
        idControlValidationEntry.set("wallet_address", walletAddress);
        idControlValidationEntry.set("ids",Ids);
        idControlValidationEntry.set("time", block.timestamp);
        idControlValidationEntry.set("token1",uint256(0));
        idControlValidationEntry.set("token","");
        idControlValidationTable.set(walletAddress, idControlValidationEntry);

        // 请求随机token
        requestRandomNumber(seed, msg.sender);
        seed += 1;
    }

    /*=========================================================
        请求一个VRF随机数。
        参数：
            userProvidedSeed：种子
            managementAccount：申请控制验证的账号
    =========================================================*/
    function requestRandomNumber(uint256 userProvidedSeed, address managementAccount) internal {
        bytes32  requestId = vrfQuery(vrfCoreAddress, keyHash, userProvidedSeed);
        KVTable randomNumberTable = tableFactory.openTable(TABLE_NAME3);
        // 创建oracle随机数订单表
        Entry randomNumberEntry = randomNumberTable.newEntry();
        string memory requestIdString = LibString.bytes32ToString(requestId);
        randomNumberEntry.set("request_id", requestIdString);
        randomNumberEntry.set("valid",uint256(1));
        randomNumberEntry.set("management_account", managementAccount);
        randomNumberTable.set(requestIdString, randomNumberEntry);
    }

    /*=========================================================
        来自VRFCore的回调，写会随机数
        参数：
            requestId：VRF随机数请求时的请求ID
            randomness：VRF随机数
    =========================================================*/
    function __callbackRandomness(bytes32 requestId, uint256 randomness) internal override {
        // 查询requestID是否有效
        string memory requestIdString = LibString.bytes32ToString(requestId);
        KVTable randomNumberTable = tableFactory.openTable(TABLE_NAME3);
        Entry randomNumberEntry;
        bool ok;
        (ok, randomNumberEntry) = randomNumberTable.get(requestIdString);
        // 没有requestid记录，那么是非法的
        // 有reqeustid记录，但是状态是无效，那么也是非法的
        require(randomNumberEntry.getUInt("valid") == 1 && ok , "E3");

        // 记录随机token
        address managementAccount = randomNumberEntry.getAddress("management_account");
        KVTable idControlValidationTable = tableFactory.openTable(TABLE_NAME5);
        Entry idControlValidationEntry = idControlValidationTable.newEntry();
        idControlValidationEntry.set("token1", randomness);
        idControlValidationTable.set(LibString.addressToString(managementAccount),idControlValidationEntry);
        emit RandomToken(managementAccount,randomness);

        // 完成随机数订单，设置随机数订单有效位为0
        randomNumberEntry.set("valid",uint256(0));
        randomNumberTable.set(requestIdString,randomNumberEntry);
    }

    /*=========================================================
        开始ID控制权验证
        参数：
            Token2：ACME客户端生成的Token2
    =========================================================*/
    function startValidation(uint256 token2) public {
        // 缓解DOS攻击，如果token已经被设置就不能继续操作，即之前已经有过一次的http验证了
        string memory walletAddress = LibString.addressToString(msg.sender);
        bool ok;
        Entry idControlValidationEntry;
        KVTable idControlValidationTable = tableFactory.openTable(TABLE_NAME5);
        // 控制权订单不存在禁止继续
        (ok, idControlValidationEntry) = idControlValidationTable.get(walletAddress);
        // token已经被设置过，说明已经调用过startValidation，禁止重复调用
        require(ok && LibString.equals(idControlValidationEntry.getString("token"),""), "E4");

        // 获取id列表
        string[] memory ids_list = LibString.split(idControlValidationEntry.getString("ids"),",");

        // 计算出Token值，并记录到控制权验证订单中
        idControlValidationEntry.set("token", LibString.bytes32ToHexString(sha256(abi.encodePacked(idControlValidationEntry.getUInt("token1"), token2))));
        idControlValidationTable.set(walletAddress,idControlValidationEntry);

        // 对每一个ID进行http验证
        for (uint256 i = 0; i < ids_list.length; i++){
            string memory url = string(abi.encodePacked(httpSchema, ids_list[i], idValidationURL));
            request(url, ids_list[i], msg.sender);
        }
    }

    /*=========================================================
        HTTP 验证请求(调用预言机)
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
        接收来自Oracle的反馈
    =========================================================*/
    function __callback(bytes32 _requestId, bytes memory _result) internal override
    {
        // 检查requestID是否合法
        string memory requestIdString = LibString.bytes32ToString(_requestId);
        bool ok;
        Entry httpValidEntry;
        KVTable httpValidTable = tableFactory.openTable(TABLE_NAME4);
        (ok, httpValidEntry) = httpValidTable.get(requestIdString);
        require(httpValidEntry.getUInt("valid") == 1 && ok, "E5");
        string memory id = httpValidEntry.getString("id");
        address managementAccount = httpValidEntry.getAddress("management_account");

        // 打开控制权验证订单，查询其中的token
        Entry idControlValidationEntry;
        (ok, idControlValidationEntry) = tableFactory.openTable(TABLE_NAME5).get(LibString.addressToString(managementAccount));
        require(ok, "E6");
        string memory token = idControlValidationEntry.getString("token");

        //打开id控制权记录表
        KVTable idControlRightTable = tableFactory.openTable(TABLE_NAME1);
        Entry idControlRightEntry = idControlRightTable.newEntry();

        // 检查控制权验证
        if(LibString.compareHashWithHexString(token, string(_result))){
            // 添加控制权信息记录
            idControlRightEntry.set("id", id);
            idControlRightEntry.set("validate_time", block.timestamp);
            idControlRightEntry.set("owner_wallet", managementAccount);
            idControlRightTable.set(id,idControlRightEntry);

            // 清理http请求订单信息
            httpValidEntry.set("valid", uint256(0));
            httpValidTable.set(requestIdString,httpValidEntry);
            emit ValidationResult(managementAccount,  id,true);
        }
        else{
            //失败则告知http验证失败
            emit ValidationResult(managementAccount, id, false);
        }
    }

    /*=========================================================
        证书申请接口
        参数：
            Ids：需要保护的所有域名
            CertHash：证书hash
        返回值：false（失败，控制权不具有）, true（成功，证书申请成功）
    =========================================================*/
    function applyCertificate(string memory Ids, string memory CertHash, address CertPKHash) public returns(bool) {
        // 分割ids为列表
        string[] memory idList = LibString.split(Ids,",");
        bool ok;
        Entry idControlRightEntry;
        // 所有ID的控制权属于申请者账户，且在有效期30日内
        for(uint256 i=0; i<idList.length; i++){
            (ok, idControlRightEntry) = tableFactory.openTable(TABLE_NAME1).get(idList[i]);
            // 记录存在，检查记录是否合规。必须在有效期内，且控制权归属于申请者。记录不存在直接返回失败。
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

        // 创建证书元数据
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
        检查需要操作的证书是否是有效的证书且属于请求账户
        参数：
            oldCertHash：需要操作的证书hash值
            certMetaDataTable：证书元数据表
        返回值：
            （有效状态，所有ID）
    */
    function isValidCert(string memory oldCertHash, KVTable certMetaDataTable) private view returns(bool, string memory){
        bool ok;
        Entry certMetaDataEntry;
        (ok, certMetaDataEntry) = certMetaDataTable.get(oldCertHash);
        if (ok){
            // 检索到证书元数据，需要证书管理账户属于请求者，且证书状态有效
            if (certMetaDataEntry.getAddress("management_account")==msg.sender && certMetaDataEntry.getUInt("valid") == 1){
                return (true,certMetaDataEntry.getString("ids"));
            }
            else{
                return (false,"");
            }
        }
        else{
            // 检索不到证书元数据，直接无效
            return (false,"");
        }

    }


    /*=========================================================
        证书更新接口
        参数：
            oldCertHash：旧证书hash值
            CertHash：新证书hash值
            CertPKHash：新证书公钥hash值
        返回值：false（失败，证书管理账户不匹配）, true（成功，证书申请成功）
    =========================================================*/
    function updateCertificate(string memory oldCertHash, string memory CertHash, address CertPKHash) public returns(bool){
        bool ok;
        string memory ids;
        KVTable certMetaDataTable = tableFactory.openTable(TABLE_NAME2);
        (ok,ids) = isValidCert(oldCertHash,certMetaDataTable);
        // 如果证书有效且管理地址都是请求者地址，就直接更新信息。
        if (ok){
            // 创建新的证书元数据
            Entry certMetaDataEntry = certMetaDataTable.newEntry();
            certMetaDataEntry.set("cert_hash", CertHash);
            certMetaDataEntry.set("ids", ids);
            certMetaDataEntry.set("cert_publicKey_hash", CertPKHash);
            certMetaDataEntry.set("management_account", msg.sender);
            certMetaDataEntry.set("valid", uint256(1));
            certMetaDataTable.set(CertHash, certMetaDataEntry);

            //设置旧的证书元数据为无效
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
        证书撤销接口
        参数：
            oldCertHash：待撤销的旧证书hash值
        返回值：false（失败，证书管理账户不匹配）, true（成功，证书申请成功）
    =========================================================*/
    function revokeCertificate(string memory oldCertHash) public returns(bool){
        // 证书属于该请求账户且是有效证书，就从证书元数据表中删除该证书
        bool ok;
        string memory ids;
        KVTable certMetaDataTable = tableFactory.openTable(TABLE_NAME2);
        (ok,ids) = isValidCert(oldCertHash,certMetaDataTable);
        // 修改旧证书状态为无效
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
        通过私钥证明来撤销证书
        参数：
            oldCertHash：待撤销的证书hash
            v，r, s：签名
    =========================================================*/
    function revokeCertificateWithPrivateKey(string memory oldCertHash, uint8 v, bytes32 r, bytes32 s) public returns(bool){
        //验证签名
        bytes32 messageHash = keccak256(abi.encodePacked(oldCertHash));
        address recoveredAddress = ecrecover(messageHash, v, r, s);
        Entry certMetaDataEntry;
        bool ok;
        KVTable certMetaDataTable = tableFactory.openTable(TABLE_NAME2);
        (ok, certMetaDataEntry) = certMetaDataTable.get(oldCertHash);
        uint256 valid = certMetaDataEntry.getUInt("valid");
        require(ok && valid == 1, "E7");
        // 如果签名验证成功，就撤销证书
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
        通过控制权验证撤销证书
    =========================================================*/
    function revokeCertificateWithDV(string memory oldCertHash) public returns(bool){
        bool ok;
        Entry idControlRightEntry;
        Entry certMetaDataEntry;
        KVTable certMetaDataTable = tableFactory.openTable(TABLE_NAME2);
        // 证书如果不存在或本就无效，无需撤销
        (ok, certMetaDataEntry) = certMetaDataTable.get(oldCertHash);
        require(ok && certMetaDataEntry.getUInt("valid") == 1, "E8");
        // 获取证书元数据中的ids，并分割为数组
        string[] memory idList = LibString.split(certMetaDataEntry.getString("ids"),",");
        for(uint256 i = 0; i < idList.length; i++){
            (ok, idControlRightEntry) = tableFactory.openTable(TABLE_NAME1).get(idList[i]);
            // 记录存在，检查记录是否合规。记录不存在直接返回失败。
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
        // 所有ID控制权检查都通过了，撤销证书
        certMetaDataEntry.set("valid", uint256(0));
        certMetaDataTable.set(oldCertHash, certMetaDataEntry);
        return true;
    }

    /*=========================================================
        证书验证
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