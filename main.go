package main

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"flag"
	"fmt"
	"github.com/BurntSushi/toml"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"io"
	"log"
	"mime"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

type Config struct {
	Server struct {
		Addr string `toml:"addr"`
		Key  string `toml:"key"`
	} `toml:"server"`
	Storage struct {
		Endpoint string `toml:"endpoint"`
		AccessId string `toml:"access_id"`
		Secret   string `toml:"secret"`
		Bucket   string `toml:"bucket"`
		Region   string `toml:"region"`
	} `toml:"storage"`
}

var cfg Config

type UploadRequest struct {
	Filepath string `form:"filepath" query:"filepath"`
	Password string `form:"password" query:"password"`
}

var IV = make([]byte, 16)

var (
	client *s3.S3
)

func generateEncryptFilenameNonce(name string) []byte {
	hash := sha256.Sum256([]byte(name))
	return hash[:16]
}

func encryptFilename(name string) string {
	segments := strings.Split(name, "/")
	for i := range segments {
		hash := sha256.Sum256([]byte(segments[i] + cfg.Server.Key))
		segments[i] = strings.ReplaceAll(base64.URLEncoding.EncodeToString(hash[:]), "=", "")
	}

	return strings.Join(segments, "/")
}

func uploadFileToS3(path string, file io.ReadSeeker) (string, error) {

	bucket := aws.String(cfg.Storage.Bucket)
	uploadPath := aws.String(path)

	// generate chunk upload id
	resp, err := client.CreateMultipartUpload(&s3.CreateMultipartUploadInput{
		Bucket: bucket,
		Key:    uploadPath,
	})
	if err != nil {
		return "", err
	}
	uploadId := resp.UploadId

	block, err := aes.NewCipher([]byte(cfg.Server.Key))
	if err != nil {
		return "", err
	}

	counter := make([]byte, len(IV))
	copy(counter, IV)
	stream := cipher.NewCTR(block, counter)

	var partNumber int64 = 1
	var completedParts []*s3.CompletedPart

	buffer := make([]byte, 100*1024*1024)
	for {
		n, err := file.Read(buffer)
		if n > 0 {
			part := buffer[:n]

			// encrypt each block data
			stream.XORKeyStream(part, part)

			partResp, err := client.UploadPart(&s3.UploadPartInput{
				Bucket:     bucket,
				Key:        uploadPath,
				PartNumber: aws.Int64(partNumber),
				UploadId:   uploadId,
				Body:       bytes.NewReader(part),
			})
			if err != nil {
				return "", err
			}
			completedParts = append(completedParts, &s3.CompletedPart{
				ETag:       partResp.ETag,
				PartNumber: aws.Int64(partNumber),
			})

			partNumber++
		}

		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}
	}

	uploadResp, err := client.CompleteMultipartUpload(&s3.CompleteMultipartUploadInput{
		Bucket:   bucket,
		Key:      uploadPath,
		UploadId: uploadId,
		MultipartUpload: &s3.CompletedMultipartUpload{
			Parts: completedParts,
		},
	})

	if err != nil {
		return "", err
	}

	return *uploadResp.Key, nil

}

func parseRangeHeader(rangeHeader string) (int64, int64, error) {
	var start, end int64
	//var err error

	r := regexp.MustCompile(`bytes=(\d+)-(\d+)?/?(\d+)?`)
	match := r.FindStringSubmatch(rangeHeader)
	if len(match) <= 0 {
		return 0, 0, errors.New("invalid range header")
	}

	if match[1] != "" {
		start, _ = strconv.ParseInt(match[1], 10, 64)
	}

	if match[2] == "" {
		end = -1
	} else {
		end, _ = strconv.ParseInt(match[2], 10, 64)
	}

	return start, end, nil
}

func calculateCounter(start int64, blockSize int64) ([]byte, int64) {

	counter := make([]byte, len(IV))
	copy(counter, IV)

	blockOffset := start / blockSize
	ctrOffset := start % blockSize

	// counter align
	binaryCounter := make([]byte, 8)
	binaryCounter[7] = byte(blockOffset & 0xFF)
	binaryCounter[6] = byte((blockOffset >> 8) & 0xFF)
	binaryCounter[5] = byte((blockOffset >> 16) & 0xFF)
	binaryCounter[4] = byte((blockOffset >> 24) & 0xFF)
	copy(counter[8:], binaryCounter)
	return counter, ctrOffset
}

func handleUpload(c echo.Context) error {
	var req UploadRequest
	if err := c.Bind(&req); err != nil {
		return c.String(400, err.Error())
	}

	if req.Filepath == "" {
		return c.String(400, "The filepath is required")
	}

	formFile, err := c.FormFile("file")
	if err != nil {
		return c.String(400, err.Error())
	}

	if !strings.HasPrefix(c.Request().Header.Get("Content-Type"), "multipart/form-data") {
		return c.String(400, "invalid request Content-Type")
	}
	file, err := formFile.Open()
	if err != nil {
		panic(err)
	}
	defer file.Close()

	if result, err := uploadFileToS3(req.Filepath, file); err != nil {
		_err := err.Error()
		if aserr, ok := err.(awserr.Error); ok {
			_err = aserr.Error()
		}
		return c.String(500, _err)
	} else {
		return c.String(200, result)
	}
}

func handleFile(c echo.Context) error {
	bucket := aws.String(cfg.Storage.Bucket)
	path := aws.String(c.Request().URL.Path[1:])

	partialRequest := false
	statusCode := 200

	headResp, err := client.HeadObject(&s3.HeadObjectInput{
		Bucket: bucket,
		Key:    path,
	})
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok && aerr.Code() == "NotFound" {
			return c.String(404, "file not found")
		}
		return c.String(500, err.Error())
	}
	totalSize := headResp.ContentLength

	start := int64(0)
	end := *totalSize - 1

	rangeRequest := c.Request().Header.Get("Range")
	if rangeRequest != "" {
		start, end, err = parseRangeHeader(rangeRequest)
		if err != nil {
			return c.String(400, err.Error())
		}
		if end == -1 {
			end = *totalSize - 1
		}
		partialRequest = true
	}

	if partialRequest {
		statusCode = 206
		// realign
		alignedStart := (start / int64(aes.BlockSize)) * int64(aes.BlockSize)
		c.Response().Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", alignedStart, end, *totalSize))
		c.Response().Header().Set("Content-Length", strconv.FormatInt(end-alignedStart+1, 10))
		start = alignedStart
	} else {
		c.Response().Header().Set("Accept-Ranges", "bytes")
		c.Response().Header().Set("Content-Length", strconv.FormatInt(end-start+1, 10))
	}

	object, err := client.GetObjectWithContext(c.Request().Context(), &s3.GetObjectInput{
		Bucket: bucket,
		Key:    path,
		Range:  aws.String(fmt.Sprintf("bytes=%d-%d", start, end)),
	})
	if err != nil {
		return c.String(500, err.Error())
	}
	defer object.Body.Close()

	block, err := aes.NewCipher([]byte(cfg.Server.Key))
	if err != nil {
		return c.String(500, err.Error())
	}

	counter, offset := calculateCounter(start, int64(aes.BlockSize))
	stream := cipher.NewCTR(block, counter)

	skip := make([]byte, offset)
	stream.XORKeyStream(skip, skip)

	reader := cipher.StreamReader{S: stream, R: object.Body}

	mimeType := mime.TypeByExtension(filepath.Ext(*path))
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	return c.Stream(statusCode, mimeType, reader)
}

func main() {
	configFilepath := flag.String("config", "config.toml", "--config=config.toml")
	flag.Parse()

	if err := setupConfiguration(*configFilepath); err != nil {
		log.Fatal(err)
	}

	awsCfg := &aws.Config{
		Region:      aws.String(cfg.Storage.Region),
		Credentials: credentials.NewStaticCredentials(cfg.Storage.AccessId, cfg.Storage.Secret, ""),
	}
	if cfg.Storage.Endpoint != "" {
		awsCfg.Endpoint = aws.String(cfg.Storage.Endpoint)
	}
	opts, _ := session.NewSession(awsCfg)
	client = s3.New(opts)

	app := echo.New()
	app.HidePort = true
	app.HideBanner = true
	app.HTTPErrorHandler = func(err error, c echo.Context) {
		if e, ok := err.(*echo.HTTPError); ok {
			_ = c.String(e.Code, e.Error())
			return
		}

		_ = c.String(500, err.Error())
	}
	app.Use(middleware.Recover(), middleware.Logger())

	app.POST("/upload", handleUpload)
	app.GET("/*", handleFile)

	app.Logger.Info(app.Start(cfg.Server.Addr))
}

func setupConfiguration(f string) error {
	_, err := toml.DecodeFile(f, &cfg)
	if err != nil {
		return err
	}

	if cfg.Server.Addr == "" {
		cfg.Server.Addr = ":8000"
	}
	if cfg.Server.Key == "" {
		return fmt.Errorf("the encryption key must be specified")
	}
	if len(cfg.Server.Key) != 32 {
		return fmt.Errorf("the encryption key must be 32 characters")
	}
	if cfg.Storage.AccessId == "" || cfg.Storage.Secret == "" || cfg.Storage.Region == "" || cfg.Storage.Bucket == "" {
		return fmt.Errorf("invalid configuration, please check the required paramerter: storage.access_id, storage.secret, storage.region, storage.bucket")
	}

	return nil
}
