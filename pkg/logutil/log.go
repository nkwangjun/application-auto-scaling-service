/*
Package logutil 对logrus进行二次封装，用于提供日志相关接口
*/
package logutil

import (
	"bytes"
	"fmt"
	"os"
	"path"
	"strings"
	"sync"

	log "github.com/sirupsen/logrus"
)

var (
	defaultLogger *log.Logger
	once          sync.Once
)

// LogConf 相关配置项
type LogConf struct {
	Level      string `ini:"level"`
	Path       string `ini:"path"`
	MaxSize    int    `ini:"max_size"`
	MaxBackups int    `ini:"max_backups"`
	MaxDays    int    `ini:"max_days"`
	Compress   bool   `ini:"compress"`
}

// GetLogger 用于获取logger实例
func GetLogger() *log.Logger {
	once.Do(func() {
		defaultLogger = log.New()
	})
	return defaultLogger
}

// Init ...
func Init(config *LogConf) {
	initLogger(GetLogger(), config)
}

func initLogger(logger *log.Logger, config *LogConf) {
	var level log.Level
	logger.SetReportCaller(true)
	logger.SetFormatter(&LogFormatterWithCaller{})
	//logger.SetOutput(&lumberjack.Logger{
	//	Filename:   config.Path,
	//	MaxSize:    config.MaxSize,
	//	MaxBackups: config.MaxBackups,
	//	Compress:   config.Compress,
	//})

	level, err := log.ParseLevel(config.Level)
	if err != nil {
		logger.Warnf("Invalid logging level %s", config.Level)
		level = log.InfoLevel
	}
	logger.SetLevel(level)
	logger.Infof("LogConf level set to %s", strings.ToUpper(level.String()))
}

// LogFormatterWithCaller ...
type LogFormatterWithCaller struct {
}

// Format ...
func (f *LogFormatterWithCaller) Format(entry *log.Entry) ([]byte, error) {
	var (
		result bytes.Buffer
	)
	if entry.Caller != nil {
		result.WriteString(fmt.Sprintf("%s %s [pid:%d] [%s:%s:%v] ",
			entry.Time, strings.ToUpper(entry.Level.String()), os.Getpid(),
			path.Base(entry.Caller.File), entry.Caller.Function, entry.Caller.Line))
	}
	for key, val := range entry.Data {
		result.WriteString(fmt.Sprintf("[%s:%s] ", key, val))
	}
	if _, err := result.WriteString(entry.Message); err != nil {
		return nil, err
	}
	if _, err := result.WriteRune('\n'); err != nil {
		return nil, err
	}
	return result.Bytes(), nil
}
