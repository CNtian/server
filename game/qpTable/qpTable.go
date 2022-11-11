package qpTable

import (
	"encoding/json"
	"fmt"
	"github.com/golang/glog"
	"go.mongodb.org/mongo-driver/bson/primitive"
	commonDef "qpGame/commonDefine"
	"qpGame/commonDefine/mateProto"
	"qpGame/commonDefine/mateProto/protoGameBasic"
	"qpGame/commonDefine/mateProto/protoInnerServer"
	"qpGame/db"
	"qpGame/wrapMQ"
	"time"
)

type QPGameTable interface {
	OnMessage(protocol *mateProto.MessageMaTe) int32
	GetBaseQPTable() *QPTable
	GetMaxRound() int32
}

type NewSeatFunc func(playerID PlayerID, seatNumber SeatNumber) (QPSeat, error)

type TableStatus uint32

const (
	TS_Invalid            TableStatus = 0x8000000 // 无效
	TS_Idle               TableStatus = 0         // 空闲
	TS_WaitingPlayerEnter TableStatus = 1         // 1<<0	等待玩家进入(游戏未开始)
	TS_WaitingReady       TableStatus = 2         // 1<<1	游戏后,等待玩家准备
	TS_Playing            TableStatus = 4         // 1<<3	游戏进行中
	TS_Dissolve           TableStatus = 8         // 1<<4 是否正在解散房间
	TS_CustomDefineBase   TableStatus = 32        // 自定义状态起始值
)

type TableDissolveType int32

const (
	DT_Vote        = 1 // 投票解散
	DT_LiveTimeout = 2 // 桌子生命周期已到
	DT_Playing     = 3 // 正常打完
	DT_ScoreLess   = 4 // 分数不够
	DT_Enforcement = 5 // 强制解散
	DT_Trusteeship = 6 // 托管
)

type PlayerPosInfo struct {
	AUID     int64   `json:"auid"`
	BUID     int64   `json:"buid"`
	Distance float64 `json:"distance"`
}

type QPTable struct {
	GameID             int32                  // 玩法ID
	PlayName           string                 // 玩法名称
	TableNum           int32                  // 桌子编号
	MaxPlayers         int32                  // 最大座位人数
	CurXRound          int32                  // 当前玩局数
	Status             TableStatus            // 桌子状态
	GameStartTime      time.Time              // 游戏开始时间
	SeatArr            []QPSeat               // 座位
	SpectatorMap       map[PlayerID]*QPPlayer // 旁观者
	OperateRec         PlayerOperRec          // 游戏步骤记录
	TableRule          *TableRuleConfig       // 桌子配置
	GameTimer          QPTimer                // 定时器
	DissolveSeatNum    SeatNumber             // 解散发起人
	LaunchDissolveTime int64                  // 发起解散时,时间戳
	CreateTime         time.Time              // 创建时间戳
	RecordID           primitive.ObjectID     // 记录ID
	RootTable          QPGameTable            // 顶层桌子
	DissolveType       int32                  // 桌子解散原因
	GpsInfo            []PlayerPosInfo        // GPS距离信息
	GameOverFunc       func()                 // 游戏 结束 回调
	PayUID             int64                  // 支付(房卡|钻石)的玩家
	Consumables        int32                  // 消耗(房卡|钻石)
	ClubRule           *DBClubRule            // 俱乐部规则
	ClubRuleText       string                 // 俱乐部规则
	LookerCount        int32                  // 观看人数
	FirstRoundReadTime int64                  // 首局准备的时间
	IsUnReady          bool                   // 是否遗弃准备
	MaxTZCount         int                    // 最大同桌数
	TZKEY              string                 // 同桌数Key

	MZClubID   int32 // 盟主俱乐部ID
	ClubPlayID int64 // 盟主俱乐部 玩法ID

	curPlayerCount int32       // 当前在座人数
	newSeatFunc    NewSeatFunc // 新建座位 回调
	IsPlaying      bool        // 是否在玩

	lastNoticeRobotTime              time.Time
	RobotJoinPlaying, RobotJoinReady int32
	RobotInviteTimer                 int64
}

// 清空每一小局数据
func (this *QPTable) CleanRoundData() {

	for _, v := range this.SeatArr {
		if v == nil {
			continue
		}
		v.CleanRoundData()
	}
}

func NewQPTable(tableNumber int32, tableCfg string, newSeatFunc NewSeatFunc) (*QPTable, int32, error) {
	rule, err := ParseTableConfig(tableCfg)
	if err != nil {
		return nil, mateProto.Err_CreateTableParam, err
	}

	t := &QPTable{
		curPlayerCount: 0,
		TableNum:       tableNumber,
		TableRule:      rule,
		Status:         TS_WaitingPlayerEnter,
		SpectatorMap:   make(map[PlayerID]*QPPlayer),
		newSeatFunc:    newSeatFunc,
		CreateTime:     time.Now(),
		RecordID:       primitive.NewObjectID(),
	}

	t.TableRule = rule
	t.IsUnReady = false

	return t, 0, nil
}

func (this *QPTable) SetMaxPlayers(value int32) {
	this.MaxPlayers = value
	this.SeatArr = make([]QPSeat, value)
}

// 消息处理
func (this *QPTable) OnMessage(msg *mateProto.MessageMaTe) int32 {

	switch msg.MessageID {
	case protoGameBasic.ID_PlayerInteractive:
		return this.onPlayerInteractive(msg)
	case protoGameBasic.ID_ReqSitDown:
		return this.OnSitDown(msg)
	case protoGameBasic.ID_GameReady:
		return this.OnReady(msg)
	case protoGameBasic.ID_ReqLeaveTable:
		return this.OnLeave(msg)
	case protoGameBasic.ID_PlayerNetStatus:
		return this.OnPlayerNetStatus(msg)
	case protoGameBasic.ID_LaunchDissolveTable:
		return this.OnPlayerDissolveTable(msg)
	case protoGameBasic.ID_DissolveTableVote:
		return this.OnDissolveTableVote(msg)
	case protoGameBasic.ID_GameStart:
		return this.OnGameStart(msg)
	case protoGameBasic.ID_CancelTrusteeship:
		return this.OnCancelTrusteeship(msg)
	case protoGameBasic.ID_TableExpire:
		return this.OnTableExpire(msg)
	case protoGameBasic.ID_GetRoundOverMsg:
		return this.onGetRoundOverMsg(msg)
	case protoGameBasic.ID_GetGPSInfo:
		return this.onGetSeatGPSInfo(msg)
	case protoGameBasic.ID_ActiveTrusteeship:
		return this.OnActiveTrusteeship(msg)
	case protoGameBasic.ID_LookerLeave:
		return this.onLookerLeave(msg)
	default:

	}
	return 0
}

func (this *QPTable) GetBaseQPTable() *QPTable {
	return this
}

func (this *QPTable) GetMaxRound() int32 {
	return 0
}

func (this *QPTable) GetTableNumber() int32 {
	return this.TableNum
}

// 检查桌子的状态
func (this QPTable) IsAssignTableState(value TableStatus) bool {
	if (this.Status & value) == value {
		return true
	}
	return false
}

//当前坐位上的玩家人数
func (this QPTable) GetCurSeatCount() int32 {
	return this.curPlayerCount
}

// 获取座位的数据
func (this QPTable) GetSeatDataByPlayerID(playerID PlayerID) QPSeat {
	for i, _ := range this.SeatArr {
		if this.SeatArr[i] != nil &&
			this.SeatArr[i].GetSeatData().Player != nil &&
			this.SeatArr[i].GetSeatData().Player.ID == playerID {
			return this.SeatArr[i]
		}
	}
	return nil
}

// 游戏开始
func (this *QPTable) GameStart(msg *mateProto.MessageMaTe) int32 {
	if this.IsAssignTableState(TS_Playing) == true {
		return mateProto.Err_GameStarted
	}

	playings := 0
	for _, v := range this.SeatArr {
		if v == nil {
			continue
		}
		seatData := v.GetSeatData()
		if seatData.IsAssignSeatState(SS_Ready) == true {
			seatData.AppendState(SS_Playing)
			seatData.DelState(SS_Ready)
			seatData.IsPlayed = true
			playings += 1

			this.NotifyPlayerStateChange(seatData.Number)
		}
		seatData.RoundOverMsg = nil
	}

	if playings < 2 {
		return mateProto.Err_PlayerNotEnough
	}

	this.GameStartTime = time.Now()
	this.DelTableState(TS_WaitingPlayerEnter | TS_WaitingReady)
	this.AppendTableState(TS_Playing)
	this.IsPlaying = true

	this.GameTimer.RemoveByTimeID(protoGameBasic.TIMER_AutoRedy)

	if this.MZClubID > 0 && this.CurXRound < 1 {
		// 转发给 俱乐部
		var msg mateProto.MessageMaTe
		msg.To, msg.MessageID = fmt.Sprintf("%d", this.MZClubID), protoGameBasic.ID_TableStatusChanged
		msgBody := protoGameBasic.SS_TableStatusChanged{
			ClubID:      this.MZClubID,
			GameID:      this.GameID,
			TableNumber: this.TableNum,
		}

		err := wrapMQ.SendMsgTo(&msg, &msgBody)
		if err != nil {
			glog.Warning("ableStatusChanged() err. err :=", err.Error())
		}
	}
	this.BroadcastTableStatus()
	return mateProto.Err_Success
}

// 游戏结束
func (this *QPTable) GameOver() int32 {
	this.DelTableState(TS_Playing)
	this.AppendTableState(TS_WaitingReady)
	this.IsPlaying = false

	this.LookerCount = 0
	for _, v := range this.SeatArr {
		if v == nil {
			continue
		}
		tempSeatData := v.GetSeatData()

		tempSeatData.DelState(SS_Playing)
		v.CleanRoundData()
		if tempSeatData.IsAssignSeatState(SS_Trusteeship) == true {
			tempSeatData.CurTuoGuanRound += 1
		}
		if this.TableRule.TuoGuanRoundCount != 0 {
			if tempSeatData.CurTuoGuanRound >= this.TableRule.TuoGuanRoundCount {
				this.DissolveType = DT_Trusteeship
				return mateProto.Err_TuoGuanLimit
			}
		}

		if this.ClubRule != nil {
			switch this.ClubRule.GameOverCon {
			case 1: //小局负分
				if tempSeatData.SeatScore < 0 {
					this.DissolveType = DT_ScoreLess
					return mateProto.Err_ClubRuleLimit
				}
			case 2: //低于多少分结束
				if tempSeatData.SeatScore+tempSeatData.ClubScore <= this.ClubRule.OverScoreInt {
					this.DissolveType = DT_ScoreLess
					return mateProto.Err_ClubRuleLimit
				}
			case 3: //低于多少分观看
				if tempSeatData.SeatScore+tempSeatData.ClubScore <= this.ClubRule.OverScoreInt {
					tempSeatData.SetState(SS_Looker | SS_Sitdown)
					this.NotifyPlayerStateChange(tempSeatData.Number)
				}
			default:
			}
		}

		if tempSeatData.IsAssignSeatState(SS_Looker) == true {
			this.LookerCount += 1
		}
	}

	if this.curPlayerCount-this.LookerCount < 2 {
		this.DissolveType = DT_ScoreLess
		return mateProto.Err_ClubRuleLimit
	}
	this.BroadcastTableStatus()
	return 0
}

// 坐下
func (this *QPTable) SitDown(playerID PlayerID, seatNumber SeatNumber) int32 {

	// 是否已经在座位上
	for i, _ := range this.SeatArr {
		if this.SeatArr[i] == nil || this.SeatArr[i].GetSeatData().Player == nil {
			continue
		}

		if this.SeatArr[i].GetSeatData().Player.ID == playerID {
			if this.SeatArr[i].GetSeatData().IsLeave != 0 {
				return mateProto.Err_ActionNotMatchStatus
			}
			return int32(i)
		}
	}

	if this.curPlayerCount >= this.MaxPlayers {
		return mateProto.Err_TableFull
	}

	var (
		seatData QPSeat
		err      error
	)

	//系统选择座位
	if seatNumber == INVALID_SEAT_NUMBER {
		if len(this.SeatArr) == int(this.MaxPlayers) {
			//查找空位
			for i, _ := range this.SeatArr {

				if this.SeatArr[i] != nil {
					continue
				}
				seatData, err = this.newSeatFunc(playerID, SeatNumber(i))
				if err != nil {
					return mateProto.Err_NewSeatFailed
				}
				this.SeatArr[i] = seatData
				break
			}
		} else {
			seatData, err = this.newSeatFunc(playerID, SeatNumber(len(this.SeatArr)))
			if err != nil {
				return mateProto.Err_NewSeatFailed
			}
			this.SeatArr[len(this.SeatArr)-1] = seatData
		}
	} else {
		//玩家自己选择座位

		if int32(seatNumber) >= this.MaxPlayers {
			return mateProto.Err_InvalidSeatNumber
		}

		//是否已经有人
		if int(seatNumber) < len(this.SeatArr) && this.SeatArr[seatNumber] != nil {
			return mateProto.Err_SeatFull
		}

		//占位
		for i := len(this.SeatArr); i <= int(seatNumber); i++ {
			this.SeatArr = append(this.SeatArr, nil)
		}

		seatData, err = this.newSeatFunc(playerID, SeatNumber(len(this.SeatArr)))
		if err != nil {
			return mateProto.Err_NewSeatFailed
		}
		this.SeatArr[seatNumber] = seatData
	}

	//没有空位
	if seatData == nil {
		return mateProto.Err_NotFindIdleSeat
	}

	this.curPlayerCount += 1
	this.DeleteSpectator(playerID)

	playerSeatArr := make([]int64, 0, 4)
	for _, v := range this.SeatArr {
		if v != nil {
			playerSeatArr = append(playerSeatArr, int64(v.GetSeatData().Player.ID))
		}
	}
	err = db.UpdateTablePlayer(this.TableNum, playerSeatArr)
	if err != nil {
		glog.Warning("UpdateTablePlayer(). err:=", err.Error())
	}

	return int32(seatData.GetSeatData().Number)
}

// 删除旁观者
func (this *QPTable) DeleteSpectator(id PlayerID) {
	delete(this.SpectatorMap, id)
}

// 通知玩家座位状态的变化
func (this *QPTable) NotifyPlayerStateChange(seatNum SeatNumber) {

	seatData := this.SeatArr[seatNum]

	playerStatus := protoGameBasic.BroadcastPlayerStatus{
		UID:     int64(seatData.GetSeatData().Player.ID),
		SeatNum: int32(seatData.GetSeatData().Number),
		Status:  uint32(seatData.GetSeatData().Status)}

	this.SendToAllPlayer(protoGameBasic.ID_BroadPlayerStatus, &playerStatus)
}

// 发送给所有玩家
func (this *QPTable) SendToAllPlayer(msgID int32, msg interface{}) {

	for _, v := range this.SeatArr {
		if v == nil {
			continue
		}
		v.GetSeatData().Player.SendData(msgID, msg)
	}
}

// 单独发送给一个人
func (this *QPTable) SendToSeat(seatNumber SeatNumber, msgID int32, msgData interface{}) {
	this.SeatArr[seatNumber].GetSeatData().Player.SendData(msgID, msgData)
}

func (this *QPTable) SendMsgToSeat(seatNumber SeatNumber, msg *mateProto.MessageMaTe) {
	this.SeatArr[seatNumber].GetSeatData().Player.SendMsg(msg)
}

func (this *QPTable) BroadCastGameEvent(msgID int32, msg interface{}) {

	this.OperateRec.PutBroadStep(msgID, msg)

	for _, v := range this.SeatArr {
		if v == nil {
			continue
		}
		v.GetSeatData().Player.SendData(msgID, msg)
	}
}

func (this *QPTable) SendGameEventToSeat(seatNumber SeatNumber, msgID int32, msgData interface{}) {
	this.OperateRec.PutServiceStep(int32(seatNumber), msgID, msgData)

	this.SeatArr[seatNumber].GetSeatData().Player.SendData(msgID, msgData)
}

// 清理座位
func (this *QPTable) ClearSeat(value SeatNumber) {
	this.SeatArr[value] = nil
	this.curPlayerCount -= 1
	this.GameTimer.RemoveBySeatNum(int32(value))
}

// 获取正在玩的下一位玩家
func (this *QPTable) GetNextPlayingSeat(cur SeatNumber) QPSeat {

	for i := cur + 1; int(i) < len(this.SeatArr); i++ {
		if this.SeatArr[i] != nil &&
			this.SeatArr[i].GetSeatData().IsAssignSeatState(SS_Playing) == true {
			return this.SeatArr[i]
		}
	}

	for i := 0; SeatNumber(i) < cur; i++ {
		if this.SeatArr[i] != nil &&
			this.SeatArr[i].GetSeatData().IsAssignSeatState(SS_Playing) == true {
			return this.SeatArr[i]
		}
	}
	return nil
}

// 获取 下一位有效座位
func (this *QPTable) GetNextValidSeat(cur SeatNumber) QPSeat {

	for i := cur + 1; int(i) < len(this.SeatArr); i++ {
		if this.SeatArr[i] != nil &&
			this.SeatArr[i].GetSeatData().IsAssignSeatState(SS_Looker) == false {
			return this.SeatArr[i]
		}
	}

	for i := 0; SeatNumber(i) < cur; i++ {
		if this.SeatArr[i] != nil &&
			this.SeatArr[i].GetSeatData().IsAssignSeatState(SS_Looker) == false {
			return this.SeatArr[i]
		}
	}
	return nil
}

// 获取正在玩的上一位玩家
func (this *QPTable) GetPrePlayingSeat(cur SeatNumber) QPSeat {
	for i := cur - 1; i >= 0; i-- {
		if this.SeatArr[i] != nil &&
			this.SeatArr[i].GetSeatData().IsAssignSeatState(SS_Playing) == true {
			return this.SeatArr[i]
		}
	}

	for i := len(this.SeatArr) - 1; i > int(cur); i-- {
		if this.SeatArr[i] != nil &&
			this.SeatArr[i].GetSeatData().IsAssignSeatState(SS_Playing) == true {
			return this.SeatArr[i]
		}
	}

	return nil
}

// 添加桌子的状态
func (this *QPTable) AppendTableState(value TableStatus) {
	this.Status |= value
}

// 删除某个状态
func (this *QPTable) DelTableState(value TableStatus) {
	this.Status &= ^value
}

// 设置某个状态
func (this *QPTable) SetTableState(value TableStatus) {
	this.Status = value
}

func (this *QPTable) BroadcastTableStatus() {
	this.BroadCastGameEvent(protoGameBasic.ID_GameTableStatusChanged,
		&protoGameBasic.SC_GameTableStatusChanged{Status: uint32(this.Status)})
}

// 玩家坐下
func (this *QPTable) OnSitDown(msg *mateProto.MessageMaTe) int32 {

	var rspCode int32 = -1

	if this.MaxTZCount > 0 && msg.Source != "robot" {
		arr := make([]db.PlayerTongZhuoCount, 0, 8)
		for _, v := range this.SeatArr {
			if v == nil {
				continue
			}
			uid := v.GetSeatData().Player.ID
			arr = append(arr, db.PlayerTongZhuoCount{UID: int64(uid)})
		}
		err := db.GetMaxTongZhuo(this.TZKEY, msg.SenderID, &arr)
		if err != nil {
			glog.Warning("GetMaxTongZhuo() err.", err.Error())
		} else {
			for i, _ := range arr {
				if arr[i].Value > this.MaxTZCount {
					return mateProto.Err_MaxTZ
				}
			}
		}
	}

	// 先占位
	storeRes, err := db.StorePlayerGameIntro(msg.SenderID, this.TableNum, this.GameID)
	if err != nil {
		glog.Warning("uid:=", msg.SenderID, " StorePlayerGameIntro(). err:=", err.Error())
		return mateProto.Err_SystemError
	}
	if storeRes == false {
		return mateProto.Err_AlreadyPlaying
	}

	// 坐下失败时,释放
	defer func() {
		if rspCode < 0 {
			_, err := db.RemovePlayerGameIntro(msg.SenderID)
			if err != nil {
				glog.Error("uid:=", msg.SenderID, " RemovePlayerGameIntro(). err:=", err.Error())
			}
		}
	}()

	reqSitDown, _ := msg.MsgBody.(*protoGameBasic.CS_ReqSitDown)
	rspCode = this.SitDown(PlayerID(msg.SenderID), SeatNumber(reqSitDown.SeatNumber))
	if rspCode < 0 {
		return rspCode
	}

	player := this.SeatArr[rspCode].GetSeatData().Player
	player.LoginSrc = msg.Source

	player.Head, player.Nick, player.Sex, err = db.GetPlayerIntro(msg.SenderID)
	if err != nil {
		glog.Warning("uid:=", msg.SenderID, " not get player intro. err:=", err.Error())
	}

	this.TableRule.OnJoinTable()

	return rspCode
}

// 玩家准备
func (this *QPTable) OnReady(msg *mateProto.MessageMaTe) int32 {
	playerID := PlayerID(msg.SenderID)

	if this.IsAssignTableState(TS_WaitingPlayerEnter) == false &&
		this.IsAssignTableState(TS_WaitingReady) == false {
		return mateProto.Err_ActionNotMatchStatus
	}

	seat := this.GetSeatDataByPlayerID(playerID)
	if seat == nil {
		return mateProto.Err_NotFindPlayer
	}
	seatData := seat.GetSeatData()
	if seatData.IsAssignSeatState(SS_Ready) == true {
		return mateProto.Err_ActionNotMatchStatus
	}

	if seatData.IsAssignSeatState(SS_Looker) == true {
		return mateProto.Err_ActionNotMatchStatus
	}

	seatData.AppendState(SS_Ready)
	seatData.RoundOverMsg = nil

	this.NotifyPlayerStateChange(seatData.Number)

	this.GameTimer.RemoveBySeatNum(int32(seatData.Number))

	return mateProto.Err_Success
}

// 玩家离开
func (this *QPTable) OnLeave(msg *mateProto.MessageMaTe) int32 {
	playerID := PlayerID(msg.SenderID)

	this.DeleteSpectator(playerID)

	seatData := this.GetSeatDataByPlayerID(playerID)
	if seatData == nil {
		return mateProto.Err_NotFindPlayer
	}

	this.SendToSeat(seatData.GetSeatData().Number, protoGameBasic.ID_ReqLeaveTable, &protoGameBasic.JsonResponse{Status: 0})

	this.ClearSeat(seatData.GetSeatData().Number)

	this.GameTimer.RemoveByTimeID(protoGameBasic.TIMER_Leave)
	this.GameTimer.RemoveByTimeID(protoGameBasic.TIMER_FirstRoundReady)
	this.FirstRoundReadTime = 0

	playerSeatArr := make([]int64, 0, 4)

	for _, v := range this.SeatArr {
		if v == nil {
			continue
		}
		playerSeatArr = append(playerSeatArr, int64(v.GetSeatData().Player.ID))

		if this.IsUnReady == true {
			seat := v.GetSeatData()
			if seat.IsAssignSeatState(SS_Ready) == true {
				seat.DelState(SS_Ready)
				this.NotifyPlayerStateChange(seat.Number)
			}
		}
	}
	err := db.UpdateTablePlayer(this.TableNum, playerSeatArr)
	if err != nil {
		glog.Warning("UpdateTablePlayer(). err:=", err.Error())
	}

	for i, v := range this.GpsInfo {
		if v.BUID == msg.SenderID || v.AUID == msg.SenderID {
			this.GpsInfo[i].AUID, this.GpsInfo[i].BUID = 0, 0
			this.GpsInfo[i].Distance = 0.0
		}
	}

	this.SendToAllPlayer(protoGameBasic.ID_BroadPlayerLeaveTable,
		&protoGameBasic.BroadPlayerLeaveTable{
			UID: msg.SenderID, SeatNum: int32(seatData.GetSeatData().Number),
		})

	return mateProto.Err_Success
}

// 游戏开始
func (this *QPTable) OnGameStart(msg *mateProto.MessageMaTe) int32 {
	if this.GetCurSeatCount() < 1 {
		return mateProto.Err_PlayerNotEnough
	}
	return this.GameStart(msg)
}

// 玩家网络断开
func (this *QPTable) OnPlayerNetStatus(msg *mateProto.MessageMaTe) int32 {
	playerID := PlayerID(msg.SenderID)

	msgNetStatus := protoGameBasic.PlayerNetStatus{}
	err := json.Unmarshal(msg.Data, &msgNetStatus)
	if err != nil {
		return mateProto.Err_ProtocolDataErr
	}

	this.DeleteSpectator(playerID)

	seat := this.GetSeatDataByPlayerID(playerID)
	if seat == nil {
		return mateProto.Err_NotFindPlayer
	}
	seatData := seat.GetSeatData()
	//if seatData.IsAssignSeatState(SS_Looker) == true {
	//	return 0
	//}

	if msgNetStatus.IsOnline == false {
		if seatData.Player.LoginSrc == msg.Source {
			seatData.AppendState(SS_Offline)
		}
	} else {
		seatData.Player.LoginSrc = msg.Source
		seatData.DelState(SS_Offline)
	}

	this.NotifyPlayerStateChange(seatData.Number)

	return mateProto.Err_Success
}

// 玩家发起投票解散
func (this *QPTable) OnPlayerDissolveTable(msg *mateProto.MessageMaTe) int32 {
	if this.TableRule.JieSanMode == 0 {
		return mateProto.Err_NotMatchTableRule
	}

	if this.Status < TS_WaitingReady || this.IsAssignTableState(TS_Dissolve) == true {
		return mateProto.Err_ActionNotMatchStatus
	}

	seat := this.GetSeatDataByPlayerID(PlayerID(msg.SenderID))
	if seat == nil {
		return mateProto.Err_NotFindPlayer
	}

	seat.GetSeatData().DissolveVote = 1
	this.DissolveSeatNum = seat.GetSeatData().Number
	this.LaunchDissolveTime = time.Now().Unix()

	this.AppendTableState(TS_Dissolve)

	replyMsg := protoGameBasic.BroadcastDissolveTableResult{
		SeatNum: int32(seat.GetSeatData().Number)}

	this.SendToAllPlayer(protoGameBasic.ID_LaunchDissolveTable, &replyMsg)

	// 超时解散
	this.GameTimer.PutTableTimer(protoGameBasic.TIMER_DissolveTable, 60*1000, func() {

		this.SetTableState(TS_Invalid)

		this.SendToAllPlayer(protoGameBasic.ID_DissolveTableVoteReslut,
			&protoGameBasic.DissolveTableVoteResult{IsDissolveTable: true})

		if this.GameOverFunc != nil {
			this.GameOverFunc()
		}
	})

	return mateProto.Err_Success
}

func (this *QPTable) OnDissolveTableVote(msg *mateProto.MessageMaTe) int32 {

	if this.IsAssignTableState(TS_Dissolve) == false {
		return mateProto.Err_ActionNotMatchStatus
	}

	seat := this.GetSeatDataByPlayerID(PlayerID(msg.SenderID))
	if seat == nil {
		return mateProto.Err_NotFindPlayer
	}
	var msgDissolveTableVote protoGameBasic.CS_DissolveTableVote
	err := json.Unmarshal(msg.Data, &msgDissolveTableVote)
	if err != nil {
		return mateProto.Err_ProtocolDataErr
	}

	if msgDissolveTableVote.Vote != 1 && msgDissolveTableVote.Vote != 2 {
		return mateProto.Err_OperationParamErr
	}

	this.SendToAllPlayer(protoGameBasic.ID_DissolveTableVote,
		&protoGameBasic.BroadcastDissolveTableVoteResult{
			SeatNumber: int32(seat.GetSeatData().Number), Vote: msgDissolveTableVote.Vote,
		})

	seat.GetSeatData().DissolveVote = msgDissolveTableVote.Vote

	this.TableRule.OnDissolveTableVote()

	return mateProto.Err_Success
}

// 取消托管
func (this *QPTable) OnCancelTrusteeship(msg *mateProto.MessageMaTe) int32 {
	playerID := PlayerID(msg.SenderID)

	seat := this.GetSeatDataByPlayerID(playerID)
	if seat == nil {
		return mateProto.Err_NotFindPlayer
	}
	seat.GetSeatData().DelState(SS_Trusteeship)
	seat.GetSeatData().CurTuoGuanRound = 0

	this.NotifyPlayerStateChange(seat.GetSeatData().Number)

	return int32(seat.GetSeatData().Number)
}

// 激活托管
func (this *QPTable) OnActiveTrusteeship(msg *mateProto.MessageMaTe) int32 {
	playerID := PlayerID(msg.SenderID)

	seat := this.GetSeatDataByPlayerID(playerID)
	if seat == nil {
		return mateProto.Err_NotFindPlayer
	}
	if seat.GetSeatData().IsAssignSeatState(SS_Trusteeship) == true {
		return mateProto.Err_OperationParamErr
	}
	seat.GetSeatData().AppendState(SS_Trusteeship)

	this.NotifyPlayerStateChange(seat.GetSeatData().Number)

	return int32(seat.GetSeatData().Number)
}

func (this *QPTable) onLookerLeave(msg *mateProto.MessageMaTe) int32 {
	playerID := PlayerID(msg.SenderID)

	seat := this.GetSeatDataByPlayerID(playerID)
	if seat == nil {
		return mateProto.Err_NotFindPlayer
	}
	sv := seat.GetSeatData()
	if sv.IsLeave == 0 && sv.IsAssignSeatState(SS_Looker) == true {
		sv.Player.IsLeave = true
		sv.IsLeave += 1
		return 0
	}

	return mateProto.Err_NotFindPlayer
}

func (this *QPTable) UpdatePlayerSource(seat *SeatData, source string) {

	//glog.Warning("onTableData() tabID:=", this.TableNum, ",uid:=", seat.Player.ID, ",cur source:=", source,
	//	",preSource:=", seat.Player.LoginSrc)

	seat.Player.LoginSrc = source
}

// 座位游戏分变化
func (this *QPTable) ChangeRoundScore(win SeatNumber, score float64, isNotice bool) {

	msgBody := protoGameBasic.BroadcastPlayerScoreChanged{
		WinnerSeatNum: int32(win),
		LoserSeatNum:  make([]int32, 0, 4),
		Score:         commonDef.Float64ToString(score)}

	for i, v := range this.SeatArr {
		if v == nil {
			continue
		}
		if v.GetSeatData().IsAssignSeatState(SS_Playing) == false {
			continue
		}
		if SeatNumber(i) == win {
			continue
		}

		v.GetSeatData().RoundScore -= score
		msgBody.LoserSeatNum = append(msgBody.LoserSeatNum, int32(v.GetSeatData().Number))

		this.SeatArr[win].GetSeatData().RoundScore += score
	}

	if isNotice == true {
		this.BroadCastGameEvent(protoGameBasic.ID_PlayerRoundScoreChanged, &msgBody)
	}
}

// 座位游戏分变化
func (this *QPTable) ChangeWinner_Loser_RoundScore(win, lose SeatNumber, score float64, isNotice bool) {

	//this.CheckBroke(this.SeatArr[lose].GetSeatData(), &score)

	this.SeatArr[win].GetSeatData().RoundScore += score
	this.SeatArr[lose].GetSeatData().RoundScore -= score

	if isNotice == true {
		msgBody := protoGameBasic.BroadcastPlayerScoreChanged{
			WinnerSeatNum: int32(win),
			LoserSeatNum:  make([]int32, 0, 1),
			Score:         commonDef.Float64ToString(score)}

		msgBody.LoserSeatNum = append(msgBody.LoserSeatNum, int32(lose))

		this.BroadCastGameEvent(protoGameBasic.ID_PlayerRoundScoreChanged, &msgBody)
	}
}

//func (this *QPTable) CheckBroke(seat *SeatData, loseGameScore *float64) {
//	if this.TableRule.BrokeType != 1 {
//		return
//	}
//
//	if *loseGameScore > 0 {
//		ts_ := seat.SeatScore + seat.RoundScore
//		if ts_-*loseGameScore <= seat.ClubScore*-1 {
//			*loseGameScore = seat.ClubScore + ts_
//			seat.IsBroke = true
//			return
//		}
//	} else if *loseGameScore < 0 {
//		ts_ := seat.SeatScore + seat.RoundScore
//		if ts_+*loseGameScore <= seat.ClubScore*-1 {
//			*loseGameScore = seat.ClubScore + ts_
//			seat.IsBroke = true
//			*loseGameScore *= -1
//			return
//		}
//	}
//	seat.IsBroke = false
//}

func (this *QPTable) OnTableExpire(msg *mateProto.MessageMaTe) int32 {
	for _, v := range this.SeatArr {
		if v == nil {
			continue
		}
		this.SendToSeat(v.GetSeatData().Number,
			protoGameBasic.ID_ReqLeaveTable,
			&protoGameBasic.JsonResponse{Status: DT_LiveTimeout})
	}
	return 0
}

// 解散桌子 状态
//():同意人数,非同意人数
func (this *QPTable) DissolveTableStatus() (int32, int32) {

	var agreeCount, notDidCount int32
	for _, v := range this.SeatArr {
		if v == nil {
			continue
		}
		switch v.GetSeatData().DissolveVote {
		case 1:
			agreeCount += 1
		case 2:
			notDidCount += 1
		}
	}
	return agreeCount, notDidCount
}

func (this *QPTable) onPlayerInteractive(msg *mateProto.MessageMaTe) int32 {

	seat := this.GetSeatDataByPlayerID(PlayerID(msg.SenderID))
	if seat == nil {
		return mateProto.Err_NotFindPlayer
	}
	msgBody := protoGameBasic.CS_PlayerInteractive{}
	err := json.Unmarshal(msg.Data, &msgBody)
	if err != nil {
		return mateProto.Err_ProtocolDataErr
	}

	msg.Data, _ = json.Marshal(&protoGameBasic.SC_PlayerInteractive{
		SendSeatNum: int32(seat.GetSeatData().Number),
		To:          msgBody.To,
		Type:        msgBody.Type,
		Content:     msgBody.Content,
	})
	this.SendToAllPlayer(protoGameBasic.ID_PlayerInteractive, msg)

	return 0
}

// 非正常打完
func (this *QPTable) RecordNonNormalGameOverRound() {
	if this.IsPlaying == false {
		return
	}

	recPlayerArr := make([]*protoGameBasic.PlayerGameScore, 0, 3)
	for _, v := range this.GetBaseQPTable().SeatArr {
		if v == nil {
			continue
		}
		seat := v.GetSeatData()
		if seat.IsAssignSeatState(SS_Looker) == true {
			continue
		}
		if seat.IsAssignSeatState(SS_Playing) == false {
			continue
		}

		temp := protoGameBasic.PlayerGameScore{
			UID:    int64(seat.Player.ID),
			Nick:   seat.Player.Nick,
			ClubID: seat.ClubID,
			SScore: commonDef.Float64Mul1000ToService(seat.RoundScore),
		}
		recPlayerArr = append(recPlayerArr, &temp)
	}

	gameStepRec, _ := this.OperateRec.Pack()

	roundRecData := protoGameBasic.SS_RoundRecord{
		Begin:    this.GameStartTime,
		End:      time.Now(),
		RoundID:  this.RecordID,
		ClubID:   this.MZClubID,
		TableID:  this.TableNum,
		CurRound: this.CurXRound,
		Players:  recPlayerArr,
		GameStep: string(gameStepRec),
	}
	wrapMQ.SendMsgTo(&mateProto.MessageMaTe{To: "db", MessageID: protoGameBasic.ID_RoundOver}, &roundRecData)
}

func (this *QPTable) onGetRoundOverMsg(msg *mateProto.MessageMaTe) int32 {
	seat := this.GetSeatDataByPlayerID(PlayerID(msg.SenderID))
	if seat == nil {
		return mateProto.Err_NotFindPlayer
	}

	seatData := seat.GetSeatData()

	// 是否要推送 小结算
	if this.IsAssignTableState(TS_WaitingReady) == true && seatData.RoundOverMsg != nil {
		this.SendMsgToSeat(seatData.Number, seatData.RoundOverMsg)
	} else {
		this.SendToSeat(seatData.Number, msg.MessageID, nil)
	}
	return 0
}

func (this *QPTable) onGetSeatGPSInfo(msg *mateProto.MessageMaTe) int32 {
	seat := this.GetSeatDataByPlayerID(PlayerID(msg.SenderID))
	if seat == nil {
		return mateProto.Err_NotFindPlayer
	}

	this.SendToSeat(seat.GetSeatData().Number, protoGameBasic.ID_GetGPSInfo, this.GpsInfo)
	return 0
}

// 通知 机器人
func (this *QPTable) TimerInvitationRobot() {

	if this.MaxPlayers == this.curPlayerCount {
		return
	}

	isSendInvitation := false
	tableStatus := 0
	if this.IsAssignTableState(TS_WaitingPlayerEnter) { // || this.IsAssignTableState(TS_WaitingReady)
		tableStatus = 1
	} else if this.IsAssignTableState(TS_Playing) {
		tableStatus = 2
	}

	if this.RobotJoinReady > 0 && tableStatus == 1 {
		if this.curPlayerCount <= this.RobotJoinReady {
			isSendInvitation = true
		}
	}
	if this.RobotJoinPlaying > 0 && tableStatus == 2 {
		if this.curPlayerCount <= this.RobotJoinPlaying {
			isSendInvitation = true
		}
	}

	if isSendInvitation == true {
		sendMsg := mateProto.MessageMaTe{MessageID: protoInnerServer.ID_CallRobotComeIn, To: "robot",
			MsgBody: &protoInnerServer.MsgCallRobotComeIn{GameID: this.GameID,
				TableID: this.TableNum, MZClubID: this.MZClubID, PlayID: this.ClubPlayID}}

		wrapMQ.SendMsg(&sendMsg)
	}
}
