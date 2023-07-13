package log

import (
	"fmt"
	"io"
	"log"
	"os"
	"sync"

	"github.com/ThreeKing2018/gocolor"
)

// 日志等级
const (
	DEBUG = iota
	INFO
	WARNING
	ERROR
	DEFAULT_FLAG    = log.LstdFlags
	LSHORTFILE_FLAG = log.Lshortfile | log.LstdFlags
)

type ColorLogger interface {
	Debug(format string, s ...interface{})
	Info(format string, s ...interface{})
	Warn(format string, s ...interface{})
	Error(format string, s ...interface{})
	SetLevel(level int)
}

type Logger struct {
	Level   int
	debug   *log.Logger
	info    *log.Logger
	warning *log.Logger
	error   *log.Logger
	IsColor bool
	Depth   int
	Prefix  string
	wg      *sync.WaitGroup
}

var defaultLogger = InitWriteLogger(os.Stdout, 2, DEFAULT_FLAG, true)

func Debug(format string, s ...interface{}) {
	defaultLogger.Debug(format, s...)
}
func Info(format string, s ...interface{}) {
	defaultLogger.Info(format, s...)
}
func Warn(format string, s ...interface{}) {
	defaultLogger.Warn(format, s...)
}
func Error(format string, s ...interface{}) {
	defaultLogger.Error(format, s...)
}

// New 返回一个 ColorLogger.
func New(w io.Writer, isColor bool) ColorLogger {
	return InitWriteLogger(w, 2, LSHORTFILE_FLAG, isColor)
}

func InitWriteLogger(w io.Writer, depth, flag int, isColor bool) ColorLogger {
	logger := new(Logger)
	logger.wg = new(sync.WaitGroup)
	logger.IsColor = isColor
	logger.Depth = depth
	logger.debug = log.New(w, logger.setColorString(DEBUG, "[DEBUG🛠]"), flag)
	logger.info = log.New(w, logger.setColorString(INFO, "[INFO📝]"), flag)
	logger.warning = log.New(w, logger.setColorString(WARNING, "[WARNING❗]"), flag)
	logger.error = log.New(w, logger.setColorString(ERROR, "[ERROR❌]"), flag)

	logger.SetLevel(DEBUG)
	return logger
}

func (l *Logger) Debug(format string, s ...interface{}) {
	if l.Level > DEBUG {
		return
	}
	l.debug.Output(l.Depth, l.setColor(DEBUG, format, s...))
}

func (l *Logger) Info(format string, s ...interface{}) {
	if l.Level > INFO {
		return
	}
	l.info.Output(l.Depth, l.setColor(INFO, format, s...))
}

func (l *Logger) Warn(format string, s ...interface{}) {
	if l.Level > WARNING {
		return
	}
	l.warning.Output(l.Depth, l.setColor(WARNING, format, s...))
}

func (l *Logger) Error(format string, s ...interface{}) {
	if l.Level > ERROR {
		return
	}
	l.error.Output(l.Depth, l.setColor(ERROR, format, s...))
}

// SetLevel 配置日志等级 默认是 DEBUG.
func (l *Logger) SetLevel(level int) {
	l.Level = level
}

// setColorString 设置字体背景颜色.
func (l *Logger) setColorString(level int, format string, args ...interface{}) string {
	if false == l.IsColor {
		return fmt.Sprintf(format, args...)
	}
	switch level {
	case DEBUG:
		return gocolor.SMagentaBG(format, args...)
	case INFO:
		return gocolor.SGreenBG(format, args...)
	case WARNING:
		return gocolor.SYellowBG(format, args...)
	case ERROR:
		return gocolor.SRedBG(format, args...)
	default:
		return fmt.Sprintf(format, args...)
	}
}

// setColor 设置不同字体颜色.
func (l *Logger) setColor(level int, format string, args ...interface{}) string {
	if l.IsColor == false {
		return fmt.Sprintf(format, args...)
	}
	switch level {
	case DEBUG:
		return gocolor.SMagenta(format, args...)
	case INFO:
		return gocolor.SGreen(format, args...)
	case WARNING:
		return gocolor.SYellow(format, args...)
	case ERROR:
		return gocolor.SRed(format, args...)
	default:
		return fmt.Sprintf(format, args...)
	}
}
