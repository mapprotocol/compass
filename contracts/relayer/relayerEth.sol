pragma solidity ^0.7.0;

// SPDX-License-Identifier: UNLICENSED


import "./interface/IData.sol";
import "./Zeppelin/token/ERC20/IERC20.sol";

library SafeMath {
    /**
     * @dev Multiplies two unsigned integers, reverts on overflow.
     */
    function mul(uint256 a, uint256 b) internal pure returns (uint256) {
        // Gas optimization: this is cheaper than requiring 'a' not being zero, but the
        // benefit is lost if 'b' is also tested.
        // See: https://github.com/OpenZeppelin/openzeppelin-solidity/pull/522
        if (a == 0) {
            return 0;
        }

        uint256 c = a * b;
        require(c / a == b);

        return c;
    }

    /**
     * @dev Integer division of two unsigned integers truncating the quotient, reverts on division by zero.
     */
    function div(uint256 a, uint256 b) internal pure returns (uint256) {
        // Solidity only automatically asserts when dividing by 0
        require(b > 0);
        uint256 c = a / b;
        // assert(a == b * c + a % b); // There is no case in which this doesn't hold

        return c;
    }

    /**
     * @dev Subtracts two unsigned integers, reverts on overflow (i.e. if subtrahend is greater than minuend).
     */
    function sub(uint256 a, uint256 b) internal pure returns (uint256) {
        require(b <= a);
        uint256 c = a - b;

        return c;
    }

    /**
     * @dev Adds two unsigned integers, reverts on overflow.
     */
    function add(uint256 a, uint256 b) internal pure returns (uint256) {
        uint256 c = a + b;
        require(c >= a);

        return c;
    }

    /**
     * @dev Divides two unsigned integers and returns the remainder (unsigned integer modulo),
     * reverts when dividing by zero.
     */
    function mod(uint256 a, uint256 b) internal pure returns (uint256) {
        require(b != 0);
        return a % b;
    }
}


contract relayer {
    using SafeMath for uint256;
    
    IData data ;
    mapping(address => bool) private manager;
    IERC20 mapCoin ;
    
    
    modifier onlyManager() {
        require(manager[msg.sender],"onlyManager");
        _;
    }
    
    modifier checkEnd(address _address){
        (,uint256 _daycount,uint256 _daySign)=data.getUserInfo(_address);
        require(_daycount <=_daySign,"sign is not end");
        _;
    }
    
    
    constructor(IData _data,address map) {
        data = _data;
        mapCoin = IERC20(map);
        manager[msg.sender] = true;
    } 
    
    
    function staking(uint256 _amount,uint256 _dayCount) public {
        mapCoin.transferFrom(msg.sender,address(this),_amount);
        (uint256 amount,,) = data.getUserInfo(msg.sender);
        if (amount > 0){
            data.setUserInfo(_dayCount,0,_amount.add(amount),msg.sender);
        }else{
            data.setUserInfo(_dayCount,0,_amount,msg.sender);
        }
    } 
    
    function withdraw(address _sender) public onlyManager checkEnd(_sender){
        (uint256 amount,,) = data.getUserInfo(_sender);
        mapCoin.transfer(_sender,amount);
    }
}