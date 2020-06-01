package service

import (
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/LinkinStars/golang-util/gu"
	"github.com/dgrijalva/jwt-go"
	"github.com/valyala/fasthttp"
	"go.uber.org/zap"

	"github.com/aassaadd/sgfs/config"
	"github.com/aassaadd/sgfs/util/date_util"
)

// Reader 下载进度实体
type Reader struct {
	io.Reader
	Total   int64
	Current int64
}

func (r *Reader) Read(p []byte) (n int, err error) {
	n, err = r.Reader.Read(p)
	r.Current += int64(n)
	zap.S().Info(fmt.Sprintf("进度 %.2f%%", float64(r.Current*10000/r.Total)/100))
	return
}
func downloadFileProgress(url, filename string) (written int64, err error) {
	r, err := http.Get(url)
	if err != nil {
		zap.S().Error(err)
	}
	defer r.Body.Close()
	f, err := os.Create(filename)
	if err != nil {
		zap.S().Error(err)
	}
	defer f.Close()
	reader := &Reader{
		Reader: r.Body,
		Total:  r.ContentLength,
	}
	return io.Copy(f, reader)

}
func copy(oldpath string, newpath string) {
	//打开源文件
	fileRead, err := os.Open(oldpath)
	if err != nil {
		zap.S().Error(err)
		return
	}
	defer fileRead.Close()
	//创建目标文件
	fileWrite, err := os.Create(newpath)
	if err != nil {
		zap.S().Error(err)
		return
	}
	defer fileWrite.Close()

	//从源文件获取数据，放到缓冲区
	buf := make([]byte, 4096)
	//循环从源文件中获取数据，全部写到目标文件中
	for {
		n, err := fileRead.Read(buf)
		if err != nil && err == io.EOF {
			zap.S().Info(fmt.Sprintf("备份完毕，n = %d \n:", n))
			return
		}
		fileWrite.Write(buf[:n]) //读多少、写多少
	}

}

// Strips 'Bearer ' prefix from bearer token string
func stripBearerPrefixFromTokenString(tok string) (string, error) {
	// Should be a bearer token
	if len(tok) > 6 && strings.ToUpper(tok[0:7]) == "BEARER " {
		return tok[7:], nil
	}
	return tok, nil
}

// SaveFile 通过url保存文件
var SaveFile = make(chan AsyncSaveFile)

// AsyncSaveFile 图片保存
type AsyncSaveFile struct {
	Durl        string  // 下载地址
	FileAllPath string  // 存放地址
	Bak         *string // 是否备份只要有数就行
	Token       string  // token
	Suffix      string  // 后缀
	CallbackUrl *string // 回调地址
}

func cal(callbackUrl string) {
	client := http.DefaultClient
	resp, err := client.Get(callbackUrl)
	if err != nil {
		zap.S().Error(err)
	}
	if resp.ContentLength <= 0 {
		zap.S().Error("No file was found.")
	}
	raw := resp.Body
	zap.S().Info(raw)
	defer raw.Close()
}

// UploadFileHandlerCopyByF 从其他url copy
// func UploadFileHandlerCopyByF(ff AsyncSaveFile) {
// 	bak := ff.Bak
// 	durl := ff.Durl
// 	client := http.DefaultClient
// 	callbackUrl := ff.CallbackUrl
// 	client.Timeout = time.Second * 60 //设置超时时间
// 	resp, err := client.Get(durl)
// 	if err != nil {
// 		zap.S().Error(err)
// 	}
// 	if resp.ContentLength <= 0 {
// 		zap.S().Error("No file was found.")
// 	}
// 	raw := resp.Body
// 	defer raw.Close()
// 	//
// 	// Check File Size
// 	// if header.Size > int64(config.GlobalConfig.MaxUploadSize) {
// 	// 	SendResponse(ctx, -1, "File size exceeds limit.", nil)
// 	// 	return
// 	// }

// 	// authentication
// 	// token := string(ctx.FormValue("token"))
// 	// if strings.Compare(token, config.GlobalConfig.OperationToken) != 0 {
// 	// 	SendResponse(ctx, -1, "Token error.", nil)
// 	// 	return
// 	// }
// 	// buf := ctx.Request.Header.Peek("Authorization")
// 	tokenString, err := stripBearerPrefixFromTokenString(ff.Token)
// 	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
// 		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
// 			// SendResponse(ctx, -1, "not authorization.", nil)
// 			return nil, fmt.Errorf("not authorization")
// 		}
// 		return []byte(config.GlobalConfig.OperationToken), nil
// 	})
// 	if err != nil {
// 		zap.S().Error("not token.")
// 		return
// 	}
// 	if !token.Valid {
// 		zap.S().Error("Token error.")
// 		return
// 	}
// 	// 保存文件之前 先备份
// 	if bak != nil {
// 		copy(ff.FileAllPath, ff.FileAllPath+"."+createFileName(ff.Suffix))
// 	}
// 	// 保存文件
// 	reader := bufio.NewReaderSize(raw, 1024*32)
// 	file, err := os.Create(ff.FileAllPath)
// 	if err != nil {
// 		zap.S().Error("Save file fail.")
// 	}
// 	writer := bufio.NewWriter(file)
// 	buff := make([]byte, 32*1024)
// 	written := 0
// 	zap.S().Info(ff.Durl)
// 	zap.S().Info(ff.FileAllPath)
// 	go func() {
// 		for {
// 			nr, er := reader.Read(buff)
// 			if nr > 0 {
// 				nw, ew := writer.Write(buff[0:nr])
// 				if nw > 0 {
// 					written += nw
// 				}
// 				if ew != nil {
// 					err = ew
// 					break
// 				}
// 				if nr != nw {
// 					err = io.ErrShortWrite
// 					break
// 				}
// 			}
// 			if er != nil {
// 				if er != io.EOF {
// 					err = er
// 				}
// 				break
// 			}
// 		}
// 		if err != nil {
// 			zap.S().Error(err)
// 		}
// 	}()
// 	spaceTime := time.Second * 1
// 	ticker := time.NewTicker(spaceTime)
// 	lastWtn := 0
// 	stop := false
// 	for {
// 		select {
// 		case <-ticker.C:
// 			speed := written - lastWtn
// 			zap.S().Info(fmt.Sprintf("[*] Speed %s / %s \n", bytesToSize(speed), spaceTime.String()))
// 			if written-lastWtn == 0 {
// 				ticker.Stop()
// 				stop = true
// 				break
// 			}
// 			lastWtn = written
// 		}
// 		if stop {
// 			break
// 		}
// 	}
// 	// if err := fasthttp.SaveMultipartFile(header, fileAllPath); err != nil {
// 	// 	zap.S().Error(err)
// 	// 	SendResponse(ctx, -1, "Save file fail.", err.Error())
// 	// }
// 	// 如果有回调 更新回调
// 	if callbackUrl != nil {
// 		cal(*callbackUrl)
// 	}
// 	zap.S().Info("Save file success.")
// 	zap.S().Info(ff.FileAllPath)
// 	// SendResponse(ctx, 1, "Save file success.", visitPath+"/"+filename)
// 	return
// }
func UploadFileHandlerCopyByF(ff AsyncSaveFile) {
	bak := ff.Bak
	durl := ff.Durl
	// client := http.DefaultClient
	callbackUrl := ff.CallbackUrl
	// client.Timeout = time.Second * 60 //设置超时时间
	// resp, err := client.Get(durl)
	// if err != nil {
	// zap.S().Error(err)
	// }
	// if resp.ContentLength <= 0 {
	// 	zap.S().Error("No file was found.")
	// }
	// raw := resp.Body
	// defer raw.Close()
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
	// buf := ctx.Request.Header.Peek("Authorization")
	tokenString, err := stripBearerPrefixFromTokenString(ff.Token)
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			// SendResponse(ctx, -1, "not authorization.", nil)
			return nil, fmt.Errorf("not authorization")
		}
		return []byte(config.GlobalConfig.OperationToken), nil
	})
	if err != nil {
		zap.S().Error("not token.")
		return
	}
	if !token.Valid {
		zap.S().Error("Token error.")
		return
	}
	// 保存文件之前 先备份
	if bak != nil {
		copy(ff.FileAllPath, ff.FileAllPath+"."+createFileName(ff.Suffix))
	}
	// 保存文件
	// reader := bufio.NewReaderSize(raw, 1024*32)
	// file, err := os.Create(ff.FileAllPath)
	// if err != nil {
	// 	zap.S().Error("Save file fail.")
	// }
	_, err = downloadFileProgress(durl, ff.FileAllPath)
	if err != nil {
		zap.S().Error("Save file fail.")
		return
	}
	// 如果有回调 更新回调
	if callbackUrl != nil {
		cal(*callbackUrl)
	}
	zap.S().Info("Save file success.")
	zap.S().Info(ff.FileAllPath)
	// SendResponse(ctx, 1, "Save file success.", visitPath+"/"+filename)
	return
}

// UploadFileHandlerCopy 从其他url copy
func UploadFileHandlerCopy(ctx *fasthttp.RequestCtx) {
	fileUrl := ctx.FormValue("fileUrl")
	bak := ctx.FormValue("bak")
	callbackUrl := ctx.FormValue("callbackUrl")
	durl := string(fileUrl)
	uri, err := url.ParseRequestURI(durl)
	if err != nil {
		SendResponse(ctx, -1, "No file was found.", nil)
	}
	dfileNmae := path.Base(uri.Path)
	buf := ctx.Request.Header.Peek("Authorization")
	// tokenString, err := stripBearerPrefixFromTokenString(string(buf))
	// token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
	// 	if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
	// 		SendResponse(ctx, -1, "not authorization.", nil)
	// 		return nil, fmt.Errorf("not authorization")
	// 	}
	// 	return []byte(config.GlobalConfig.OperationToken), nil
	// })
	// if err != nil {
	// 	SendResponse(ctx, -1, "not token.", nil)
	// 	return
	// }
	// if !token.Valid {
	// 	SendResponse(ctx, -1, "Token error.", nil)
	// 	return
	// }
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
	ff := AsyncSaveFile{
		Durl:        durl,
		FileAllPath: fileAllPath,
		Token:       string(buf),
		Suffix:      suffix,
	}
	if bak != nil {
		b := string(bak)
		ff.Bak = &b
	}
	if callbackUrl != nil {
		c := string(callbackUrl)
		ff.CallbackUrl = &c
	}
	SaveFile <- ff
	SendResponse(ctx, 1, "Save file success.", visitPath+"/"+filename)
	return
}

// func UploadFileHandlerCopy(ctx *fasthttp.RequestCtx) {
// 	fileUrl := ctx.FormValue("fileUrl")
// 	bak := ctx.FormValue("bak")
// 	durl := string(fileUrl)
// 	uri, err := url.ParseRequestURI(durl)
// 	if err != nil {
// 		SendResponse(ctx, -1, "No file was found.", nil)
// 	}
// 	dfileNmae := path.Base(uri.Path)
// 	client := http.DefaultClient
// 	client.Timeout = time.Second * 60 //设置超时时间
// 	resp, err := client.Get(durl)
// 	if err != nil {
// 		SendResponse(ctx, -1, "No file was found.", nil)
// 	}
// 	if resp.ContentLength <= 0 {
// 		SendResponse(ctx, -1, "No file was found.", nil)
// 	}
// 	raw := resp.Body
// 	defer raw.Close()
// 	//
// 	// Check File Size
// 	// if header.Size > int64(config.GlobalConfig.MaxUploadSize) {
// 	// 	SendResponse(ctx, -1, "File size exceeds limit.", nil)
// 	// 	return
// 	// }

// 	// authentication
// 	// token := string(ctx.FormValue("token"))
// 	// if strings.Compare(token, config.GlobalConfig.OperationToken) != 0 {
// 	// 	SendResponse(ctx, -1, "Token error.", nil)
// 	// 	return
// 	// }
// 	buf := ctx.Request.Header.Peek("Authorization")
// 	tokenString, err := stripBearerPrefixFromTokenString(string(buf))
// 	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
// 		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
// 			SendResponse(ctx, -1, "not authorization.", nil)
// 			return nil, fmt.Errorf("not authorization")
// 		}
// 		return []byte(config.GlobalConfig.OperationToken), nil
// 	})
// 	if err != nil {
// 		SendResponse(ctx, -1, "not token.", nil)
// 		return
// 	}
// 	if !token.Valid {
// 		SendResponse(ctx, -1, "Token error.", nil)
// 		return
// 	}
// 	// Check upload File Path
// 	upFileName := ctx.FormValue("upFileName")
// 	upFilePath := ctx.FormValue("upFilePath")
// 	uploadSubPath := string(ctx.FormValue("uploadSubPath"))
// 	visitPath := "/" + uploadSubPath + "/" + date_util.GetCurTimeFormat(date_util.YYYYMMDD)
// 	if upFilePath != nil {
// 		// 如果规定了文件路径和文件
// 		visitPath = "/" + uploadSubPath + "/" + string(upFilePath)
// 	}

// 	dirPath := config.GlobalConfig.UploadPath + visitPath
// 	if err := gu.CreateDirIfNotExist(dirPath); err != nil {
// 		zap.S().Error(err)
// 		SendResponse(ctx, -1, "Failed to create folder.", nil)
// 		return
// 	}

// 	suffix := path.Ext(dfileNmae)
// 	ext := ctx.FormValue("ext")
// 	if ext != nil {
// 		suffix = "." + string(ext)
// 	}
// 	filename := createFileName(suffix)
// 	if upFileName != nil {
// 		filename = createFileNameByName(string(upFileName), suffix)
// 	}
// 	fileAllPath := dirPath + "/" + filename

// 	// Guarantee that the filename does not duplicate
// 	if upFileName == nil {
// 		for {
// 			if !gu.CheckPathIfNotExist(fileAllPath) {
// 				break
// 			}
// 			filename = createFileName(suffix)
// 			fileAllPath = dirPath + "/" + filename
// 		}
// 	}
// 	// 保存文件之前 先备份
// 	if bak != nil {
// 		copy(fileAllPath, fileAllPath+"."+createFileName(suffix))
// 	}
// 	// 保存文件
// 	reader := bufio.NewReaderSize(raw, 1024*32)
// 	file, err := os.Create(fileAllPath)
// 	if err != nil {
// 		SendResponse(ctx, -1, "Save file fail.", err.Error())
// 	}
// 	writer := bufio.NewWriter(file)
// 	buff := make([]byte, 32*1024)
// 	written := 0
// 	zap.S().Info(durl)
// 	zap.S().Info(fileAllPath)
// 	go func() {
// 		for {
// 			nr, er := reader.Read(buff)
// 			if nr > 0 {
// 				nw, ew := writer.Write(buff[0:nr])
// 				if nw > 0 {
// 					written += nw
// 				}
// 				if ew != nil {
// 					err = ew
// 					break
// 				}
// 				if nr != nw {
// 					err = io.ErrShortWrite
// 					break
// 				}
// 			}
// 			if er != nil {
// 				if er != io.EOF {
// 					err = er
// 				}
// 				break
// 			}
// 		}
// 		if err != nil {
// 			zap.S().Error(err)
// 		}
// 	}()
// 	spaceTime := time.Second * 1
// 	ticker := time.NewTicker(spaceTime)
// 	lastWtn := 0
// 	stop := false
// 	for {
// 		select {
// 		case <-ticker.C:
// 			speed := written - lastWtn
// 			zap.S().Info(fmt.Sprintf("[*] Speed %s / %s \n", bytesToSize(speed), spaceTime.String()))
// 			if written-lastWtn == 0 {
// 				ticker.Stop()
// 				stop = true
// 				break
// 			}
// 			lastWtn = written
// 		}
// 		if stop {
// 			break
// 		}
// 	}
// 	// if err := fasthttp.SaveMultipartFile(header, fileAllPath); err != nil {
// 	// 	zap.S().Error(err)
// 	// 	SendResponse(ctx, -1, "Save file fail.", err.Error())
// 	// }

// 	SendResponse(ctx, 1, "Save file success.", visitPath+"/"+filename)
// 	return
// }

// UploadFileHandler 上传文件 需要高延时
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
