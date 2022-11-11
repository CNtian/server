package tableSnapshot

import (
	"encoding/json"
	"github.com/golang/glog"
	"sort"
	"unsafe"
	clubProto "vvService/appClub/protoDefine"
	"vvService/appClub/worker/clubEvent"
)

type tablePool struct {
	deskMap map[*TableData]*TableData
}

func (this *tablePool) Put(table *TableData) {
	this.deskMap[table] = table
}
func (this *tablePool) Get() *TableData {

	for _, v := range this.deskMap {
		delete(this.deskMap, v)
		v.JsonTable = v.JsonTable[0:0]
		return v
	}

	return new(TableData)
}

var idleDeskPool tablePool

func init() {
	idleDeskPool.deskMap = make(map[*TableData]*TableData)
}

// ---------------------------------------------------

type ChangeCategory uint32

const (
	defCC      ChangeCategory = 0x0 // 默认值
	updateCC   ChangeCategory = 0x1 // 更新变化
	addCC      ChangeCategory = 0x2 // 新增的桌子
	playingCC  ChangeCategory = 0x4 // 已经在玩的桌子
	dissolveCC ChangeCategory = 0x8 // 解散的桌子
)

type TableMap map[int32]*TableData // key:桌子
type TableDataArr []*TableData     // 桌子数组

type PlayerInfo struct {
	Uid  int64  `json:u`
	Head string `json:"h"`
	Nick string `json:"n"`
}

type TableData struct {
	TableNum          int32          `json:"tabNum"`
	ClubPlayID        int64          `json:"clubPlayID"`
	PlayerArr         []PlayerInfo   `json:"playerArr"` // 最多 10个
	CurChangeCategory ChangeCategory `json:"change"`    // 桌子状态变更
	GameID            int32          `json:"gID"`
	Inc               uint64         `json:"inc"` // 排序依据

	MaxPlayers    int32  `json:"-"`
	OnlinePlayers int32  `json:"-"`
	ServiceID     string `json:"-"`
	JsonTable     []byte `json:"-"`
}

type SortTable []*TableData

func (s SortTable) Len() int      { return len(s) }
func (s SortTable) Swap(i, j int) { s[i], s[j] = s[j], s[i] }
func (s SortTable) Less(i, j int) bool {
	if s[i].CurChangeCategory == s[j].CurChangeCategory {
		if s[i].Inc > s[j].Inc {
			return true
		}
		return false
	}
	if s[i].CurChangeCategory < s[j].CurChangeCategory {
		return true
	}
	return false
}

type groupTableInfo struct {
	allTable []string

	tableMap map[int32]*TableData // key:桌子ID
}

type clubTableData struct {
	playerCount int32 // 玩家总数
	tableCount  int32 // 桌子总数

	allTable []string

	gameTableMap map[int32]*groupTableInfo // key:gameID
	clubPlayMap  map[int64]*groupTableInfo // key:clubPlay
	// 所有桌子
	allTableMap map[int32]*TableData // key:tableID
	// 上次变化过的桌子
	lastChangedTableMap map[int32]*TableData // key:tableID

	// 等待中
	waitingTableMap map[int64]TableMap // key:玩法

	inc    uint64
	clubID int32
}

func NewClubTableData(clubID int32) *clubTableData {
	temp := clubTableData{
		playerCount:         0,
		tableCount:          0,
		waitingTableMap:     make(map[int64]TableMap),
		gameTableMap:        make(map[int32]*groupTableInfo),
		clubPlayMap:         make(map[int64]*groupTableInfo),
		allTableMap:         make(map[int32]*TableData),
		lastChangedTableMap: make(map[int32]*TableData),
		inc:                 0,
		clubID:              clubID,
	}

	return &temp
}

var (
	clubMap = make(map[int32]*clubTableData) // key:俱乐部ID
)

// 同步 onRecoverTable()    同步 onRecoverTable()    同步 onRecoverTable()
func (this *clubTableData) PutNewTable(msg *clubProto.SS_PutNewTable) {
	newTable := idleDeskPool.Get()

	newTable.TableNum = msg.TableNumber
	newTable.ClubPlayID = msg.ClubPlayID
	newTable.MaxPlayers = msg.MaxPlayers
	tPlayer := clubEvent.LoadPlayerNick_Name(msg.UID)
	newTable.PlayerArr = []PlayerInfo{{msg.UID, tPlayer.HeadURL, tPlayer.Nick}}
	newTable.GameID = msg.GameID
	newTable.CurChangeCategory = addCC
	newTable.OnlinePlayers = 1
	newTable.ServiceID = msg.ServiceID
	newTable.Inc = this.inc

	// wait
	{
		tableMap, ok := this.waitingTableMap[newTable.ClubPlayID]
		if ok == false || tableMap == nil {
			tableMap = make(TableMap)
			this.waitingTableMap[newTable.ClubPlayID] = tableMap
		}
		tableMap[newTable.TableNum] = newTable
	}

	// gameID
	groupTable, ok := this.gameTableMap[msg.GameID]
	if ok == false || groupTable == nil {
		groupTable = &groupTableInfo{tableMap: make(map[int32]*TableData)}
		this.gameTableMap[msg.GameID] = groupTable
	}
	groupTable.tableMap[newTable.TableNum] = newTable
	// gameID

	// clubPlayID
	groupTable, ok = this.clubPlayMap[newTable.ClubPlayID]
	if ok == false || groupTable == nil {
		groupTable = &groupTableInfo{tableMap: make(map[int32]*TableData)}
		this.clubPlayMap[newTable.ClubPlayID] = groupTable
	}
	groupTable.tableMap[newTable.TableNum] = newTable
	// clubPlayID

	this.allTableMap[newTable.TableNum] = newTable

	this.lastChangedTableMap[newTable.TableNum] = newTable

	this.playerCount += 1
	this.tableCount += 1
	this.inc += 1
}

func (this *clubTableData) PutPlayer(msg *clubProto.SS_PutPlayerToTable) {
	table, ok := this.allTableMap[msg.TableNumber]
	if ok == false {
		glog.Warning("PutPlayer() not find GameID. TableNumber:=", msg.TableNumber)
		return
	}

	isFind := false
	for i, _ := range table.PlayerArr {
		if table.PlayerArr[i].Uid == 0 {
			tPlayer := clubEvent.LoadPlayerNick_Name(msg.UID)
			table.PlayerArr[i] = PlayerInfo{msg.UID, tPlayer.HeadURL, tPlayer.Nick}
			isFind = true
			break
		}
	}
	if isFind == false {
		if table.PlayerArr == nil {
			table.PlayerArr = make([]PlayerInfo, 0, 10)
		}
		tPlayer := clubEvent.LoadPlayerNick_Name(msg.UID)
		table.PlayerArr = append(table.PlayerArr, PlayerInfo{msg.UID, tPlayer.HeadURL, tPlayer.Nick})
	}
	table.OnlinePlayers += 1
	this.playerCount += 1

	table.CurChangeCategory |= updateCC

	if table.OnlinePlayers >= table.MaxPlayers {
		waitTableMap, ok3 := this.waitingTableMap[table.ClubPlayID]
		if ok3 == true {
			delete(waitTableMap, table.TableNum)
		}
	}

	this.lastChangedTableMap[msg.TableNumber] = table
}

func (this *clubTableData) DeletePlayer(msg *clubProto.SS_DelPlayerInTable) {
	table, ok := this.allTableMap[msg.TableNumber]
	if ok == false {
		glog.Warning("DeletePlayer() not find GameID. TableNumber:=", msg.TableNumber)
		return
	}

	for i, _ := range table.PlayerArr {
		if table.PlayerArr[i].Uid == msg.UID {
			table.PlayerArr[i].Uid = 0
			table.PlayerArr[i].Nick = ""
			table.PlayerArr[i].Head = ""
			table.OnlinePlayers -= 1
			this.playerCount -= 1
			break
		}
	}

	table.CurChangeCategory |= updateCC

	if table.OnlinePlayers < table.MaxPlayers {
		waitTableMap, ok3 := this.waitingTableMap[table.ClubPlayID]
		if ok3 == true {
			waitTableMap[table.TableNum] = table
		}
	}

	this.lastChangedTableMap[msg.TableNumber] = table
}

func (this *clubTableData) DeleteTable(msg *clubProto.SS_DelTable) {

	table, ok := this.allTableMap[msg.TableNumber]
	if ok == false {
		glog.Warning("DeleteTable() not find GameID. TableNumber:=", msg.TableNumber)
		return
	}
	table.CurChangeCategory |= dissolveCC
	this.tableCount -= 1
	this.playerCount -= table.OnlinePlayers

	// gameID
	groupTable, ok := this.gameTableMap[msg.GameID]
	if ok == false {
		glog.Warning("DeleteTable() not find GameID. GameID:=", msg.GameID)
	} else {
		delete(groupTable.tableMap, msg.TableNumber)
	}

	// clubPlayID
	groupTable, ok = this.clubPlayMap[table.ClubPlayID]
	if ok == false {
		glog.Warning("DeleteTable() not find GameID. TableNumber:=", msg.TableNumber)
	} else {
		delete(groupTable.tableMap, msg.TableNumber)
	}

	// wait
	if v, ok := this.waitingTableMap[table.ClubPlayID]; ok == true {
		delete(v, msg.TableNumber)
	}

	delete(this.allTableMap, msg.TableNumber)

	this.lastChangedTableMap[msg.TableNumber] = table
}

func (this *clubTableData) TableStatusChanged(msg *clubProto.SS_TableStatusChanged) {

	table, ok := this.allTableMap[msg.TableNumber]
	if ok == false {
		glog.Warning("TableStatusChanged() not find GameID. TableNumber:=", msg.TableNumber)
		return
	}

	table.CurChangeCategory |= playingCC

	waitTableMap, ok3 := this.waitingTableMap[table.ClubPlayID]
	if ok3 == true {
		delete(waitTableMap, table.TableNum)
	}

	this.lastChangedTableMap[table.TableNum] = table
}

// 记录一秒的变化
func makeTableJson(clubTable *clubTableData) int {

	if len(clubTable.lastChangedTableMap) < 1 {
		return len(clubTable.allTableMap)
	}

	for _, v := range clubTable.lastChangedTableMap {

		if v.CurChangeCategory&dissolveCC == dissolveCC {
			continue
		}
		if v.CurChangeCategory&playingCC == playingCC {
			v.CurChangeCategory = playingCC
		} else {
			v.CurChangeCategory = defCC
		}

		v.JsonTable, _ = json.Marshal(v)
	}

	allTableArr := make(SortTable, 0, 1000)

	// 按游戏类型排序
	for _, v := range clubTable.gameTableMap {
		v.allTable = make([]string, 0, len(v.tableMap))
		sortTableArr := make(SortTable, 0, len(v.tableMap))

		for _, t := range v.tableMap {
			sortTableArr = append(sortTableArr, t)
			allTableArr = append(allTableArr, t)
		}

		sort.Sort(sortTableArr)
		for _, t := range sortTableArr {
			v.allTable = append(v.allTable, *(*string)(unsafe.Pointer(&t.JsonTable)))
		}
		//glog.Warning("gameID:=", k, ", count:=", len(v.allTable))
	}

	// 所有
	clubTable.allTable = make([]string, 0, len(allTableArr))
	sort.Sort(allTableArr)
	for _, t := range allTableArr {
		clubTable.allTable = append(clubTable.allTable, *(*string)(unsafe.Pointer(&t.JsonTable)))
	}
	//glog.Warning("all table count:=", len(clubTable.allTable))

	// 俱乐部玩法
	for _, v := range clubTable.clubPlayMap {
		v.allTable = make([]string, 0, len(v.tableMap))
		sortTableArr := make(SortTable, 0, len(v.tableMap))

		for _, t := range v.tableMap {
			sortTableArr = append(sortTableArr, t)
			allTableArr = append(allTableArr, t)
		}

		sort.Sort(sortTableArr)
		for _, t := range sortTableArr {
			v.allTable = append(v.allTable, *(*string)(unsafe.Pointer(&t.JsonTable)))
		}
		//glog.Warning("clubPlayID:=", k, ", count:=", len(v.allTable))
	}

	// 清空
	clubTable.lastChangedTableMap = map[int32]*TableData{}

	return len(clubTable.allTableMap)
}
