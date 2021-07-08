pragma solidity ^0.8.0;

// SPDX-License-Identifier: UNLICENSED


import "@openzeppelin/contracts/token/ERC20/IERC20.sol";
import "@openzeppelin/contracts/utils/math/SafeMath.sol";
import "./EthData.sol";
import "./utils/Managers.sol";


contract EthStaking is Managers{
    using SafeMath for uint256;
    
    EthData data ;
    IERC20 mapCoin ;

    uint256 rate = 2600;
    event stakingE(address sender, uint256 amount, uint256 dayCount);
    event withdrawE(address sender, uint256 amount);
    event bindingE(address sender, address bindAddress);
    
    modifier checkEnd(address _address){
        (,,uint256 _status)=data.getUserInfo(_address);
        require(_status > 0,"sign is not end");
        _;
    }
    
    
    constructor(EthData _data,address map) {
        data = _data;
        mapCoin = IERC20(map);
        manager[msg.sender] = true;
    } 
    
    
    function staking(uint256 _amount,uint256 _dayCount) public {
        (uint256 amount,uint256 dayCount,) = data.getUserInfo(msg.sender);
        if(amount > 0){
            require(_dayCount == dayCount, "only choose first dayCount");
        }
        amount = amount.add(_amount);
        data.setUserInfo(_dayCount,amount,msg.sender);
        mapCoin.transferFrom(msg.sender,address(this),_amount);
        emit stakingE(msg.sender,_amount,_dayCount);
    } 
    
    function withdraw() public checkEnd(msg.sender){
        (uint256 amount,,) = data.getUserInfo(msg.sender);
        mapCoin.transfer(msg.sender,amount);
        emit withdrawE(msg.sender,amount);
    }

    function setCanWithdraw(address _sender) public onlyManager{
        if (data.getStakingStatus(_sender) == 0){
            data.setCanWithdraw(_sender);
        }
    }

    function bindingWorker(address worker) public{
        data.setBindAddress(msg.sender,worker);
        emit bindingE(worker,msg.sender);
    }
}