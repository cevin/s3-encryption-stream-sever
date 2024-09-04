# S3 加密中间件

简单易用的自动加解密S3上传下载流量

使用AES-256-CTR算法，自动加密上传流量，自动解密下载流量。支持HTTP分块请求


# ⚠️ 警告

未经充分测试请勿用于生产环境



# 使用示范

## 上传

自动加密一个文件并上传到S3 使用AES-256-CTR加密算法

密码可以通过FormData、FormParams和Query发送

```shell
# POST /在S3的路径
# multipart/form-data
curl -F 'password=' -F 'file=@/location/file.dat' http://localhost:8000/upload
```

## 下载

自动下载一个加密文件，在下载的过程中自动解密

```shell
# GET /S3路径(加密前)
curl --output file.dat http://localhost:8000/path/file
```

### 分块下载

支持 HTTP Range 请求，通常用于音视频播放、并行下载等场景

```shell
curl -H "Range: bytes=1-2" http://localhost:8000/path/file.txt
```

## 删除文件

密码可以通过FormData、FormParams和Query发送

```shell
# DELETE /S3路径(加密前)
curl -XDELETE http://localhost:8000/path/file[?password=]
```

# 配置

使用 `toml`

## 亚马逊S3

```toml
[server]
addr=":8000"
key="11111111111111111111111111111111" # 加密密钥 必须是32个字符
password="密码" # 如果设置 在上传、删除文件时需要传递
[storage]
access_id="你的accessid"
secret="你的密钥"
bucket="你的Bucket名字"
region="你的bucket区域"
```

## Cloudflare R2

```toml
[server]
addr=":8000"
key="11111111111111111111111111111111" # 必须是32个字符
password="密码" # 如果设置 在上传、删除文件时需要传递
[storage]
enpoint="https://<你的accountid在cloudfalre的R2控制面板可查看到>.r2.cloudflarestorage.com/"
access_id="你的R2访问key"
secret="你的R2访问密钥"
bucket="你的存储桶名称"
region="auto" # 必须是 auto
```

## 阿里云 OSS

```toml
[server]
addr=":8000"
key="11111111111111111111111111111111" # must be 32 characters
password="密码" # 如果设置 在上传、删除文件时需要传递
[storage]
enpoint="http://<region>.aliyuncs.com"
access_id="your access key id"
secret="your access secret key"
bucket="your bucket name"
region="<region>" # 比如: oss-cn-hangzhou
```


# 捐助

XMR: 4Ay7eEeA13R82Ff11EN6WXA6wHsZcD15u71at1RGyzhhPqhj4Hd2sQKiKWc3UVXECxLpugirRgE2YfWTmsJPCdY3DJjYqym

BTC: bc1qmdae24nwg5ckeh4xlmtzh88gjygcrynqs8sz0j