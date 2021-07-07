pragma solidity ^0.8.0;

// SPDX-License-Identifier: UNLICENSED

import "./interface/IDataMatic.sol";


contract MaticStaking {
    IDataMatic data ;
    
    mapping(address => bool) private manager;

    event signE(address sender, uint256 dayCount, uint256 daySign);
    event stakingE(address sender, uint256 amount, uint256 dayCount);
    event withdrawE(address sender, uint256 amount);
    event bindingE(address sender, address bindAddress);

    modifier onlyManager() {
        require(manager[msg.sender],"onlyManager");
        _;
    }
    
    constructor(IDataMatic _data) {
        data = _data;
        manager[msg.sender] = true;
    } 

    function getSender(address _worker) public view returns (address){
        address sender = data.getBindAddress(_worker);
        if (sender == address(0)){
            return _worker;
        }
        return sender;
    }
    
    function staking(uint256 _dayCount, uint256 _amount, address _sender) public onlyManager {
        (uint256 amount,,,,) = data.getUserInfo(_sender);


    }
    
    function binding(address _sender, address _binding) public onlyManager{
        data.setBindAddress(_binding,_sender);
        emit bindingE(_sender,_binding);
    }

    function withdraw(address _sender) public onlyManager{
        address sender = getSender(_sender);
        data.setUserWithdraw(_sender);
    }
    
    function sign() public{
        
    }
}