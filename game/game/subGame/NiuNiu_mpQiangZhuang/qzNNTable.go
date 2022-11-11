package NiuNiu_mpQiangZhuang

import (
	"encoding/json"
	"github.com/golang/glog"
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

const timerAutoBiPai = protoGameBasic.NiuNiu
const timerAutoXiaZhu = protoGameBasic.NiuNiu + 1
const timerAutoQiangZhuang = protoGameBasic.NiuNiu + 2
const timerCheckUnreadyPlayer = protoGameBasic.NiuNiu + 3

const TS_QiangZhuang = qpTable.TS_CustomDefineBase
const TS_XiaZhu = qpTable.TS_CustomDefineBase * 2
const TS_BiPai = qpTable.TS_CustomDefineBase * 4

type NiuNiuMPQZTable struct {
	PaiMgr      *NiuNiuPaiMgr // 牌的管理器
	logic       niuNiuLogic
	table       *qpTable.QPTable
	gameRule    NiuNiuRule
	launchTimer bool

	// 小局 待清理 成员
	tuiZhu           []int32 // 推注数量
	tuizhuSeat       []int32
	stageTime        int64   // 时间戳
	playingSeatArr   []int32 // 在玩座位
	bankerSeatNumber qpTable.SeatNumber
}

// 清空每一小局数据
func (this *NiuNiuMPQZTable) CleanRoundData() {
	this.tuiZhu = []int32{}
	this.tuizhuSeat = nil
	this.stageTime = 0
	this.playingSeatArr = nil
	this.bankerSeatNumber = qpTable.INVALID_SEAT_NUMBER
	this.table.CleanRoundData()
}

func (this *NiuNiuMPQZTable) SetTableNumber(tabNumber int32) {
	this.table.TableNum = tabNumber
}

func (this *NiuNiuMPQZTable) GetStatus() int32 {
	return int32(this.table.Status)
}

func (this *NiuNiuMPQZTable) ParseTableOptConfig(gameRuleCfg string) (rspCode int32, err error) {

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

func (this *NiuNiuMPQZTable) GetMaxRound() int32 {
	return this.gameRule.MaxRoundCount
}

func (this *NiuNiuMPQZTable) OnMessage(msg *mateProto.MessageMaTe) int32 {

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
	case ID_GetRemainingPai:
		return this.onGetRemainingPai(msg)
	case ID_XiaZhu:
		return this.onXiaZHu(msg)
	case ID_PlayerLiangPai:
		return this.onPlayerLiangPai(msg)
	case ID_PlayerQiangZhuang:
		return this.onQiangZhuang(msg)
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

func (this *NiuNiuMPQZTable) GetBaseQPTable() *qpTable.QPTable {
	return this.table
}

func (this *NiuNiuMPQZTable) onPrivateJoinTable(msg *mateProto.MessageMaTe) int32 {

	if this.table.Status > qpTable.TS_WaitingPlayerEnter {
		return mateProto.Err_GameStarted
	}
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

func (this *NiuNiuMPQZTable) onClubJoinTable(msg *mateProto.MessageMaTe) int32 {

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

func (this *NiuNiuMPQZTable) onPlayerLeave(pro *mateProto.MessageMaTe) int32 {

	if this.table.IsAssignTableState(qpTable.TS_WaitingPlayerEnter) == false {
		return mateProto.Err_TableStatusNotMatch
	}
	return this.table.OnLeave(pro)
}

func (this *NiuNiuMPQZTable) onReady(pro *mateProto.MessageMaTe) int32 {
	funRes := this.GetBaseQPTable().OnMessage(pro)
	if funRes != mateProto.Err_Success {
		return funRes
	}

	var readyCount, lookerCount int32
	this.playingSeatArr = make([]int32, 0, 10)
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
			//this.GetBaseQPTable().BroadCastGameEvent(ID_StartCountdownClock, nil)
		}
	} else {
		if readyCount == this.table.GetCurSeatCount()-lookerCount {
			return this.onGameStart(pro)
		}
	}

	return funRes
}

func (this *NiuNiuMPQZTable) onGameStart(pro *mateProto.MessageMaTe) int32 {

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

	this.FaPai()

	this.table.TableRule.TuoGuanTime = 8
	if this.table.TableRule.TuoGuanTime > 0 {
		this.table.GameTimer.PutTableTimer(timerAutoQiangZhuang, this.table.TableRule.TuoGuanTime*1000, this.autoQiangZhuang)
	}

	return mateProto.Err_Success
}

func (this *NiuNiuMPQZTable) onQiangZhuang(pro *mateProto.MessageMaTe) int32 {
	if this.table.IsAssignTableState(TS_QiangZhuang) == false {
		return mateProto.Err_TableStatusNotMatch
	}
	operQiangZhuang := CS_QiangZhuang{}
	err := json.Unmarshal(pro.Data, &operQiangZhuang)
	if err != nil {
		return mateProto.Err_ProtocolDataErr
	}

	if operQiangZhuang.Value < 0 || operQiangZhuang.Value > 4 || operQiangZhuang.Value == 1 {
		return mateProto.Err_OperationParamErr
	}

	seat := this.table.GetSeatDataByPlayerID(qpTable.PlayerID(pro.SenderID))
	if seat == nil {
		return mateProto.Err_NotFindPlayer
	}
	seatData := seat.(*NiuNiuMPQZSeat)
	if seatData.seatData.IsAssignSeatState(qpTable.SS_Looker) == true {
		return mateProto.Err_ActionNotMatchStatus
	}

	if seatData.qiangZhuang > 0 {
		return mateProto.Err_OperationRepeat
	}

	//if (seatData.seatData.SeatScore+seatData.seatData.ClubScore) < this.gameRule.ZhuangMinScoreFloat64 &&
	//	operQiangZhuang.Value > 0 {
	//	return mateProto.Err_SeatScoreLittle
	//}

	nnSeat := seat.(*NiuNiuMPQZSeat)
	nnSeat.qiangZhuang = operQiangZhuang.Value

	this.table.BroadCastGameEvent(ID_BroacastQiangZhuang,
		CS_BroacastQiangZhuang{SeatNum: int32(seatData.seatData.Number), Value: operQiangZhuang.Value})

	// 是否所有人都已操作
	maxQiangZhuangArr := make([]int32, 0, 10)
	var (
		maxQiangZhuangVale             int32
		playingCount, qiangZhuangCount int32
	)
	for _, v := range this.table.SeatArr {
		if v == nil {
			continue
		}
		if v.GetSeatData().IsAssignSeatState(qpTable.SS_Playing) == false {
			continue
		}
		playingCount += 1
		nnSeat := v.(*NiuNiuMPQZSeat)

		if nnSeat.qiangZhuang < 0 {
			continue
		}

		qiangZhuangCount += 1
		if nnSeat.qiangZhuang > maxQiangZhuangVale {
			maxQiangZhuangVale = v.(*NiuNiuMPQZSeat).qiangZhuang
			maxQiangZhuangArr = []int32{int32(nnSeat.seatData.Number)}
		} else if nnSeat.qiangZhuang == maxQiangZhuangVale {
			maxQiangZhuangArr = append(maxQiangZhuangArr, int32(nnSeat.seatData.Number))
		}
	}

	if qiangZhuangCount != playingCount {
		return 0
	}

	// 所有人抢庄后，开始确定庄家
	this.table.GameTimer.RemoveByTimeID(timerAutoQiangZhuang)
	this.table.DelTableState(TS_QiangZhuang)
	this.table.AppendTableState(TS_XiaZhu)
	this.table.BroadcastTableStatus()
	this.stageTime = time.Now().Unix()

	if len(maxQiangZhuangArr) < 1 {
		glog.Warning("onQiangZhuang().....,", this.table.TableNum)
		return 0
	}

	if len(maxQiangZhuangArr) > 1 {
		index := rand.Intn(len(maxQiangZhuangArr))
		if index < 0 {
			index = 0
		}
		this.bankerSeatNumber = qpTable.SeatNumber(maxQiangZhuangArr[index])
		bankerSeat := this.table.SeatArr[this.bankerSeatNumber].(*NiuNiuMPQZSeat)

		// 推注
		if bankerSeat.qiangZhuang == 4 && this.gameRule.IsTuiZhu == true {
			this.tuizhuSeat = make([]int32, 0, len(maxQiangZhuangArr)-1)
			for _, v := range maxQiangZhuangArr {
				if this.bankerSeatNumber == qpTable.SeatNumber(v) {
					continue
				}
				this.tuizhuSeat = append(this.tuizhuSeat, v)
			}
		}
	} else {
		this.bankerSeatNumber = qpTable.SeatNumber(maxQiangZhuangArr[0])
	}
	bankerNNSeat := this.table.SeatArr[this.bankerSeatNumber].(*NiuNiuMPQZSeat)
	if bankerNNSeat.qiangZhuang < 1 {
		bankerNNSeat.qiangZhuang = 1
	}
	bankerNNSeat.zhuangCount += 1

	this.table.BroadCastGameEvent(ID_NoticeZhuangJia,
		&SC_ZhuangJia{SeatNumber: int32(this.bankerSeatNumber),
			Value:            bankerNNSeat.qiangZhuang,
			MaxXiaZhuSeatArr: maxQiangZhuangArr})

	if len(this.tuizhuSeat) > 0 {
		switch this.gameRule.XiaZhuOpt {
		case 0:
			this.tuiZhu = []int32{10}
		case 1:
			this.tuiZhu = []int32{20}
		case 2:
			this.tuiZhu = []int32{30}
		case 3:
			this.tuiZhu = []int32{80}
		}
	}

	this.table.BroadCastGameEvent(ID_NoticeXiaZhu, &SC_NoticeXiaZhu{
		PlayingSeatArr: this.playingSeatArr,
		TuiZhuSeatArr:  this.tuizhuSeat,
		TuiZhu:         this.tuiZhu})

	if this.table.TableRule.TuoGuanTime > 0 {
		this.table.GameTimer.PutTableTimer(timerAutoXiaZhu, this.table.TableRule.TuoGuanTime*1000, this.autoXiaZhu)
	}
	return 0
}

func (this *NiuNiuMPQZTable) onXiaZHu(pro *mateProto.MessageMaTe) int32 {
	if this.table.IsAssignTableState(TS_XiaZhu) == false {
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
	seatData := seat.(*NiuNiuMPQZSeat)
	if seatData.seatData.IsAssignSeatState(qpTable.SS_Looker) == true {
		return mateProto.Err_ActionNotMatchStatus
	}

	// 庄家不能下注
	if seatData.seatData.Number == this.bankerSeatNumber {
		return mateProto.Err_ActionNotMatchStatus
	}

	ok := false
	isTuiZhu := false
	//isMinXiaZhu := false
	for i, v := range this.gameRule.XiaZhuArr[this.gameRule.XiaZhuOpt] {
		if v == operXiaZhu.Value {
			ok = true
			if i == 0 {
				//isMinXiaZhu = true
			}
			break
		}
	}
	//if (seatData.seatData.SeatScore+seatData.seatData.ClubScore) < this.gameRule.ZhuangMinScoreFloat64 &&
	//	isMinXiaZhu == false {
	//	return mateProto.Err_SeatScoreLittle
	//}

	if ok == false {
		for _, v := range this.tuizhuSeat {
			if v == int32(seatData.seatData.Number) {
				ok = true
				break
			}
		}
		if ok == false {
			return mateProto.Err_OperationParamErr
		}

		for _, v := range this.tuiZhu {
			if v == operXiaZhu.Value {
				ok, isTuiZhu = true, true
				break
			}
		}
		if ok == false {
			return mateProto.Err_OperationParamErr
		}
	}

	nnSeat := seat.(*NiuNiuMPQZSeat)
	nnSeat.xiaZhu = operXiaZhu.Value

	if isTuiZhu == true {
		nnSeat.tuiZhuCount += 1
	}

	this.table.BroadCastGameEvent(ID_BroadcastXiaZhu,
		SC_XiaZhu{SeatNumber: int32(seatData.seatData.Number), XiaZhu: operXiaZhu.Value})

	// 是否所有人都 下注了
	var playingCount, beiShuCount int32
	for _, v := range this.table.SeatArr {
		if v == nil {
			continue
		}
		if v.GetSeatData().IsAssignSeatState(qpTable.SS_Playing) == false {
			continue
		}
		if v.GetSeatData().Number == this.bankerSeatNumber {
			continue
		}

		playingCount += 1

		if v.(*NiuNiuMPQZSeat).xiaZhu > 0 {
			beiShuCount += 1
		}
	}

	// 所有人下注后，开始发牌
	if beiShuCount == playingCount {
		this.FaOnePai()

		if this.table.TableRule.TuoGuanTime > 0 {
			this.table.GameTimer.PutTableTimer(timerAutoBiPai, this.table.TableRule.TuoGuanTime*1000, this.autoLiang)
		}
	}

	return 0
}

func (this *NiuNiuMPQZTable) autoXiaZhu() {

	if this.table.IsAssignTableState(TS_XiaZhu) == false {
		return
	}

	for _, v := range this.table.SeatArr {
		if v == nil {
			continue
		}
		if v.GetSeatData().IsAssignSeatState(qpTable.SS_Playing) == false {
			continue
		}
		if v.GetSeatData().Number == this.bankerSeatNumber {
			continue
		}

		if v.(*NiuNiuMPQZSeat).xiaZhu > 0 {
			continue
		}

		beiShuData, _ := json.Marshal(&CS_XiaZhu{this.gameRule.XiaZhuArr[this.gameRule.XiaZhuOpt][0]})
		msg := mateProto.MessageMaTe{
			SenderID:  int64(v.GetSeatData().Player.ID),
			MessageID: ID_XiaZhu,
			Data:      beiShuData,
		}
		this.OnMessage(&msg)
	}
}

func (this *NiuNiuMPQZTable) autoQiangZhuang() {

	if this.table.IsAssignTableState(TS_QiangZhuang) == false {
		return
	}

	for _, v := range this.table.SeatArr {
		if v == nil {
			continue
		}
		if v.GetSeatData().IsAssignSeatState(qpTable.SS_Playing) == false {
			continue
		}
		if v.(*NiuNiuMPQZSeat).qiangZhuang > 0 {
			continue
		}

		qiangZhuanData, _ := json.Marshal(&CS_QiangZhuang{})
		msg := mateProto.MessageMaTe{
			SenderID:  int64(v.GetSeatData().Player.ID),
			MessageID: ID_PlayerQiangZhuang,
			Data:      qiangZhuanData,
		}
		this.OnMessage(&msg)
	}
}

func (this *NiuNiuMPQZTable) autoLiang() {

	if this.table.IsAssignTableState(TS_BiPai) == false {
		return
	}

	for _, v := range this.table.SeatArr {
		if v == nil {
			continue
		}
		if v.GetSeatData().IsAssignSeatState(qpTable.SS_Playing) == false {
			continue
		}
		if v.(*NiuNiuMPQZSeat).isLiang == true {
			continue
		}

		msg := mateProto.MessageMaTe{
			SenderID:  int64(v.GetSeatData().Player.ID),
			MessageID: ID_PlayerLiangPai,
			Data:      nil,
		}
		this.OnMessage(&msg)
	}
}

func (this *NiuNiuMPQZTable) FaPai() {

	this.logic.lzPai =
		this.PaiMgr.XiPai(this.table.GetCurSeatCount()-this.table.LookerCount, 5, this.gameRule.IsDoublePai, this.gameRule.IsLaiZi)

	maxPXIndex := 0
	maxPaiXing := int32(0)
	maxPai := int8(0)
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
		s_ := v.(*NiuNiuMPQZSeat)

		paiArr := this.PaiMgr.GetGroupPai(int32(v.GetSeatData().Number), 5)
		faPaiArr = append(faPaiArr, paiArr)
		px, _, mp, _ := this.logic.GetLaiZiPaiXing(paiArr)
		if px > maxPaiXing {
			maxPaiXing, maxPXIndex = px, len(faPaiArr)-1
			maxPai = mp
		} else if px == maxPaiXing && mp > maxPai {
			maxPaiXing, maxPXIndex = px, len(faPaiArr)-1
			maxPai = mp
		}

		if s_.pro > maxPro {
			maxPro, maxProIndex = s_.pro, len(faPaiArr)-1
		}
	}

	if maxPro > 0 && maxProIndex != maxPXIndex {
		r_ := rand.Intn(100)
		if r_ < maxPro {
			faPaiArr[maxProIndex], faPaiArr[maxPXIndex] = faPaiArr[maxPXIndex], faPaiArr[maxProIndex]
		}
		//t_ := faPaiArr[maxPXIndex]
		//faPaiArr[maxPXIndex] = faPaiArr[maxProIndex]
		//faPaiArr[maxProIndex] = t_
	}

	faPaiIndex := 0
	tempPlayingSeatArr := make([]int32, 0, len(this.table.SeatArr))
	for i, v := range this.table.SeatArr {
		if v == nil {
			continue
		}
		this.table.OperateRec.PutPlayer(v.GetSeatData())

		if v.GetSeatData().IsAssignSeatState(qpTable.SS_Playing) == false {
			this.table.SendToSeat(qpTable.SeatNumber(i),
				ID_FaShouPai,
				&SC_FaShouPai{SeatNumber: int32(i), Pai: []int8{InvalidPai, InvalidPai, InvalidPai, InvalidPai}, PlayingSeat: this.playingSeatArr})
			continue
		}
		tempPlayingSeatArr = append(tempPlayingSeatArr, int32(i))

		//paiArr := this.PaiMgr.GetGroupPai(int32(v.GetSeatData().Number), 5)
		paiArr := faPaiArr[faPaiIndex]
		faPaiIndex++

		seat := v.(*NiuNiuMPQZSeat)
		seat.SetShouPai(paiArr)

		this.table.SendGameEventToSeat(qpTable.SeatNumber(i),
			ID_FaShouPai,
			&SC_FaShouPai{SeatNumber: int32(i), Pai: paiArr[:4], PlayingSeat: this.playingSeatArr, LaiZiPai: this.logic.lzPai})
	}

	this.table.AppendTableState(TS_QiangZhuang)
	this.table.BroadcastTableStatus()
	this.stageTime = time.Now().Unix()
}

func (this *NiuNiuMPQZTable) FaOnePai() {

	this.table.DelTableState(TS_XiaZhu)
	this.table.AppendTableState(TS_BiPai)
	this.table.BroadcastTableStatus()
	this.table.GameTimer.RemoveByTimeID(timerAutoXiaZhu)
	this.stageTime = time.Now().Unix()

	for i, v := range this.table.SeatArr {
		if v == nil {
			continue
		}
		if v.GetSeatData().IsAssignSeatState(qpTable.SS_Playing) == false {
			this.table.SendToSeat(qpTable.SeatNumber(i),
				ID_FaShouPai,
				&SC_FaShouPai{SeatNumber: int32(i), Pai: []int8{InvalidPai}})
			continue
		}

		this.table.SendGameEventToSeat(qpTable.SeatNumber(i),
			ID_FaShouPai,
			&SC_FaShouPai{SeatNumber: int32(i), Pai: v.(*NiuNiuMPQZSeat).shouPai[4:5]})
	}

	//this.BroadcastPaiXing()
}

func (this *NiuNiuMPQZTable) onPlayerLiangPai(pro *mateProto.MessageMaTe) int32 {
	if this.table.IsAssignTableState(TS_BiPai) == false {
		return mateProto.Err_ActionNotMatchStatus
	}

	seat := this.table.GetSeatDataByPlayerID(qpTable.PlayerID(pro.SenderID))
	if seat == nil {
		return mateProto.Err_NotFindPlayer
	}
	nnSeat := seat.(*NiuNiuMPQZSeat)
	if nnSeat.isLiang == true {
		return mateProto.Err_OperationRepeat
	}
	if nnSeat.seatData.IsAssignSeatState(qpTable.SS_Playing) == false {
		return mateProto.Err_ActionNotMatchStatus
	}

	var LaiZiChanged []int8 //arrangePaiArr
	nnSeat.isLiang = true
	nnSeat.paiXing, _, nnSeat.maxPai, LaiZiChanged = this.logic.GetLaiZiPaiXing(nnSeat.shouPai)

	if nnSeat.paiXing > nnSeat.maxPaiXing {
		nnSeat.maxPaiXing = nnSeat.paiXing
	}

	this.table.BroadCastGameEvent(ID_BroadcastLiangPai,
		&SC_BroadcastLiangPai{LiangPaiXing{
			SeatNumber:   int32(nnSeat.seatData.Number),
			PaiArr:       nnSeat.shouPai,
			LaiZiChanged: LaiZiChanged,
			PaiXing:      nnSeat.paiXing,
			LastPai:      nnSeat.shouPai[4]}})

	// 全部人 亮牌后, 游戏结束
	var liangPaiCount, playingCount int32
	for _, v := range this.table.SeatArr {
		if v == nil {
			continue
		}
		if v.GetSeatData().IsAssignSeatState(qpTable.SS_Playing) == false {
			continue
		}
		playingCount += 1
		if v.(*NiuNiuMPQZSeat).isLiang == true {
			liangPaiCount += 1
		}
	}

	if liangPaiCount == playingCount {
		this.BroadcastPaiXing()
	}

	return 0
}

func (this *NiuNiuMPQZTable) BroadcastPaiXing() {

	this.table.GameTimer.RemoveByTimeID(timerAutoBiPai)

	tempLiangPaiXing := make([]LiangPaiXing, 0, 10)
	for _, v := range this.GetBaseQPTable().SeatArr {
		if v == nil {
			continue
		}
		niuNiuSeat := v.(*NiuNiuMPQZSeat)
		if len(niuNiuSeat.shouPai) < 5 {
			continue
		}
		if niuNiuSeat.isLiang == true {
			continue
		}
		var LaiZiChanged []int8 //arrangePaiArr
		niuNiuSeat.paiXing, _, niuNiuSeat.maxPai, LaiZiChanged = this.logic.GetLaiZiPaiXing(niuNiuSeat.shouPai)

		tempLiangPaiXing = append(tempLiangPaiXing,
			LiangPaiXing{int32(niuNiuSeat.seatData.Number),
				niuNiuSeat.shouPai,
				niuNiuSeat.paiXing,
				niuNiuSeat.shouPai[4],
				LaiZiChanged})
	}
	if len(tempLiangPaiXing) > 0 {
		this.table.BroadCastGameEvent(ID_BroadcastLiangPaiXing, &SS_LiangPaiXing{tempLiangPaiXing})
	}

	this.RoundOverFun()
}

func (this *NiuNiuMPQZTable) RoundOverFun() {

	this.table.SetTableState(qpTable.TS_Playing)

	bankerNNSeat := this.table.SeatArr[this.bankerSeatNumber].(*NiuNiuMPQZSeat)
	paiXingBeiShu := float64(1)
	cBankerScore := bankerNNSeat.seatData.ClubScore + bankerNNSeat.seatData.SeatScore

	winnerArr := make([]qpTable.SeatNumber, 0, 10)
	//winnerMap := make(map[qpTable.SeatNumber]*NiuNiuMPQZSeat)

	loseArr := make([]qpTable.SeatNumber, 0, 10)
	loseMap := make(map[qpTable.SeatNumber]*NiuNiuMPQZSeat)

	// 牌型分
	{
		var tempScore float64
		tempSeatNum := bankerNNSeat.seatData.Number
		for {
			v := this.table.GetNextPlayingSeat(tempSeatNum)
			if v == nil {
				continue
			}
			playerSeat := v.(*NiuNiuMPQZSeat)
			if playerSeat.seatData.Number == this.bankerSeatNumber {
				break
			}
			if len(playerSeat.shouPai) < 5 {
				continue
			}
			tempSeatNum = playerSeat.seatData.Number

			if this.logic.Compare(bankerNNSeat.paiXing, bankerNNSeat.maxPai, playerSeat.paiXing, playerSeat.maxPai) == true {
				if this.gameRule.IsSuper {
					if v, ok := superBeiShuMap[bankerNNSeat.paiXing]; ok == true {
						paiXingBeiShu = v
					}
				} else {
					if v, ok := normalBeiShuMap[bankerNNSeat.paiXing]; ok == true {
						paiXingBeiShu = v
					}
				}
				loseArr = append(loseArr, playerSeat.seatData.Number)
				loseMap[playerSeat.seatData.Number] = playerSeat
				tempScore = float64(bankerNNSeat.qiangZhuang) * float64(playerSeat.xiaZhu) * paiXingBeiShu * this.gameRule.MultipleFloat64

				bankerNNSeat.seatData.RoundScore += tempScore

				if tempScore > playerSeat.seatData.ClubScore+playerSeat.seatData.SeatScore {
					tempScore = playerSeat.seatData.ClubScore + playerSeat.seatData.SeatScore
				}
				playerSeat.roundScore = tempScore
			} else {
				if this.gameRule.IsSuper {
					if v, ok := superBeiShuMap[playerSeat.paiXing]; ok == true {
						paiXingBeiShu = v
					}
				} else {
					if v, ok := normalBeiShuMap[playerSeat.paiXing]; ok == true {
						paiXingBeiShu = v
					}
				}
				winnerArr = append(winnerArr, playerSeat.seatData.Number)
				//winnerMap[playerSeat.seatData.Number] = playerSeat
				tempScore = float64(bankerNNSeat.qiangZhuang) * float64(playerSeat.xiaZhu) * paiXingBeiShu * this.gameRule.MultipleFloat64

				bankerNNSeat.seatData.RoundScore -= tempScore

				if tempScore > playerSeat.seatData.ClubScore+playerSeat.seatData.SeatScore {
					tempScore = playerSeat.seatData.ClubScore + playerSeat.seatData.SeatScore
				}
				playerSeat.roundScore += tempScore
			}
		}
	}

	//for _, v := range this.table.SeatArr {
	//	if v == nil {
	//		continue
	//	}
	//	playerSeat := v.(*NiuNiuMPQZSeat)
	//	glog.Warning("uid:", playerSeat.seatData.Player.ID, ", round score:=", playerSeat.roundScore, ",club score:=", playerSeat.seatData.ClubScore)
	//}

	bankerNNSeat.seatData.RoundScore = 0
	tempBankerScore := cBankerScore

	// 给赢家赔付
	for _, wv := range winnerArr {
		ws := this.table.SeatArr[wv].(*NiuNiuMPQZSeat).roundScore
		// 自己赔付
		if tempBankerScore >= ws {
			tempBankerScore -= ws
			bankerNNSeat.seatData.RoundScore -= ws
			this.table.SeatArr[wv].(*NiuNiuMPQZSeat).seatData.RoundScore += ws
			//glog.Warning("win := ")
			continue
		}

		for _, lv := range loseArr {
			loseSeat, ok := loseMap[lv]
			if ok == false {
				continue
			}

			ls := loseSeat.roundScore + loseSeat.seatData.RoundScore
			temp := tempBankerScore + ls
			if temp >= ws {
				tt := ws - tempBankerScore
				tempBankerScore += tt
				loseSeat.seatData.RoundScore -= tt
				bankerNNSeat.seatData.RoundScore += tt

				//glog.Warning("lose uid:= ", loseSeat.seatData.Player.ID, ",RoundScore:=", loseSeat.seatData.RoundScore, "....", ls)
			} else {
				tempBankerScore += ls
				loseSeat.seatData.RoundScore -= ls
				bankerNNSeat.seatData.RoundScore += ls
				//glog.Warning("lose uid:= ", loseSeat.seatData.Player.ID, ",RoundScore:=", loseSeat.seatData.RoundScore, "....", ls)
			}
			if loseSeat.roundScore+loseSeat.seatData.RoundScore < 0.001 {
				delete(loseMap, lv)
			}

			if tempBankerScore >= ws {
				break
			}
		}

		temp := float64(0)
		if tempBankerScore > ws {
			temp = ws
		} else {
			temp = tempBankerScore
		}
		tempBankerScore -= temp
		bankerNNSeat.seatData.RoundScore -= temp
		this.table.SeatArr[wv].(*NiuNiuMPQZSeat).seatData.RoundScore += temp
	}

	for _, lv := range loseArr {
		loseSeat, ok := loseMap[lv]
		if ok == false {
			continue
		}
		//if loseSeat.roundScore-loseSeat.seatData.RoundScore < 0 {
		//	fmt.Println("vvvvvvvvvvvv")
		//}
		if tempBankerScore > (cBankerScore * 2) {
			break
		}

		ls := loseSeat.roundScore + loseSeat.seatData.RoundScore
		temp := tempBankerScore + ls
		if temp >= (cBankerScore * 2) {
			ls = cBankerScore*2 - tempBankerScore
			temp = ls
		} else {
			temp = ls
		}

		tempBankerScore += temp
		loseSeat.seatData.RoundScore -= temp
		bankerNNSeat.seatData.RoundScore += temp
		//glog.Warning("lose uid:= ", loseSeat.seatData.Player.ID, ",RoundScore:=", loseSeat.seatData.RoundScore, ".....", temp)
		if tempBankerScore >= (cBankerScore * 2) {
			break
		}
	}

	// 结算
	for _, v := range this.table.SeatArr {
		if v == nil {
			continue
		}
		player := v.(*NiuNiuMPQZSeat)
		if len(player.shouPai) < 5 {
			continue
		}
		player.seatData.SeatScore += player.seatData.RoundScore
	}

	recPlayerGameScoreArr := this.handleXiaoJieSuan()
	gameStepRec, _ := this.GetBaseQPTable().OperateRec.Pack()

	// 小局记录
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

	if 0 != this.table.GameOver() {
		this.handleDaJieSuan()
		return
	}

	//this.table.BroadcastTableStatus()

	if this.table.CurXRound >= this.gameRule.MaxRoundCount {
		this.table.DissolveType = qpTable.DT_Playing
		this.handleDaJieSuan()
		return
	}

	this.CleanRoundData()
	this.table.TableRule.TimerAutoReady()
}

func (this *NiuNiuMPQZTable) handleXiaoJieSuan() []*protoGameBasic.PlayerGameScore {
	if this.table.CurXRound < 1 {
		return nil
	}

	roundOver := BroadcastRoundOver{
		ClubID:      this.table.MZClubID,
		ClubPlayID:  this.table.ClubPlayID,
		TableNumber: this.GetBaseQPTable().TableNum,
		Timestamp:   time.Now().Unix(),
	}
	msgRoundOver := mateProto.MessageMaTe{MessageID: ID_RoundOver, MsgBody: &roundOver}

	roundSeatScoreArr := make([]*RoundSeatScore, 0, 10)
	recPlayerGameScoreArr := make([]*protoGameBasic.PlayerGameScore, 0, 10)

	for _, v := range this.GetBaseQPTable().SeatArr {
		if v == nil {
			continue
		}
		seat := v.GetSeatData()
		if seat.IsAssignSeatState(qpTable.SS_Looker) == true {
			continue
		}
		if seat.IsAssignSeatState(qpTable.SS_Playing) == false {
			continue
		}
		seat.RoundOverMsg = &msgRoundOver

		temp := protoGameBasic.PlayerGameScore{
			UID:    int64(seat.Player.ID),
			Nick:   seat.Player.Nick,
			ClubID: seat.ClubID,
			SScore: commonDef.Float64Mul1000ToService(seat.RoundScore),
		}
		recPlayerGameScoreArr = append(recPlayerGameScoreArr, &temp)

		roundSeatScoreArr = append(roundSeatScoreArr, &RoundSeatScore{
			ClubID:     seat.ClubID,
			UID:        int64(seat.Player.ID),
			NickName:   seat.Player.Nick,
			Head:       seat.Player.Head,
			SeatNumber: int32(seat.Number),
			Pai:        v.(*NiuNiuMPQZSeat).shouPai,
			GameScore:  commonDef.Float64ToString(seat.RoundScore),
			SeatScore:  commonDef.Float64ToString(seat.SeatScore)})

		//v.CleanRoundData()
	}
	roundOver.SeatData = roundSeatScoreArr

	this.GetBaseQPTable().BroadCastGameEvent(ID_RoundOver, &roundOver)

	return recPlayerGameScoreArr
}

// 大结算
func (this *NiuNiuMPQZTable) handleDaJieSuan() {
	if this.table.CurXRound < 1 {
		return
	}

	msgBody := BroadcastGameOver{TableNumber: this.table.TableNum,
		CurRound:     this.table.CurXRound,
		MaxRound:     this.gameRule.MaxRoundCount,
		EndTime:      time.Now().Unix(),
		SeatData:     make([]*GameOverSeatData, 0, this.table.GetCurSeatCount()),
		DissolveType: this.table.DissolveType,
		ClubID:       this.table.MZClubID,
		ClubPlayID:   this.table.ClubPlayID}

	for _, v := range this.table.SeatArr {
		if v == nil {
			continue
		}

		pdkSeat := v.(*NiuNiuMPQZSeat)

		tempSeat := &GameOverSeatData{
			ClubID:           pdkSeat.seatData.ClubID,
			UID:              int64(pdkSeat.seatData.Player.ID),
			Nick:             pdkSeat.seatData.Player.Nick,
			Head:             pdkSeat.seatData.Player.Head,
			TuiZhuCount:      pdkSeat.tuiZhuCount,
			QiangZhuangCount: pdkSeat.zhuangCount,
			MaxPaiXing:       pdkSeat.maxPaiXing,
			SeatScore:        commonDef.Float64ToString(pdkSeat.seatData.SeatScore),
			SeatScoreInt:     commonDef.Float64Mul1000ToService(pdkSeat.seatData.SeatScore),
			IsMaxWin:         false}

		msgBody.SeatData = append(msgBody.SeatData, tempSeat)
	}
	sort.Sort(msgBody.SeatData)
	msgBody.SeatData[0].IsMaxWin = true
	for i := 1; i < len(msgBody.SeatData); i++ {
		if msgBody.SeatData[i].SeatScoreInt < msgBody.SeatData[0].SeatScoreInt {
			break
		}
		if msgBody.SeatData[i].SeatScoreInt == msgBody.SeatData[0].SeatScoreInt {
			msgBody.SeatData[i].IsMaxWin = true
		}
	}

	this.table.SendToAllPlayer(ID_BroadcastGameOver, &msgBody)

	this.GetBaseQPTable().SetTableState(qpTable.TS_Invalid)
}

func (this *NiuNiuMPQZTable) onTableExpire(pro *mateProto.MessageMaTe) int32 {

	this.table.DissolveType = qpTable.DT_LiveTimeout

	this.handleXiaoJieSuan()

	this.handleDaJieSuan()

	return this.table.OnMessage(pro)
}

func (this *NiuNiuMPQZTable) onTableData(pro *mateProto.MessageMaTe) int32 {

	seat := this.table.GetSeatDataByPlayerID(qpTable.PlayerID(pro.SenderID))
	if seat == nil {
		return mateProto.Err_NotFindPlayer
	}

	nnSeatArr := make([]*NiuNiuSeatData, 0)
	for _, v := range this.table.SeatArr {
		if v == nil {
			continue
		}

		pkNNSeat := v.(*NiuNiuMPQZSeat)
		tempNNSeat := NiuNiuSeatData{
			UID:           int64(pkNNSeat.seatData.Player.ID),
			Nick:          pkNNSeat.seatData.Player.Nick,
			HeadURL:       pkNNSeat.seatData.Player.Head,
			IP:            pkNNSeat.seatData.Player.IP,
			Sex:           pkNNSeat.seatData.Player.Sex,
			SeatNumber:    int32(pkNNSeat.seatData.Number),
			SeatStatus:    uint32(pkNNSeat.seatData.Status),
			ClubID:        pkNNSeat.seatData.ClubID,
			ClubScore:     commonDef.Float64ToString(pkNNSeat.seatData.ClubScore),
			SeatScore:     commonDef.Float64ToString(pkNNSeat.seatData.SeatScore),
			RoundScore:    commonDef.Float64ToString(pkNNSeat.seatData.RoundScore),
			VoteStatus:    v.GetSeatData().DissolveVote,
			OperationTime: time.Now().Unix() - v.GetSeatData().OperationStart,
			PaiXing:       pkNNSeat.paiXing,
			XiaZhu:        pkNNSeat.xiaZhu,
			QZValue:       pkNNSeat.qiangZhuang,
			Liang:         pkNNSeat.isLiang,
		}

		if pkNNSeat.seatData.Lng > 0.1 && pkNNSeat.seatData.Lat > 0.1 {
			tempNNSeat.IsGPS = true
		}
		if pkNNSeat.seatData.IsAssignSeatState(qpTable.SS_Playing) == true {
			if this.table.IsAssignTableState(TS_BiPai) == true {
				if pkNNSeat.isLiang == true {
					tempNNSeat.ShouPai = pkNNSeat.shouPai
					tempNNSeat.LastPai = pkNNSeat.shouPai[4]
				} else {
					tempNNSeat.ShouPai = []int8{InvalidPai, InvalidPai, InvalidPai, InvalidPai, InvalidPai}
				}
			} else if len(pkNNSeat.shouPai) > 0 {
				tempNNSeat.ShouPai = []int8{InvalidPai, InvalidPai, InvalidPai, InvalidPai}
			}
		}

		nnSeatArr = append(nnSeatArr, &tempNNSeat)
	}

	pdkSeat := seat.(*NiuNiuMPQZSeat)

	nowTT := time.Now().Unix()
	tableData := SC_TableData{
		MZCID:              this.table.MZClubID,
		TableNumber:        this.table.TableNum,
		TableStatus:        uint32(this.table.Status),
		RoundCount:         this.table.CurXRound,
		TableRuleText:      this.table.TableRule.TableCfgJson,
		ClubRuleText:       this.table.ClubRuleText,
		BankerSeatNum:      int32(this.bankerSeatNumber),
		ClubID:             pdkSeat.seatData.ClubID,
		SeatData:           nnSeatArr,
		GameRuleText:       this.gameRule.RuleJson,
		ClubScore:          commonDef.Float64ToString(seat.GetSeatData().ClubScore),
		DissolveID:         int32(this.table.DissolveSeatNum),
		LaunchDissolveTime: nowTT - this.table.LaunchDissolveTime,
		FirstRoundReadTime: nowTT - this.table.FirstRoundReadTime,
		StageTime:          nowTT - this.stageTime,
		TuiZhuSeatArr:      this.tuizhuSeat,
		TuiZhuArr:          this.tuiZhu,
		LaiZiPai:           this.logic.lzPai,
	}

	if this.table.IsAssignTableState(TS_BiPai) == true {
		tableData.ShouPai = pdkSeat.shouPai
	} else if len(pdkSeat.shouPai) > 0 {
		tableData.ShouPai = pdkSeat.shouPai[:4]
	}

	this.table.UpdatePlayerSource(pdkSeat.seatData, pro.Source)

	this.table.SendToSeat(seat.GetSeatData().Number, ID_TableData, tableData)

	return mateProto.Err_Success
}

func (this *NiuNiuMPQZTable) onDissolveTableVote(pro *mateProto.MessageMaTe) int32 {
	rspCode := this.table.OnMessage(pro)

	if this.table.Status != qpTable.TS_Invalid {
		return rspCode
	}

	this.handleXiaoJieSuan()
	this.handleDaJieSuan()

	return rspCode
}

func (this *NiuNiuMPQZTable) timerCheckUnreadyPlayer() {
	var readyCount, lookerCount int32
	this.playingSeatArr = make([]int32, 0, 10)
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

func (this *NiuNiuMPQZTable) onCustomShouPai(pro *mateProto.MessageMaTe) int32 {
	if this.table.IsAssignTableState(qpTable.TS_Playing) == false {
		return mateProto.Err_TableStatusNotMatch
	}

	msgCustomShouPai := CS_CustomShouPai{}
	err := json.Unmarshal(pro.Data, &msgCustomShouPai)
	if err != nil {
		return mateProto.Err_ProtocolDataErr
	}

	seat := this.table.GetSeatDataByPlayerID(qpTable.PlayerID(pro.SenderID))
	if seat == nil {
		return mateProto.Err_NotFindPlayer
	}
	nnSeat := seat.(*NiuNiuMPQZSeat)
	if len(nnSeat.shouPai) != 5 {
		return mateProto.Err_ActionNotMatchStatus
	}

	powerMap, _ := db.GetPlayerPower(pro.SenderID)
	if powerMap == nil {
		return 0
	}

	if v, ok := powerMap[strconv.Itoa(int(this.table.GameID))]; ok == false || v == 0 {
		return 0
	}

	rspMsg := protoGameBasic.JsonResponse{}

	ok := this.PaiMgr.Reserve(msgCustomShouPai.ShouPai)
	if ok == false {
		rspMsg.Status = mateProto.Err_CustomPai
	} else {
		nnSeat.shouPai[4] = msgCustomShouPai.ShouPai
	}

	this.table.SendToSeat(nnSeat.seatData.Number, pro.MessageID, &rspMsg)

	return 0
}

func (this *NiuNiuMPQZTable) onGetRemainingPai(pro *mateProto.MessageMaTe) int32 {
	playerID := qpTable.PlayerID(pro.SenderID)

	seat := this.table.GetSeatDataByPlayerID(playerID)
	if seat == nil {
		return mateProto.Err_ProtocolDataErr
	}
	if seat.GetSeatData().IsAssignSeatState(qpTable.SS_Playing) == false {
		return mateProto.Err_ActionNotMatchStatus
	}

	powerMap, _ := db.GetPlayerPower(pro.SenderID)
	if powerMap == nil {
		return 0
	}

	if v, ok := powerMap[strconv.Itoa(int(this.table.GameID))]; ok == false || v == 0 {
		return 0
	}

	//type PaiInfo struct {
	//	Pai   int8
	//	Count int8
	//}
	//arr := make([]PaiInfo, 0, 27)
	//for k, v := range this.PaiMgr.GetSurplusPai() {
	//	arr = append(arr, PaiInfo{Pai: k, Count: v})
	//}

	wrapMQ.ReplyToSource(pro, this.PaiMgr.GetSurplusPai())

	return 0
}

func (this *NiuNiuMPQZTable) onForceDissolveTable(pro *mateProto.MessageMaTe) int32 {
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

func (this *NiuNiuMPQZTable) onGetPai(pro *mateProto.MessageMaTe) int32 {
	if pro.Source != "robot" {
		return 0
	}

	r_ := SC_GetPai{Pai: make([]GetPai, 0, 8)}

	for _, v := range this.table.SeatArr {
		if v == nil {
			continue
		}
		s_ := v.(*NiuNiuMPQZSeat)
		if len(s_.shouPai) < 1 {
			continue
		}
		r_.Pai = append(r_.Pai, GetPai{UID: int64(s_.seatData.Player.ID), Pai: s_.shouPai})
	}
	s_ := this.table.GetSeatDataByPlayerID(qpTable.PlayerID(pro.SenderID))
	if s_ == nil {
		return 0
	}

	this.table.SendMsgToSeat(s_.GetSeatData().Number,
		&mateProto.MessageMaTe{MessageID: pro.MessageID, MsgBody: &r_})

	return 0
}

func (this *NiuNiuMPQZTable) onChangePai(pro *mateProto.MessageMaTe) int32 {
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

	ns_ := s_.(*NiuNiuMPQZSeat)
	ns_.pro = param.Pro

	return 0
}
