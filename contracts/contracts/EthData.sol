pragma solidity ^0.8.0;

// SPDX-License-Identifier: UNLICENSED

contract EthData{
    address master;
    // data manager
    mapping(address => bool) private manager;
    mapping(address => userInfo) private userInfos;
    //bind address
    mapping(address => address) private bindAddress;

    uint256 stakingAmount;

    //data userinfo
    struct userInfo {
        //0 staking 1  1 can withDraw 2 withDraw done
        uint256 stakingStatus;
        uint256 dayCount;
        uint256 amount;
    }

    modifier onlyManager() {
        require(manager[msg.sender],"onlyManager");
        _;
    }

    constructor() {
        manager[msg.sender] = true;
        master = msg.sender;
    }
    
    function addManager(address _address) public{
        manager[_address] = true;
    }

    function setUserInfo(uint256 _dayCount,uint256 _amount, address _sender) public onlyManager{
        userInfo memory u = userInfos[_sender];
        u.amount = _amount;
        u.dayCount = _dayCount;
        userInfos[_sender] = u;
    }

    function getUserInfo(address _sender) public view
    returns(uint256 amount, uint256 dayCount, uint256 stakingStatus){
        userInfo memory u = userInfos[_sender];
        return (u.amount,u.dayCount,u.stakingStatus);
    }

    function setBindAddress(address _source,address _bind) public onlyManager{
        bindAddress[_source] = _bind;
    }

    function getBindAddress(address _source) public view returns(address){
        return bindAddress[_source];
    }
    
    function setCanWithdraw(address _source) public onlyManager{
        userInfo storage u = userInfos[_source];
        u.stakingStatus = 1;
    }
    
    function getStakingStatus(address _source) public view returns (uint256){
         userInfo memory u = userInfos[_source];
         return u.stakingStatus;
    }
}