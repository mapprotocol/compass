pragma solidity ^0.8.0;

// SPDX-License-Identifier: UNLICENSED

import "@openzeppelin/contracts/utils/math/SafeMath.sol";
import "./MaticData.sol";


contract MaticStaking {
    using SafeMath for uint256;
    MaticData data ;
    
    mapping(address => bool) private manager;

    event signE(address sender, uint256 dayCount, uint256 daySign);
    event stakingE(address sender, uint256 amount, uint256 dayCount);
    event withdrawE(address sender);
    event bindingE(address sender, address bindAddress);

    modifier onlyManager() {
        require(manager[msg.sender],"onlyManager");
        _;
    }
    
    constructor(MaticData _data) {
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
        data.setUserInfo(_dayCount,0,amount.add(_amount),_sender);
        emit stakingE(msg.sender,amount.add(_amount),_dayCount);

        if (amount == 0){
            data.setAddressCount(data.getAddressCount().add(1));
        }

        data.setStakingAmount(data.getStakingAmount().add(_amount));
    }
    
    function binding(address _sender, address _binding) public onlyManager{
        data.setBindAddress(_binding,_sender);
        emit bindingE(_sender,_binding);
    }

    function withdraw(address _sender) public onlyManager{
        (uint256 amount,,,uint256 stakingStatus,) = data.getUserInfo(_sender);
        require(stakingStatus != 2,"user is withdraw");
        emit withdrawE(_sender);
        if (amount == 0){
            data.setAddressCount(data.getAddressCount().sub(1));
        }
        data.setStakingAmount(data.getStakingAmount().sub(amount));
    }

    function getTmDayHour(uint256 tm) public pure returns(uint256 day,uint256 hour){
        if (tm == 0){
            return(0,0);
        }
        day = tm.div(3600*24);
        hour = tm.sub(day.mul(3600*24)).div(3600);
    }

    function sign() public{
        address sender = getSender(msg.sender);
        uint256 last = data.getLastSign(sender);
        (uint256 lastDay,) = getTmDayHour(last);
        (uint256 day,uint256 hour) = getTmDayHour(block.timestamp);
        require(day > lastDay,"today is sign");
        data.sign(sender,day,hour);
    }
}