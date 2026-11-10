// SPDX-License-Identifier: MIT

pragma solidity ^0.6.0;

library LibString {

    using LibString for *;

    /*=========================================================
       String comparison
    =========================================================*/
    function equals(string memory _self, string memory _str) internal pure returns (bool _ret) {
        if (bytes(_self).length != bytes(_str).length) {
            return false;
        }

        for (uint i=0; i<bytes(_self).length; ++i) {
            if (bytes(_self)[i] != bytes(_str)[i]) {
                return false;
            }
        }

        return true;
    }

    /*=========================================================
        Split a string into an array
    =========================================================*/
    function split(string memory src, string memory separator) internal pure returns (string[] memory splitArr) {
        bytes memory srcBytes = bytes(src);

        uint offset = 0;
        uint splitsCount = 1;
        int limit = -1;
        while (offset < srcBytes.length - 1) {
            limit = indexOf(src, separator, offset);
            if (limit == -1)
                break;
            else {
                splitsCount++;
                offset = uint(limit) + 1;
            }
        }

        splitArr = new string[](splitsCount);

        offset = 0;
        splitsCount = 0;
        while (offset < srcBytes.length - 1) {

            limit = indexOf(src, separator, offset);
            if (limit == - 1) {
                limit = int(srcBytes.length);
            }

            string memory tmp = new string(uint(limit) - offset);
            bytes memory tmpBytes = bytes(tmp);

            uint j = 0;
            for (uint i = offset; i < uint(limit); i++) {
                tmpBytes[j++] = srcBytes[i];
            }
            offset = uint(limit) + 1;
            splitArr[splitsCount++] = string(tmpBytes);
        }
        return splitArr;
    }

    /*=========================================================
        Character position
    =========================================================*/
    function indexOf(string  memory src, string memory value, uint offset)
    internal
    pure
    returns (int) {
        bytes memory srcBytes = bytes(src);
        bytes memory valueBytes = bytes(value);

        assert(valueBytes.length == 1);

        for (uint i = offset; i < srcBytes.length; i++) {
            if (srcBytes[i] == valueBytes[0]) {
                return int(i);
            }
        }

        return -1;
    }

    /*=========================================================
        Bytes to hexadecimal string
    =========================================================*/
    function bytes32ToHexString(bytes32 _bytes32) internal pure returns (string memory) {
        bytes memory hexString = new bytes(64);
        bytes memory alphabet = "0123456789abcdef";

        for (uint i = 0; i < 32; i++) {
            hexString[i * 2] = alphabet[uint(uint8(_bytes32[i] >> 4))];
            hexString[i * 2 + 1] = alphabet[uint(uint8(_bytes32[i] & 0x0f))];
        }

        return string(hexString);
    }

    /*=========================================================
    Utility: convert an address to a string
    Parameters:
        _address: address
    =========================================================*/
    function addressToString(address _address) internal pure returns (string memory) {
        bytes memory addressBytes = abi.encodePacked(_address);
        bytes memory hexString = new bytes(42); // 40 characters + "0x"
        hexString[0] = '0';
        hexString[1] = 'x';

        for (uint i = 0; i < 20; i++) {
            uint8 byteValue = uint8(addressBytes[i]);
            hexString[2 + i * 2] = _byteToHexChar(byteValue / 16);
            hexString[3 + i * 2] = _byteToHexChar(byteValue % 16);
        }

        return string(hexString);
    }

    /*=========================================================
        Byte -> hexadecimal character
    =========================================================*/
    function _byteToHexChar(uint8 _byte) internal pure returns (bytes1) {
        if (_byte < 10) {
            return bytes1(_byte + 48); // 48 is the ASCII code of '0'
        } else {
            return bytes1(_byte + 87); // 10 + 87 = 97 is the ASCII code of 'a'
        }
    }

    /*=========================================================
     Convert bytes to a string.
     Parameters:
         _bytes32: bytes
    =========================================================*/
    function bytes32ToString(bytes32 _bytes32) internal pure returns(string memory){
        bytes memory bytesString = new bytes(32);
        uint charCount = 0 ;
        for(uint i = 0 ; i<32;i++){
            byte char = byte(bytes32(uint(_bytes32) *2 **(8*i)));
            if(char !=0){
                bytesString[charCount] = char;
                charCount++;
            }
        }
        bytes memory bytesStringTrimmed = new bytes(charCount);
        for(uint j=0;j<charCount;j++){
            bytesStringTrimmed[j]=bytesString[j];
        }
        return string(bytesStringTrimmed);
    }

    function compareHashWithHexString(string memory computedHash, string memory hexStringHash) internal pure returns (bool) {
        // Compare the strings
        return keccak256(abi.encodePacked(computedHash)) == keccak256(abi.encodePacked(hexStringHash));
    }
}