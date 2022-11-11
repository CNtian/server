package NiuNiu_wzTongBi

import (
	"encoding/json"
	"github.com/golang/glog"
	commonDef "qpGame/commonDefine"
	"qpGame/commonDefine/mateProto"
	"qpGame/commonDefine/mateProto/protoGameBasic"
	"qpGame/db"
	"qpGame/qpTable"
	"qpGame/wrapMQ"
	"strconv"
	"time"
)

const timerAutoXiaZhu = protoGameBasic.NiuNiu
const timerAutoBiPai = protoGameBasic.NiuNiu + 1
const timerCheckUnreadyPlayer = protoGameBasic.NiuNiu + 2

const TS_XiaZhu = qpTable.TS_CustomDefineBase * 2
const TS_BiPai = qpTable.TS_CustomDefineBase * 4

type NiuNiuTable struct {
	PaiMgr   *NiuNiuPaiMgr // 牌的管理器
	logic    niuNiuLogic
	table    *qpTable.QPTable
	gameRule NiuNiuRule

	// 小局 待清理 成员
	stageTime      int64   // 时间戳
	playingSeatArr []int32 // 在玩座位
}

// 清空每一小局数据
func (this *NiuNiuTable) CleanRoundData() {
	this.playingSeatArr = nil
	this.table.CleanRoundData()
}

func (this *NiuNiuTable) SetTableNumber(tabNumber int32) {
	this.table.TableNum = tabNumber
}

func (this *NiuNiuTable) GetStatus() int32 {
	return int32(this.table.Status)
}

func (this *NiuNiuTable) ParseTableOptConfig(gameRuleCfg string) (rspCode int32, err error) {

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

func (this *NiuNiuTable) GetMaxRound() int32 {
	return this.gameRule.MaxRoundCount
}

func (this *NiuNiuTable) OnMessage(msg *mateProto.MessageMaTe) int32 {

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
	default:
		return this.table.OnMessage(msg)
	}
}

func (this *NiuNiuTable) GetBaseQPTable() *qpTable.QPTable {
	return this.table
}

func (this *NiuNiuTable) onPrivateJoinTable(msg *mateProto.MessageMaTe) int32 {

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

func (this *NiuNiuTable) onClubJoinTable(msg *mateProto.MessageMaTe) int32 {

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

func (this *NiuNiuTable) onPlayerLeave(pro *mateProto.MessageMaTe) int32 {

	if this.table.IsAssignTableState(qpTable.TS_WaitingPlayerEnter) == false {
		return mateProto.Err_TableStatusNotMatch
	}
	return this.table.OnLeave(pro)
}

func (this *NiuNiuTable) onReady(pro *mateProto.MessageMaTe) int32 {
	funRes := this.GetBaseQPTable().OnMessage(pro)
	if funRes != mateProto.Err_Success {
		return funRes
	}

	var readyCount, lookerCount int32
	this.playingSeatArr = make([]int32, 0, 8)
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

			this.table.GameTimer.PutTableTimer(timerCheckUnreadyPlayer, 5*1000, this.timerCheckUnreadyPlayer)
			//this.GetBaseQPTable().BroadCastGameEvent(ID_StartCountdownClock, nil)
		}
	} else {
		if readyCount == this.table.GetCurSeatCount()-lookerCount {
			return this.onGameStart(pro)
		}
	}

	return funRes
}

func (this *NiuNiuTable) onGameStart(pro *mateProto.MessageMaTe) int32 {

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

	this.table.AppendTableState(TS_XiaZhu)
	this.table.BroadcastTableStatus()
	this.stageTime = time.Now().Unix()

	this.table.BroadCastGameEvent(ID_NoticeXiaZhu, &SC_NoticeXiaZhu{this.playingSeatArr})

	this.table.TableRule.TuoGuanTime = 10
	if this.table.TableRule.TuoGuanTime > 0 {
		this.table.GameTimer.PutTableTimer(timerAutoXiaZhu, this.table.TableRule.TuoGuanTime*1000, this.autoXiaZhu)
	}

	return mateProto.Err_Success
}

func (this *NiuNiuTable) onXiaZHu(pro *mateProto.MessageMaTe) int32 {
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
	seatData := seat.(*NiuNiuSeat)
	if seatData.seatData.IsAssignSeatState(qpTable.SS_Playing) == false {
		return mateProto.Err_ActionNotMatchStatus
	}

	ok := false
	for _, v := range this.gameRule.XiaZhuArr[this.gameRule.XiaZhuOpt] {
		if v == operXiaZhu.Value {
			ok = true
			break
		}
	}
	if ok == false {
		return mateProto.Err_OperationParamErr
	}

	nnSeat := seat.(*NiuNiuSeat)
	nnSeat.xiaZhu = operXiaZhu.Value

	this.table.BroadCastGameEvent(ID_BroadcastXiaZhu,
		SC_XiaZhu{SeatNumber: int32(seatData.seatData.Number), XiaZhu: operXiaZhu.Value})

	// 是否所有人都 下注了
	var playingCount, xiaZhuShuCount int32
	for _, v := range this.table.SeatArr {
		if v == nil {
			continue
		}
		if v.GetSeatData().IsAssignSeatState(qpTable.SS_Playing) == false {
			continue
		}

		playingCount += 1

		if v.(*NiuNiuSeat).xiaZhu > 0 {
			xiaZhuShuCount += 1
		}
	}

	// 所有人下注后，开始比牌
	if xiaZhuShuCount == playingCount {
		this.FaOnePai()
	}

	return 0
}

func (this *NiuNiuTable) autoXiaZhu() {

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

		if v.(*NiuNiuSeat).xiaZhu > 0 {
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

func (this *NiuNiuTable) autoLiang() {

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
		if v.(*NiuNiuSeat).isLiang == true {
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

func (this *NiuNiuTable) FaPai() {

	this.PaiMgr.XiPai(this.table.GetCurSeatCount()-this.table.LookerCount, 5)

	for i, v := range this.table.SeatArr {
		if v == nil {
			continue
		}
		if v.GetSeatData().IsAssignSeatState(qpTable.SS_Playing) == false {
			this.table.SendToSeat(qpTable.SeatNumber(i),
				ID_FaShouPai,
				&SC_FaShouPai{SeatNumber: int32(i), Pai: []int8{InvalidPai, InvalidPai, InvalidPai, InvalidPai}, PlayingSeat: this.playingSeatArr})
			continue
		}

		paiArr := this.PaiMgr.GetGroupPai(int32(v.GetSeatData().Number), 5)

		seat := v.(*NiuNiuSeat)
		seat.SetShouPai(paiArr)

		this.table.SendGameEventToSeat(qpTable.SeatNumber(i),
			ID_FaShouPai,
			&SC_FaShouPai{SeatNumber: int32(i), Pai: paiArr[:4], PlayingSeat: this.playingSeatArr})
	}
}

func (this *NiuNiuTable) FaOnePai() {

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
			&SC_FaShouPai{SeatNumber: int32(i), Pai: v.(*NiuNiuSeat).shouPai[4:5]})
	}

	if this.table.TableRule.TuoGuanTime > 0 {
		this.table.GameTimer.PutTableTimer(timerAutoBiPai, this.table.TableRule.TuoGuanTime*1000, this.autoLiang)
	}
}

func (this *NiuNiuTable) onPlayerLiangPai(pro *mateProto.MessageMaTe) int32 {
	if this.table.IsAssignTableState(TS_BiPai) == false {
		return mateProto.Err_ActionNotMatchStatus
	}

	seat := this.table.GetSeatDataByPlayerID(qpTable.PlayerID(pro.SenderID))
	if seat == nil {
		return mateProto.Err_NotFindPlayer
	}
	nnSeat := seat.(*NiuNiuSeat)
	if nnSeat.isLiang == true {
		return mateProto.Err_OperationRepeat
	}
	if nnSeat.seatData.IsAssignSeatState(qpTable.SS_Playing) == false {
		return mateProto.Err_ActionNotMatchStatus
	}

	var arrangePaiArr []int8
	nnSeat.isLiang = true
	nnSeat.paiXing, arrangePaiArr, nnSeat.maxPai = this.logic.GetPaiXing(nnSeat.shouPai)

	this.table.BroadCastGameEvent(ID_BroadcastLiangPai,
		&SC_BroadcastLiangPai{LiangPaiXing{
			SeatNumber: int32(nnSeat.seatData.Number),
			PaiArr:     arrangePaiArr,
			PaiXing:    nnSeat.paiXing,
			LastPai:    nnSeat.shouPai[4]}})

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
		if v.(*NiuNiuSeat).isLiang == true {
			liangPaiCount += 1
		}
	}

	if liangPaiCount == playingCount {
		this.BroadcastPaiXing()
	}

	return 0
}

func (this *NiuNiuTable) BroadcastPaiXing() {

	this.table.GameTimer.RemoveByTimeID(timerAutoBiPai)

	tempLiangPaiXing := make([]LiangPaiXing, 0, 8)
	for _, v := range this.GetBaseQPTable().SeatArr {
		if v == nil {
			continue
		}
		niuNiuSeat := v.(*NiuNiuSeat)
		if len(niuNiuSeat.shouPai) < 5 {
			continue
		}
		if niuNiuSeat.isLiang == true {
			continue
		}
		var arrangePaiArr []int8
		niuNiuSeat.paiXing, arrangePaiArr, niuNiuSeat.maxPai = this.logic.GetPaiXing(niuNiuSeat.shouPai)

		tempLiangPaiXing = append(tempLiangPaiXing,
			LiangPaiXing{int32(v.GetSeatData().Number), arrangePaiArr, niuNiuSeat.paiXing, niuNiuSeat.shouPai[4]})

		if niuNiuSeat.paiXing > niuNiuSeat.maxPaiXing {
			niuNiuSeat.maxPaiXing = niuNiuSeat.paiXing
		}
	}
	if len(tempLiangPaiXing) > 0 {
		this.table.BroadCastGameEvent(ID_BroadcastLiangPaiXing, &SS_LiangPaiXing{tempLiangPaiXing})
	}

	this.RoundOverFun()
}

func (this *NiuNiuTable) RoundOverFun() {

	this.table.SetTableState(qpTable.TS_Playing)

	baseTable := this.GetBaseQPTable()

	var winnerNNSeat *NiuNiuSeat
	for i := 0; i < len(baseTable.SeatArr); i++ {
		if baseTable.SeatArr[i] == nil {
			continue
		}
		playerSeatA := baseTable.SeatArr[i].(*NiuNiuSeat)
		if len(playerSeatA.shouPai) < 5 {
			continue
		}

		winnerNNSeat = playerSeatA

		for j := i + 1; j < len(baseTable.SeatArr); j++ {
			if baseTable.SeatArr[j] == nil {
				continue
			}
			playerSeatB := baseTable.SeatArr[j].(*NiuNiuSeat)
			if len(playerSeatB.shouPai) < 5 {
				continue
			}

			if this.logic.Compare(winnerNNSeat.paiXing, winnerNNSeat.maxPai, playerSeatB.paiXing, playerSeatB.maxPai) == false {
				winnerNNSeat = playerSeatB
			}
		}
		break
	}
	if winnerNNSeat == nil {
		glog.Warning("not find winner... tabID:=", this.table.TableNum)
		return
	}
	winnerNNSeat.winCount += 1
	cWinnerScore := winnerNNSeat.seatData.ClubScore + winnerNNSeat.seatData.SeatScore + winnerNNSeat.seatData.RoundScore

	paiXingBeiShu := float64(1)
	if this.gameRule.IsSuper {
		if v, ok := superBeiShuMap[winnerNNSeat.paiXing]; ok == true {
			paiXingBeiShu = v
		}
	} else {
		if v, ok := normalBeiShuMap[winnerNNSeat.paiXing]; ok == true {
			paiXingBeiShu = v
		}
	}

	// 算分
	for i := 0; i < len(baseTable.SeatArr); i++ {
		if baseTable.SeatArr[i] == nil {
			continue
		}
		playerSeat := baseTable.SeatArr[i].(*NiuNiuSeat)
		if len(playerSeat.shouPai) < 5 {
			continue
		}
		if i == int(winnerNNSeat.seatData.Number) {
			continue
		}

		tempScore := float64(winnerNNSeat.xiaZhu) * paiXingBeiShu * float64(playerSeat.xiaZhu) * this.gameRule.MultipleFloat64

		// 赢家分数不够
		if cWinnerScore < tempScore {
			tempScore = cWinnerScore
		}
		// 输家数是否足够
		actualScore := playerSeat.seatData.RoundScore + playerSeat.seatData.SeatScore + playerSeat.seatData.ClubScore
		if actualScore < tempScore {
			if actualScore > 0 {
				tempScore = actualScore
			} else {
				tempScore = 0
			}
		}

		playerSeat.seatData.RoundScore -= tempScore
		playerSeat.seatData.SeatScore -= tempScore

		winnerNNSeat.seatData.RoundScore += tempScore
		winnerNNSeat.seatData.SeatScore += tempScore
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
	this.table.BroadcastTableStatus()

	if this.table.CurXRound >= this.gameRule.MaxRoundCount {
		this.table.DissolveType = qpTable.DT_Playing
		this.handleDaJieSuan()
		return
	}

	this.CleanRoundData()
	this.table.TableRule.TimerAutoReady()
}

func (this *NiuNiuTable) handleXiaoJieSuan() []*protoGameBasic.PlayerGameScore {
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

	roundSeatScoreArr := make([]*RoundSeatScore, 0, 8)
	recPlayerGameScoreArr := make([]*protoGameBasic.PlayerGameScore, 0, 8)

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
			Pai:        v.(*NiuNiuSeat).shouPai,
			GameScore:  commonDef.Float64ToString(seat.RoundScore),
			SeatScore:  commonDef.Float64ToString(seat.SeatScore)})

		v.CleanRoundData()
	}
	roundOver.SeatData = roundSeatScoreArr

	this.GetBaseQPTable().BroadCastGameEvent(ID_RoundOver, &roundOver)

	return recPlayerGameScoreArr
}

// 大结算
func (this *NiuNiuTable) handleDaJieSuan() {
	if this.table.CurXRound < 1 {
		return
	}

	//msg := BroadcastGameOver{TableNumber: this.table.TableNum,
	//	CurRound:     this.table.CurXRound,
	//	MaxRound:     this.gameRule.MaxRoundCount,
	//	EndTime:      time.Now().Unix(),
	//	SeatData:     make([]*GameOverSeatData, 0, this.table.GetCurSeatCount()),
	//	DissolveType: this.table.DissolveType,
	//	ClubID:       this.table.MZClubID,
	//	ClubPlayID:   this.table.ClubPlayID}
	//msgGameOver := mateProto.MessageMaTe{MessageID: ID_BroadcastGameOver, MsgBody: &msg}

	//for _, v := range this.table.SeatArr {
	//	if v == nil {
	//		continue
	//	}
	//
	//	pdkSeat := v.(*NiuNiuMPQZSeat)
	//
	//	tempSeat := &GameOverSeatData{
	//		ClubID:       pdkSeat.seatData.ClubID,
	//		UID:          int64(pdkSeat.seatData.Player.ID),
	//		Nick:         pdkSeat.seatData.Player.Nick,
	//		Head:         pdkSeat.seatData.Player.Head,
	//		WinCount:     pdkSeat.winCount,
	//		MaxPaiXing:   pdkSeat.maxPaiXing,
	//		SeatScore:    commonDef.Float64ToString(pdkSeat.seatData.SeatScore),
	//		SeatScoreInt: commonDef.Float64Mul1000ToService(pdkSeat.seatData.SeatScore),
	//		IsMaxWin:     false}
	//
	//	msg.SeatData = append(msg.SeatData, tempSeat)
	//}
	//sort.Sort(msg.SeatData)
	//msg.SeatData[0].IsMaxWin = true
	//for i := 1; i < len(msg.SeatData); i++ {
	//	if msg.SeatData[i].SeatScoreInt < msg.SeatData[0].SeatScoreInt {
	//		break
	//	}
	//	if msg.SeatData[i].SeatScoreInt == msg.SeatData[0].SeatScoreInt {
	//		msg.SeatData[i].IsMaxWin = true
	//	}
	//}
	//
	//this.table.SendToAllPlayer(ID_BroadcastGameOver, &msg)

	this.GetBaseQPTable().SetTableState(qpTable.TS_Invalid)
}

func (this *NiuNiuTable) onTableExpire(pro *mateProto.MessageMaTe) int32 {

	this.table.DissolveType = qpTable.DT_LiveTimeout

	this.handleXiaoJieSuan()

	this.handleDaJieSuan()

	return this.table.OnMessage(pro)
}

func (this *NiuNiuTable) onTableData(pro *mateProto.MessageMaTe) int32 {

	seat := this.table.GetSeatDataByPlayerID(qpTable.PlayerID(pro.SenderID))
	if seat == nil {
		return mateProto.Err_NotFindPlayer
	}

	nnSeatArr := make([]*NiuNiuSeatData, 0)
	for _, v := range this.table.SeatArr {
		if v == nil {
			continue
		}

		pkNNSeat := v.(*NiuNiuSeat)
		tempNNSeat := NiuNiuSeatData{
			UID:           int64(pkNNSeat.seatData.Player.ID),
			Nick:          pkNNSeat.seatData.Player.Nick,
			HeadURL:       pkNNSeat.seatData.Player.Head,
			IP:            pkNNSeat.seatData.Player.IP,
			Sex:           pkNNSeat.seatData.Player.Sex,
			SeatNumber:    int32(pkNNSeat.seatData.Number),
			SeatStatus:    uint32(pkNNSeat.seatData.Status),
			ClubScore:     commonDef.Float64ToString(pkNNSeat.seatData.ClubScore),
			SeatScore:     commonDef.Float64ToString(pkNNSeat.seatData.SeatScore),
			RoundScore:    commonDef.Float64ToString(pkNNSeat.seatData.RoundScore),
			VoteStatus:    v.GetSeatData().DissolveVote,
			OperationTime: time.Now().Unix() - v.GetSeatData().OperationStart,
			PaiXing:       pkNNSeat.paiXing,
			XiaZhu:        pkNNSeat.xiaZhu,
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

	pdkSeat := seat.(*NiuNiuSeat)

	nowTT := time.Now().Unix()
	tableData := SC_TableData{
		MZCID:              this.table.MZClubID,
		TableNumber:        this.table.TableNum,
		TableStatus:        uint32(this.table.Status),
		RoundCount:         this.table.CurXRound,
		TableRuleText:      this.table.TableRule.TableCfgJson,
		ClubRuleText:       this.table.ClubRuleText,
		BankerSeatNum:      int32(qpTable.INVALID_SEAT_NUMBER),
		ClubID:             pdkSeat.seatData.ClubID,
		SeatData:           nnSeatArr,
		GameRuleText:       this.gameRule.RuleJson,
		ClubScore:          commonDef.Float64ToString(seat.GetSeatData().ClubScore),
		DissolveID:         int32(this.table.DissolveSeatNum),
		LaunchDissolveTime: nowTT - this.table.LaunchDissolveTime,
		FirstRoundReadTime: nowTT - this.table.FirstRoundReadTime,
		StageTime:          nowTT - this.stageTime,
	}

	if this.table.IsAssignTableState(TS_BiPai) == true {
		tableData.ShouPai = pdkSeat.shouPai
	} else if len(pdkSeat.shouPai) > 0 {
		tableData.ShouPai = pdkSeat.shouPai[:4]
	}

	this.table.SendToSeat(seat.GetSeatData().Number, ID_TableData, tableData)

	return mateProto.Err_Success
}

func (this *NiuNiuTable) onDissolveTableVote(pro *mateProto.MessageMaTe) int32 {
	rspCode := this.table.OnMessage(pro)

	if this.table.Status != qpTable.TS_Invalid {
		return rspCode
	}

	this.handleXiaoJieSuan()
	this.handleDaJieSuan()

	return rspCode
}

func (this *NiuNiuTable) timerCheckUnreadyPlayer() {
	var readyCount, lookerCount int32
	this.playingSeatArr = make([]int32, 0, 8)
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
			if readyCount < this.gameRule.FirstRoundReady {
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

func (this *NiuNiuTable) onCustomShouPai(pro *mateProto.MessageMaTe) int32 {

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
	nnSeat := seat.(*NiuNiuSeat)
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

func (this *NiuNiuTable) onGetRemainingPai(pro *mateProto.MessageMaTe) int32 {
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
