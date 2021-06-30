pragma solidity ^0.7.0;

// SPDX-License-Identifier: UNLICENSED


contract relayerData{
    //address count 
    uint256 addressCount = 0;
    address master;
    
    // data manager
    mapping(address => bool) private manager;
    mapping(address => userInfo) private userInfos;
    //bind address
    mapping(address => address) private bindAddresss;
    
    //data userinfo
    struct userInfo {
        //0 staking 1  1 can withDraw 2 withDraw done
        uint256 stakingStatus;
        uint256 dayCount;
        uint256 daySign;
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
    
    function setUserInfo(uint256 _dayCount,uint256 _daySign,uint256 _amount, address _sender) public onlyManager{
        userInfo memory u = userInfos[_sender];
        u.amount = _amount;
        u.dayCount = _dayCount;
        u.daySign = _daySign;
        userInfos[_sender] = u;
    }
    
    function getUserInfo(address _sender) public view returns(uint256, uint256,uint256){
        userInfo memory u = userInfos[_sender];
        return (u.amount,u.dayCount,u.daySign);
    }
    
    
    function setBindAddress(address _source,address _bind) public onlyManager{
        bindAddresss[_source] = _bind;
    }
    
    function getBindAddress(address _source) public view returns(address){
        return bindAddresss[_source];
    }
    
    
}