package ZhaJinHua

import (
	"encoding/json"
	"fmt"
	"github.com/golang/glog"
	"github.com/shopspring/decimal"
	"math/rand"
	commonDef "qpGame/commonDefine"
	"qpGame/commonDefine/mateProto"
	"qpGame/commonDefine/mateProto/protoGameBasic"
	"qpGame/db"
	"qpGame/qpTable"
	"qpGame/wrapMQ"
	"sort"
	"strconv"
	"time"
)

const timerFaPai = protoGameBasic.ZhaJinHua
const timerCheckUnreadyPlayer = protoGameBasic.ZhaJinHua + 1
const timerBankerOperation = protoGameBasic.ZhaJinHua + 2
const timerQiPai = protoGameBasic.ZhaJinHua + 3
const timerMenPai = protoGameBasic.ZhaJinHua + 4

type ZhaJinHuaTable struct {
	PaiMgr      *zhaJinHuaPaiMgr // 牌的管理器
	logic       zhaJinHuaLogic
	table       *qpTable.QPTable
	gameRule    ZhaJinHuaRule
	launchTimer bool

	playingSeatMap map[qpTable.SeatNumber]struct{} // 当前还在玩的 玩家数量

	// 小局 待清理 成员
	playingSeatArr   []int32 // 在玩座位
	minXiaZhuIndex1  int32   // 最小下注
	minXiaZhuIndex2  int32   // 最小下注
	xRound           int32
	curOperatorSeat  qpTable.SeatNumber // 当前操作的座位
	bankerSeatNumber qpTable.SeatNumber
	xiaZhuCount      float64 // 下注总数
}

// 清空每一小局数据
func (this *ZhaJinHuaTable) CleanRoundData() {
	this.table.CleanRoundData()

	this.playingSeatArr = nil
	this.minXiaZhuIndex1 = this.gameRule.XiaZhuOpt
	this.minXiaZhuIndex2 = 0
	this.xRound = 0
	this.curOperatorSeat = qpTable.INVALID_SEAT_NUMBER
	this.xiaZhuCount = 0.0
}

func (this *ZhaJinHuaTable) SetTableNumber(tabNumber int32) {
	this.table.TableNum = tabNumber
}

func (this *ZhaJinHuaTable) GetStatus() int32 {
	return int32(this.table.Status)
}

func (this *ZhaJinHuaTable) ParseTableOptConfig(gameRuleCfg string) (rspCode int32, err error) {

	err = json.Unmarshal([]byte(gameRuleCfg), &this.gameRule)
	if err != nil {
		rspCode = mateProto.Err_CreateTableParam
		return
	}

	err = this.gameRule.CheckField()
	if err != nil {
		rspCode = mateProto.Err_CreateTableParam
		return
	}

	this.gameRule.RuleJson = gameRuleCfg
	return
}

func (this *ZhaJinHuaTable) GetMaxRound() int32 {
	return this.gameRule.MaxRoundCount
}

func (this *ZhaJinHuaTable) OnMessage(msg *mateProto.MessageMaTe) int32 {

	switch msg.MessageID {
	case protoGameBasic.ID_PrivateJoinGameTable:
		return this.onPrivateJoinTable(msg)
	case protoGameBasic.ID_ClubJoinTable:
		return this.onClubJoinTable(msg)
	case protoGameBasic.ID_ReqLeaveTable:
		return this.onPlayerLeave(msg)
	case protoGameBasic.ID_GameReady:
		return this.onReady(msg)
	case protoGameBasic.ID_TableExpire:
		return this.onTableExpire(msg)
	case ID_TableData:
		return this.onTableData(msg)
	case protoGameBasic.ID_DissolveTableVote:
		return this.onDissolveTableVote(msg)
	case ID_CustomShouPai:
		return this.onCustomShouPai(msg)
	case ID_XiaZhu:
		return this.onXiaZhu(msg)
	case ID_KanPai:
		return this.onKanPai(msg)
	case ID_QiPai:
		return this.onQiPai(msg)
	case ID_BiPai:
		return this.onBiPai(msg)
	case ID_GenDaoDi:
		return this.onGenDaoDi(msg)
	case protoGameBasic.ID_ForceDissolveTable:
		return this.onForceDissolveTable(msg)
	case ID_GetPai:
		return this.onGetPai(msg)
	case ID_ChangePai:
		return this.onChangePai(msg)
	default:
		return this.table.OnMessage(msg)
	}
}

func (this *ZhaJinHuaTable) GetBaseQPTable() *qpTable.QPTable {
	return this.table
}

func (this *ZhaJinHuaTable) onPrivateJoinTable(msg *mateProto.MessageMaTe) int32 {

	//if this.table.Status > qpTable.TS_WaitingPlayerEnter {
	//	return mateProto.Err_GameStarted
	//}
	if this.table.GetCurSeatCount() >= this.table.MaxPlayers {
		return mateProto.Err_TableFull
	}

	joinTable := msg.MsgBody.(*protoGameBasic.CS_PrivateJoinGameTable)
	if this.table.TableRule.CheckIP(joinTable.IP) == false {
		return mateProto.Err_FindIPRepeat
	}

	if tempInt32 := this.table.TableRule.CheckGPS(msg.SenderID, joinTable.Latitude, joinTable.Longitude); tempInt32 != 0 {
		return tempInt32
	}

	msgSitdown := protoGameBasic.CS_ReqSitDown{SeatNumber: int32(qpTable.INVALID_SEAT_NUMBER)}

	// 临时转换成请求坐下
	msg.MessageID = protoGameBasic.ID_ReqSitDown
	msg.MsgBody = &msgSitdown
	rspCode := this.table.OnMessage(msg)

	if rspCode >= 0 {
		seatData := this.table.SeatArr[rspCode].GetSeatData()
		seatData.Player.IP = joinTable.IP
		seatData.Lat, seatData.Lng = joinTable.Latitude, joinTable.Longitude
	}

	// 还原成 原始ID
	msg.MessageID = protoGameBasic.ID_PrivateJoinGameTable

	return rspCode
}

func (this *ZhaJinHuaTable) onClubJoinTable(msg *mateProto.MessageMaTe) int32 {

	//if this.table.Status > qpTable.TS_WaitingPlayerEnter {
	//	return mateProto.Err_GameStarted
	//}
	if this.table.GetCurSeatCount() >= this.table.MaxPlayers {
		return mateProto.Err_TableFull
	}

	clubJoinTable := msg.MsgBody.(*protoGameBasic.CS_ClubJoinTable)
	if this.table.TableRule.CheckIP(clubJoinTable.IP) == false {
		return mateProto.Err_FindIPRepeat
	}

	if tempInt32 := this.table.TableRule.CheckGPS(msg.SenderID, clubJoinTable.Latitude, clubJoinTable.Longitude); tempInt32 != 0 {
		return tempInt32
	}

	_, ok := this.table.TableRule.CheckPlayerMutex(msg.SenderID)
	if ok == false {
		return mateProto.Err_CheckMutex
	}

	msgSitdown := protoGameBasic.CS_ReqSitDown{SeatNumber: int32(qpTable.INVALID_SEAT_NUMBER)}

	// 临时转换成请求坐下
	msg.MessageID = protoGameBasic.ID_ReqSitDown
	msg.MsgBody = &msgSitdown
	rspCode := this.table.OnMessage(msg)

	// 还原成 原始ID
	msg.MessageID = protoGameBasic.ID_ClubJoinTable

	if rspCode >= 0 {
		seatData := this.table.SeatArr[rspCode].GetSeatData()
		seatData.ClubID = clubJoinTable.ClubID
		seatData.ClubScore = clubJoinTable.PlayerClubScore
		seatData.Player.IP = clubJoinTable.IP
		seatData.Lat, seatData.Lng = clubJoinTable.Latitude, clubJoinTable.Longitude
	}

	return rspCode
}

func (this *ZhaJinHuaTable) onPlayerLeave(pro *mateProto.MessageMaTe) int32 {

	if this.table.IsAssignTableState(qpTable.TS_WaitingPlayerEnter) == false {
		return mateProto.Err_TableStatusNotMatch
	}
	return this.table.OnLeave(pro)
}

func (this *ZhaJinHuaTable) onReady(pro *mateProto.MessageMaTe) int32 {
	funRes := this.GetBaseQPTable().OnMessage(pro)
	if funRes != mateProto.Err_Success {
		return funRes
	}

	var readyCount, lookerCount int32
	this.playingSeatArr = make([]int32, 0, this.gameRule.MaxPlayer)
	for _, v := range this.table.SeatArr {
		if v == nil {
			continue
		}
		if v.GetSeatData().IsAssignSeatState(qpTable.SS_Ready) == true {
			readyCount += 1
			this.playingSeatArr = append(this.playingSeatArr, int32(v.GetSeatData().Number))
		}
		if v.GetSeatData().IsAssignSeatState(qpTable.SS_Looker) == true {
			lookerCount += 1
		}
	}

	if this.table.CurXRound == 0 {
		if this.gameRule.FirstRoundReady == 0 {
			// 所有人准备后，游戏开始
			if readyCount == this.table.MaxPlayers-lookerCount {
				return this.onGameStart(pro)
			}
		} else {
			if readyCount < this.gameRule.FirstRoundReady {
				return funRes
			}

			if this.table.GetCurSeatCount() == readyCount {
				return this.onGameStart(pro)
			}

			if this.launchTimer == false {
				this.table.GameTimer.PutTableTimer(timerCheckUnreadyPlayer, 6*1000, this.timerCheckUnreadyPlayer)
				this.launchTimer = true
			}
		}
	} else {
		if readyCount == this.table.GetCurSeatCount()-lookerCount {
			return this.onGameStart(pro)
		}
	}

	return funRes
}

func (this *ZhaJinHuaTable) onGameStart(pro *mateProto.MessageMaTe) int32 {

	this.table.GameTimer.RemoveByTimeID(timerCheckUnreadyPlayer)

	if this.table.IsAssignTableState(qpTable.TS_WaitingPlayerEnter) == false &&
		this.table.IsAssignTableState(qpTable.TS_WaitingReady) == false {
		return mateProto.Err_TableStatusNotMatch
	}

	errNumber := this.table.OnGameStart(pro)
	if errNumber != mateProto.Err_Success {
		return errNumber
	}
	this.table.CurXRound += 1
	this.table.OperateRec.SetTableInfo(this.GetBaseQPTable().TableNum, this.table.CurXRound, this.gameRule.RuleJson, this.table.TableRule.TableCfgJson)
	this.playingSeatMap = make(map[qpTable.SeatNumber]struct{})
	this.minXiaZhuIndex1 = this.gameRule.XiaZhuOpt
	this.minXiaZhuIndex2 = 0

	diZhu := this.gameRule.DiZhu[this.minXiaZhuIndex1][this.minXiaZhuIndex2] * this.gameRule.MultipleFloat64
	playingSeatNumArr := make([]int32, 0, len(this.table.SeatArr))
	for i, v := range this.table.SeatArr {
		if v == nil {
			continue
		}
		seat := v.GetSeatData()
		if seat.IsAssignSeatState(qpTable.SS_Playing) == false {
			continue
		}
		this.table.OperateRec.PutPlayer(seat)

		// 下低注
		v.(*ZhaJinHuaSeat).XiaZhuScore = diZhu
		this.xiaZhuCount += v.(*ZhaJinHuaSeat).XiaZhuScore

		playingSeatNumArr = append(playingSeatNumArr, int32(i))
		this.playingSeatMap[seat.Number] = struct{}{}
	}
	if this.bankerSeatNumber == qpTable.INVALID_SEAT_NUMBER {
		randIndex := rand.Intn(len(playingSeatNumArr))
		if randIndex < 0 {
			this.bankerSeatNumber = qpTable.SeatNumber(playingSeatNumArr[0])
		} else {
			this.bankerSeatNumber = qpTable.SeatNumber(playingSeatNumArr[randIndex])
		}
	}

	this.table.BroadCastGameEvent(ID_NoticeXiaDiZhu, &SC_NoticeXiaDiZhu{BankerSeatNum: int32(this.bankerSeatNumber), DiZhu: diZhu})

	this.table.GameTimer.PutTableTimer(timerFaPai, 500, this.FaPai)

	return mateProto.Err_Success
}

func (this *ZhaJinHuaTable) FaPai() {

	this.PaiMgr.XiPai(this.table.GetCurSeatCount()-this.table.LookerCount, 3)

	maxPXIndex := 0
	maxPX := zhaJinHuaPaiXing{}

	faPaiArr := make([][]int8, 0, 10)
	maxPro := 0
	maxProIndex := 0

	for _, v := range this.table.SeatArr {
		if v == nil {
			continue
		}
		if v.GetSeatData().IsAssignSeatState(qpTable.SS_Playing) == false {
			continue
		}
		s_ := v.(*ZhaJinHuaSeat)

		paiArr := this.PaiMgr.GetGroupPai(int32(v.GetSeatData().Number), 3)
		faPaiArr = append(faPaiArr, paiArr)
		px := this.logic.GetPaiXing(paiArr)
		if maxPX.paiXing == 0 {
			maxPX, maxPXIndex = *px, len(faPaiArr)-1
		} else if this.logic.initiatorACompareB(px, &maxPX) {
			maxPX, maxPXIndex = *px, len(faPaiArr)-1
		}

		if s_.pro > maxPro {
			maxPro, maxProIndex = s_.pro, len(faPaiArr)-1
		}
	}

	if maxPro > 0 && maxProIndex != maxPXIndex {
		r_ := rand.Intn(100)
		if r_ < maxPro {
			faPaiArr[maxProIndex], faPaiArr[maxPXIndex] = faPaiArr[maxPXIndex], faPaiArr[maxProIndex]
			//glog.Warning("robot Win.", this.table.CurXRound)
		}
	} else {
		//glog.Warning("robot Win.", maxProIndex, ",  ", maxPXIndex, ", ", this.table.CurXRound)
	}

	faPaiIndex := 0

	tempPlayingSeatArr := make([]int32, 0, len(this.table.SeatArr))
	for i, v := range this.table.SeatArr {
		if v == nil {
			continue
		}
		seat := v.GetSeatData()
		if seat.IsAssignSeatState(qpTable.SS_Playing) == false {
			continue
		}
		tempPlayingSeatArr = append(tempPlayingSeatArr, int32(i))

		//paiArr := this.PaiMgr.GetGroupPai(int32(seat.Number), 3)
		paiArr := faPaiArr[faPaiIndex]
		faPaiIndex++

		v.(*ZhaJinHuaSeat).SetShouPai(paiArr)
		this.table.OperateRec.PutServiceStep(int32(seat.Number), ID_PlayBack, SC_PlayBackFaShouPai{SeatNumber: int32(seat.Number), Pai: paiArr})
	}

	this.table.BroadCastGameEvent(SC_FaShouPai, &MsgGameStart{SeatNumber: tempPlayingSeatArr})

	this.table.GameTimer.PutTableTimer(timerBankerOperation, 500, func() {
		this.noticePlayerOperation(this.bankerSeatNumber)
	})
}

func (this *ZhaJinHuaTable) noticePlayerOperation(seatNum qpTable.SeatNumber) {
	this.curOperatorSeat = seatNum
	this.table.SeatArr[seatNum].(*ZhaJinHuaSeat).seatData.OperationStart = time.Now().Unix()
	this.table.BroadCastGameEvent(ID_NoticeOperation, &CS_NoticeOperation{TargetSeat: int32(seatNum)})

	if this.gameRule.TimeOut > 0 {
		if this.table.SeatArr[seatNum].(*ZhaJinHuaSeat).xiaZhuTime < this.gameRule.MenPaiRound {
			this.table.GameTimer.PutTableTimer(timerMenPai, this.gameRule.TimeOut*1000, func() {
				this.autoMen(seatNum)
			})
		} else {
			this.table.GameTimer.PutTableTimer(timerQiPai, this.gameRule.TimeOut*1000, func() {
				this.autoQiPai(seatNum)
			})
		}
	}
}

func (this *ZhaJinHuaTable) findNextPlayer() qpTable.SeatNumber {
	for i := this.curOperatorSeat + 1; int(i) < len(this.table.SeatArr); i++ {
		if this.table.SeatArr[i] == nil {
			continue
		}
		zjhSeat := this.table.SeatArr[i].(*ZhaJinHuaSeat)
		if len(zjhSeat.shouPai) < 3 {
			continue
		}
		if i == this.bankerSeatNumber {
			this.xRound += 1
			this.table.BroadCastGameEvent(ID_XiaZhuRoundChanged, SC_XiaZhuRoundChanged{
				XiaZhuRound: this.xRound,
			})
		}
		if zjhSeat.isQiPai == true || zjhSeat.isBiPaiLose == true {
			continue
		}
		return i
	}

	for i := qpTable.SeatNumber(0); i < this.curOperatorSeat; i++ {
		if this.table.SeatArr[i] == nil {
			continue
		}
		zjhSeat := this.table.SeatArr[i].(*ZhaJinHuaSeat)
		if len(zjhSeat.shouPai) < 3 {
			continue
		}
		if i == this.bankerSeatNumber {
			this.xRound += 1
			this.table.BroadCastGameEvent(ID_XiaZhuRoundChanged, SC_XiaZhuRoundChanged{
				XiaZhuRound: this.xRound,
			})
		}
		if zjhSeat.isQiPai == true || zjhSeat.isBiPaiLose == true {
			continue
		}
		return i
	}
	return qpTable.INVALID_SEAT_NUMBER
}

func (this *ZhaJinHuaTable) onKanPai(pro *mateProto.MessageMaTe) int32 {
	if this.table.IsAssignTableState(qpTable.TS_Playing) == false {
		return mateProto.Err_ActionNotMatchStatus
	}

	seat := this.table.GetSeatDataByPlayerID(qpTable.PlayerID(pro.SenderID))
	if seat == nil {
		return mateProto.Err_NotFindPlayer
	}
	zjhSeat := seat.(*ZhaJinHuaSeat)
	if zjhSeat.isQiPai == true || zjhSeat.isBiPaiLose == true {
		return mateProto.Err_ActionNotMatchStatus
	}
	if zjhSeat.isKanPai == true {
		return mateProto.Err_OperationRepeat
	}

	if zjhSeat.xiaZhuTime < this.gameRule.MenPaiRound {
		return mateProto.Err_TableStatusNotMatch
	}
	if len(zjhSeat.shouPai) < 3 {
		return mateProto.Err_OperationNotExist
	}

	if zjhSeat.zjhPaiXing == nil {
		zjhSeat.zjhPaiXing = this.logic.GetPaiXing(zjhSeat.shouPai)
		if zjhSeat.zjhPaiXing.paiXing > zjhSeat.MaxPaiXing {
			zjhSeat.MaxPaiXing = zjhSeat.zjhPaiXing.paiXing
		}
	}

	zjhSeat.isKanPai = true
	this.table.SendGameEventToSeat(seat.GetSeatData().Number, ID_KanPai, SC_KanPai{PaiArr: zjhSeat.shouPai, PaiXing: zjhSeat.zjhPaiXing.paiXing})
	this.table.BroadCastGameEvent(ID_BroadcastKanPai, &SC_BroadcastKanPai{
		SeatNumber: int32(zjhSeat.seatData.Number),
	})

	return 0
}

func (this *ZhaJinHuaTable) onQiPai(pro *mateProto.MessageMaTe) int32 {
	if this.table.IsAssignTableState(qpTable.TS_Playing) == false {
		return mateProto.Err_ActionNotMatchStatus
	}

	seat := this.table.GetSeatDataByPlayerID(qpTable.PlayerID(pro.SenderID))
	if seat == nil {
		return mateProto.Err_NotFindPlayer
	}
	zjhSeat := seat.(*ZhaJinHuaSeat)
	if zjhSeat.isQiPai == true || zjhSeat.isBiPaiLose == true {
		return mateProto.Err_OperationRepeat
	}
	if zjhSeat.xiaZhuTime < this.gameRule.MenPaiRound {
		return mateProto.Err_TableStatusNotMatch
	}

	if this.curOperatorSeat != zjhSeat.seatData.Number {
		return mateProto.Err_NotYouOperation
	}
	if len(zjhSeat.shouPai) < 3 {
		return mateProto.Err_OperationNotExist
	}

	zjhSeat.zjhPaiXing = this.logic.GetPaiXing(zjhSeat.shouPai)
	if zjhSeat.zjhPaiXing.paiXing > zjhSeat.MaxPaiXing {
		zjhSeat.MaxPaiXing = zjhSeat.zjhPaiXing.paiXing
	}

	this.table.GameTimer.RemoveByTimeID(timerQiPai)
	zjhSeat.isQiPai = true
	this.table.BroadCastGameEvent(ID_BroadcastQiPai, SC_BroadcastQiPai{
		SeatNumber: int32(zjhSeat.seatData.Number),
	})
	delete(this.playingSeatMap, zjhSeat.seatData.Number)

	if zjhSeat.isKanPai == false {
		this.table.SendGameEventToSeat(seat.GetSeatData().Number,
			ID_KanPai, SC_KanPai{PaiArr: zjhSeat.shouPai, PaiXing: zjhSeat.zjhPaiXing.paiXing})
	}

	if len(this.playingSeatMap) < 2 {
		this.gameOver()
		return 0
	}

	tempSeatNum := this.findNextPlayer()
	if tempSeatNum == qpTable.INVALID_SEAT_NUMBER {
		glog.Warning("not find next player...", ",", this.table.TableNum)
		return 0
	}

	if this.xRound >= this.gameRule.FengDingKaiPai {
		this.gameOver()
		return 0
	}

	this.noticePlayerOperation(tempSeatNum)
	//this.timerGenDaoDi(tempSeatNum)

	return 0
}

func (this *ZhaJinHuaTable) onBiPai(pro *mateProto.MessageMaTe) int32 {
	if this.table.IsAssignTableState(qpTable.TS_Playing) == false {
		return mateProto.Err_ActionNotMatchStatus
	}

	biPai := CS_BiPai{}
	err := json.Unmarshal(pro.Data, &biPai)
	if err != nil {
		return mateProto.Err_ProtocolDataErr
	}

	seat := this.table.GetSeatDataByPlayerID(qpTable.PlayerID(pro.SenderID))
	if seat == nil {
		return mateProto.Err_NotFindPlayer
	}
	if this.curOperatorSeat != seat.GetSeatData().Number {
		return mateProto.Err_NotYouOperation
	}

	zjhSeat := seat.(*ZhaJinHuaSeat)
	if zjhSeat.isBiPaiLose == true || zjhSeat.isQiPai == true {
		return mateProto.Err_OperationRepeat
	}
	if zjhSeat.xiaZhuTime < this.gameRule.MenPaiRound {
		return mateProto.Err_TableStatusNotMatch
	}

	var (
		tempXiaZhuScore          float64
		levelXiaZhu, indexXiaZhu int32
	)

	if zjhSeat.isKanPai == false {
		levelXiaZhu = this.minXiaZhuIndex1
		tempXiaZhuScore = this.gameRule.DiZhu[this.minXiaZhuIndex1][this.minXiaZhuIndex2]
	} else {
		levelXiaZhu = this.minXiaZhuIndex1 + 1
		tempXiaZhuScore = this.gameRule.DiZhu[this.minXiaZhuIndex1+1][this.minXiaZhuIndex2]
	}
	indexXiaZhu = this.minXiaZhuIndex2

	if this.gameRule.ShuangBeiBiPai == true {
		tempXiaZhuScore *= 2
	}

	if this.table.MZClubID > 0 {
		if zjhSeat.seatData.ClubScore+zjhSeat.seatData.SeatScore-zjhSeat.XiaZhuScore < tempXiaZhuScore*this.gameRule.MultipleFloat64 {
			return mateProto.Err_SeatScoreLittle
		}
	} else {
		if zjhSeat.seatData.SeatScore-zjhSeat.XiaZhuScore < tempXiaZhuScore {
			return mateProto.Err_SeatScoreLittle
		}
	}

	// 座位 是否有效
	if biPai.TargetSeat < 0 || biPai.TargetSeat >= len(this.table.SeatArr) {
		return mateProto.Err_InvalidSeatNumber
	}
	if this.table.SeatArr[biPai.TargetSeat] == nil {
		return mateProto.Err_InvalidSeatNumber
	}

	targetSeat := this.table.SeatArr[biPai.TargetSeat].(*ZhaJinHuaSeat)
	if targetSeat.isBiPaiLose == true || targetSeat.isQiPai == true {
		return mateProto.Err_OperationNotExist
	}

	if zjhSeat.zjhPaiXing == nil {
		zjhSeat.zjhPaiXing = this.logic.GetPaiXing(zjhSeat.shouPai)
		if zjhSeat.zjhPaiXing.paiXing > zjhSeat.MaxPaiXing {
			zjhSeat.MaxPaiXing = zjhSeat.zjhPaiXing.paiXing
		}
	}
	if targetSeat.zjhPaiXing == nil {
		targetSeat.zjhPaiXing = this.logic.GetPaiXing(targetSeat.shouPai)
		if targetSeat.zjhPaiXing.paiXing > targetSeat.MaxPaiXing {
			targetSeat.MaxPaiXing = targetSeat.zjhPaiXing.paiXing
		}
	}

	biPaiResult := SC_BroadcastBiPai{InitiatorSeat: int32(zjhSeat.seatData.Number)}
	if this.logic.initiatorACompareB(zjhSeat.zjhPaiXing, targetSeat.zjhPaiXing) == true {
		targetSeat.isBiPaiLose = true
		delete(this.playingSeatMap, targetSeat.GetSeatData().Number)

		biPaiResult.WinSeat = int32(zjhSeat.GetSeatData().Number)
		biPaiResult.LoseSeat = int32(targetSeat.GetSeatData().Number)
	} else {
		zjhSeat.isBiPaiLose = true
		delete(this.playingSeatMap, zjhSeat.GetSeatData().Number)

		biPaiResult.WinSeat = int32(targetSeat.GetSeatData().Number)
		biPaiResult.LoseSeat = int32(zjhSeat.GetSeatData().Number)
	}

	zjhSeat.XiaZhuScore += tempXiaZhuScore * this.gameRule.MultipleFloat64
	this.xiaZhuCount += tempXiaZhuScore * this.gameRule.MultipleFloat64

	this.table.GameTimer.RemoveByTimeID(timerQiPai)
	this.table.BroadCastGameEvent(ID_BroadcastBiPai, &biPaiResult)
	if tempXiaZhuScore > 0.0 {
		this.table.BroadCastGameEvent(ID_BroadcastXiaZhu,
			SC_XiaZhu{SeatNumber: int32(zjhSeat.seatData.Number),
				LeaveXiaZhu: levelXiaZhu, IndexXiaZhu: indexXiaZhu, XiaZhu: tempXiaZhuScore})
	}

	if len(this.playingSeatMap) < 2 {
		this.gameOver()
	} else if zjhSeat.isBiPaiLose == true { // 自己比输了
		tempSeatNum := this.findNextPlayer()
		if tempSeatNum == qpTable.INVALID_SEAT_NUMBER {
			glog.Warning("not find next player...", this.table.TableNum)
		} else {
			if this.xRound >= this.gameRule.FengDingKaiPai {
				this.gameOver()
			} else {
				this.table.GameTimer.PutTableTimer(timerBankerOperation, 3*1000, func() {
					this.noticePlayerOperation(tempSeatNum)
				})
			}
		}
	} else {
		if this.gameRule.TimeOut > 0 {
			seatNum := zjhSeat.seatData.Number
			this.table.GameTimer.PutTableTimer(timerQiPai, this.gameRule.TimeOut*1000+3000, func() {
				this.autoQiPai(seatNum)
			})
		}
	}

	return 0
}

func (this *ZhaJinHuaTable) onGenDaoDi(pro *mateProto.MessageMaTe) int32 {
	if this.table.IsAssignTableState(qpTable.TS_Playing) == false {
		return mateProto.Err_TableStatusNotMatch
	}
	operGenDaoDi := CS_GenDaoDi{}
	err := json.Unmarshal(pro.Data, &operGenDaoDi)
	if err != nil {
		return mateProto.Err_ProtocolDataErr
	}

	seat := this.table.GetSeatDataByPlayerID(qpTable.PlayerID(pro.SenderID))
	if seat == nil {
		return mateProto.Err_NotFindPlayer
	}

	nnSeat := seat.(*ZhaJinHuaSeat)
	nnSeat.isGenDaoDi = operGenDaoDi.On

	this.table.SendGameEventToSeat(seat.GetSeatData().Number, ID_GenDaoDi, &operGenDaoDi)

	return 0
}

func (this *ZhaJinHuaTable) onXiaZhu(pro *mateProto.MessageMaTe) int32 {
	if this.table.IsAssignTableState(qpTable.TS_Playing) == false {
		return mateProto.Err_TableStatusNotMatch
	}
	operXiaZhu := CS_XiaZhu{}
	err := json.Unmarshal(pro.Data, &operXiaZhu)
	if err != nil {
		return mateProto.Err_ProtocolDataErr
	}

	seat := this.table.GetSeatDataByPlayerID(qpTable.PlayerID(pro.SenderID))
	if seat == nil {
		return mateProto.Err_NotFindPlayer
	}
	zjhSeat := seat.(*ZhaJinHuaSeat)

	if zjhSeat.isBiPaiLose == true || zjhSeat.isQiPai == true {
		return mateProto.Err_ActionNotMatchStatus
	}

	if this.curOperatorSeat != zjhSeat.seatData.Number {
		return mateProto.Err_NotYouOperation
	}

	seatData := seat.GetSeatData()
	if this.table.MZClubID > 0 {
		if seatData.ClubScore+seatData.SeatScore-zjhSeat.XiaZhuScore < operXiaZhu.XiaZhu*this.gameRule.MultipleFloat64 {
			return mateProto.Err_SeatScoreLittle
		}
	}

	levelXiaZhu := int32(0)
	curIndexXiaZhu := int32(0)
	isJiaZhu := false
	if zjhSeat.isKanPai == true {
		ok := false
		levelXiaZhu = this.minXiaZhuIndex1 + 1
		for i := this.minXiaZhuIndex2; int(i) < len(this.gameRule.DiZhu[this.minXiaZhuIndex1+1]); i++ {
			if this.gameRule.DiZhu[this.minXiaZhuIndex1+1][i] != operXiaZhu.XiaZhu {
				continue
			}
			curIndexXiaZhu = i
			ok = true
			if i == this.minXiaZhuIndex2 {
				// 跟注
			} else if i > this.minXiaZhuIndex2 {
				// 加注
				this.minXiaZhuIndex2 = i
				isJiaZhu = true
			}
		}
		if ok == false {
			return mateProto.Err_OperationParamErr
		}
	} else {
		ok := false
		levelXiaZhu = this.minXiaZhuIndex1
		for i := this.minXiaZhuIndex2; int(i) < len(this.gameRule.DiZhu[this.minXiaZhuIndex1]); i++ {
			if this.gameRule.DiZhu[this.minXiaZhuIndex1][i] != operXiaZhu.XiaZhu {
				continue
			}
			curIndexXiaZhu = i
			ok = true
			if i == this.minXiaZhuIndex2 {
				// 跟注
			} else if i > this.minXiaZhuIndex2 {
				// 加注
				this.minXiaZhuIndex2 = i
				isJiaZhu = true
			}
		}
		if ok == false {
			return mateProto.Err_OperationParamErr
		}
	}

	zjhSeat.xiaZhuTime += 1
	zjhSeat.XiaZhuScore += operXiaZhu.XiaZhu * this.gameRule.MultipleFloat64
	zjhSeat.XiaZhuScore, _ = decimal.NewFromFloat(zjhSeat.XiaZhuScore).Truncate(3).Float64()

	this.xiaZhuCount += operXiaZhu.XiaZhu * this.gameRule.MultipleFloat64
	this.xiaZhuCount, _ = decimal.NewFromFloat(this.xiaZhuCount).Truncate(3).Float64()

	this.table.BroadCastGameEvent(ID_BroadcastXiaZhu,
		SC_XiaZhu{SeatNumber: int32(seatData.Number),
			LeaveXiaZhu: levelXiaZhu,
			IndexXiaZhu: curIndexXiaZhu,
			IsJiaZhu:    isJiaZhu,
			XiaZhu:      operXiaZhu.XiaZhu,
			XiaZhuCount: zjhSeat.XiaZhuScore})
	this.table.GameTimer.RemoveByTimeID(timerQiPai)
	this.table.GameTimer.RemoveByTimeID(timerMenPai)

	tempSeatNum := this.findNextPlayer()
	if tempSeatNum == qpTable.INVALID_SEAT_NUMBER {
		glog.Warning("not find next player...,", this.table.TableNum)
	} else {
		if this.xRound >= this.gameRule.FengDingKaiPai {
			this.gameOver()
		} else {
			this.noticePlayerOperation(tempSeatNum)
			//this.timerGenDaoDi(tempSeatNum)
		}
	}

	return 0
}

//func (this *ZhaJinHuaTable) timerGenDaoDi(seatNum qpTable.SeatNumber) {
//	zjhSeat := this.table.SeatArr[seatNum].(*ZhaJinHuaSeat)
//	if zjhSeat.isGenDaoDi == false {
//		return
//	}
//
//	this.table.GameTimer.PutSeatTimer(int32(seatNum), timerGenDaoDi, 700, func() {
//		zjhSeat := this.table.SeatArr[seatNum].(*ZhaJinHuaSeat)
//		if zjhSeat.isBiPaiLose || zjhSeat.isQiPai || zjhSeat.isGenDaoDi == false {
//			return
//		}
//
//		var xiaZhuScore float64
//		if zjhSeat.isKanPai == true {
//			xiaZhuScore = this.gameRule.DiZhu[this.minXiaZhuIndex1+1][this.minXiaZhuIndex2]
//		} else {
//			xiaZhuScore = this.gameRule.DiZhu[this.minXiaZhuIndex1][this.minXiaZhuIndex2]
//		}
//
//		msgXiaZhu := CS_XiaZhu{xiaZhuScore}
//		msgBody_, _ := json.Marshal(&msgXiaZhu)
//		msg := mateProto.MessageMaTe{
//			SenderID:  int64(zjhSeat.GetSeatData().Player.ID),
//			MessageID: ID_XiaZhu,
//			MsgBody:   msgBody_,
//		}
//		this.OnMessage(&msg)
//	})
//}

func (this *ZhaJinHuaTable) autoQiPai(seatNum qpTable.SeatNumber) {

	zjhSeat := this.table.SeatArr[seatNum].(*ZhaJinHuaSeat)
	if zjhSeat.isBiPaiLose || zjhSeat.isQiPai {
		return
	}

	msg := mateProto.MessageMaTe{
		SenderID:  int64(zjhSeat.GetSeatData().Player.ID),
		MessageID: ID_QiPai,
	}
	this.OnMessage(&msg)
}

func (this *ZhaJinHuaTable) autoMen(seatNum qpTable.SeatNumber) {

	zjhSeat := this.table.SeatArr[seatNum].(*ZhaJinHuaSeat)
	if zjhSeat.isBiPaiLose || zjhSeat.isQiPai {
		return
	}
	if zjhSeat.xiaZhuTime >= this.gameRule.MenPaiRound {
		return
	}
	xiaZhu := CS_XiaZhu{}

	xiaZhu.XiaZhu = this.gameRule.DiZhu[this.minXiaZhuIndex1][this.minXiaZhuIndex2]

	msg := mateProto.MessageMaTe{
		SenderID:  int64(zjhSeat.GetSeatData().Player.ID),
		MessageID: ID_XiaZhu,
	}

	msg.Data, _ = json.Marshal(&xiaZhu)

	this.OnMessage(&msg)
}

type GameOverPaiXingCompareArr struct {
	SeatArr []*ZhaJinHuaSeat
	Logic   *zhaJinHuaLogic
}

func (s GameOverPaiXingCompareArr) Len() int { return len(s.SeatArr) }
func (s GameOverPaiXingCompareArr) Swap(i, j int) {
	s.SeatArr[i], s.SeatArr[j] = s.SeatArr[j], s.SeatArr[i]
}
func (s GameOverPaiXingCompareArr) Less(i, j int) bool {
	r := s.Logic.compareInGameOver(s.SeatArr[i].zjhPaiXing, s.SeatArr[j].zjhPaiXing)

	return r > 0
}

func (this *ZhaJinHuaTable) gameOver() {

	for _, v := range this.table.SeatArr {
		if v == nil {
			continue
		}
		zjhSeat := v.(*ZhaJinHuaSeat)
		if len(zjhSeat.shouPai) > 0 && zjhSeat.zjhPaiXing == nil {
			zjhSeat.zjhPaiXing = this.logic.GetPaiXing(zjhSeat.shouPai)
			if zjhSeat.zjhPaiXing.paiXing > zjhSeat.MaxPaiXing {
				zjhSeat.MaxPaiXing = zjhSeat.zjhPaiXing.paiXing
			}
		}
	}

	this.gameOverBiPai()

	this.handleXiaoJieSuan()

	if 0 != this.table.GameOver() {
		this.handleDaJieSuan()
		return
	}

	// 不够 底分的人数
	this.table.LookerCount = int32(0)
	if this.table.MZClubID > 0 {
		for _, v := range this.table.SeatArr {
			if v == nil {
				continue
			}
			zjhSeat := v.(*ZhaJinHuaSeat)

			if zjhSeat.seatData.IsAssignSeatState(qpTable.SS_Looker) == true {
				this.table.LookerCount += 1
				continue
			}
			if zjhSeat.seatData.ClubScore-zjhSeat.seatData.SeatScore < this.gameRule.DiZhu[this.gameRule.XiaZhuOpt][2] {
				zjhSeat.seatData.SetState(qpTable.SS_Looker | qpTable.SS_Sitdown)
				this.table.NotifyPlayerStateChange(zjhSeat.seatData.Number)
				this.table.LookerCount += 1
			}
		}
	}

	if this.table.GetCurSeatCount()-this.table.LookerCount < 2 {
		this.table.DissolveType = qpTable.DT_ScoreLess
		this.handleDaJieSuan()
		return
	}

	if this.table.CurXRound >= this.gameRule.MaxRoundCount {
		this.table.DissolveType = qpTable.DT_Playing
		this.handleDaJieSuan()
		return
	}

	{
		winSeatNum := qpTable.SeatNumber(0)
		for k, _ := range this.playingSeatMap {
			winSeatNum = k
			break
		}
		if this.table.SeatArr[winSeatNum].GetSeatData().IsAssignSeatState(qpTable.SS_Looker) == true {
			for _, v := range this.table.SeatArr {
				if v == nil {
					continue
				}
				zjhSeat := v.(*ZhaJinHuaSeat)
				if zjhSeat.GetSeatData().IsAssignSeatState(qpTable.SS_Looker) == false {
					winSeatNum = v.GetSeatData().Number
					break
				}
			}
		}

		this.bankerSeatNumber = winSeatNum
	}

	this.CleanRoundData()
	this.table.TableRule.TimerAutoReady()
}

func (this *ZhaJinHuaTable) gameOverBiPai() {
	if len(this.playingSeatMap) < 1 {
		glog.Warning("RoundOverFun() .....len := ", len(this.playingSeatMap), ",", this.table.TableNum)
		return
	}
	if len(this.playingSeatMap) > 1 {
		var zjhSeat *ZhaJinHuaSeat
		var zjhSeatSort GameOverPaiXingCompareArr
		zjhSeatSort.SeatArr = make([]*ZhaJinHuaSeat, 0, len(this.playingSeatMap))
		zjhSeatSort.Logic = &this.logic

		for k, _ := range this.playingSeatMap {
			zjhSeat = this.table.SeatArr[k].(*ZhaJinHuaSeat)
			if zjhSeat.zjhPaiXing == nil {
				zjhSeat.zjhPaiXing = this.logic.GetPaiXing(zjhSeat.shouPai)
				if zjhSeat.zjhPaiXing.paiXing > zjhSeat.MaxPaiXing {
					zjhSeat.MaxPaiXing = zjhSeat.zjhPaiXing.paiXing
				}
			}
			zjhSeatSort.SeatArr = append(zjhSeatSort.SeatArr, zjhSeat)
		}

		sort.Sort(&zjhSeatSort)

		for i := 0; i < len(zjhSeatSort.SeatArr)-1; i++ {
			if zjhSeatSort.SeatArr[i].zjhPaiXing.paiXing > zjhSeatSort.SeatArr[i+1].zjhPaiXing.paiXing {
				zjhSeatSort.SeatArr = zjhSeatSort.SeatArr[:i+1]
				break
			}
			isFindMax := false
			for j := 0; j < 3; j++ {
				a := zjhSeatSort.SeatArr[i].zjhPaiXing.paiArr[j].dianShu
				b := zjhSeatSort.SeatArr[i+1].zjhPaiXing.paiArr[j].dianShu
				if a > b {
					zjhSeatSort.SeatArr = zjhSeatSort.SeatArr[:i+1]
					isFindMax = true
					break
				}
			}
			if isFindMax {
				break
			}

			if this.logic.rule.DaXiao == 0 {
				continue
			}

			// 同牌型 比 花色
			if zjhSeatSort.SeatArr[i].zjhPaiXing.paiXing == zjh_DuiZi {
				if (zjhSeatSort.SeatArr[i].zjhPaiXing.paiArr[0].huaSe*0x10) == HeiTao ||
					(zjhSeatSort.SeatArr[i].zjhPaiXing.paiArr[1].huaSe*0x10) == HeiTao {
					zjhSeatSort.SeatArr = zjhSeatSort.SeatArr[:i+1]
					break
				}
			}
			if zjhSeatSort.SeatArr[i].zjhPaiXing.paiArr[0].huaSe > zjhSeatSort.SeatArr[i+1].zjhPaiXing.paiArr[0].huaSe {
				zjhSeatSort.SeatArr = zjhSeatSort.SeatArr[:i+1]
				break
			}
		}

		this.playingSeatMap = make(map[qpTable.SeatNumber]struct{})
		for _, v := range zjhSeatSort.SeatArr {
			this.playingSeatMap[v.seatData.Number] = struct{}{}
		}
	}

	winScore := this.xiaZhuCount / float64(len(this.playingSeatMap))

	for _, v := range this.GetBaseQPTable().SeatArr {
		if v == nil {
			continue
		}

		seat := v.GetSeatData()
		playerSeat := v.(*ZhaJinHuaSeat)
		if len(playerSeat.shouPai) < 3 {
			continue
		}

		if _, ok := this.playingSeatMap[seat.Number]; ok == false {
			seat.RoundScore -= playerSeat.XiaZhuScore
			seat.SeatScore += seat.RoundScore
		}
	}

	for k, _ := range this.playingSeatMap {
		seat := this.table.SeatArr[k].(*ZhaJinHuaSeat)
		seat.seatData.RoundScore += winScore - seat.XiaZhuScore
		seat.seatData.SeatScore += seat.seatData.RoundScore
	}

	if this.gameRule.BaoZiJiangLi == true {
		baoZiScore := this.gameRule.DiZhu[this.gameRule.XiaZhuOpt][0] * 10 * this.gameRule.MultipleFloat64
		for _, v := range this.GetBaseQPTable().SeatArr {
			if v == nil {
				continue
			}

			baoZiSeat := v.GetSeatData()
			playerSeat := v.(*ZhaJinHuaSeat)
			if playerSeat.zjhPaiXing == nil {
				continue
			}
			if playerSeat.zjhPaiXing.paiXing != zjh_BaoZi {
				continue
			}

			for _, v1 := range this.GetBaseQPTable().SeatArr {
				if v1 == nil {
					continue
				}
				if v1.GetSeatData().Number == v.GetSeatData().Number {
					continue
				}

				seat := v.GetSeatData()
				playerSeat := v.(*ZhaJinHuaSeat)
				if len(playerSeat.shouPai) < 3 {
					continue
				}

				baoZiSeat.RoundScore += baoZiScore
				baoZiSeat.SeatScore += baoZiScore

				seat.RoundScore -= baoZiScore
				seat.SeatScore -= baoZiScore
			}
		}
	}
}

func (this *ZhaJinHuaTable) handleXiaoJieSuan() {
	if this.table.CurXRound < 1 {
		return
	}

	roundOver := BroadcastRoundOver{
		TableNumber: this.GetBaseQPTable().TableNum,
		Timestamp:   time.Now().Unix(),

		ClubID:     this.table.MZClubID,
		ClubPlayID: this.table.ClubPlayID,
	}
	//msgRoundOver := mateProto.MessageMaTe{MessageID: ID_RoundOver, MsgBody: &roundOver}

	recPlayerGameScoreArr := make([]*protoGameBasic.PlayerGameScore, 0, 8)

	for _, v := range this.GetBaseQPTable().SeatArr {
		if v == nil {
			continue
		}

		seat := v.GetSeatData()
		playerSeat := v.(*ZhaJinHuaSeat)
		if len(playerSeat.shouPai) < 3 {
			continue
		}

		if seat.RoundScore > v.(*ZhaJinHuaSeat).MaxGetScore {
			v.(*ZhaJinHuaSeat).MaxGetScore = seat.RoundScore
		}
		if seat.RoundScore > 0 {
			v.(*ZhaJinHuaSeat).WinCount += 1
		}

		temp := protoGameBasic.PlayerGameScore{
			UID:    int64(seat.Player.ID),
			Nick:   seat.Player.Nick,
			ClubID: seat.ClubID,
			SScore: commonDef.Float64Mul1000ToService(seat.RoundScore),
		}
		recPlayerGameScoreArr = append(recPlayerGameScoreArr, &temp)
	}

	for _, v1 := range this.GetBaseQPTable().SeatArr {
		if v1 == nil {
			continue
		}

		roundSeatScoreArr := make([]*RoundSeatScore, 0, 8)
		selfSeat := v1.(*ZhaJinHuaSeat)
		for _, v := range this.GetBaseQPTable().SeatArr {
			if v == nil {
				continue
			}

			seat := v.GetSeatData()
			playerSeat := v.(*ZhaJinHuaSeat)
			if len(playerSeat.shouPai) < 3 || playerSeat.zjhPaiXing == nil {
				continue
			}

			tempShouPai := playerSeat.shouPai
			tempPaiXing := playerSeat.zjhPaiXing.paiXing

			if playerSeat.isQiPai == true && selfSeat.seatData.Number != playerSeat.seatData.Number {
				tempShouPai = []int8{InvalidPai, InvalidPai, InvalidPai}
				tempPaiXing = zjh_NULL
			}
			roundSeatScoreArr = append(roundSeatScoreArr, &RoundSeatScore{
				ClubID:     seat.ClubID,
				UID:        int64(seat.Player.ID),
				NickName:   seat.Player.Nick,
				Head:       seat.Player.Head,
				SeatNumber: int32(seat.Number),
				Pai:        tempShouPai,
				PaiXing:    tempPaiXing,
				IsQiPai:    playerSeat.isQiPai,
				GameScore:  commonDef.Float64ToString(seat.RoundScore),
				SeatScore:  commonDef.Float64ToString(seat.SeatScore)})
		}
		roundOver.SeatData = roundSeatScoreArr
		this.table.SendGameEventToSeat(selfSeat.seatData.Number, ID_RoundOver, &roundOver)
	}

	gameStepRec, _ := this.GetBaseQPTable().OperateRec.Pack()

	// 写入小局记录 至 数据库
	roundRecData := protoGameBasic.SS_RoundRecord{
		Begin:    this.table.GameStartTime,
		End:      time.Now(),
		RoundID:  this.table.RecordID,
		ClubID:   this.table.MZClubID,
		TableID:  this.table.TableNum,
		CurRound: this.table.CurXRound,
		Players:  recPlayerGameScoreArr,
		GameStep: string(gameStepRec),
	}
	wrapMQ.SendMsgTo(&mateProto.MessageMaTe{To: "db", MessageID: protoGameBasic.ID_RoundOver}, &roundRecData)
}

// 大结算
func (this *ZhaJinHuaTable) handleDaJieSuan() {
	if this.table.CurXRound < 1 {
		return
	}

	msg := BroadcastGameOver{TableNumber: this.table.TableNum,
		CurRound:     this.table.CurXRound,
		MaxRound:     this.gameRule.MaxRoundCount,
		EndTime:      time.Now().Unix(),
		SeatData:     make([]*GameOverSeatData, 0, this.table.GetCurSeatCount()),
		DissolveType: this.table.DissolveType,
		ClubID:       this.table.MZClubID,
		ClubPlayID:   this.table.ClubPlayID}
	msgGameOver := mateProto.MessageMaTe{MessageID: ID_BroadcastGameOver, MsgBody: &msg}

	for _, v := range this.table.SeatArr {
		if v == nil {
			continue
		}
		v.GetSeatData().GameOverMsg = &msgGameOver

		pdkSeat := v.(*ZhaJinHuaSeat)

		tempSeat := &GameOverSeatData{
			ClubID:       pdkSeat.seatData.ClubID,
			UID:          int64(pdkSeat.seatData.Player.ID),
			Nick:         pdkSeat.seatData.Player.Nick,
			Head:         pdkSeat.seatData.Player.Head,
			MaxPaiXing:   pdkSeat.MaxPaiXing,
			MaxGetScore:  commonDef.Float64ToString(pdkSeat.MaxGetScore),
			WinCount:     pdkSeat.WinCount,
			LoseCount:    0,
			SeatScore:    commonDef.Float64ToString(pdkSeat.seatData.SeatScore),
			SeatScoreInt: commonDef.Float64Mul1000ToService(pdkSeat.seatData.SeatScore),
			IsMaxWin:     false}

		msg.SeatData = append(msg.SeatData, tempSeat)
	}
	sort.Sort(msg.SeatData)
	msg.SeatData[0].IsMaxWin = true
	for i := 1; i < len(msg.SeatData); i++ {
		if msg.SeatData[i].SeatScoreInt < msg.SeatData[0].SeatScoreInt {
			break
		}
		if msg.SeatData[i].SeatScoreInt == msg.SeatData[0].SeatScoreInt {
			msg.SeatData[i].IsMaxWin = true
		}
	}

	this.table.SendToAllPlayer(ID_BroadcastGameOver, &msg)

	this.GetBaseQPTable().SetTableState(qpTable.TS_Invalid)
}

func (this *ZhaJinHuaTable) onTableExpire(pro *mateProto.MessageMaTe) int32 {

	this.table.DissolveType = qpTable.DT_LiveTimeout

	if this.table.IsPlaying == true && this.gameRule.JieSanTongBi == true {
		this.gameOverBiPai()
	}

	this.handleXiaoJieSuan()

	this.handleDaJieSuan()

	return this.table.OnMessage(pro)
}

func (this *ZhaJinHuaTable) onTableData(pro *mateProto.MessageMaTe) int32 {

	seat := this.table.GetSeatDataByPlayerID(qpTable.PlayerID(pro.SenderID))
	if seat == nil {
		return mateProto.Err_NotFindPlayer
	}

	zjhSeatArr := make([]*ZhaJinHuaSeatData, 0)
	for _, v := range this.table.SeatArr {
		if v == nil {
			continue
		}

		pkZJHSeat := v.GetXSeatData(0).(*ZhaJinHuaSeat)
		tempNNSeat := ZhaJinHuaSeatData{
			UID:           int64(pkZJHSeat.seatData.Player.ID),
			Nick:          pkZJHSeat.seatData.Player.Nick,
			HeadURL:       pkZJHSeat.seatData.Player.Head,
			IP:            pkZJHSeat.seatData.Player.IP,
			Sex:           pkZJHSeat.seatData.Player.Sex,
			SeatNumber:    int32(pkZJHSeat.seatData.Number),
			SeatStatus:    uint32(pkZJHSeat.seatData.Status),
			ClubID:        pkZJHSeat.seatData.ClubID,
			ClubScore:     commonDef.Float64ToString(pkZJHSeat.seatData.ClubScore),
			SeatScore:     commonDef.Float64ToString(pkZJHSeat.seatData.SeatScore),
			RoundScore:    commonDef.Float64ToString(pkZJHSeat.seatData.RoundScore),
			VoteStatus:    v.GetSeatData().DissolveVote,
			OperationTime: time.Now().Unix() - v.GetSeatData().OperationStart,
			XiaZhuScore:   pkZJHSeat.XiaZhuScore,
			IsQiPai:       pkZJHSeat.isQiPai,
			IsLose:        pkZJHSeat.isBiPaiLose,
			IsKanPai:      pkZJHSeat.isKanPai,
			XiaZhuTime:    pkZJHSeat.xiaZhuTime,
		}

		if pkZJHSeat.seatData.Lng > 0.1 && pkZJHSeat.seatData.Lat > 0.1 {
			tempNNSeat.IsGPS = true
		}
		if pkZJHSeat.seatData.IsAssignSeatState(qpTable.SS_Playing) == true {
			tempNNSeat.ShouPai = []int8{InvalidPai, InvalidPai, InvalidPai}
		}

		zjhSeatArr = append(zjhSeatArr, &tempNNSeat)
	}

	selfSeat := seat.GetXSeatData(0).(*ZhaJinHuaSeat)
	nowTT := time.Now().Unix()

	tableData := SC_TableData{
		MZCID:              this.table.MZClubID,
		TableNumber:        this.table.TableNum,
		TableStatus:        uint32(this.table.Status),
		RoundCount:         this.table.CurXRound,
		TableRuleText:      this.table.TableRule.TableCfgJson,
		ClubRuleText:       this.table.ClubRuleText,
		BankerSeatNum:      int32(this.bankerSeatNumber),
		XiaZhuRound:        this.xRound,
		SeatData:           zjhSeatArr,
		ShouPai:            selfSeat.shouPai,
		GameRuleText:       this.gameRule.RuleJson,
		ClubScore:          commonDef.Float64ToString(seat.GetSeatData().ClubScore),
		DissolveID:         int32(this.table.DissolveSeatNum),
		LaunchDissolveTime: nowTT - this.table.LaunchDissolveTime,
		FirstRoundReadTime: nowTT - this.table.FirstRoundReadTime,
		MaxXiaZhuCount:     this.minXiaZhuIndex2,
		CurSeatNumber:      int32(this.curOperatorSeat),
		ClubID:             selfSeat.seatData.ClubID,
	}
	tableData.IsGenDaoDi = selfSeat.isGenDaoDi
	if selfSeat.isKanPai == true || selfSeat.isQiPai {
		tableData.ShouPai = selfSeat.shouPai
		if selfSeat.zjhPaiXing != nil {
			tableData.PaiXing = selfSeat.zjhPaiXing.paiXing
		}
	} else if len(selfSeat.shouPai) > 0 {
		tableData.ShouPai = []int8{InvalidPai, InvalidPai, InvalidPai}
	}

	this.table.UpdatePlayerSource(selfSeat.seatData, pro.Source)
	this.table.SendToSeat(seat.GetSeatData().Number, ID_TableData, tableData)

	return mateProto.Err_Success
}

func (this *ZhaJinHuaTable) onCustomShouPai(pro *mateProto.MessageMaTe) int32 {
	msgCustomShouPai := CS_CustomShouPai{}
	err := json.Unmarshal(pro.Data, &msgCustomShouPai)
	if err != nil {
		return mateProto.Err_ProtocolDataErr
	}

	seat := this.table.GetSeatDataByPlayerID(qpTable.PlayerID(pro.SenderID))
	if seat == nil {
		return mateProto.Err_NotFindPlayer
	}

	if len(msgCustomShouPai.ShouPai) > 3 {
		return mateProto.Err_CustomPai
	}

	powerMap, _ := db.GetPlayerPower(pro.SenderID)
	if powerMap == nil {
		return 0
	}

	if v, ok := powerMap[strconv.Itoa(int(this.table.GameID))]; ok == false || v == 0 {
		return 0
	}

	rspMsg := protoGameBasic.JsonResponse{}

	pai, rsp := this.PaiMgr.Reserve(int32(seat.GetSeatData().Number), msgCustomShouPai.ShouPai)
	if rsp != 0 {
		rspMsg.Status = mateProto.Err_CustomPai
		if rsp == -1 {
			rspMsg.Detail = fmt.Sprintf("used pai:=%d", pai)
		} else if rsp == -2 {
			rspMsg.Detail = fmt.Sprintf("invalid pai:=%d", pai)
		}
	}

	this.table.SendToSeat(seat.GetSeatData().Number, pro.MessageID, &rspMsg)

	return 0
}

func (this *ZhaJinHuaTable) onDissolveTableVote(pro *mateProto.MessageMaTe) int32 {
	rspCode := this.table.OnMessage(pro)

	if this.table.Status != qpTable.TS_Invalid {
		return rspCode
	}

	if this.table.IsPlaying == true && this.gameRule.JieSanTongBi == true {
		this.gameOverBiPai()
	}

	this.handleXiaoJieSuan()
	this.handleDaJieSuan()

	return rspCode
}

func (this *ZhaJinHuaTable) onForceDissolveTable(pro *mateProto.MessageMaTe) int32 {
	rspCode := this.table.OnMessage(pro)

	if this.table.Status != qpTable.TS_Invalid {
		return rspCode
	}

	if this.table.CurXRound < 1 {
		for _, v := range this.table.SeatArr {
			if v == nil {
				continue
			}
			this.table.SendToSeat(v.GetSeatData().Number, protoGameBasic.ID_ReqLeaveTable, &protoGameBasic.JsonResponse{Status: 0})
		}
		return rspCode
	}

	this.handleXiaoJieSuan()
	this.handleDaJieSuan()

	return rspCode
}

func (this *ZhaJinHuaTable) timerCheckUnreadyPlayer() {
	var readyCount, lookerCount int32
	this.playingSeatArr = make([]int32, 0, this.gameRule.MaxPlayer)
	for _, v := range this.table.SeatArr {
		if v == nil {
			continue
		}
		if v.GetSeatData().IsAssignSeatState(qpTable.SS_Ready) == true {
			readyCount += 1
			this.playingSeatArr = append(this.playingSeatArr, int32(v.GetSeatData().Number))
		}
		if v.GetSeatData().IsAssignSeatState(qpTable.SS_Looker) == true {
			lookerCount += 1
		}
	}

	if this.table.CurXRound == 0 {
		if this.gameRule.FirstRoundReady == 0 {
			// 所有人准备后，游戏开始
			if readyCount == this.table.MaxPlayers-lookerCount {
				this.onGameStart(nil)
			}
		} else {
			//if readyCount < this.gameRule.FirstRoundReady {
			//	return
			//}
			for _, v := range this.table.SeatArr {
				if v == nil {
					continue
				}
				if v.GetSeatData().IsAssignSeatState(qpTable.SS_Ready) == true {
					continue
				}
				msgReady := mateProto.MessageMaTe{
					SenderID:  int64(v.GetSeatData().Player.ID),
					MessageID: protoGameBasic.ID_GameReady,
				}
				this.table.RootTable.OnMessage(&msgReady)
				readyCount++
			}
			if this.table.IsAssignTableState(qpTable.TS_Playing) {
				return
			}
			this.launchTimer = false
			if readyCount < this.gameRule.FirstRoundReady || readyCount < 2 {
				return
			}
			this.onGameStart(nil)
		}
	} else {
		if readyCount == this.table.GetCurSeatCount()-lookerCount {
			this.onGameStart(nil)
		}
	}
}

func (this *ZhaJinHuaTable) onGetPai(pro *mateProto.MessageMaTe) int32 {
	if pro.Source != "robot" {
		return 0
	}

	r_ := SC_GetPai{Pai: make([]GetPai, 0, 8)}

	for _, v := range this.table.SeatArr {
		if v == nil {
			continue
		}
		s_ := v.(*ZhaJinHuaSeat)
		if len(s_.shouPai) < 1 {
			continue
		}
		r_.Pai = append(r_.Pai, GetPai{UID: int64(s_.seatData.Player.ID), SeatNo: int32(s_.seatData.Number), Pai: s_.shouPai})
	}
	s_ := this.table.GetSeatDataByPlayerID(qpTable.PlayerID(pro.SenderID))
	if s_ == nil {
		return 0
	}

	this.table.SendMsgToSeat(s_.GetSeatData().Number,
		&mateProto.MessageMaTe{MessageID: pro.MessageID, MsgBody: &r_})

	return 0
}

func (this *ZhaJinHuaTable) onChangePai(pro *mateProto.MessageMaTe) int32 {
	if pro.Source != "robot" {
		return 0
	}

	param := CS_ChangePai{}
	err := json.Unmarshal(pro.Data, &param)
	if err != nil {
		return 0
	}

	s_ := this.table.GetSeatDataByPlayerID(qpTable.PlayerID(pro.SenderID))
	if s_ == nil {
		return 0
	}

	ns_ := s_.(*ZhaJinHuaSeat)
	ns_.pro = param.Pro

	return 0
}
