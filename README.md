# S3 Encrypt Wrapper

Simple and Easy-to-Use Auto Encryption/Decryption S3 Middleware Service.

Automatically encrypts upload streams and decrypts download streams using AES-256-CTR, with support for HTTP Range.

Compatible with all cloud storage services that support the S3 protocol.

# ⚠️ WARNING

Do not use in production without thorough testing.



# Usage Example

## Upload

Automatically encrypt a file using AES-256-CTR during upload.

The password can be sent through form data, query string and form params.

```shell
# POST /path/file/to/upload
# multipart/form-data
curl -F 'password=' -F 'file=@/location/file.dat'  http://localhost:8000/path/file
```

## Download

Automatically decrypt files during download.

```shell
# GET /path/to/file
curl --output file.dat http://localhost:8000/path/file
```

### Partial download

Supports HTTP Range for partial downloads. Classic use cases include streaming media playback or parallel download.

```shell
curl -H "Range: bytes=1-2" http://localhost:8000/path/file.txt
```

## Delete

The password can be sent through form data, query string and form params.

```shell
# DELETE /path/to/file
curl -XDELETE http://localhost:8000/path/file[?password=]
```

# Configuration

Use `toml`

## AWS S3

```toml
[server]
addr=":8000"
key="11111111111111111111111111111111" # must be 32 characters
password="THE_PASSWORD" # If a password is set, it must be sent when uploading or deleting files.
[storage]
access_id="your access key id"
secret="your access secret key"
bucket="your bucket name"
region="your region"
```

## Cloudflare R2

```toml
[server]
addr=":8000"
key="11111111111111111111111111111111" # must be 32 characters
password="THE_PASSWORD" # If a password is set, it must be sent when uploading or deleting files.
[storage]
enpoint="https://<your_account_id_in_cloudfalre_r2_dashboard>.r2.cloudflarestorage.com/"
access_id="your access key id"
secret="your access secret key"
bucket="your bucket name"
region="auto" # must be "auto"
```

## Alibabacloud (Aliyun) OSS

```toml
[server]
addr=":8000"
key="11111111111111111111111111111111" # must be 32 characters
password="THE_PASSWORD" # If a password is set, it must be sent when uploading or deleting files.
[storage]
enpoint="http://<region>.aliyuncs.com"
access_id="your access key id"
secret="your access secret key"
bucket="your bucket name"
region="<region>" # example: oss-cn-hangzhou
```

# Donation

XMR: 4Ay7eEeA13R82Ff11EN6WXA6wHsZcD15u71at1RGyzhhPqhj4Hd2sQKiKWc3UVXECxLpugirRgE2YfWTmsJPCdY3DJjYqym

BTC: bc1qmdae24nwg5ckeh4xlmtzh88gjygcrynqs8sz0j