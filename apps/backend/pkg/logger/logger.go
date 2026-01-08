package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	currentLogger *zap.Logger
	sugarLogger   *zap.SugaredLogger
	logMutex      sync.RWMutex
	logFile       *os.File
)

func init() {
	currentLogger = zap.NewNop()
	sugarLogger = currentLogger.Sugar()
}

// Config 日志配置（避免依赖 internal 包）
type Config struct {
	LogLevel    string
	LogFilePath string
}

// Init 根据配置初始化全局日志器，输出到文件和标准输出。
// cfg 可以为 nil，此时使用默认配置（Info 级别，输出到 stdout）
func Init(cfg *Config) error {
	logMutex.Lock()
	defer logMutex.Unlock()

	if logFile != nil {
		_ = logFile.Close()
		logFile = nil
	}

	level := zapcore.InfoLevel
	if cfg != nil {
		rawLevel := strings.ToLower(strings.TrimSpace(cfg.LogLevel))
		if rawLevel != "" {
			if err := level.UnmarshalText([]byte(rawLevel)); err != nil {
				level = zapcore.InfoLevel
			}
		}
	}

	encoderCfg := zap.NewProductionEncoderConfig()
	encoderCfg.TimeKey = "timestamp"
	encoderCfg.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderCfg.EncodeDuration = zapcore.MillisDurationEncoder
	encoderCfg.EncodeLevel = zapcore.CapitalLevelEncoder

	consoleEncoder := zapcore.NewConsoleEncoder(encoderCfg)
	consoleCore := zapcore.NewCore(consoleEncoder, zapcore.AddSync(os.Stdout), level)

	fileCore := zapcore.NewNopCore()

	if cfg != nil {
		target := strings.TrimSpace(cfg.LogFilePath)
		if target == "" {
			target = "server.log"
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return fmt.Errorf("创建日志目录失败: %w", err)
		}
		file, err := os.OpenFile(target, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
		if err != nil {
			return fmt.Errorf("打开日志文件失败: %w", err)
		}
		logFile = file
		fileEncoder := zapcore.NewJSONEncoder(encoderCfg)
		fileCore = zapcore.NewCore(fileEncoder, zapcore.AddSync(file), level)
	}

	core := zapcore.NewTee(consoleCore, fileCore)
	logger := zap.New(core, zap.AddCaller())
	currentLogger = logger
	sugarLogger = logger.Sugar()
	return nil
}

// L 返回结构化日志器。
func L() *zap.Logger {
	logMutex.RLock()
	defer logMutex.RUnlock()
	return currentLogger
}

// S 返回SugaredLogger，便于快速字段输出。
func S() *zap.SugaredLogger {
	logMutex.RLock()
	defer logMutex.RUnlock()
	return sugarLogger
}

// Sync 刷新并关闭底层输出资源。
func Sync() {
	logMutex.Lock()
	defer logMutex.Unlock()
	if currentLogger != nil {
		_ = currentLogger.Sync()
	}
	if logFile != nil {
		_ = logFile.Close()
		logFile = nil
	}
}
