package service

import (
	"time"

	"github.com/valyala/fasthttp"
	"go.uber.org/zap"

	"github.com/aassaadd/sgfs/config"
	"github.com/dgrijalva/jwt-go"
)

// LoginHandler 登录获得token
func LoginHandler(ctx *fasthttp.RequestCtx) {
	token := jwt.New(jwt.SigningMethodHS256)
	claims := make(jwt.MapClaims)
	claims["exp"] = time.Now().Add(time.Hour * time.Duration(1)).Unix()
	claims["iat"] = time.Now().Unix()
	token.Claims = claims
	tokenString, err := token.SignedString([]byte(config.GlobalConfig.OperationToken))
	if err != nil {
		zap.S().Error(err)
		SendResponse(ctx, -1, "Get token fail.", err.Error())
		return
	}
	SendResponse(ctx, 1, "Get token success.", tokenString)
	return
}
