pragma solidity ^0.7.0;

// SPDX-License-Identifier: UNLICENSED

interface IData{
    function setUserInfo(uint256 _dayCount,uint256 _daySign,uint256 _amount, address _sender) external;
    function getUserInfo(address _sender) external view returns(uint256, uint256,uint256);
    function setBindAddress(address _source,address _bind) external;
    function getBindAddress(address _source) external view returns(address);
}