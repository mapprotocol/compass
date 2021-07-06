pragma solidity ^0.8.0;

// SPDX-License-Identifier: UNLICENSED

import "./interface/IDataMatic.sol";
import "@openzeppelin/contracts/token/ERC20/IERC20.sol";


contract MaticStaking {
    IDataMatic data ;
    
     IERC20 mapCoin ;
     // 最后一次签到时间
    mapping(address=>uint256) lasted;
   // mapping(address =>Record[]) records;
     
    mapping(address => bool) private manager;

      event signE(address sender, uint256 dayCount, uint256 daySign);
      event bindingE(address sender, address bindAddress);
      event stakingE(uint256 _dayCount, uint256 amount);
      event withdrawE(address _sender, uint256 amount);

    modifier onlyManager() {
        require(manager[msg.sender],"onlyManager");
        _;
    }
    
    constructor(IDataMatic _data) {
        data = _data;
        manager[msg.sender] = true;
    } 
    
    
    function staking(uint256 _dayCount, uint256 _amount, address _sender) public onlyManager {
         data.setUserInfo(_dayCount,0,_amount,_sender);
          emit stakingE(_dayCount,_amount);
        
    }
    
    function binding(address _sender, address _binding) public onlyManager{
           data.setBindAddress(_sender,_binding);
           emit bindingE(_sender,_binding);
        
    }

    function withdraw(address _sender) public onlyManager{
    (uint256 amount, ,,,) = data.getUserInfo(_sender);
     mapCoin.transfer(_sender,amount);
    emit withdrawE(_sender,amount);
  
    }
    
//     function sign() public{
//         uint256 last = lasted[msg.sender];
//         if (last!=0){
//             require(last+1 days < block.timestamp,"checked in within 24 hours");
//         }
//         //Record memory record = Record({time:block.timestamp});
//         //records[msg.sender].push(record);
//       //  dayCout++;
//      //   (uint256 amount, uint256 dayCout, uint256 daySign)  = data.getUserInfo(msg.sender);
//     }
}