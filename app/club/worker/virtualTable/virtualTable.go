package virtualTable

import (
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/golang/glog"
	"io"
	"math/rand"
	"os"
	"strconv"
	"sync"
	"time"
	"vvService/appClub/db"
	"vvService/appClub/localConfig"
	clubProto "vvService/appClub/protoDefine"
	"vvService/appClub/worker/tableSnapshot"
	commonDef "vvService/commonPackge"
	"vvService/commonPackge/mateProto"
	collClub "vvService/dbCollectionDefine/club"
)

type virtualTableData struct {
	isDelNow  bool // 立即删除 - 玩法是否已经不存在
	isDelSlow bool

	cfgItem collClub.VirtualTableConfigItem
	c       chan *collClub.VirtualTableConfigItem
}

var (
	virtualClubPlayMap = make(map[int64]*virtualTableData)

	SelfPostEvent         SelfPostEvents
	_mzClubID_            int32
	noticeClubPlayChanged = make(chan interface{}, 10)
)

var (
	lockIndex    sync.Mutex
	_playerIDArr []int64
	_curIndex    int

	lockTableNumIndex sync.Mutex
	_tableNumberPool  []int
	_curTableNumIndex int
)

func getPlayerID() int64 {
	lockIndex.Lock()
	defer lockIndex.Unlock()

	if _curIndex >= len(_playerIDArr) {
		_curIndex = 0
	}
	_curIndex++
	return _playerIDArr[_curIndex-1]
}

func getTableNum() int {
	if len(_tableNumberPool) < 1 {
		return 0
	}
	lockTableNumIndex.Lock()
	defer lockTableNumIndex.Unlock()

	if _curTableNumIndex >= len(_tableNumberPool) {
		_curTableNumIndex = 0
	}
	_curTableNumIndex++
	return _tableNumberPool[_curTableNumIndex-1]
}

type SelfPostEvents struct {
}

func (this *SelfPostEvents) PostMaTeEvent(msg *mateProto.MessageMaTe) {
	noticeClubPlayChanged <- msg
}

func loadClubPlay() {
	virtualTableConfig := []collClub.VirtualTableConfigItem{}
	err := db.GetVirtualTableConfig(_mzClubID_, &virtualTableConfig)
	if err != nil {
		glog.Warning("GetVirtualTableConfig() ", err.Error())
		return
	}

	for _, v := range virtualClubPlayMap {
		v.isDelNow = true
	}
	for i, _ := range virtualTableConfig {
		v, ok := virtualClubPlayMap[virtualTableConfig[i].PlayID]
		if ok == false {
			if virtualTableConfig[i].Status == 0 {
				v = &virtualTableData{
					isDelNow:  false,
					isDelSlow: false,
					cfgItem:   virtualTableConfig[i],
					c:         make(chan *collClub.VirtualTableConfigItem, 8),
				}
				virtualClubPlayMap[virtualTableConfig[i].PlayID] = v
				go work(&v.cfgItem, v.c)
			}
		} else {
			v.cfgItem = virtualTableConfig[i]

			v.isDelNow = false
			if v.cfgItem.Status == 1 {
				v.isDelSlow = true
			}
		}
	}

	if len(virtualTableConfig) < 1 {
		glog.Warning("VirtualTableConfigItem equal 0. ")
		return
	}
}

func InitVirtualTable() {

	{
		temp_, _ := strconv.Atoi(localConfig.GetConfig().ID)
		_mzClubID_ = int32(temp_)
	}

	//loadClubPlay()

	// local 148 2603
	if len(localConfig.GetConfig().VirtualPlayer) < 2 {
		glog.Warning("VirtualPlayer less  ", localConfig.GetConfig().VirtualPlayer)
		return
	}

	var err error
	_playerIDArr, err = db.GetTestPlayerID(localConfig.GetConfig().VirtualPlayer[0], localConfig.GetConfig().VirtualPlayer[1])
	if err != nil {
		glog.Warning("GetTestPlayerID()  ", err.Error())
		return
	}

	err = readTableNumFromFile()
	if err != nil {
		glog.Warning("readTableNumFromFile()  ", err.Error())
		return
	}

	loadClubPlay()

	for commonDef.IsRun {
		select {
		case <-noticeClubPlayChanged:
			loadClubPlay()
			for k, v := range virtualClubPlayMap {
				if v.isDelNow {
					close(v.c)
					delete(virtualClubPlayMap, k)
				} else if v.isDelSlow {
					v.c <- nil // 通知删除
					delete(virtualClubPlayMap, k)
				} else {
					v.c <- &v.cfgItem
				}
			}
		}
	}
}

func work(curCfgItem *collClub.VirtualTableConfigItem, c chan *collClub.VirtualTableConfigItem) {

	loop := rand.Int()%(curCfgItem.Loop2-curCfgItem.Loop1+1) + curCfgItem.Loop1

	tCreateTable := time.NewTimer(time.Second * time.Duration(loop))
	tCheck := time.NewTimer(time.Second)

	showTableCount := rand.Int()%(curCfgItem.ShowTableCount2-curCfgItem.ShowTableCount1+1) + curCfgItem.ShowTableCount1

	playingTableMap := map[int32]time.Time{} // value:结束时间

	gameID := curCfgItem.GameID
	playID := curCfgItem.PlayID

	glog.Warning("open virtual table. ", curCfgItem.GameID, ", ", curCfgItem.PlayID)

	ok := false
	for {
		select {
		case curCfgItem, ok = <-c:
			if ok == false {
				deleteTable(gameID, &playingTableMap, true)
				glog.Warning("close virtual table all. ", gameID, ", ", playID)
				return
			}
			if curCfgItem == nil {
				glog.Warning("close virtual table. ", gameID, ", ", playID)
			}
		case <-tCheck.C:
			tCheck = time.NewTimer(time.Second)
			deleteTable(gameID, &playingTableMap, false)
			if curCfgItem == nil && len(playingTableMap) < 1 {
				return
			}
		case <-tCreateTable.C:
			if curCfgItem == nil {
				break
			}
			if len(playingTableMap) < showTableCount {
				tableNumber := createTable(playingTableMap, curCfgItem)

				runDuration_ := curCfgItem.RunDuration2 - curCfgItem.RunDuration1
				runDuration_ = rand.Intn(runDuration_) + 1
				playingTableMap[tableNumber] = time.Now().Add(time.Duration(runDuration_) * time.Minute)
			}

			//glog.Warning("showTableCount:=", showTableCount, ",curTableCount:=", len(playingTableMap), ",loop:=", loop)

			loop = rand.Int()%(curCfgItem.Loop2-curCfgItem.Loop1+1) + curCfgItem.Loop1
			tCreateTable = time.NewTimer(time.Second * time.Duration(loop))

			showTableCount = rand.Int()%(curCfgItem.ShowTableCount2-curCfgItem.ShowTableCount1+1) + curCfgItem.ShowTableCount1
		}
	}
}

func deleteTable(gameID int32, playing *map[int32]time.Time, isDelAll bool) {
	now_ := time.Now()
	for k, v := range *playing {
		if now_.Sub(v) > 0 || isDelAll {
			delete(*playing, k)

			msg := mateProto.MessageMaTe{MessageID: clubProto.ID_TableDelete}
			childMsg := clubProto.SS_DelTable{
				ClubID:      _mzClubID_,
				GameID:      gameID,
				TableNumber: k,
			}
			msg.Data, _ = json.Marshal(&childMsg)

			tableSnapshot.SelfPostEvent.PostMaTeEvent(&msg)
		}
	}
}

//func testTableRun() {
//
//	now := time.Now()
//
//	// 创建
//	if len(testPlayingTableMap) < 500 {
//		testCreateTable(now)
//	}
//
//	// 改变在玩
//	for k, v := range testReadyTableMap {
//		if now.Sub(v.tableData.CreateTime).Seconds() < 10 {
//			continue
//		}
//
//		msg := mateProto.MessageMaTe{MessageID: clubProto.ID_TableStatusChanged}
//		childMsg := clubProto.SS_TableStatusChanged{
//			ClubID:      _mzClubID_,
//			GameID:      v.tableData.GameID,
//			TableNumber: v.tableData.TableNumber,
//		}
//		msg.Data, _ = json.Marshal(&childMsg)
//
//		tableSnapshot.SelfPostEvent.PostMaTeEvent(&msg)
//
//		v.PlayingTime = now
//		testPlayingTableMap[k] = testReadyTableMap[k]
//		delete(testReadyTableMap, k)
//
//		tm, ok := testTableMap[v.ClubPlayID]
//		if ok == true {
//			tm.readyCount -= 1
//		}
//	}
//
//	// 删除在玩的
//	for k, v := range testPlayingTableMap {
//		ri := rand.Intn(5) + 1
//		rif := float64(ri)
//		if now.Sub(v.PlayingTime).Minutes() < rif {
//			continue
//		}
//
//		msg := mateProto.MessageMaTe{MessageID: clubProto.ID_TableDelete}
//		childMsg := clubProto.SS_DelTable{
//			ClubID:      _mzClubID_,
//			GameID:      v.tableData.GameID,
//			TableNumber: v.tableData.TableNumber,
//		}
//		msg.Data, _ = json.Marshal(&childMsg)
//
//		tableSnapshot.SelfPostEvent.PostMaTeEvent(&msg)
//
//		delete(testPlayingTableMap, k)
//
//		tm, ok := testTableMap[v.ClubPlayID]
//		if ok == true {
//			tm.playing -= 1
//		}
//	}
//}

func createTable(playingMap_ map[int32]time.Time, cfgItem *collClub.VirtualTableConfigItem) int32 {

	r := getTableNum()
	if r < 1 {
		for i := 0; i < 100; i++ {
			r = (rand.Intn(9) + 1) * 100000
			r += rand.Intn(10) * 10000
			r += rand.Intn(10) * 1000
			r += rand.Intn(10) * 100
			r += rand.Intn(10) * 10
			r += rand.Intn(10)

			if _, ok := playingMap_[int32(r)]; ok {
				continue
			}
			break
		}
	}

	uid := getPlayerID()

	{
		msg := mateProto.MessageMaTe{MessageID: clubProto.ID_TablePutNew}

		msg.Data, _ = json.Marshal(&clubProto.SS_PutNewTable{
			ClubID:      _mzClubID_,
			TableNumber: int32(r),
			ClubPlayID:  cfgItem.PlayID,
			GameID:      cfgItem.GameID,
			MaxPlayers:  cfgItem.MaxPlayers,
			UID:         uid,
			CreateTime:  time.Now()})

		tableSnapshot.SelfPostEvent.PostMaTeEvent(&msg)
	}

	for i := int32(1); i < cfgItem.MaxPlayers; i++ {

		uid = getPlayerID()

		msg := mateProto.MessageMaTe{MessageID: clubProto.ID_TablePutPlayer}
		childMsg := clubProto.SS_PutPlayerToTable{
			ClubID:      _mzClubID_,
			GameID:      cfgItem.GameID,
			TableNumber: int32(r),
			UID:         uid,
		}
		msg.Data, _ = json.Marshal(&childMsg)

		tableSnapshot.SelfPostEvent.PostMaTeEvent(&msg)
	}

	{
		msg := mateProto.MessageMaTe{MessageID: clubProto.ID_TableStatusChanged}
		childMsg := clubProto.SS_TableStatusChanged{
			ClubID:      _mzClubID_,
			GameID:      cfgItem.GameID,
			TableNumber: int32(r),
		}
		msg.Data, _ = json.Marshal(&childMsg)

		tableSnapshot.SelfPostEvent.PostMaTeEvent(&msg)
	}

	return int32(r)
}

func readTableNumFromFile() error {
	file, err := os.Open("./tableNumber.list")
	if err != nil {
		return err
	}
	defer file.Close()

	tableNumberArr := make([]string, 0, 5000)
	br := bufio.NewReader(file)

	i, j := 0, 0
	for ; i < 800000; i++ {
		_, _, c := br.ReadLine()
		if c == io.EOF {
			break
		}
	}

	for ; i < 805000; i++ {
		text, _, c := br.ReadLine()
		if c == io.EOF {
			break
		}
		tableNumberArr = append(tableNumberArr, string(text))
		j++
	}

	if len(tableNumberArr) < 5000 {
		return fmt.Errorf("int32(len(beans.Roomlist)) < cfg.RoomNumberEndIndex. %d", len(tableNumberArr))
	}
	_tableNumberPool = make([]int, 0, 5000)

	for j = 0; j < len(tableNumberArr); j++ {
		var roomNumber int
		roomNumber, err = strconv.Atoi(tableNumberArr[j])
		if err != nil {
			return fmt.Errorf("strconv.ParseInt() err.err:=%s,text:=%s", err.Error(), tableNumberArr[j])
		}
		_tableNumberPool = append(_tableNumberPool, roomNumber)
	}

	_curTableNumIndex = 0
	return nil
}
