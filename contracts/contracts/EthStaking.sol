pragma solidity ^0.8.0;

// SPDX-License-Identifier: UNLICENSED


import "./interface/IDataEth.sol";
import "@openzeppelin/contracts/token/ERC20/IERC20.sol";
import "@openzeppelin/contracts/utils/math/SafeMath.sol";


contract EthStaking {
    using SafeMath for uint256;
    
    IDataEth data ;
    mapping(address => bool) private manager;
    IERC20 mapCoin ;

    event stakingE(address sender, uint256 amount, uint256 dayCount);
    event withdrawE(address sender, uint256 amount);
    event bindingE(address sender, address bindAddress);
    
    modifier onlyManager() {
        require(manager[msg.sender],"onlyManager");
        _;
    }
    
    modifier checkEnd(address _address){
        (,,uint256 _status)=data.getUserInfo(_address);
        require(_status > 0,"sign is not end");
        _;
    }
    
    
    constructor(IDataEth _data,address map) {
        data = _data;
        mapCoin = IERC20(map);
        manager[msg.sender] = true;
    } 
    
    
    function staking(uint256 _amount,uint256 _dayCount) public {
        mapCoin.transferFrom(msg.sender,address(this),_amount);
        (uint256 amount,uint256 dayCount,) = data.getUserInfo(msg.sender);
        require(_dayCount == dayCount, "only choose first dayCount");
        amount = amount + _amount;
        data.setUserInfo(_dayCount,amount,msg.sender);
        emit stakingE(msg.sender,amount,dayCount);
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