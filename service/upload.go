package service

import (
	"fmt"
	"path"
	"strings"

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

// UploadFileHandler 上传文件
func UploadFileHandler(ctx *fasthttp.RequestCtx) {
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

	// Save file
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
