pragma solidity ^0.8.0;

// SPDX-License-Identifier: UNLICENSED

import "@openzeppelin/contracts/utils/math/SafeMath.sol";
import "./utils/managers.sol";



contract MaticData is Managers{
    using SafeMath for uint256;
    //address count 
    uint256 addressCount =0;
    uint256 stakingAmount = 0;
    dayHourSign[24] dayHourSigns;
    uint256 rate  = 2600;
 
    mapping(address => userInfo) private userInfos;
    //bind address
    mapping(address => address) public bindAddress;
    //address bind 
    mapping(address => address) public addressBind;

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
    
    constructor() {
        manager[msg.sender] = true;    
        master = msg.sender;
    }
    
    function setUserInfo(uint256 _dayCount,uint256 _daySign,uint256 _amount, uint256 status ,address _sender) public onlyManager{
        userInfo storage u = userInfos[_sender];
        u.amount = _amount;
        u.dayCount = _dayCount;
        u.daySign = _daySign;
        u.stakingStatus = status;
        delete u.signTm;
    }

    function setUserWithdraw(address _sender, uint status) public onlyManager{
        userInfo storage u = userInfos[_sender];
        u.stakingStatus = status;
    }

    function sign(address _sender, uint256 day, uint256 hour,uint256 times) public onlyManager returns(uint256){
        userInfo storage u = userInfos[_sender];
        u.signTm.push(block.timestamp);
        u.daySign = u.daySign.add(1);
        dayHourSign storage ds = dayHourSigns[hour];
        ds.day = day;
        ds.times = times;
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
        if(u.signTm.length == 0) return 0;
        return u.signTm[u.signTm.length];
    }
    
    function setBindAddress(address _source,address _bind) public onlyManager{
        bindAddress[_bind] = _source;
        addressBind[_source] = _bind;
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

    function get24HourSign() public view returns(uint){
        uint256 count = 0;
        uint256 day = block.timestamp.div(3600*24);
        for (uint i = 0;i<24 ;i++){
            uint256 daySign = dayHourSigns[i].day;
            if (daySign + 1 >= day) {
                count = count.add(dayHourSigns[i].times);
            }
        }
        return count;
    }
    
    function getAward(address _sender) public view returns(uint){
        userInfo memory u = userInfos[_sender];
        if (u.daySign > 0){
            return u.amount.mul(u.daySign).mul(rate).div(365).div(10000);
        }
        return 0;
    }

    function setRate(uint256 _rate) public{
        rate = _rate;
    }
    
    function getRate() public view returns (uint256){
        return rate;
    }

    function getDayHourSign(uint256 hour) public view returns(uint256 day, uint256 times){
        dayHourSign memory dhs = dayHourSigns[hour];
        return (dhs.day,dhs.times);
    }
}