pragma solidity ^0.8.0;

// SPDX-License-Identifier: UNLICENSED

import "./interface/IDataMatic.sol";


contract MaticStaking {
    IDataMatic data ;
    
    mapping(address => bool) private manager;

    event signE(address sender, uint256 dayCount, uint256 daySign);

    modifier onlyManager() {
        require(manager[msg.sender],"onlyManager");
        _;
    }
    
    constructor(IDataMatic _data) {
        data = _data;
        manager[msg.sender] = true;
    } 
    
    
    function staking(uint256 _dayCount, uint256 _amount, address _sender) public onlyManager {
        
    }
    
    function binding(address _sender, address _binding) public onlyManager{
        
    }

    function withdraw(address _sender) public onlyManager{

    }
    
    function sign() public{
        
    }
}