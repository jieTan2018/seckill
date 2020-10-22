package middlewares

import (
	"path"
	cfg "seckill/layer/configs"
	"time"

	"github.com/rifflock/lfshook"

	rotatelogs "github.com/lestrrat-go/file-rotatelogs"
	log "github.com/sirupsen/logrus"

	"github.com/gin-gonic/gin"
)

// 日志输出到 file
func LoggerToFiles() (gin.HandlerFunc, error) {
	logClient := cfg.Logs
	// 确定日志文件的名称、位置
	fileName := path.Join(cfg.LogDirectoryPath, cfg.LogFileName)
	logClient.SetLevel(log.DebugLevel)
	// 日志分割配置
	logWriter, err := rotatelogs.New(
		fileName+"_%Y-%m-%d"+cfg.LogFileSuffix,
		// rotatelogs.WithLinkName(fileName),         // 生成软链，指向最新日志文件  // linux上才有用
		rotatelogs.WithMaxAge(cfg.WeekTime),      // 文件最大保存时间
		rotatelogs.WithRotationTime(cfg.DayTime), // 日志切割时间间隔
	)
	if err != nil {
		return nil, err
	}
	// gin.DefaultWriter = io.MultiWriter(logWriter, os.Stdout)  // 将控制台的日志写入到文件
	writeMap := lfshook.WriterMap{ // 为不同日志级别设置不同的输出
		log.InfoLevel:  logWriter,
		log.FatalLevel: logWriter,
		log.DebugLevel: logWriter,
		log.WarnLevel:  logWriter,
		log.ErrorLevel: logWriter,
		log.PanicLevel: logWriter,
	}
	lfHook := lfshook.NewHook(writeMap, &log.JSONFormatter{
		TimestampFormat: cfg.TimeFormatString, // 格式化json中的时间显示
	})
	logClient.AddHook(lfHook)

	return func(c *gin.Context) {
		start := time.Now() // 开始时间
		c.Next()            // 处理请求
		end := time.Now()   // 结束时间
		//执行时间
		latency := end.Sub(start)
		// request相关信息
		uri := c.Request.URL.Path
		clientIP := c.ClientIP()
		method := c.Request.Method
		statusCode := c.Writer.Status()
		// 指定日志打印出来的格式。
		logClient.Infof("| %3d | %13v | %15s | %s  %s |",
			statusCode,
			latency,
			clientIP,
			method, uri,
		)
	}, nil
}

// // 日志输出到 redis

// // 日志输出 elasticsearch

// // 日志输出到 MQ
