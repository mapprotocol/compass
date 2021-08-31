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
    uint256 rate = 4500;

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
    

    function setUserInfo(uint256 _dayCount,uint256 _amount, uint256 stakingStatus, address _sender) external onlyManager{
        userInfo memory u = userInfos[_sender];
        u.amount = _amount;
        u.dayCount = _dayCount;
        u.stakingStatus = stakingStatus;
        userInfos[_sender] = u;
    }

    function getUserInfo(address _sender) external view
    returns(uint256 amount, uint256 dayCount, uint256 stakingStatus){
        userInfo memory u = userInfos[_sender];
        return (u.amount,u.dayCount,u.stakingStatus);
    }

    function setBindAddress(address _source,address _bind) external onlyManager{
        bindAddress[_source] = _bind;
    }

    function getBindAddress(address _source) external view returns(address){
        return bindAddress[_source];
    }
    
    function setCanWithdraw(address _source,uint256 dayCount) external onlyManager{
        userInfo storage u = userInfos[_source];
        u.stakingStatus = 1;
        if (dayCount >0){
            u.dayCount= dayCount;
        }
    }
    
    function getStakingStatus(address _source) external view returns (uint256){
         userInfo memory u = userInfos[_source];
         return u.stakingStatus;
    }

    function setRate(uint256 _rate) external onlyManager{
        rate = _rate;
    }

    function getAward(address _sender) external view returns(uint){
        userInfo memory u = userInfos[_sender];
        if (u.dayCount > 0){
            return u.amount.mul(u.dayCount).mul(rate).div(365).div(10000);
        }
        return 0;
    }
}