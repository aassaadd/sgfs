package main

import (
	"github.com/LinkinStars/golang-util/gu"
	"go.uber.org/zap"

	"github.com/aassaadd/sgfs/config"
	"github.com/aassaadd/sgfs/service"

	"github.com/buaazp/fasthttprouter"
	"github.com/valyala/fasthttp"
)

var (
	corsAllowHeaders     = "authorization"
	corsAllowMethods     = "HEAD,GET,POST,PUT,DELETE,OPTIONS"
	corsAllowOrigin      = "*"
	corsAllowCredentials = "true"
)

func CORS(next fasthttp.RequestHandler) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {

		ctx.Response.Header.Set("Access-Control-Allow-Credentials", corsAllowCredentials)
		ctx.Response.Header.Set("Access-Control-Allow-Headers", corsAllowHeaders)
		ctx.Response.Header.Set("Access-Control-Allow-Methods", corsAllowMethods)
		ctx.Response.Header.Set("Access-Control-Allow-Origin", corsAllowOrigin)

		next(ctx)
	}
}

func main() {

	gu.InitEasyZapDefault("sgfs")

	config.LoadConf()

	zap.S().Info("Simple golang file server is starting...")

	// Create the uploaded file directory if it does not exist
	if err := gu.CreateDirIfNotExist(config.GlobalConfig.UploadPath); err != nil {
		panic(err)
	}

	startStaticFileServer()

	startOperationServer()
	startLoginServer()
	for {
		// 这里做异步下载
		select {
		case f := <-service.SaveFile:
			go service.UploadFileHandlerCopyByF(f)
		}
	}
}

func startStaticFileServer() {
	fs := &fasthttp.FS{
		Root: config.GlobalConfig.UploadPath,

		// Generate a file directory index. If true, access to the root path can see all the files stored.
		// In a production environment, it is recommended to set false
		GenerateIndexPages: config.GlobalConfig.GenerateIndexPages,

		// Open compression for bandwidth savings
		Compress: true,
	}

	go func() {
		if err := fasthttp.ListenAndServe(config.GlobalConfig.VisitPort, CORS(fs.NewRequestHandler())); err != nil {
			panic(err)
		}
	}()
}

func startOperationServer() {
	router := fasthttprouter.New()

	// Add panic handler
	router.PanicHandler = func(ctx *fasthttp.RequestCtx, err interface{}) {
		zap.S().Error(err)
		service.SendResponse(ctx, -1, "Unexpected error", err)
	}
	// router.POST("/login", service.LoginHandler)
	router.POST("/upload-file", service.UploadFileHandler)
	router.POST("/delete-file", service.DeleteFileHandler)

	fastServer := &fasthttp.Server{
		Handler:            CORS(router.Handler),
		MaxRequestBodySize: config.GlobalConfig.MaxRequestBodySize,
	}
	go func() {
		if err := fastServer.ListenAndServe(config.GlobalConfig.OperationPort); err != nil {
			panic(err)
		}
	}()
}
func startLoginServer() {
	router := fasthttprouter.New()

	// Add panic handler
	router.PanicHandler = func(ctx *fasthttp.RequestCtx, err interface{}) {
		zap.S().Error(err)
		service.SendResponse(ctx, -1, "Unexpected error", err)
	}
	router.POST("/login", service.LoginHandler)

	fastServer := &fasthttp.Server{
		Handler:            CORS(router.Handler),
		MaxRequestBodySize: config.GlobalConfig.MaxRequestBodySize,
	}
	go func() {
		if err := fastServer.ListenAndServe(config.GlobalConfig.LoginPort); err != nil {
			panic(err)
		}
	}()
}
