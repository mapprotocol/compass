pragma solidity ^0.8.0;

// SPDX-License-Identifier: UNLICENSED

contract Managers {
    address master;
    mapping(address => bool) manager;
    
    constructor () {
        master = msg.sender;
        manager[msg.sender] = true;
    }
    
    
    modifier onlyMaster(){
        require(msg.sender == master,"only master");
        _;
    }
    
    modifier onlyManager(){
        require(manager[msg.sender],"only manager");
        _;
    }
    
    
    function addManager(address _address) public onlyMaster{
        manager[_address] = true;
    }
    
    function removeManager(address _address) public onlyMaster{
        manager[_address] = false;
    }
}