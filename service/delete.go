package service

import (
	"fmt"

	"go.uber.org/zap"

	"github.com/dgrijalva/jwt-go"
	"github.com/valyala/fasthttp"

	"github.com/aassaadd/sgfs/config"
	"github.com/aassaadd/sgfs/util/file_util"
)

//DeleteFileHandler 删除文件
func DeleteFileHandler(ctx *fasthttp.RequestCtx) {
	// authentication
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
	//
	fileUrl := string(ctx.FormValue("fileUrl"))
	if len(fileUrl) == 0 {
		SendResponse(ctx, -1, "FileUrl error.", nil)
		return
	}

	fileUrl = config.GlobalConfig.UploadPath + fileUrl
	if err := file_util.DeleteFile(fileUrl); err != nil {
		zap.S().Error(err)
		SendResponse(ctx, -1, "Delete file error.", err.Error())
		return
	}

	SendResponse(ctx, 1, "Delete file success.", nil)
	return
}
