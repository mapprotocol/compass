pragma solidity ^0.7.0;

// SPDX-License-Identifier: UNLICENSED

import "./interface/IData.sol";


contract relayer {
    IData data ;
    
    mapping(address => bool) private manager;
    
    
    modifier onlyManager() {
        require(manager[msg.sender],"onlyManager");
        _;
    }
    
    modifier checkEnd(address _address){
        (,uint256 _daycount,uint256 _daySign)=data.getUserInfo(_address);
        require(_daycount <=_daySign,"sign is not end");
        _;
    }
    
    
    constructor(IData _data) {
        data = _data;
        manager[msg.sender] = true;
    } 
    
    
    function staking() public onlyManager {
        
    } 
    
    function sign() public{
        
    }
}