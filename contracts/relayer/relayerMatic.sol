pragma solidity ^0.7.0;

// SPDX-License-Identifier: UNLICENSED

import "./interface/IData.sol";


contract relayer {
    IData data ;

    mapping(address => bool) private manager;
    // event
    event votedEvent(uint indexed_candidateId);

    struct Record{
        uint256 time;
    }


    // 最后一次签到时间
    mapping(address=>uint256) lasted;
    mapping(address =>Record[]) records;

    // data manager
    mapping(address => userInfo) private userInfos;
    //data user
    struct userInfo {
        uint256 amount;
        uint256 daySign;
    }
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


    function staking(uint256 _amount,uint256 _daySign, address _sender) public onlyManager {
        //Amount and number of terms
        userInfo memory u = userInfos[_sender];
        u.amount = _amount;
        u.daySign = _daySign;
        userInfos[_sender] = u;

    }

    function sign() public{
        uint256 last = lasted[msg.sender];
        if (last!=0){
            require(last+1 days < block.timestamp,"checked in within 24 hours");
        }
        Record memory record = Record({time:block.timestamp});
        records[msg.sender].push(record);
        dayCout++;
        (uint256 amount, uint256 dayCout, uint256 daySign)  = data.getUserInfo(msg.sender);
    }
}