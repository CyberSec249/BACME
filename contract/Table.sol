// SPDX-License-Identifier: MIT
pragma solidity >=0.4.24 <0.6.11;

//one record
contract Entry {
    function getInt(string memory) public view returns (int256) {}
    function getUInt(string memory) public view returns (uint256) {}
    function getAddress(string memory) public view returns (address) {}
    function getBytes64(string memory) public view returns (bytes1[64] memory) {}
    function getBytes32(string memory) public view returns (bytes32) {}
    function getString(string memory) public view returns (string memory) {}

    function set(string memory, int256) public {}
    function set(string memory, uint256) public {}
    function set(string memory, string memory) public {}
    function set(string memory, address) public {}
}

contract KVTableFactory {
    function openTable(string memory) public view returns (KVTable) {}
    function createTable(string memory, string memory, string memory) public returns (int256) {}
}

//KVTable per permiary key has only one Entry
contract KVTable {
    function get(string memory) public view returns (bool, Entry) {}
    function set(string memory, Entry) public returns (int256) {}
    function newEntry() public view returns (Entry) {}
}
