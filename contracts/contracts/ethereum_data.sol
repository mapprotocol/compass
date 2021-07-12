pragma solidity ^0.8.0;

// SPDX-License-Identifier: UNLICENSED

import "./utils/managers.sol";
import "@openzeppelin/contracts/utils/math/SafeMath.sol";

contract EthereumData is Managers{
    using SafeMath for uint256;
    mapping(address => userInfo) private userInfos;
    //bind address
    mapping(address => address) private bindAddress;

    uint256 stakingAmount;

    uint256 rate = 2600;

    //data userinfo
    struct userInfo {
        //0 staking 1  1 can withDraw 2 withDraw done
        uint256 stakingStatus;
        uint256 dayCount;
        uint256 amount;
    }
    
    constructor() {
        manager[msg.sender] = true;
        master = msg.sender;
    }
    

    function setUserInfo(uint256 _dayCount,uint256 _amount, uint256 stakingStatus, address _sender) public onlyManager{
        userInfo memory u = userInfos[_sender];
        u.amount = _amount;
        u.dayCount = _dayCount;
        u.stakingStatus = stakingStatus;
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

    function setRate(uint256 _rate) public{
        rate = _rate;
    }

    function getAward(address _sender) public view returns(uint){
        userInfo memory u = userInfos[_sender];
        if (u.dayCount > 0){
            return u.amount.mul(u.dayCount).mul(rate).div(365).div(10000);
        }
        return 0;
    }
}