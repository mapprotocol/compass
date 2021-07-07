pragma solidity ^0.8.0;

// SPDX-License-Identifier: UNLICENSED

contract MaticData{
    //address count 
    uint256 addressCount =0;
    uint256 stakingAmount = 0;
    dayHourSign[24] dayHourSigns;

    address master;
    
    // data manager
    mapping(address => bool) private manager;
    mapping(address => userInfo) private userInfos;
    //bind address
    mapping(address => address) private bindAddress;

    //data userinfo
    struct userInfo {
        uint256 stakingStatus;
        uint256 dayCount;
        uint256 daySign;
        uint256 amount;
        uint256 [] signTm;
    }

    struct dayHourSign{
        uint256 times;
        uint256 day;
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
        userInfo storage u = userInfos[_sender];
        u.amount = _amount;
        u.dayCount = _dayCount;
        u.daySign = _daySign;
        u.stakingStatus = 0;
    }

    function setUserWithdraw(address _sender, uint status) public onlyManager{
        userInfo storage u = userInfos[_sender];
        u.stakingStatus = status;
    }

    function sign(address _sender, uint256 day, uint256 hour) public onlyManager returns(uint256){
        userInfo storage u = userInfos[_sender];
        u.signTm.push(block.timestamp);
        u.daySign ++;
        dayHourSign storage ds = dayHourSigns[hour];
        ds.day = day;
        ds.times ++;
        return u.daySign;
    }
    
    function getUserInfo(address _sender) public view
        returns(uint256 amount, uint256 dayCount,uint256 daySign, uint256 stakingStatus,uint256[] memory signTm){
        userInfo memory u = userInfos[_sender];
        return (u.amount,u.dayCount,u.daySign,u.stakingStatus,u.signTm);
    }

    function getUserInfos(address _sender) public view returns(userInfo memory u){
        return userInfos[_sender];
    }

    function getLastSign(address _sender) public view returns(uint256){
        userInfo memory u = userInfos[_sender];
        return u.signTm[u.signTm.length];
    }
    
    function setBindAddress(address _source,address _bind) public onlyManager{
        bindAddress[_source] = _bind;
    }
    
    function getBindAddress(address _source) public view returns(address){
        return bindAddress[_source];
    }

    function getAddressCount() public view returns(uint256){
        return addressCount;
    }

    function setAddressCount(uint count) public onlyManager{
        addressCount = count;
    }

    function getStakingAmount() public view returns(uint256){
        return stakingAmount;
    }

    function setStakingAmount(uint amount) public onlyManager{
        stakingAmount = amount;
    }
}