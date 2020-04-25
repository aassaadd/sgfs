package service

import (
	"bufio"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/LinkinStars/golang-util/gu"
	"github.com/dgrijalva/jwt-go"
	"github.com/valyala/fasthttp"
	"go.uber.org/zap"

	"github.com/aassaadd/sgfs/config"
	"github.com/aassaadd/sgfs/util/date_util"
)

// Strips 'Bearer ' prefix from bearer token string
func stripBearerPrefixFromTokenString(tok string) (string, error) {
	// Should be a bearer token
	if len(tok) > 6 && strings.ToUpper(tok[0:7]) == "BEARER " {
		return tok[7:], nil
	}
	return tok, nil
}

// UploadFileHandlerCopy 从其他url copy
func UploadFileHandlerCopy(ctx *fasthttp.RequestCtx) {
	fileUrl := ctx.FormValue("fileUrl")
	durl := string(fileUrl)
	uri, err := url.ParseRequestURI(durl)
	if err != nil {
		SendResponse(ctx, -1, "No file was found.", nil)
	}
	dfileNmae := path.Base(uri.Path)
	client := http.DefaultClient
	client.Timeout = time.Second * 60 //设置超时时间
	resp, err := client.Get(durl)
	if err != nil {
		SendResponse(ctx, -1, "No file was found.", nil)
	}
	if resp.ContentLength <= 0 {
		SendResponse(ctx, -1, "No file was found.", nil)
	}
	raw := resp.Body
	defer raw.Close()
	//
	// Check File Size
	// if header.Size > int64(config.GlobalConfig.MaxUploadSize) {
	// 	SendResponse(ctx, -1, "File size exceeds limit.", nil)
	// 	return
	// }

	// authentication
	// token := string(ctx.FormValue("token"))
	// if strings.Compare(token, config.GlobalConfig.OperationToken) != 0 {
	// 	SendResponse(ctx, -1, "Token error.", nil)
	// 	return
	// }
	buf := ctx.Request.Header.Peek("Authorization")
	tokenString, err := stripBearerPrefixFromTokenString(string(buf))
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			SendResponse(ctx, -1, "not authorization.", nil)
			return nil, fmt.Errorf("not authorization")
		}
		return []byte(config.GlobalConfig.OperationToken), nil
	})
	if err != nil {
		SendResponse(ctx, -1, "not token.", nil)
		return
	}
	if !token.Valid {
		SendResponse(ctx, -1, "Token error.", nil)
		return
	}
	// Check upload File Path
	upFileName := ctx.FormValue("upFileName")
	upFilePath := ctx.FormValue("upFilePath")
	uploadSubPath := string(ctx.FormValue("uploadSubPath"))
	visitPath := "/" + uploadSubPath + "/" + date_util.GetCurTimeFormat(date_util.YYYYMMDD)
	if upFilePath != nil {
		// 如果规定了文件路径和文件
		visitPath = "/" + uploadSubPath + "/" + string(upFilePath)
	}

	dirPath := config.GlobalConfig.UploadPath + visitPath
	if err := gu.CreateDirIfNotExist(dirPath); err != nil {
		zap.S().Error(err)
		SendResponse(ctx, -1, "Failed to create folder.", nil)
		return
	}

	suffix := path.Ext(dfileNmae)
	ext := ctx.FormValue("ext")
	if ext != nil {
		suffix = "." + string(ext)
	}
	filename := createFileName(suffix)
	if upFileName != nil {
		filename = createFileNameByName(string(upFileName), suffix)
	}
	fileAllPath := dirPath + "/" + filename

	// Guarantee that the filename does not duplicate
	if upFileName == nil {
		for {
			if !gu.CheckPathIfNotExist(fileAllPath) {
				break
			}
			filename = createFileName(suffix)
			fileAllPath = dirPath + "/" + filename
		}
	}
	// 保存文件
	reader := bufio.NewReaderSize(raw, 1024*32)
	file, err := os.Create(fileAllPath)
	if err != nil {
		SendResponse(ctx, -1, "Save file fail.", err.Error())
	}
	writer := bufio.NewWriter(file)
	buff := make([]byte, 32*1024)
	written := 0
	zap.S().Info(durl)
	zap.S().Info(fileAllPath)
	go func() {
		for {
			nr, er := reader.Read(buff)
			if nr > 0 {
				nw, ew := writer.Write(buff[0:nr])
				if nw > 0 {
					written += nw
				}
				if ew != nil {
					err = ew
					break
				}
				if nr != nw {
					err = io.ErrShortWrite
					break
				}
			}
			if er != nil {
				if er != io.EOF {
					err = er
				}
				break
			}
		}
		if err != nil {
			panic(err)
		}
	}()
	spaceTime := time.Second * 1
	ticker := time.NewTicker(spaceTime)
	lastWtn := 0
	stop := false
	for {
		select {
		case <-ticker.C:
			speed := written - lastWtn
			zap.S().Info(fmt.Sprintf("[*] Speed %s / %s \n", bytesToSize(speed), spaceTime.String()))
			if written-lastWtn == 0 {
				ticker.Stop()
				stop = true
				break
			}
			lastWtn = written
		}
		if stop {
			break
		}
	}
	// if err := fasthttp.SaveMultipartFile(header, fileAllPath); err != nil {
	// 	zap.S().Error(err)
	// 	SendResponse(ctx, -1, "Save file fail.", err.Error())
	// }

	SendResponse(ctx, 1, "Save file success.", visitPath+"/"+filename)
	return
}

// UploadFileHandler 上传文件
func UploadFileHandler(ctx *fasthttp.RequestCtx) {
	fileUrl := ctx.FormValue("fileUrl")
	if fileUrl != nil {
		UploadFileHandlerCopy(ctx)
		return
	}
	// Get the file from the form
	header, err := ctx.FormFile("file")
	if err != nil {
		SendResponse(ctx, -1, "No file was found.", nil)
		return
	}

	// Check File Size
	if header.Size > int64(config.GlobalConfig.MaxUploadSize) {
		SendResponse(ctx, -1, "File size exceeds limit.", nil)
		return
	}

	// authentication
	// token := string(ctx.FormValue("token"))
	// if strings.Compare(token, config.GlobalConfig.OperationToken) != 0 {
	// 	SendResponse(ctx, -1, "Token error.", nil)
	// 	return
	// }
	buf := ctx.Request.Header.Peek("Authorization")
	tokenString, err := stripBearerPrefixFromTokenString(string(buf))
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			SendResponse(ctx, -1, "not authorization.", nil)
			return nil, fmt.Errorf("not authorization")
		}
		return []byte(config.GlobalConfig.OperationToken), nil
	})
	if err != nil {
		SendResponse(ctx, -1, "not token.", nil)
		return
	}
	if !token.Valid {
		SendResponse(ctx, -1, "Token error.", nil)
		return
	}
	// Check upload File Path
	upFileName := ctx.FormValue("upFileName")
	upFilePath := ctx.FormValue("upFilePath")
	uploadSubPath := string(ctx.FormValue("uploadSubPath"))
	visitPath := "/" + uploadSubPath + "/" + date_util.GetCurTimeFormat(date_util.YYYYMMDD)
	if upFilePath != nil {
		// 如果规定了文件路径和文件
		visitPath = "/" + uploadSubPath + "/" + string(upFilePath)
	}

	dirPath := config.GlobalConfig.UploadPath + visitPath
	if err := gu.CreateDirIfNotExist(dirPath); err != nil {
		zap.S().Error(err)
		SendResponse(ctx, -1, "Failed to create folder.", nil)
		return
	}

	suffix := path.Ext(header.Filename)
	ext := ctx.FormValue("ext")
	if ext != nil {
		suffix = "." + string(ext)
	}
	filename := createFileName(suffix)
	if upFileName != nil {
		filename = createFileNameByName(string(upFileName), suffix)
	}
	fileAllPath := dirPath + "/" + filename
	// Guarantee that the filename does not duplicate
	if upFileName == nil {
		for {
			if !gu.CheckPathIfNotExist(fileAllPath) {
				break
			}
			filename = createFileName(suffix)
			fileAllPath = dirPath + "/" + filename
		}
	}
	// 保存文件
	if err := fasthttp.SaveMultipartFile(header, fileAllPath); err != nil {
		zap.S().Error(err)
		SendResponse(ctx, -1, "Save file fail.", err.Error())
	}

	SendResponse(ctx, 1, "Save file success.", visitPath+"/"+filename)
	return
}

func createFileName(suffix string) string {
	// Date and Time + _ + Random Number + File Suffix
	return date_util.GetCurTimeFormat(date_util.YYYYMMddHHmmss) + "_" + gu.GenerateRandomNumber(10) + suffix
}
func createFileNameByName(fileName string, suffix string) string {
	// Date and Time + _ + Random Number + File Suffix
	return fileName + suffix
}
func bytesToSize(length int) string {
	var k = 1024 // or 1024
	var sizes = []string{"Bytes", "KB", "MB", "GB", "TB"}
	if length == 0 {
		return "0 Bytes"
	}
	i := math.Floor(math.Log(float64(length)) / math.Log(float64(k)))
	r := float64(length) / math.Pow(float64(k), i)
	return strconv.FormatFloat(r, 'f', 3, 64) + " " + sizes[int(i)]
}
