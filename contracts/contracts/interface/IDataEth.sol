pragma solidity ^0.8.0;

// SPDX-License-Identifier: UNLICENSED

interface IDataEth{
    function setUserInfo(uint256 _dayCount,uint256 _amount, address _sender) external;
    function getUserInfo(address _sender) external view
        returns(uint256 amount, uint256 dayCount, uint256 stakingStatus);
    function setBindAddress(address _source,address _bind) external;
    function getBindAddress(address _source) external view returns(address bind);
    function setCanWithdraw(address _source) external;
    function getStakingStatus(address _source) external view returns (uint256);
}