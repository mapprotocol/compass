# Compass
Compass is Ultra-light Verification Protocol of MAP Protocol. It is designed as a library, that other blockchains can integrated it to support MAP Protocol.

# map-rly Usage

##  Help document
```shell
./map-rly 
./map-rly help  
```
<p>The default command is help.</p>

## Configure the application.
<p>The command requires a configuration file named config.toml in the same directory as the command</p>
This is [example](./example.config.toml)


## To become relayer
```shell
./map-rly register  # Interactive
./map-rly register 200000 # Direct execution
```
<p>You can continue to call to increase the registration amount</p>

## 
```shell
./map-rly unregister  # Interactive
./map-rly unregister 200000 # Direct execution
```
<p>With the unregister transaction executed, the unregistering portion is locked in the contract for about 2 epoch. After the period, you can withdraw the unregistered coins.</p>

##  Get relayer information
```shell
./map-rly info # Read information once
./map-rly info watch # Read information Every five seconds
./map-rly info watch 10 # Read information Every ten seconds
```

## Main daemon program
```shell
./map-rly daemon
```
<p>Do relay work</p>

