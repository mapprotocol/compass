# Compass
Compass is Ultra-light Verification Protocol of MAP Protocol. It is designed as a library, that other blockchains can integrated it to support MAP Protocol.

# signmap Usage

## Main daemon program
```shell
$ ./signmap 
```
<p> Keystore file is required, Default is in the current folder keystore.json. If not, you need to type it in the input box.
<p>Password is required.

## Help document
```shell
$ ./signmap help  
```
## Configure the application.
```shell
$ ./signmap config
```
<p> Specify keystore file path.

## Read the sign-in history
```shell
$ ./signmap log
```
##  Get user information
```shell
$ ./signmap info
```
<p>Acquisition of sign-in times, pledge amount, revenue. 
<p>Keystore and password is required.

##  Configure the application chain
```shell
$ ./signmap chain
```
<p>Create or update rpc url, contract address. 
<p> User input is not safe. Please guarantee that it is correct by yourself！
