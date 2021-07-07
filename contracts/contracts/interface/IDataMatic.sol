pragma solidity ^0.8.0;
import "../MaticData.sol";


// SPDX-License-Identifier: UNLICENSED

interface IDataMatic{
    function setUserInfo(uint256 _dayCount,uint256 _daySign,uint256 _amount, address _sender) external;
    function getUserInfo(address _sender) external view returns(uint256 amount, uint256 dayCount,uint256 daySign, uint256 stakingStatus,uint256[] memory signTm);
    function setBindAddress(address _source,address _bind) external;
    function getBindAddress(address _source) external view returns(address bind);
    function sign(address _sender, uint256 day, uint256 hour, uint256 times) external returns(uint256 daySign);
    function getLastSign(address _sender) external returns(uint256 lastTime);
    function dayHourSigns(uint256 _hour) external view returns(uint256 times, uint256 day);
    function setUserWithdraw(address _sender) external;
}