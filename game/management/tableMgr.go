package management

import (
	"sync"
)

type ServiceStatus int32

const (
	SS_Run           = iota // 0:运行中
	SS_NotCreatTable        // 1:不能创建新的桌子
	SS_NotJoinTable         // 2:不能加入桌子
	SS_Stop                 // 3:停止游戏服务
)

var (
	tableMap      sync.Map
	curTableCount int32
	serviceStatus ServiceStatus
)

func SetServiceStatus(value int32) {
	serviceStatus = ServiceStatus(value)
}

//
//func init() {
//}
//
//func putTable(tableNumber int32, value *rootTable) bool {
//	_, ok := tableMap.Load(tableNumber)
//	if ok == true {
//		return false
//	}
//	tableMap.Store(tableNumber, value)
//
//	tableCount += 1
//	return true
//}
//
//func getTable(tableNumber int32) (interface{}, bool) {
//	return tableMap.Load(tableNumber)
//}
//
//func delTable(tableNum int32) (interface{}, bool) {
//
//	v, ok := tableMap.LoadAndDelete(tableNum)
//	if ok == true {
//		tableCount -= 1
//	}
//	return v, ok
//}
