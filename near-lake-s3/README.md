# near-lake-s3
near-lake-s3 is used to sync the certain blocks from AWS S3 buckets to redis server. The block must contain transactions/receipts related to specified account.

## How to build?

```shell
./build.sh
```

The binary file "near-lake-s3" will be generated in ./target/release.

## How to Run

### AWS S3 Credentials

In order to be able to get objects from the AWS S3 bucket you need to provide the AWS credentials.

AWS default profile configuration with aws configure looks similar to the following:

`~/.aws/credentials`
```
[default]
aws_access_key_id=
aws_secret_access_key=
```

[AWS docs: Configuration and credential file settings](https://docs.aws.amazon.com/cli/latest/userguide/cli-configure-files.html)

### Env Config

You can rename .env.example to .env, modify it and put it in the same directory with near-lake-s3 (or it's parent directory).

```
// Start from cache if true ignore START_BLOCK_HEIGHT
START_BLOCK_HEIGHT_FROM_CACHE=false
START_BLOCK_HEIGHT=0

// Push block to Redis list if false ignore PUSH_ENGINE_URL
ENABLE_REDIS=true
REDIS_URL="redis://127.0.0.1:6379"
// redis list name
PUB_CHANNEL="blocks"

// The account name to watch
MCS="mcs.testnet"
```


### Run
```shell
./target/release/near-lake-s3
```