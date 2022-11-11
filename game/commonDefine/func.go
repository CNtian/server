package commonDef

import "github.com/golang/glog"

var (
	Info    glog.Verbose
	Warning glog.Verbose
)

//func LOG_Info(args ...interface{}) {
//	if glog.V(3) {
//		glog.InfoDepth(1, args...)
//	}
//}
//
//func LOG_Warning(args ...interface{}) {
//	if glog.V(2) {
//		glog.WarningDepth(1, args...)
//	}
//}
//
//func LOG_Error(args ...interface{}) {
//	if glog.V(1) {
//		glog.ErrorDepth(1, args...)
//	}
//}
