1.event stakingE(address sender, uint256 amount, uint256 dayCount);
    当以太网用户质押时触发 
    需要转移状态到matic
    调用：
    Matic :function staking(uint256 _dayCount, uint256 _amount, address _sender)
2.event bindingE(address sender, address bindAddress);
    当绑定worker时触发
    需要转移状态到matic
    调用：
    Matic function binding(address _sender, address _binding) 
3.     event signE(address sender, uint256 dayCount, uint256 daySign);
    matic 每次质押时触发
    当daycount <= daySign 时
    调用
    Eth  function setCanWithdraw(address _sender)

