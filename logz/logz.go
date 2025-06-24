package logz


// 这里使用logrus作为日志库
import (
	"github.com/sirupsen/logrus"
)

// 全局配置Logrus
var Logrus *logrus.Logger

// 全局配置Logrus