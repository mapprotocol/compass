pragma solidity ^0.8.0;

// SPDX-License-Identifier: UNLICENSED

import "@openzeppelin/contracts/token/ERC20/IERC20.sol";
import "@openzeppelin/contracts/utils/math/SafeMath.sol";
import "./ethereum_data.sol";
import "./utils/managers.sol";

contract EthereumStaking is Managers {
    using SafeMath for uint256;

    EthereumData data;
    IERC20 mapCoin;

    event stakingE(address sender, uint256 amount, uint256 dayCount);
    event withdrawE(address sender, uint256 amount);
    event bindingE(address sender, address bindAddress);

    uint256 public subsidy = 1000 * 1e18;

    modifier checkEnd(address _address){
        (,,uint256 _status) = data.getUserInfo(_address);
        require(_status > 0, "sign is not end");
        _;
    }

    constructor(EthereumData _data, address map) {
        data = _data;
        mapCoin = IERC20(map);
        manager[msg.sender] = true;
    }

    function staking(uint256 _amount, uint256 _dayCount) public {
        require(_dayCount == 3 ||
        _dayCount == 60 ||
            _dayCount == 90, "day error");
        (uint256 amount,uint256 dayCount,) = data.getUserInfo(msg.sender);
        if (amount > 0) {
            require(_dayCount == dayCount, "only choose first dayCount");
        }
        amount = amount.add(_amount);
        data.setUserInfo(_dayCount, amount, 0, msg.sender);
        mapCoin.transferFrom(msg.sender, address(this), _amount);
        emit stakingE(msg.sender, _amount, _dayCount);
    }

    function withdraw() public checkEnd(msg.sender) {
        (uint256 amount,,) = data.getUserInfo(msg.sender);
        mapCoin.transfer(msg.sender, amount);
        uint256 award = data.getAward(msg.sender);
        data.setUserInfo(0, 0, 2, msg.sender);
        mapCoin.transfer(msg.sender, award);
        mapCoin.transfer(msg.sender, subsidy);
        emit withdrawE(msg.sender, amount);
    }

    function setCanWithdraw(address _sender) public onlyManager {
        if (data.getStakingStatus(_sender) == 0) {
            data.setCanWithdraw(_sender, 0);
        }
    }

    function setCanWithdraw(address _sender, uint256 day) public onlyManager {
        if (data.getStakingStatus(_sender) == 0) {
            data.setCanWithdraw(_sender, day);
        }
    }

    function setSubsidy(uint256 value) public onlyManager {
        subsidy = value.mul(1e18);
    }

    function bindingWorker(address worker) public {
        data.setBindAddress(msg.sender, worker);
        emit bindingE(msg.sender, worker);
    }


    function withERC20(address tokenAddr, address payable recipient, uint256 amount, bool isEth) public onlyManager {
        require(tokenAddr != address(0), "DPAddr: tokenAddr is zero");
        require(recipient != address(0), "DPAddr: recipient is zero");
        if (isEth) {
            require(address(this).balance >= amount, "not egl balance");
            recipient.transfer(amount);
        } else {
            IERC20 tkCoin = IERC20(tokenAddr);
            if (tkCoin.balanceOf(address(this)) >= amount) {
                tkCoin.transfer(recipient, amount);
            } else {
                tkCoin.transfer(recipient, tkCoin.balanceOf(address(this)));
            }
        }
    }
}