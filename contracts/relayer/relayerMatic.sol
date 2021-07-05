pragma solidity ^0.8.0;

// SPDX-License-Identifier: UNLICENSED

import "./interface/IData.sol";

library SafeMath {
    /**
     * @dev Adds two unsigned integers, reverts on overflow.
     */
    function add(uint256 a, uint256 b) internal pure returns (uint256) {
        uint256 c = a + b;
        require(c >= a);

        return c;
    }
}


contract relayer {
    using SafeMath for uint256;
    IData data ;


    mapping(address => bool) private manager;
    // event
    event votedEvent(uint indexed_candidateId);



    // 最后一次签到时间
    mapping(address=>uint256) lasted;



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

    function staking(uint256 _amount,uint256 _daySign) public onlyManager {
        (uint256 amount,uint256 daysign,) = data.getUserInfo(msg.sender);
        if (amount > 0 && daysign > 0 ){
            data.setUserInfo(0,_daySign.add(daysign),_amount.add(amount),msg.sender);
        }else{
            data.setUserInfo(0,_daySign,_amount,msg.sender);
        }
    }

    function sign() public{
        uint256 last = lasted[msg.sender];
        if (last!=0){
            require(last+1 days < block.timestamp,"checked in within 24 hours");
        }
        (uint256 daysign) = data.sign(msg.sender);
        (uint256 amount, uint256 dayCount, uint256 daySign)  = data.getUserInfo(msg.sender);
        return;
    }
}