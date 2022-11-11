package pokerPDKTable

import (
	"encoding/json"
	"fmt"
	"github.com/golang/glog"
	"math/rand"
	commonDef "qpGame/commonDefine"
	"qpGame/commonDefine/mateProto"
	"qpGame/commonDefine/mateProto/protoGameBasic"
	"qpGame/db"
	pokerTable "qpGame/game/poker"
	"qpGame/qpTable"
	"qpGame/wrapMQ"
	"sort"
	"strconv"
	"time"
)

const timerAutoPlayPai = protoGameBasic.PaoDeKuai

type PokerPDKTable struct {
	PaiMgr   *pokerTable.PokerPaiBaseMgr // 牌的管理器
	logic    paoDeKuaiLogic
	table    *qpTable.QPTable
	gameRule PDKRule

	lastPlaySeatNumber qpTable.SeatNumber // 上次出牌的座位号(关联下次出牌)

	// 小局 待清理 成员
	niaoPai           int8               // 鸟牌（随机\红桃10）
	niaoPaiSeatNumber qpTable.SeatNumber // 鸟牌 座位号
	curPlaySeatNumber qpTable.SeatNumber // 当前出牌座位号
	bankerSeatNum     qpTable.SeatNumber // 本轮最先出牌的座位号
}

// 清空每一小局数据
func (this *PokerPDKTable) CleanRoundData() {
	this.table.CleanRoundData()

	this.niaoPai = pokerTable.InvalidPai
	this.niaoPaiSeatNumber = qpTable.INVALID_SEAT_NUMBER
	this.curPlaySeatNumber = qpTable.INVALID_SEAT_NUMBER
}

func (this *PokerPDKTable) SetTableNumber(tabNumber int32) {
	this.table.TableNum = tabNumber
}

func (this *PokerPDKTable) GetStatus() int32 {
	return int32(this.table.Status)
}

func (this *PokerPDKTable) ParseTableOptConfig(gameRuleCfg string) (rspCode int32, err error) {

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

func (this *PokerPDKTable) GetMaxRound() int32 {
	return this.gameRule.MaxRoundCount
}

func (this *PokerPDKTable) OnMessage(msg *mateProto.MessageMaTe) int32 {

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
	case protoGameBasic.ID_CancelTrusteeship:
		return this.onCancelTrusteeship(msg)
	case CS_TableData:
		return this.onTableData(msg)
	case ID_Play:
		return this.OnPlayerPlayPai(msg)
	case protoGameBasic.ID_ActiveTrusteeship:
		return this.OnActiveTrusteeship(msg)
	case protoGameBasic.ID_DissolveTableVote:
		return this.onDissolveTableVote(msg)
	case protoGameBasic.ID_ForceDissolveTable:
		return this.onForceDissolveTable(msg)
	case ID_CustomShouPai:
		return this.onCustomShouPai(msg)
	default:
		return this.table.OnMessage(msg)
	}
}

func (this *PokerPDKTable) GetBaseQPTable() *qpTable.QPTable {
	return this.table
}

func (this *PokerPDKTable) onPrivateJoinTable(msg *mateProto.MessageMaTe) int32 {

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
		this.table.SeatArr[rspCode].(*PokerPDKSeat).paiLogic.rule = &this.gameRule
		seatData := this.table.SeatArr[rspCode].GetSeatData()
		seatData.Player.IP = joinTable.IP
		seatData.Lat, seatData.Lng = joinTable.Latitude, joinTable.Longitude
	}

	// 还原成 原始ID
	msg.MessageID = protoGameBasic.ID_PrivateJoinGameTable

	return rspCode
}

func (this *PokerPDKTable) onClubJoinTable(msg *mateProto.MessageMaTe) int32 {

	if this.table.Status > qpTable.TS_WaitingPlayerEnter {
		return mateProto.Err_GameStarted
	}
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
		//seatData.MutexMap = playerMutex

		this.table.SeatArr[rspCode].(*PokerPDKSeat).paiLogic.rule = &this.gameRule
	}

	return rspCode
}

func (this *PokerPDKTable) onPlayerLeave(pro *mateProto.MessageMaTe) int32 {

	if this.table.IsAssignTableState(qpTable.TS_WaitingPlayerEnter) == false {
		return mateProto.Err_TableStatusNotMatch
	}
	return this.table.OnLeave(pro)
}

func (this *PokerPDKTable) onReady(pro *mateProto.MessageMaTe) int32 {
	funRes := this.GetBaseQPTable().OnMessage(pro)
	if funRes != mateProto.Err_Success {
		return funRes
	}

	var readyCount, lookerCount int32
	for _, v := range this.table.SeatArr {
		if v == nil {
			continue
		}
		if v.GetSeatData().IsAssignSeatState(qpTable.SS_Ready) == true {
			readyCount += 1
		}
		if v.GetSeatData().IsAssignSeatState(qpTable.SS_Looker) == true {
			lookerCount += 1
		}
	}

	// 所有人准备后，游戏开始
	if readyCount == this.table.MaxPlayers-lookerCount {
		return this.onGameStart(pro)
	}
	return funRes
}

func (this *PokerPDKTable) OnPlayerPlayPai(pro *mateProto.MessageMaTe) int32 {

	if this.table.IsAssignTableState(qpTable.TS_Playing) == false {
		return mateProto.Err_ActionNotMatchStatus
	}

	operPlayPai := CS_PlayPai{}
	err := json.Unmarshal(pro.Data, &operPlayPai)
	if err != nil {
		return mateProto.Err_ProtocolDataErr
	}

	seat, errCode := this.checkOperation(pro.SenderID, operPlayPai.OperationID)
	if errCode != mateProto.Err_Success {
		return errCode
	}

	pdkSeat := seat.(*PokerPDKSeat)

	for _, v := range operPlayPai.ChuPai {
		if pdkSeat.GetPaiCount(v) < 1 {
			return mateProto.Err_PaiNotExist
		}
	}

	// 如果下家只剩下 单张,上家出单牌必须是最大的
	if len(operPlayPai.ChuPai) == 1 {
		nextSeat := this.table.GetNextPlayingSeat(pdkSeat.seatData.Number)
		if nextSeat.(*PokerPDKSeat).shouPaiCount == 1 {
			for k, v := range pdkSeat.shouPai {
				if v < 1 {
					continue
				}
				if (k & 0x0F) > (operPlayPai.ChuPai[0] & 0x0F) {
					return mateProto.Err_PaiXingError
				}
			}
		}
	}

	// 牌型 检查
	if len(operPlayPai.ChuPai) > 0 {
		assignPaiXing := int32(PDK_PX_Invalid)
		if this.lastPlaySeatNumber != pdkSeat.seatData.Number {
			lastPlaySeat := this.table.SeatArr[this.lastPlaySeatNumber].(*PokerPDKSeat)
			assignPaiXing = lastPlaySeat.paiLogic.PaiXing
		}

		if pdkSeat.paiLogic.ParsePaiXing(pdkSeat.shouPaiCount, operPlayPai.ChuPai, assignPaiXing) == false {
			return mateProto.Err_PaiXingError
		}
	} else {
		pdkSeat.paiLogic.CleanStatus()
	}

	// 是否比上次出牌者 大
	if this.lastPlaySeatNumber != pdkSeat.seatData.Number {
		lastPlaySeat := this.table.SeatArr[this.lastPlaySeatNumber].(*PokerPDKSeat)
		compareRes := pdkSeat.paiLogic.IsGreaterX(&lastPlaySeat.paiLogic)
		if compareRes < 0 {
			return mateProto.Err_PaiXingYaoBuQi
		}
		if compareRes == 1 {
			if this.gameRule.IsShaoDaiTouPao == false {
				return mateProto.Err_PaiXingYaoBuQi
			}
			if pdkSeat.shouPaiCount != int32(len(operPlayPai.ChuPai)) {
				return mateProto.Err_PaiXingYaoBuQi
			}
		}
	} else {
		// 别人都要不起时
	}
	// 广播出牌
	broadPlayerPlayMsg := MsgBroadcastPlayerPai{
		SeatNum:   int32(pdkSeat.seatData.Number),
		Operation: operPlayPai.Operation,
		ChuPai:    operPlayPai.ChuPai,
		PaiXing:   pdkSeat.paiLogic.PaiXing,
		MinValue:  pdkSeat.paiLogic.PaiXingStartDianShu}
	this.table.BroadCastGameEvent(B_PlayerPlay, &broadPlayerPlayMsg)

	for _, v := range operPlayPai.ChuPai {
		pdkSeat.DeleteShouPai(v)
	}

	// 日志
	{
		//chuPaiTest := fmt.Sprintf("seatNo:=%d play Pai:= ", seat.GetSeatData().Number)
		//for i, _ := range operPlayPai.ChuPai {
		//	chuPaiTest += pokerTable.PaiValueToString(operPlayPai.ChuPai[i])
		//	if i < len(operPlayPai.ChuPai)-1 {
		//		chuPaiTest += ","
		//	}
		//}
		//commonDef.LOG_Info(chuPaiTest)
	}

	// 移走 自动出牌定时器
	this.table.GameTimer.RemoveByTimeID(timerAutoPlayPai)
	// 记录步骤
	this.table.OperateRec.PutPlayerStep(int32(pdkSeat.seatData.Number), ID_Play, &operPlayPai)

	// 记录出牌者
	this.lastPlaySeatNumber = pdkSeat.seatData.Number

	pdkSeat.PlayPai = operPlayPai.ChuPai
	pdkSeat.seatData.CleanOperationID()
	pdkSeat.playPaiCount += 1

	// 炸弹分
	bombScoreFunc := func() {
		bombScore := 10 * this.gameRule.MultipleFloat64
		this.ChangeBombScore(pdkSeat.seatData.Number, bombScore)
	}

	// 游戏结束
	if pdkSeat.shouPaiCount < 1 {
		if pdkSeat.paiLogic.PaiXing == PDK_PX_ZhaDan {
			bombScoreFunc()
		}
		this.RoundOverFun()
		return 0
	}

	// 查找下一个出牌者(管的上的)
	nextSeatNumber := pdkSeat.seatData.Number
	for i := int32(0); i < this.gameRule.MaxPlayer; i++ {
		tempSeat := this.table.GetNextPlayingSeat(nextSeatNumber)

		if pdkSeat.seatData.Number == tempSeat.GetSeatData().Number {
			// 都要不起,下一个还是自己出牌
			nextSeatNumber = pdkSeat.seatData.Number
			break
		}

		if this.findGreaterPai(tempSeat.(*PokerPDKSeat)) == true {
			//tempTestArr := make([]int8, len(tempSeat.(*PokerPDKSeat).autoPlayPaiArr))
			//for i, v := range tempSeat.(*PokerPDKSeat).autoPlayPaiArr {
			//	tempTestArr[i] = v & 0x0F
			//}
			//commonDef.LOG_Info("预计出牌：=", tempTestArr)

			nextSeatNumber = tempSeat.(*PokerPDKSeat).seatData.Number
			break
		}
		nextSeatNumber = tempSeat.GetSeatData().Number

		this.NoticeOperation(tempSeat.GetSeatData().Number, PKOperation_YaoBuQi)
		tempSeat.GetSeatData().CleanOperationID()

		time.Sleep(time.Millisecond * 800)

		broadPlayerPlayMsg := MsgBroadcastPlayerPai{
			SeatNum:   int32(tempSeat.GetSeatData().Number),
			Operation: PKOperation_YaoBuQi,
			ChuPai:    nil}
		this.table.BroadCastGameEvent(B_PlayerPlay, &broadPlayerPlayMsg)
	}

	// 最后一炸才得分
	if nextSeatNumber == pdkSeat.seatData.Number && pdkSeat.paiLogic.PaiXing == PDK_PX_ZhaDan {
		bombScoreFunc()
	}

	this.NoticeOperation(nextSeatNumber, PKOperation_PlayPai)

	// 检查是否 能 一次性 出完
	{
		tempLogic := paoDeKuaiLogic{rule: &this.gameRule}
		nextSeat := this.table.SeatArr[nextSeatNumber].(*PokerPDKSeat)
		tempPlayPaiArr := nextSeat.GetAllPai()

		// 是否 符合牌型
		if tempLogic.ParseYiShouChu(nextSeat.shouPaiCount, tempPlayPaiArr) == false {
			this.timerAutoTrusteeship()
			return 0
		}

		if nextSeatNumber == pdkSeat.seatData.Number {

		} else {
			compareRes := tempLogic.IsGreaterX(&pdkSeat.paiLogic)
			if compareRes < 0 {
				this.timerAutoTrusteeship()
				return 0
			}
			if compareRes == 1 {
				if this.gameRule.IsShaoDaiTouPao == false {
					this.timerAutoTrusteeship()
					return 0
				}
			}
		}

		time.Sleep(time.Millisecond * 800)

		// 自动 打出最后 一手牌, 模拟消息
		// 更换 发送消息的玩家ID
		pro.SenderID = int64(nextSeat.GetSeatData().Player.ID)
		operPlayPai.Operation = PKOperation_PlayPai
		operPlayPai.OperationID = this.table.SeatArr[nextSeatNumber].GetSeatData().OperationID
		operPlayPai.ChuPai = tempPlayPaiArr
		pro.Data, _ = json.Marshal(&operPlayPai)
		rspCode := this.OnPlayerPlayPai(pro)
		if rspCode != 0 {
			glog.Warning("end play pai code:=", rspCode)
		}
		return rspCode
	}

	return mateProto.Err_Success
}

func (this *PokerPDKTable) onGameStart(pro *mateProto.MessageMaTe) int32 {
	if this.table.IsAssignTableState(qpTable.TS_WaitingPlayerEnter) == false &&
		this.table.IsAssignTableState(qpTable.TS_WaitingReady) == false {
		return mateProto.Err_TableStatusNotMatch
	}

	errNumber := this.table.OnGameStart(pro)
	if errNumber != mateProto.Err_Success {
		return errNumber
	}

	this.table.CurXRound += 1
	this.table.OperateRec.SetTableInfo(this.table.TableNum, this.table.CurXRound, this.gameRule.RuleJson, this.table.TableRule.TableCfgJson)

	this.PaiMgr.XiPai(this.table.GetCurSeatCount()-this.table.LookerCount, this.gameRule.ShouPaiCount)

	switch this.gameRule.ZhuaNiaoMul {
	case 0:
		this.niaoPai = pokerTable.InvalidPai
	case 1:
		this.niaoPai = pokerTable.HongTao | 0x0A
	case 2:
		this.niaoPai = this.PaiMgr.RandomPai()
	}

	luckPai := pokerTable.InvalidPai
	heiTao3SeatNumber := qpTable.SeatNumber(0)
	tempPlayingSeatArr := make([]int, 0, len(this.table.SeatArr))

	// 发手牌
	for i, v := range this.table.SeatArr {
		if v == nil {
			continue
		}
		seat := v.GetSeatData()
		this.table.OperateRec.PutPlayer(seat)

		if seat.IsAssignSeatState(qpTable.SS_Looker) == true {
			this.table.SendGameEventToSeat(qpTable.SeatNumber(i),
				SC_FaShouPai,
				&MsgFaShouPai{SeatNumber: int32(i)})
			continue
		}
		if seat.IsAssignSeatState(qpTable.SS_Playing) == false {
			continue
		}
		tempPlayingSeatArr = append(tempPlayingSeatArr, i)

		paiArr := this.PaiMgr.GetGroupPai(int32(seat.Number), this.gameRule.ShouPaiCount,
			func(pai int8) {
				if pai == (pokerTable.HeiTao | 0x03) {
					heiTao3SeatNumber = seat.Number
				}
				if pai == this.niaoPai {
					this.niaoPaiSeatNumber = seat.Number
				}
			})

		pdkSeat := v.(*PokerPDKSeat)

		for _, v := range paiArr {
			pdkSeat.PushShouPai(v)
		}

		this.table.SendGameEventToSeat(qpTable.SeatNumber(i),
			SC_FaShouPai,
			&MsgFaShouPai{SeatNumber: int32(i), Pai: paiArr})
	}

	getLuckPaiFunc := func() int32 {
		seatNumber := rand.Intn(len(tempPlayingSeatArr))
		if seatNumber < 0 {
			seatNumber = tempPlayingSeatArr[0]
		}
		for k, _ := range this.table.SeatArr[seatNumber].(*PokerPDKSeat).shouPai {
			luckPai = k
			break
		}
		return int32(seatNumber)
	}

	this.table.GameTimer.PutTableTimer(timerAutoPlayPai, 1700, func() {
		// 首局出牌方式 幸运牌(0)\黑桃3(1)
		if this.table.CurXRound == 1 {
			if this.gameRule.FirstRoundChuPai == 0 {
				this.lastPlaySeatNumber = qpTable.SeatNumber(getLuckPaiFunc())
			} else if this.gameRule.FirstRoundChuPai == 1 {
				this.lastPlaySeatNumber = heiTao3SeatNumber
			}
		}
		if this.table.SeatArr[this.lastPlaySeatNumber].GetSeatData().IsAssignSeatState(qpTable.SS_Looker) == true {
			chuPaiSeat := this.table.GetNextPlayingSeat(this.lastPlaySeatNumber)
			if chuPaiSeat != nil {
				this.lastPlaySeatNumber = chuPaiSeat.GetSeatData().Number
			}
		}

		//if luckPai != pokerTable.InvalidPai || this.niaoPai != pokerTable.InvalidPai {
		//	time.Sleep(time.Second * 2)
		//}
		this.table.BroadCastGameEvent(ID_NoticeLuckPai, &BroadcastLuckPai{LuckPai: luckPai, NiaoPai: this.niaoPai})

		this.bankerSeatNum = this.lastPlaySeatNumber
		this.NoticeOperation(this.lastPlaySeatNumber, PKOperation_PlayPai)
		this.timerAutoTrusteeship()
	})

	return mateProto.Err_Success
}

func (this *PokerPDKTable) RoundOverFun() {

	// 算分
	winNiaoBei := int64(1)
	if this.niaoPaiSeatNumber == this.lastPlaySeatNumber {
		winNiaoBei = 2
	}
	winSeatData := this.table.SeatArr[this.lastPlaySeatNumber].(*PokerPDKSeat)

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
		curSeatNumber := seat.Number

		// 赢家 不参与计算
		if this.lastPlaySeatNumber == curSeatNumber {
			v.(*PokerPDKSeat).winCount += 1
			continue
		}
		v.(*PokerPDKSeat).loseCount += 1

		var tempScore float64
		// 手牌数 > 1
		if v.(*PokerPDKSeat).shouPaiCount > 1 {
			tempScore = float64(v.(*PokerPDKSeat).shouPaiCount)
			// 赢家中鸟
			tempScore *= float64(winNiaoBei)
			// 自己中鸟
			if this.niaoPaiSeatNumber == curSeatNumber {
				tempScore *= 2
			}
			// 春天
			if this.bankerSeatNum == this.lastPlaySeatNumber && v.(*PokerPDKSeat).playPaiCount < 1 {
				v.(*PokerPDKSeat).chunTianCount += 1
				v.(*PokerPDKSeat).isChunTian = true
				tempScore *= 2
			}
			// 反春
			if curSeatNumber == this.bankerSeatNum && v.(*PokerPDKSeat).playPaiCount < 2 {
				v.(*PokerPDKSeat).isFanChun = true
				tempScore *= 2
			}
			tempScore = tempScore * this.gameRule.MultipleFloat64
		}

		//commonDef.LOG_Info("uid:=", v.GetSeatData().Player.ID, ", paiScore:=", commonDef.ScoreToClient(tempScore), ",bombScore:=", commonDef.ScoreToClient(v.GetSeatData().RoundScore))

		v.(*PokerPDKSeat).TotalBombScore += v.(*PokerPDKSeat).bombScore
		v.(*PokerPDKSeat).TotalPaiScore -= tempScore
		v.(*PokerPDKSeat).PaiScore = -tempScore
		// 输家 -分
		seat.RoundScore -= tempScore
		seat.SeatScore += seat.RoundScore

		winSeatData.TotalPaiScore += tempScore
		winSeatData.PaiScore += tempScore
		// 赢家 +分
		winSeatData.GetSeatData().RoundScore += tempScore
	}
	winSeatData.GetSeatData().SeatScore += winSeatData.GetSeatData().RoundScore
	winSeatData.TotalBombScore += winSeatData.bombScore

	roundOver := BroadcastRoundOver{
		TableNumber: this.GetBaseQPTable().TableNum,
		Timestamp:   time.Now().Unix(),
	}
	msgRoundOver := mateProto.MessageMaTe{MessageID: ID_RoundOver, MsgBody: &roundOver}

	roundSeatScoreArr := make([]*RoundSeatScore, 0, 3)
	recPlayerGameScoreArr := make([]*protoGameBasic.PlayerGameScore, 0, 3)

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

		isNiaoPai := false
		if seat.Number == this.niaoPaiSeatNumber {
			isNiaoPai = true
		}

		roundSeatScoreArr = append(roundSeatScoreArr, &RoundSeatScore{
			UID:           int64(seat.Player.ID),
			NickName:      seat.Player.Nick,
			Head:          seat.Player.Head,
			Pai:           v.(*PokerPDKSeat).GetAllPai(),
			BombScore:     commonDef.Float64ToString(v.(*PokerPDKSeat).bombScore),
			GameScore:     commonDef.Float64ToString(seat.RoundScore),
			SeatScore:     commonDef.Float64ToString(seat.SeatScore),
			IsNiaoPai:     isNiaoPai,
			IsChunTian:    v.(*PokerPDKSeat).isChunTian,
			IsFanChun:     v.(*PokerPDKSeat).isFanChun,
			GameScoreStep: seat.GameScoreRecStep,
			RecChuPai:     v.(*PokerPDKSeat).recChuPai})

		//v.CleanRoundData()
	}
	roundOver.SurplusPaiArr = this.PaiMgr.GetSurplusPai()
	roundOver.SeatData = roundSeatScoreArr

	this.table.BroadCastGameEvent(ID_RoundOver, &roundOver)
	gameStepRec, _ := this.table.OperateRec.Pack()

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

	if this.table.CurXRound >= this.gameRule.MaxRoundCount {
		this.table.DissolveType = qpTable.DT_Playing
		this.handleDaJieSuan()
		return
	}

	this.CleanRoundData()
	this.table.TableRule.TimerAutoReady()
}

func (this *PokerPDKTable) handleXiaoJieSuan() {
	if this.table.CurXRound < 1 {
		return
	}

	roundOver := BroadcastRoundOver{
		TableNumber: this.GetBaseQPTable().TableNum,
		Timestamp:   time.Now().Unix(),
	}

	roundSeatScoreArr := make([]*RoundSeatScore, 0, 3)

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

		isNiaoPai := false
		if seat.Number == this.niaoPaiSeatNumber {
			isNiaoPai = true
		}

		roundSeatScoreArr = append(roundSeatScoreArr, &RoundSeatScore{
			UID:           int64(seat.Player.ID),
			NickName:      seat.Player.Nick,
			Head:          seat.Player.Head,
			Pai:           v.(*PokerPDKSeat).GetAllPai(),
			BombScore:     commonDef.Float64ToString(v.(*PokerPDKSeat).bombScore),
			GameScore:     commonDef.Float64ToString(seat.RoundScore),
			SeatScore:     commonDef.Float64ToString(seat.SeatScore),
			IsNiaoPai:     isNiaoPai,
			IsChunTian:    v.(*PokerPDKSeat).isChunTian,
			IsFanChun:     v.(*PokerPDKSeat).isFanChun,
			GameScoreStep: seat.GameScoreRecStep,
			RecChuPai:     v.(*PokerPDKSeat).recChuPai,
			PaiXinScore:   commonDef.Float64ToString(v.(*PokerPDKSeat).PaiScore)})
	}

	roundOver.SurplusPaiArr = this.PaiMgr.GetSurplusPai()
	roundOver.SeatData = roundSeatScoreArr

	this.GetBaseQPTable().BroadCastGameEvent(ID_RoundOver, &roundOver)
}

// 大结算
func (this *PokerPDKTable) handleDaJieSuan() {
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

		pdkSeat := v.(*PokerPDKSeat)

		tempSeat := &GameOverSeatData{
			ClubID:        pdkSeat.seatData.ClubID,
			UID:           int64(pdkSeat.seatData.Player.ID),
			Nick:          pdkSeat.seatData.Player.Nick,
			Head:          pdkSeat.seatData.Player.Head,
			ChunTianCount: pdkSeat.chunTianCount,
			BombCount:     pdkSeat.bombCount,
			WinCount:      pdkSeat.winCount,
			LoseCount:     pdkSeat.loseCount,
			SeatScore:     commonDef.Float64ToString(pdkSeat.seatData.SeatScore),
			SeatScoreInt:  commonDef.Float64Mul1000ToService(pdkSeat.seatData.SeatScore),
			IsMaxWin:      false,
			BombScore:     commonDef.Float64ToString(pdkSeat.TotalBombScore),
			PaiXinScore:   commonDef.Float64ToString(pdkSeat.TotalPaiScore)}

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

func (this *PokerPDKTable) onTableExpire(pro *mateProto.MessageMaTe) int32 {

	this.table.DissolveType = qpTable.DT_LiveTimeout

	this.handleXiaoJieSuan()

	this.handleDaJieSuan()

	return this.table.OnMessage(pro)
}

func (this *PokerPDKTable) onTableData(pro *mateProto.MessageMaTe) int32 {

	seat := this.table.GetSeatDataByPlayerID(qpTable.PlayerID(pro.SenderID))
	if seat == nil {
		return mateProto.Err_NotFindPlayer
	}

	pdkSeatArr := make([]*PdkSeatData, 0)
	for _, v := range this.table.SeatArr {
		if v == nil {
			continue
		}

		pkPDKSeat := v.GetXSeatData(0).(*PokerPDKSeat)
		tempPDKSeat := PdkSeatData{
			UID:               int64(pkPDKSeat.seatData.Player.ID),
			Nick:              pkPDKSeat.seatData.Player.Nick,
			HeadURL:           pkPDKSeat.seatData.Player.Head,
			IP:                pkPDKSeat.seatData.Player.IP,
			Sex:               pkPDKSeat.seatData.Player.Sex,
			SeatNumber:        int32(pkPDKSeat.seatData.Number),
			SeatStatus:        uint32(pkPDKSeat.seatData.Status),
			ClubID:            pkPDKSeat.seatData.ClubID,
			ClubScore:         commonDef.Float64ToString(pkPDKSeat.seatData.ClubScore),
			SeatScore:         commonDef.Float64ToString(pkPDKSeat.seatData.SeatScore),
			RoundScore:        commonDef.Float64ToString(pkPDKSeat.seatData.RoundScore),
			ShouPaiCount:      255,
			CurPlayCard:       pkPDKSeat.PlayPai,
			LastOperationItem: int32(pkPDKSeat.CurOperationItem),
			VoteStatus:        v.GetSeatData().DissolveVote,
			OperationTime:     time.Now().Unix() - v.GetSeatData().OperationStart,
			PaiXing:           pkPDKSeat.paiLogic.PaiXing,
			MinValue:          pkPDKSeat.paiLogic.PaiXingStartDianShu,
		}
		if pkPDKSeat.shouPaiCount <= 1 {
			tempPDKSeat.ShouPaiCount = pkPDKSeat.shouPaiCount
		}
		if this.gameRule.IsShowPaiShu == true {
			tempPDKSeat.ShouPaiCount = pkPDKSeat.shouPaiCount
		}
		if pkPDKSeat.seatData.Lng > 0.1 && pkPDKSeat.seatData.Lat > 0.1 {
			tempPDKSeat.IsGPS = true
		}

		pdkSeatArr = append(pdkSeatArr, &tempPDKSeat)
	}

	pdkSeat := seat.GetXSeatData(0).(*PokerPDKSeat)
	selfShouPai := make([]int8, 0)

	for k, v := range pdkSeat.shouPai {
		for i := 0; i < v; i++ {
			selfShouPai = append(selfShouPai, k)
		}
	}

	tableData := MsgTableData{
		MZCID:              this.table.MZClubID,
		TableNumber:        this.table.TableNum,
		TableStatus:        uint32(this.table.Status),
		RoundCount:         this.table.CurXRound,
		TableRuleText:      this.table.TableRule.TableCfgJson,
		ClubRuleText:       this.table.ClubRuleText,
		CurPlaySeatNumber:  int32(this.curPlaySeatNumber),
		SurplusPai:         this.PaiMgr.GetSurplusPai(),
		NiaoPai:            this.niaoPai,
		SeatData:           pdkSeatArr,
		ShouPai:            selfShouPai,
		OperationID:        seat.GetSeatData().OperationID,
		OperationItem:      int32(pdkSeat.CurOperationItem),
		GameRuleText:       this.gameRule.RuleJson,
		ClubScore:          commonDef.Float64ToString(seat.GetSeatData().ClubScore),
		DissolveID:         int32(this.table.DissolveSeatNum),
		LaunchDissolveTime: time.Now().Unix() - this.table.LaunchDissolveTime,
	}

	this.table.UpdatePlayerSource(pdkSeat.seatData, pro.Source)

	this.table.SendToSeat(seat.GetSeatData().Number, CS_TableData, tableData)

	return mateProto.Err_Success
}

func (this *PokerPDKTable) NoticeOperation(seatNumber qpTable.SeatNumber, item OperationItem) {
	pdkSeat := this.table.SeatArr[seatNumber].(*PokerPDKSeat)
	pdkSeat.SetOperationItem(item)

	// 通知操作
	this.table.SendGameEventToSeat(seatNumber,
		SC_NoticeOperation,
		&MsgNoticeOperation{SeatNumber: int32(seatNumber),
			OperationID:   pdkSeat.GetSeatData().GetOperationID(),
			OperationItem: item})

	// 广播 轮到谁出牌
	broadcastPlayPai := MsgBroadcastOperation{SeatNumber: int32(seatNumber)}
	this.table.BroadCastGameEvent(B_CurOperationSeatNumber, &broadcastPlayPai)

	// 当前出牌座位号
	this.curPlaySeatNumber = seatNumber
}

// 座位游戏分变化
func (this *PokerPDKTable) ChangeBombScore(win qpTable.SeatNumber, score float64) {

	rec := qpTable.GameScoreRec{}

	tempScore := float64(0)
	msgBody := protoGameBasic.BroadcastPlayerScoreChanged{
		WinnerSeatNum: int32(win),
		LoserSeatNum:  make([]int32, 0, 4),
		Score:         commonDef.Float64ToString(score)}

	giveSeatArr := []qpTable.SeatNumber{}
	for i, v := range this.table.SeatArr {
		if v == nil {
			continue
		}
		if v.GetSeatData().IsAssignSeatState(qpTable.SS_Playing) == false {
			continue
		}
		if qpTable.SeatNumber(i) == win {
			this.table.SeatArr[win].(*PokerPDKSeat).bombCount += 1
			continue
		}
		giveSeatArr = append(giveSeatArr, qpTable.SeatNumber(i))

		rec.Category, rec.Score = ZhaDan, -score
		rec.TargetUID = []qpTable.SeatNumber{win}
		v.GetSeatData().PutGameScoreItem(&rec, 1)
		v.GetSeatData().RoundScore -= score
		v.(*PokerPDKSeat).bombScore -= score

		tempScore += score
		this.table.SeatArr[win].GetSeatData().RoundScore += score
		this.table.SeatArr[win].(*PokerPDKSeat).bombScore += score

		msgBody.LoserSeatNum = append(msgBody.LoserSeatNum, int32(v.GetSeatData().Number))
	}

	rec.Category, rec.Score = ZhaDan, tempScore
	rec.TargetUID = giveSeatArr
	this.table.SeatArr[win].GetSeatData().PutGameScoreItem(&rec, 1)

	this.table.BroadCastGameEvent(protoGameBasic.ID_PlayerRoundScoreChanged, &msgBody)
}

// 检查操作
func (this *PokerPDKTable) checkOperation(playerID int64, operationID string) (qpTable.QPSeat, int32) {
	seat := this.table.GetSeatDataByPlayerID(qpTable.PlayerID(playerID))
	if seat == nil {
		return nil, mateProto.Err_NotFindPlayer
	}
	if len(seat.GetSeatData().OperationID) < 1 {
		return nil, mateProto.Err_OperationIDErr
	}

	if seat.GetSeatData().OperationID != operationID {
		return nil, mateProto.Err_OperationIDErr
	}

	if seat.GetSeatData().Number != this.curPlaySeatNumber {
		return nil, mateProto.Err_NotYouOperation
	}

	return seat, mateProto.Err_Success
}

func (this *PokerPDKTable) timerAutoTrusteeship() {

	tempTime := this.table.TableRule.TuoGuanTime * 1000

	if this.table.TableRule.TuoGuanTime < 1 {
		// 是否 主动开启了 托管
		if this.table.SeatArr[this.curPlaySeatNumber].GetSeatData().IsAssignSeatState(qpTable.SS_Trusteeship) == false {
			return
		}
		tempTime = 800
	} else {
		if this.table.SeatArr[this.curPlaySeatNumber].GetSeatData().IsAssignSeatState(qpTable.SS_Trusteeship) == true {
			tempTime = 800
		}
	}

	this.table.GetBaseQPTable().GameTimer.PutSeatTimer(
		int32(this.curPlaySeatNumber),
		timerAutoPlayPai,
		tempTime, this.autoTrusteeshipPlayPai)
}

func (this *PokerPDKTable) findGreaterPai(curPDKPlayPaiSeat *PokerPDKSeat) bool {
	lastPDKPlayPaiSeat := this.table.SeatArr[this.lastPlaySeatNumber].(*PokerPDKSeat)

	var expectPlayPaiArr []int8

	defer func() {
		if expectPlayPaiArr != nil {
			curPDKPlayPaiSeat.autoPlayPaiArr = expectPlayPaiArr
		}
	}()

	// 牌的数量
	const PaiCount = 4
	const UseIndex = 5
	minPaiDianShu := pokerTable.MaxDianShu
	maxPaiDianShu := pokerTable.MinDianShu
	shouPaiDianShuArr := [pokerTable.MaxDianShu + 1][PaiCount + 2]int8{}
	for k, v := range curPDKPlayPaiSeat.shouPai {
		if v < 1 {
			continue
		}
		tempDianShu := k & 0x0F
		if tempDianShu < minPaiDianShu {
			minPaiDianShu = tempDianShu
		}
		if tempDianShu > maxPaiDianShu {
			maxPaiDianShu = tempDianShu
		}
		paiCount := shouPaiDianShuArr[tempDianShu][PaiCount]
		shouPaiDianShuArr[tempDianShu][paiCount] = k
		shouPaiDianShuArr[tempDianShu][PaiCount] += 1
	}

	findBombFunc := func(min int8) bool {
		if this.gameRule.Is3ABomb == true && shouPaiDianShuArr[pokerTable.ADianShu][PaiCount] == 3 {
			expectPlayPaiArr = make([]int8, 3)
			for j := int8(0); j < shouPaiDianShuArr[pokerTable.ADianShu][PaiCount]; j++ {
				expectPlayPaiArr[j] = shouPaiDianShuArr[pokerTable.ADianShu][j]
			}
			shouPaiDianShuArr[pokerTable.ADianShu][PaiCount] = 0
			return true
		}
		for i := min; i <= maxPaiDianShu; i++ {
			if shouPaiDianShuArr[i][PaiCount] < 4 {
				continue
			}
			expectPlayPaiArr = make([]int8, 4)
			for j := int8(0); j < shouPaiDianShuArr[i][PaiCount]; j++ {
				expectPlayPaiArr[j] = shouPaiDianShuArr[i][j]
			}
			shouPaiDianShuArr[i][PaiCount] = 0
			return true
		}
		return false
	}

	switch lastPDKPlayPaiSeat.paiLogic.PaiXing {
	case PDK_PX_ZhaDan:
		return findBombFunc(lastPDKPlayPaiSeat.paiLogic.PaiXingStartDianShu + 1)
	case PDK_PX_SiDaiEr:
		return findBombFunc(pokerTable.MinDianShu)
	case PDK_PX_SiDaiSan:
		return findBombFunc(pokerTable.MinDianShu)
	case PDK_PX_FeiJi:
		if findBombFunc(pokerTable.MinDianShu) == true {
			return true
		}

		for i := lastPDKPlayPaiSeat.paiLogic.PaiXingStartDianShu + 1; i <= maxPaiDianShu && len(expectPlayPaiArr) < 1; i++ {
			if shouPaiDianShuArr[i][PaiCount] < 3 {
				continue
			}

			var tempCC int8 = 1
			for j := i + 1; j <= maxPaiDianShu; j++ {
				if shouPaiDianShuArr[j][PaiCount] < 3 {
					break
				}
				tempCC += 1
				if tempCC < int8(lastPDKPlayPaiSeat.paiLogic.SequenceCount) {
					continue
				}
				expectPlayPaiArr = make([]int8, 0, 20)

				for ; i <= j; i++ {
					for shouPaiDianShuArr[i][UseIndex] < 3 {
						l := shouPaiDianShuArr[i][UseIndex]
						expectPlayPaiArr = append(expectPlayPaiArr, shouPaiDianShuArr[i][l])
						shouPaiDianShuArr[i][UseIndex] += 1
					}
				}
				break
			}
		}
		for i := minPaiDianShu; i <= maxPaiDianShu && len(expectPlayPaiArr) > 0; i++ {
			for shouPaiDianShuArr[i][UseIndex] < shouPaiDianShuArr[i][PaiCount] {
				l := shouPaiDianShuArr[i][UseIndex]
				expectPlayPaiArr = append(expectPlayPaiArr, shouPaiDianShuArr[i][l])
				shouPaiDianShuArr[i][UseIndex] += 1
				if len(expectPlayPaiArr) >= len(lastPDKPlayPaiSeat.PlayPai) {
					return true
				}
			}
		}

		if this.gameRule.IsShaoDaiTouPao == true && len(expectPlayPaiArr) >= 6 {
			return true
		}
	case PDK_PX_LianDui:
		if findBombFunc(pokerTable.MinDianShu) == true {
			return true
		}
		for i := lastPDKPlayPaiSeat.paiLogic.PaiXingStartDianShu + 1; i <= maxPaiDianShu && len(expectPlayPaiArr) < 1; i++ {
			if shouPaiDianShuArr[i][PaiCount] < 2 {
				continue
			}

			var tempCC int8
			for j := i + 1; j <= maxPaiDianShu; j++ {
				if shouPaiDianShuArr[j][PaiCount] < 2 {
					break
				}
				tempCC += 1
				if tempCC < int8(lastPDKPlayPaiSeat.paiLogic.SequenceCount) {
					continue
				}
				expectPlayPaiArr = make([]int8, 0, 16)

				for ; i <= j; i++ {
					for shouPaiDianShuArr[i][UseIndex] < 2 {
						l := shouPaiDianShuArr[i][UseIndex]
						expectPlayPaiArr = append(expectPlayPaiArr, shouPaiDianShuArr[i][l])
						shouPaiDianShuArr[i][UseIndex] += 1
					}
				}
				return true
			}
		}
	case PDK_PX_ShunZi:
		if findBombFunc(pokerTable.MinDianShu) == true {
			return true
		}
		if maxPaiDianShu == pokerTable.MaxDianShu {
			maxPaiDianShu = pokerTable.ADianShu
		}
		for i := lastPDKPlayPaiSeat.paiLogic.PaiXingStartDianShu + 1; i <= maxPaiDianShu && len(expectPlayPaiArr) < 1; i++ {
			if shouPaiDianShuArr[i][PaiCount] < 1 {
				continue
			}

			var tempCC int8
			for j := i + 1; j <= maxPaiDianShu; j++ {
				if shouPaiDianShuArr[j][PaiCount] < 1 {
					break
				}
				tempCC += 1
				if tempCC < int8(lastPDKPlayPaiSeat.paiLogic.SequenceCount) {
					continue
				}
				expectPlayPaiArr = make([]int8, 0, 16)

				for ; i <= j; i++ {
					for shouPaiDianShuArr[i][UseIndex] < 1 {
						l := shouPaiDianShuArr[i][UseIndex]
						expectPlayPaiArr = append(expectPlayPaiArr, shouPaiDianShuArr[i][l])
						shouPaiDianShuArr[i][UseIndex] += 1
					}
				}
				return true
			}
		}
	case PDK_PX_SanDai_Er:
		if findBombFunc(pokerTable.MinDianShu) == true {
			return true
		}
		for i := lastPDKPlayPaiSeat.paiLogic.PaiXingStartDianShu + 1; i <= maxPaiDianShu; i++ {
			if shouPaiDianShuArr[i][PaiCount] < 3 {
				continue
			}

			expectPlayPaiArr = make([]int8, 0, 5)
			for j := int8(0); j < 3; j++ {
				expectPlayPaiArr = append(expectPlayPaiArr, shouPaiDianShuArr[i][j])
				shouPaiDianShuArr[i][UseIndex] += 1
			}
			break
		}
		for i := minPaiDianShu; i <= maxPaiDianShu && len(expectPlayPaiArr) > 0; i++ {
			for shouPaiDianShuArr[i][UseIndex] < shouPaiDianShuArr[i][PaiCount] {
				l := shouPaiDianShuArr[i][UseIndex]
				expectPlayPaiArr = append(expectPlayPaiArr, shouPaiDianShuArr[i][l])
				shouPaiDianShuArr[i][UseIndex] += 1
				if len(expectPlayPaiArr) >= len(lastPDKPlayPaiSeat.PlayPai) {
					return true
				}
			}
		}
		if this.gameRule.IsShaoDaiTouPao == true &&
			len(expectPlayPaiArr) >= 3 && len(expectPlayPaiArr) < 5 {
			return true
		}
	case PDK_PX_SanDai_Yi:
		if findBombFunc(pokerTable.MinDianShu) == true {
			return true
		}
		for i := lastPDKPlayPaiSeat.paiLogic.PaiXingStartDianShu + 1; i <= maxPaiDianShu; i++ {
			if shouPaiDianShuArr[i][PaiCount] < 3 {
				continue
			}

			expectPlayPaiArr = make([]int8, 0, 4)
			for j := int8(0); j < 3; j++ {
				expectPlayPaiArr = append(expectPlayPaiArr, shouPaiDianShuArr[i][j])
				shouPaiDianShuArr[i][UseIndex] += 1
			}
			break
		}
		for i := minPaiDianShu; i <= maxPaiDianShu && len(expectPlayPaiArr) > 0; i++ {
			for shouPaiDianShuArr[i][UseIndex] < shouPaiDianShuArr[i][PaiCount] {
				l := shouPaiDianShuArr[i][UseIndex]
				expectPlayPaiArr = append(expectPlayPaiArr, shouPaiDianShuArr[i][l])
				shouPaiDianShuArr[i][UseIndex] += 1
				if len(expectPlayPaiArr) >= len(lastPDKPlayPaiSeat.PlayPai) {
					return true
				}
			}
		}
		if this.gameRule.IsShaoDaiTouPao == true &&
			len(expectPlayPaiArr) >= 3 && len(expectPlayPaiArr) < 4 {
			return true
		}
	case PDK_PX_YiDui:
		if findBombFunc(pokerTable.MinDianShu) == true {
			return true
		}
		for i := lastPDKPlayPaiSeat.paiLogic.PaiXingStartDianShu + 1; i <= maxPaiDianShu; i++ {
			if shouPaiDianShuArr[i][PaiCount] < 2 {
				continue
			}

			expectPlayPaiArr = make([]int8, 0, 2)
			for j := int8(0); j < 2; j++ {
				expectPlayPaiArr = append(expectPlayPaiArr, shouPaiDianShuArr[i][j])
			}
			return true
		}
	case PDK_PX_DandZhang:
		for i := maxPaiDianShu; i > lastPDKPlayPaiSeat.paiLogic.PaiXingStartDianShu; i-- {
			if shouPaiDianShuArr[i][PaiCount] < 1 {
				continue
			}
			expectPlayPaiArr = make([]int8, 1)
			expectPlayPaiArr[0] = shouPaiDianShuArr[i][0]
			return true
		}
		if findBombFunc(pokerTable.MinDianShu) == true {
			return true
		}
	}

	return false
}

func (this *PokerPDKTable) autoTrusteeshipPlayPai() {
	curPlayPaiSeat := this.table.SeatArr[this.curPlaySeatNumber]

	if curPlayPaiSeat.GetSeatData().IsAssignSeatState(qpTable.SS_Trusteeship) == false {
		curPlayPaiSeat.GetSeatData().AppendState(qpTable.SS_Trusteeship)
		this.table.NotifyPlayerStateChange(this.curPlaySeatNumber)
	}

	var tempPlayPaiArr []int8

	tempAutoPlayPaiFun := func() int32 {

		// 日志
		//commonDef.LOG_Info("autoTrusteeshipPlayPai() seatNo:=", this.curPlaySeatNumber)

		if len(tempPlayPaiArr) < 1 {
			glog.Warning("autoTrusteeshipPlayPai() not find pai.tableNumber:=", this.table.TableNum)
			return 0
		}
		pro := mateProto.MessageMaTe{}
		pro.SenderID = int64(curPlayPaiSeat.GetSeatData().Player.ID)
		pro.MessageID = ID_Play
		operPlayPai := CS_PlayPai{}
		operPlayPai.Operation = PKOperation_PlayPai
		operPlayPai.OperationID = curPlayPaiSeat.GetSeatData().OperationID
		operPlayPai.ChuPai = tempPlayPaiArr
		pro.Data, _ = json.Marshal(&operPlayPai)
		return this.OnPlayerPlayPai(&pro)
	}

	curPDKPlayPaiSeat := curPlayPaiSeat.(*PokerPDKSeat)
	if this.curPlaySeatNumber == this.lastPlaySeatNumber {
		tempPlayPaiArr = make([]int8, 1)

		// 下一个人的手牌数是否 == 1
		nextPlaySeat := this.table.GetNextPlayingSeat(this.curPlaySeatNumber)
		if nextPlaySeat.(*PokerPDKSeat).shouPaiCount == 1 {
			maxSinglePai := pokerTable.MinDianShu
			for k, v := range curPDKPlayPaiSeat.shouPai {
				if v < 1 {
					continue
				}
				if (k & 0x0F) > (maxSinglePai & 0x0F) {
					maxSinglePai = k
				}
			}
			tempPlayPaiArr[0] = maxSinglePai
		} else {
			minSinglePai := pokerTable.MaxDianShu
			for k, v := range curPDKPlayPaiSeat.shouPai {
				if v < 1 {
					continue
				}
				if (k & 0x0F) < (minSinglePai & 0x0F) {
					minSinglePai = k
				}
			}
			tempPlayPaiArr[0] = minSinglePai
		}
	} else {
		tempPlayPaiArr = curPlayPaiSeat.(*PokerPDKSeat).autoPlayPaiArr
	}

	rsp := tempAutoPlayPaiFun()
	if rsp != 0 {
		glog.Warning("tempAutoPlayPaiFun() error.tableNumber:=", this.table.TableNum, ",err:=", rsp)
	}
}

func (this *PokerPDKTable) onCancelTrusteeship(msg *mateProto.MessageMaTe) int32 {
	seatNumber := this.table.OnCancelTrusteeship(msg)
	if seatNumber < 0 {
		return seatNumber
	}

	timerArr := this.table.GameTimer.RemoveBySeatNum(seatNumber)
	for _, v := range timerArr {
		if v.TimerID == timerAutoPlayPai && this.table.TableRule.TuoGuanTime > 0 {
			this.table.GetBaseQPTable().GameTimer.PutSeatTimer(
				seatNumber,
				timerAutoPlayPai,
				this.table.TableRule.TuoGuanTime*1000, v.DoFunc)
		} else {
			glog.Warning("onCancelTrusteeship() timerID:=", v.TimerID)
		}
	}
	return 0
}

func (this *PokerPDKTable) onCustomShouPai(pro *mateProto.MessageMaTe) int32 {
	msgCustomShouPai := CS_CustomShouPai{}
	err := json.Unmarshal(pro.Data, &msgCustomShouPai)
	if err != nil {
		return mateProto.Err_ProtocolDataErr
	}

	if len(msgCustomShouPai.ShouPai) > int(this.gameRule.ShouPaiCount) {
		return mateProto.Err_CustomPai
	}

	seat := this.table.GetSeatDataByPlayerID(qpTable.PlayerID(pro.SenderID))
	if seat == nil {
		return mateProto.Err_NotFindPlayer
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

func (this *PokerPDKTable) onDissolveTableVote(pro *mateProto.MessageMaTe) int32 {
	rspCode := this.table.OnMessage(pro)

	if this.table.Status != qpTable.TS_Invalid {
		return rspCode
	}

	this.handleXiaoJieSuan()
	this.handleDaJieSuan()

	return rspCode
}

func (this *PokerPDKTable) onForceDissolveTable(pro *mateProto.MessageMaTe) int32 {
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

// 激活托管
func (this *PokerPDKTable) OnActiveTrusteeship(msg *mateProto.MessageMaTe) int32 {
	seatNum := this.table.OnActiveTrusteeship(msg)
	if seatNum < 0 {
		return seatNum
	}

	if this.curPlaySeatNumber != qpTable.SeatNumber(seatNum) {
		return seatNum
	}

	this.timerAutoTrusteeship()

	return seatNum
}
