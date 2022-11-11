package management

import (
	"qpGame/db"
	"sync"
	"sync/atomic"
)

var playerMap sync.Map
var playerCount int32

func putPlayer(playerID int64, value *rootTable) {
	playerMap.Store(playerID, value)
	atomic.AddInt32(&playerCount, 1)
}

func deletePlayer(playerID int64) (bool, error) {
	if _, ok := playerMap.LoadAndDelete(playerID); ok == true {
		atomic.AddInt32(&playerCount, -1)
	}

	return db.RemovePlayerGameIntro(playerID)
}

func getPlayer(playerID int64) *rootTable {
	v, ok := playerMap.Load(playerID)
	if ok == false {
		return nil
	}
	return v.(*rootTable)
}
