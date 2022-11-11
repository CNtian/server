package mjXZDDTable

import (
	"encoding/json"
	"github.com/golang/glog"
	commonDef "qpGame/commonDefine"
	"qpGame/commonDefine/mateProto"
	"qpGame/commonDefine/mateProto/protoGameBasic"
	"qpGame/game/gameMaJiang"
	"qpGame/qpTable"
	"qpGame/wrapMQ"
	"sort"
	"time"
)

const (
	TIMER_ChanePai     = protoGameBasic.KaWuXing
	TIMER_DingQue      = protoGameBasic.KaWuXing + 1
	TIMER_PlayPai      = protoGameBasic.KaWuXing + 2
	TIMER_DelayDingQue = protoGameBasic.KaWuXing + 3
)

const TS_ChangePai qpTable.TableStatus = 32   // 自定义状态起始值 换牌
const TS_DingQue qpTable.TableStatus = 32 * 2 // 自定义状态起始值 定缺

type groupWinner struct {
	Flag    string
	SeatArr []qpTable.SeatNumber
}

type GameXZDDTable struct {
	table        *qpTable.QPTable
	playPaiLogic XueZhanDaoDiPlayLogic
	gameRule     XZDDPlayRule

	huOrder     int32
	firstHuSeat qpTable.SeatNumber
}

// 清空每一小局数据
func (this *GameXZDDTable) CleanRoundData() {
	this.firstHuSeat = qpTable.INVALID_SEAT_NUMBER
	this.huOrder = -1
	this.playPaiLogic.CleanRoundData()
	this.table.CleanRoundData()
}

func (this *GameXZDDTable) GetStatus() int32 {
	return int32(this.table.Status)
}

func (this *GameXZDDTable) ParseTableOptConfig(playCfg string) (rspCode int32, err error) {

	err = json.Unmarshal([]byte(playCfg), &this.gameRule)
	if err != nil {
		rspCode = mateProto.Err_CreateTableParam
		return
	}
	err = this.gameRule.CheckField()
	if err != nil {
		rspCode = mateProto.Err_CreateTableParam
		return
	}
	this.gameRule.RuleJson = playCfg
	return
}

func (this *GameXZDDTable) GetBaseQPTable() *qpTable.QPTable {
	return this.table
}

func (this *GameXZDDTable) GetMaxRound() int32 {
	return this.gameRule.MaxRoundCount
}

// 事件处理
func (this *GameXZDDTable) OnMessage(msg *mateProto.MessageMaTe) int32 {

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
	case ID_TableData:
		return this.onGetTableData(msg)
	case gameMaJiang.ID_Hu:
		return this.OnHu(msg)
	case protoGameBasic.ID_DissolveTableVote:
		return this.onDissolveTableVote(msg)
	default:
		return this.playPaiLogic.OnMessage(msg)
	}

	return mateProto.Err_Success
}

func (this *GameXZDDTable) onGameStart() int32 {
	rspCode := this.playPaiLogic.OnGameStart(nil)
	if rspCode != 0 {
		return rspCode
	}

	return 0
}

// 游戏结束
func (this *GameXZDDTable) RoundOver() {

	roundOver := BroadRoundGameOver{TableNumber: this.table.TableNum,
		CurRoundCount:      this.table.CurXRound,
		MaxRoundCount:      this.gameRule.MaxRoundCount,
		BankerSeatNumber:   int32(this.playPaiLogic.BankerSeatNum),
		CurPlayCard:        this.playPaiLogic.CurPlayPai,
		CurPlaySeatNumber:  int32(this.playPaiLogic.CurPlaySeatNum),
		CurMoPaiSeatNumber: int32(this.playPaiLogic.CurMoPaiSeatNum),
		RemainCardCount:    this.playPaiLogic.PaiMgr.GetTheRestOfPaiCount(),
		Timestamp:          time.Now().Unix()}
	msgRoundOver := mateProto.MessageMaTe{MessageID: ID_RoundGameOver, MsgBody: &roundOver}

	recPlayerGameScoreArr := make([]*protoGameBasic.PlayerGameScore, 0, this.table.GetCurSeatCount())

	// 游戏分
	{
		this.calculateScore()

		roundOver.SeatDataArr = make([]*HuPaiSeat, 0, 3)
		// 清算
		for _, v := range this.table.SeatArr {
			if v == nil {
				continue
			}
			if v.GetSeatData().IsAssignSeatState(qpTable.SS_Looker) == true {
				continue
			}
			v.GetSeatData().SeatScore += v.GetSeatData().RoundScore
		}
	}

	for _, v := range this.table.SeatArr {
		if v == nil {
			continue
		}
		tempSeat := v.GetSeatData()
		tempSeat.DelState(SS_HU)
		if tempSeat.IsAssignSeatState(qpTable.SS_Looker) == true {
			continue
		}
		if tempSeat.IsAssignSeatState(qpTable.SS_Playing) == false {
			continue
		}
		tempSeat.RoundOverMsg = &msgRoundOver

		temp := protoGameBasic.PlayerGameScore{
			UID:    int64(tempSeat.Player.ID),
			Nick:   tempSeat.Player.Nick,
			ClubID: tempSeat.ClubID,
			SScore: commonDef.Float64Mul1000ToService(tempSeat.RoundScore),
		}
		recPlayerGameScoreArr = append(recPlayerGameScoreArr, &temp)

		mjSeat := v.GetXSeatData(0).(*gameMaJiang.MJSeat)

		tempHuSeat := HuPaiSeat{
			UID:           int64(tempSeat.Player.ID),
			NickName:      tempSeat.Player.Nick,
			Head:          tempSeat.Player.Head,
			SeatNumber:    int32(tempSeat.Number),
			HuMode:        v.(*XZDDSeat).HuMode,
			ShouPai:       mjSeat.GetShouPai(),
			OperationPai:  mjSeat.OperationPai,
			RoundScore:    commonDef.Float64ToString(tempSeat.RoundScore),
			SeatScore:     commonDef.Float64ToString(tempSeat.SeatScore),
			GangScore:     commonDef.Float64ToString(v.(*XZDDSeat).GangScore),
			HuPai:         v.(*XZDDSeat).HuPai,
			HuOrder:       v.(*XZDDSeat).HuOrder,
			GameScoreStep: tempSeat.GameScoreRecStep,
		}

		if _, ok := this.playPaiLogic.huSeatMap[tempHuSeat.SeatNumber]; ok == true {
			tempHuSeat.HuPaiXing = mjSeat.HuPaiXing
		}

		roundOver.SeatDataArr = append(roundOver.SeatDataArr, &tempHuSeat)
	}

	this.table.BroadCastGameEvent(ID_RoundGameOver, &roundOver)

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

	// 确定 下次庄
	{
		if this.firstHuSeat != qpTable.INVALID_SEAT_NUMBER {
			this.playPaiLogic.BankerSeatNum = this.firstHuSeat
		}
		for k, v := range this.playPaiLogic.dianPaoSeatMap {
			if len(v) > 1 {
				this.playPaiLogic.BankerSeatNum = k
				break
			}
		}

		if this.table.SeatArr[this.playPaiLogic.BankerSeatNum].GetSeatData().IsAssignSeatState(qpTable.SS_Looker) == true {
			seat := this.table.GetNextValidSeat(this.playPaiLogic.BankerSeatNum)
			this.playPaiLogic.BankerSeatNum = seat.GetSeatData().Number
		}
	}

	this.CleanRoundData()

	this.table.TableRule.TimerAutoReady()
}

func (this *GameXZDDTable) handleXiaoJieSuan() {
	if this.table.CurXRound < 1 {
		return
	}

	roundOver := BroadRoundGameOver{TableNumber: this.table.TableNum,
		CurRoundCount:      this.table.CurXRound,
		MaxRoundCount:      this.gameRule.MaxRoundCount,
		BankerSeatNumber:   int32(this.playPaiLogic.BankerSeatNum),
		CurPlayCard:        this.playPaiLogic.CurPlayPai,
		CurPlaySeatNumber:  int32(this.playPaiLogic.CurPlaySeatNum),
		CurMoPaiSeatNumber: int32(this.playPaiLogic.CurMoPaiSeatNum),
		RemainCardCount:    this.playPaiLogic.PaiMgr.GetTheRestOfPaiCount(),
		SeatDataArr:        make([]*HuPaiSeat, 0, 3),
		Timestamp:          time.Now().Unix()}

	for _, v := range this.table.SeatArr {
		if v == nil {
			continue
		}
		tempSeat := v.GetSeatData()
		if tempSeat.IsAssignSeatState(qpTable.SS_Looker) == true {
			continue
		}
		if tempSeat.IsAssignSeatState(qpTable.SS_Playing) == false {
			continue
		}

		mjSeat := v.GetXSeatData(0).(*gameMaJiang.MJSeat)

		tempHuSeat := HuPaiSeat{
			UID:           int64(tempSeat.Player.ID),
			NickName:      tempSeat.Player.Nick,
			Head:          tempSeat.Player.Head,
			SeatNumber:    int32(tempSeat.Number),
			HuMode:        -1,
			ShouPai:       mjSeat.GetShouPai(),
			OperationPai:  mjSeat.OperationPai,
			RoundScore:    commonDef.Float64ToString(tempSeat.RoundScore),
			SeatScore:     commonDef.Float64ToString(tempSeat.SeatScore),
			GangScore:     commonDef.Float64ToString(v.(*XZDDSeat).GangScore),
			GameScoreStep: tempSeat.GameScoreRecStep,
		}

		roundOver.SeatDataArr = append(roundOver.SeatDataArr, &tempHuSeat)
	}

	this.table.BroadCastGameEvent(ID_RoundGameOver, &roundOver)
}

func (this *GameXZDDTable) handleDaJieSuan() {

	if this.table.CurXRound < 1 {
		return
	}

	broadcastGameover := BroadGameOverData{
		TableNumber:   this.table.TableNum,
		CurRoundCount: this.table.CurXRound,
		MaxRoundCount: this.gameRule.MaxRoundCount,
		Timestamp:     time.Now().Unix(),
		DissolveType:  this.table.DissolveType,
		ClubID:        this.table.MZClubID,
		ClubPlayID:    this.table.ClubPlayID,
	}
	msgGameOver := mateProto.MessageMaTe{MessageID: ID_RoundGameOver, MsgBody: &broadcastGameover}

	broadcastGameover.SeatData = make([]*GameOverSeatData, 0, this.table.GetCurSeatCount())
	for _, v := range this.table.SeatArr {
		if v == nil {
			continue
		}
		seat := v.GetSeatData()
		seat.GameOverMsg = &msgGameOver
		mjSeat := v.GetXSeatData(0).(*gameMaJiang.MJSeat)

		tempSeat := &GameOverSeatData{
			ClubID:       seat.ClubID,
			UID:          int64(seat.Player.ID),
			NickName:     seat.Player.Nick,
			Head:         seat.Player.Head,
			GangCount:    mjSeat.GangCount,
			DianPaoCount: mjSeat.DianPaoCount,
			JiePaoCount:  mjSeat.JiePaoCount,
			HuPaiCount:   mjSeat.HuPaiCount,
			SeatScore:    commonDef.Float64ToString(seat.SeatScore),
			SeatScoreInt: commonDef.Float64ScoreToInt64(seat.SeatScore),
			IsMaxWin:     false,
		}

		broadcastGameover.SeatData = append(broadcastGameover.SeatData, tempSeat)
	}

	sort.Sort(broadcastGameover.SeatData)
	broadcastGameover.SeatData[0].IsMaxWin = true
	for i := 1; i < len(broadcastGameover.SeatData); i++ {
		if broadcastGameover.SeatData[i].SeatScoreInt < broadcastGameover.SeatData[0].SeatScoreInt {
			break
		}
		if broadcastGameover.SeatData[i].SeatScoreInt == broadcastGameover.SeatData[0].SeatScoreInt {
			broadcastGameover.SeatData[i].IsMaxWin = true
		}
	}

	this.table.BroadCastGameEvent(ID_GameOver, &broadcastGameover)

	this.table.SetTableState(qpTable.TS_Invalid)
}

func (this *GameXZDDTable) onTableExpire(pro *mateProto.MessageMaTe) int32 {

	this.table.DissolveType = qpTable.DT_LiveTimeout

	this.handleXiaoJieSuan()
	this.handleDaJieSuan()

	return this.table.OnMessage(pro)
}

func (this *GameXZDDTable) onReady(pro *mateProto.MessageMaTe) int32 {
	funRes := this.table.OnMessage(pro)
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
		return this.onGameStart()
	}
	return funRes
}

func (this *GameXZDDTable) onPlayerLeave(pro *mateProto.MessageMaTe) int32 {

	if this.table.IsAssignTableState(qpTable.TS_WaitingPlayerEnter) == false {
		return mateProto.Err_TableStatusNotMatch
	}
	return this.table.OnLeave(pro)
}

func (this *GameXZDDTable) onPrivateJoinTable(msg *mateProto.MessageMaTe) int32 {

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

func (this *GameXZDDTable) onClubJoinTable(msg *mateProto.MessageMaTe) int32 {

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
	}

	return rspCode
}

func (this *GameXZDDTable) OnHu(pro *mateProto.MessageMaTe) int32 {
	playerID := qpTable.PlayerID(pro.SenderID)

	operationHu := gameMaJiang.CS_Hu{}
	err := json.Unmarshal(pro.Data, &operationHu)
	if err != nil {
		return mateProto.Err_ProtocolDataErr
	}

	seat, errCode := this.playPaiLogic.CheckOperation(playerID, operationHu.OperationID)
	if errCode != mateProto.Err_Success {
		return errCode
	}

	winnerSeat := seat.(*XZDDSeat)

	if (winnerSeat.MJSeat.OperationItem & gameMaJiang.OPI_HU) == 0 {
		return mateProto.Err_OperationNotExist
	}

	huPlayers := 0
	for _, v := range this.table.SeatArr {
		if v == nil {
			continue
		}
		if v.(*XZDDSeat).MJSeat.OperationItem&gameMaJiang.OPI_HU == gameMaJiang.OPI_HU {
			huPlayers += 1
		}
	}

	broadcastHu := SC_BroadcastHu{HuSeatNum: int32(winnerSeat.MJSeat.SeatData.Number), GangSeat: -1}

	if this.playPaiLogic.sRoundHuSeatMap == nil {
		this.playPaiLogic.sRoundHuSeatMap = make(map[int32]*gameMaJiang.MJSeat)
	}
	this.playPaiLogic.sRoundHuSeatMap[broadcastHu.HuSeatNum] = winnerSeat.MJSeat
	winnerSeat.MJSeat.SeatData.CleanOperationID()
	winnerSeat.MJSeat.OperationItem = 0
	winnerSeat.MJSeat.SeatData.AppendState(SS_HU)
	this.table.GameTimer.RemoveBySeatNum(int32(winnerSeat.MJSeat.SeatData.Number))
	winnerSeat.WinMap = make(map[int32]struct{})

	// 记录 点炮的座位，点了几家
	dianPaoSeatNum := qpTable.INVALID_SEAT_NUMBER
	if this.playPaiLogic.CurPlaySeatNum != qpTable.INVALID_SEAT_NUMBER { // 正常点炮
		playPaiSeat := this.table.SeatArr[this.playPaiLogic.CurPlaySeatNum].(*XZDDSeat)
		if len(playPaiSeat.MJSeat.PlayPai) > 0 &&
			this.playPaiLogic.CurPlayPai == playPaiSeat.MJSeat.PlayPai[len(playPaiSeat.MJSeat.PlayPai)-1] {
			playPaiSeat.MJSeat.PlayPai = playPaiSeat.MJSeat.PlayPai[:len(playPaiSeat.MJSeat.PlayPai)-1]
		}
		dianPaoSeatNum = this.playPaiLogic.CurPlaySeatNum
		broadcastHu.HuMode = 2
		broadcastHu.HuPai = this.playPaiLogic.CurPlayPai
	} else if this.playPaiLogic.buGangSeatNum != qpTable.INVALID_SEAT_NUMBER { // 抢杠胡
		dianPaoSeatNum = this.playPaiLogic.buGangSeatNum
		broadcastHu.HuMode = 3
		broadcastHu.HuPai = this.playPaiLogic.buGangPai
		broadcastHu.GangSeat = int32(this.playPaiLogic.buGangSeatNum)
	} else if this.gameRule.IsDGHZiMo == false &&
		this.playPaiLogic.lastGangSeatNum == winnerSeat.MJSeat.SeatData.Number &&
		winnerSeat.DGHPlayPaiSeat != qpTable.INVALID_SEAT_NUMBER {
		dianPaoSeatNum = winnerSeat.DGHPlayPaiSeat
		broadcastHu.HuMode = 4
		broadcastHu.HuPai = this.playPaiLogic.CurMoPai
	}

	if dianPaoSeatNum != qpTable.INVALID_SEAT_NUMBER {
		dpSeat := this.table.SeatArr[dianPaoSeatNum].GetSeatData()
		if this.playPaiLogic.dianPaoSeatMap == nil {
			this.playPaiLogic.dianPaoSeatMap = make(map[qpTable.SeatNumber][]*groupWinner)
		}
		groupWinnerArr, ok := this.playPaiLogic.dianPaoSeatMap[dianPaoSeatNum]
		if ok == false {
			this.huOrder += 1
			groupWinnerArr = []*groupWinner{{Flag: dpSeat.OperationIDBak, SeatArr: []qpTable.SeatNumber{}}}
		} else {
			if groupWinnerArr[len(groupWinnerArr)-1].Flag != dpSeat.OperationIDBak {
				this.huOrder += 1
				groupWinnerArr = append(groupWinnerArr, &groupWinner{Flag: dpSeat.OperationIDBak, SeatArr: []qpTable.SeatNumber{}})
			}
		}
		groupWinnerArr[len(groupWinnerArr)-1].SeatArr = append(groupWinnerArr[len(groupWinnerArr)-1].SeatArr, winnerSeat.MJSeat.SeatData.Number)

		this.playPaiLogic.dianPaoSeatMap[dianPaoSeatNum] = groupWinnerArr

		winnerSeat.WinMap[int32(dianPaoSeatNum)] = struct{}{}
	} else {
		this.huOrder += 1
		winnerSeat.isZiMo = true
		broadcastHu.HuPai = this.playPaiLogic.CurMoPai
		broadcastHu.HuMode = 1

		for _, v := range this.table.SeatArr {
			if v == nil {
				continue
			}
			xzddSeat := v.(*XZDDSeat)
			if xzddSeat.MJSeat.SeatData.IsContainSeatState(qpTable.SS_Looker|SS_HU) == true {
				continue
			}
			if xzddSeat.MJSeat.SeatData.Number == winnerSeat.MJSeat.SeatData.Number {
				continue
			}
			winnerSeat.WinMap[int32(v.GetSeatData().Number)] = struct{}{}
		}
	}

	broadcastHu.HuOrder = this.huOrder
	winnerSeat.HuOrder = this.huOrder
	winnerSeat.HuPai = broadcastHu.HuPai
	winnerSeat.HuMode = broadcastHu.HuMode

	noticeChangeScore := BroadcastHuNoticeChangeScore{}
	this.huCalculateScore(winnerSeat, &noticeChangeScore)

	if this.playPaiLogic.huSeatMap == nil {
		this.firstHuSeat = winnerSeat.MJSeat.SeatData.Number
		this.playPaiLogic.huSeatMap = make(map[int32]*gameMaJiang.MJSeat)
	}
	this.playPaiLogic.huSeatMap[int32(winnerSeat.MJSeat.SeatData.Number)] = winnerSeat.MJSeat

	// 多人胡牌,延迟发送
	if huPlayers > 1 {
		this.playPaiLogic.TempHuPlayerData.winSeat = winnerSeat.MJSeat.SeatData.Number
		this.playPaiLogic.TempHuPlayerData.dianPaoSeat = dianPaoSeatNum
		this.playPaiLogic.TempHuPlayerData.f = append(this.playPaiLogic.TempHuPlayerData.f, func() {
			this.table.BroadCastGameEvent(ID_BroadcastHu, &broadcastHu)
			this.table.BroadCastGameEvent(ID_HuNoticeChangeScore, &noticeChangeScore)
		})
		pro.MsgBody = &protoGameBasic.JsonResponse{Status: 0}
		this.table.SendMsgToSeat(winnerSeat.MJSeat.SeatData.Number, pro)
		return mateProto.Err_Success
	}

	for _, v := range this.playPaiLogic.TempHuPlayerData.f {
		v()
	}
	this.table.BroadCastGameEvent(ID_BroadcastHu, &broadcastHu)
	this.table.BroadCastGameEvent(ID_HuNoticeChangeScore, &noticeChangeScore)

	//if this.playPaiLogic.FindPriorityOperation(qpTable.INVALID_SEAT_NUMBER, gameMaJiang.OPI_HU) == true {
	//	return mateProto.Err_Success
	//}

	if dianPaoSeatNum != qpTable.INVALID_SEAT_NUMBER {
		this.playPaiLogic.findHuJiaoZhuanYi(dianPaoSeatNum)
	}

	// 自摸，点炮一人胡， 胡的下一个摸牌 ，  一炮多响  点炮人摸牌
	nextPlayer := this.playPaiLogic.GetNextPlayingSeat(winnerSeat.MJSeat.SeatData.Number).GetSeatData().Number
	if dianPaoSeatNum != qpTable.INVALID_SEAT_NUMBER {
		v, _ := this.playPaiLogic.dianPaoSeatMap[dianPaoSeatNum]
		if len(v[len(v)-1].SeatArr) > 1 {
			nextPlayer = dianPaoSeatNum
		}
	}

	this.playPaiLogic.huNextStep(nextPlayer)

	return mateProto.Err_Success
}

func (this *GameXZDDTable) onGetTableData(pro *mateProto.MessageMaTe) int32 {

	playerID := qpTable.PlayerID(pro.SenderID)
	seat := this.table.GetSeatDataByPlayerID(playerID)
	if seat == nil {
		return mateProto.Err_NotFindPlayer
	}

	nowTT := time.Now().Unix()

	tableData := SC_TableData{
		MZCID:               this.table.MZClubID,
		TableNumber:         this.table.GetTableNumber(),
		TableStatus:         int32(this.table.Status),
		BankerSeatNumber:    int32(this.playPaiLogic.BankerSeatNum),
		RoundCount:          this.table.CurXRound,
		TableRuleText:       this.table.TableRule.TableCfgJson,
		ClubRuleText:        this.table.ClubRuleText,
		CurPlayCard:         int32(this.playPaiLogic.CurPlayPai),
		CurPlaySeatNumber:   int32(this.playPaiLogic.CurPlaySeatNum),
		CurMoPaiSeatNumber:  int32(this.playPaiLogic.CurMoPaiSeatNum),
		CurPengSeatNumber:   int32(this.playPaiLogic.CurPengSeatNum),
		RemainCardCount:     this.playPaiLogic.PaiMgr.GetTheRestOfPaiCount(),
		ClubScore:           commonDef.Float64ToString(seat.GetSeatData().ClubScore),
		DissolveID:          int32(this.table.DissolveSeatNum),
		LaunchDissolveTime:  nowTT - this.table.LaunchDissolveTime,
		PlayerOperationTime: nowTT - this.playPaiLogic.OperationTime,
		FirstRoundReadTime:  nowTT - this.table.FirstRoundReadTime,
	}

	var selfSeat *XZDDSeat
	tableData.SeatData = make([]*MsgSeatData, 0)
	for _, v := range this.table.SeatArr {
		if v == nil {
			continue
		}

		xzddSeat := v.(*XZDDSeat)
		mjSeat := xzddSeat.MJSeat
		qpSeat := mjSeat.SeatData

		if int64(qpSeat.Player.ID) == pro.SenderID {
			selfSeat = xzddSeat
		}

		proSeat := &MsgSeatData{UID: int64(qpSeat.Player.ID),
			Nick:         qpSeat.Player.Nick,
			HeadURL:      qpSeat.Player.Head,
			Sex:          qpSeat.Player.Sex,
			IP:           qpSeat.Player.IP,
			SeatNumber:   int32(qpSeat.Number),
			SeatStatus:   uint32(qpSeat.Status),
			ClubID:       mjSeat.SeatData.ClubID,
			ClubScore:    commonDef.Float64ToString(qpSeat.ClubScore),
			SeatScore:    commonDef.Float64ToString(qpSeat.SeatScore),
			RoundScore:   commonDef.Float64ToString(qpSeat.RoundScore),
			ShouPaiCount: mjSeat.ShouPaiCount,
			ChuPai:       mjSeat.PlayPai,
			VoteStatus:   v.GetSeatData().DissolveVote,
			HuOrder:      xzddSeat.HuOrder,
			HuPai:        xzddSeat.HuPai,
			HuMode:       xzddSeat.HuMode,
			DingQue:      -1,
			//OperationTime: time.Now().Unix() - v.GetSeatData().OperationStart,
		}
		if len(xzddSeat.ChangePai) > 0 {
			proSeat.ChanedPai = true
		}
		if this.table.IsAssignTableState(TS_DingQue) == true {
			if xzddSeat.DingQue != -1 {
				proSeat.DingQue = 5
			}
		} else {
			proSeat.DingQue = xzddSeat.DingQue
		}

		proSeat.ReadyDingQue = xzddSeat.ReadyDingQue
		proSeat.ChangePai = xzddSeat.ReadyChangePai

		proSeat.OperationPai = make([]*gameMaJiang.OperationPaiInfo, 0)
		for _, v := range mjSeat.OperationPai {
			switch v.OperationPXItem {
			//case gameMaJiang.OPX_AN_GANG:
			//	tempOperPai := *v
			//	tempOperPai.PaiArr = nil
			//	proSeat.OperationPai = append(proSeat.OperationPai, &tempOperPai)
			default:
				proSeat.OperationPai = append(proSeat.OperationPai, v)
			}
		}

		if qpSeat.Lng > 0.1 && qpSeat.Lat > 0.1 {
			proSeat.IsGPS = true
		}

		tableData.SeatData = append(tableData.SeatData, proSeat)
	}

	mjSeat := selfSeat.MJSeat

	tableData.ChangePai = selfSeat.ChangePai
	tableData.CurMoPai = mjSeat.CurMoPai
	tableData.ShouPai = mjSeat.GetShouPai()
	tableData.OperationID = mjSeat.SeatData.OperationID
	tableData.OperationItem = uint32(mjSeat.OperationItem)
	tableData.GangArr = mjSeat.GangArr
	tableData.DingQue = selfSeat.DingQue

	//tableData.AnGangCard = make([]int8, 0)
	//for _, v := range mjSeat.OperationPai {
	//	if v.OperationPXItem == gameMaJiang.OPX_AN_GANG {
	//		tableData.AnGangCard = append(tableData.AnGangCard, v.PaiArr[0])
	//	}
	//}

	tableData.GameRuleText = this.gameRule.RuleJson

	this.table.UpdatePlayerSource(mjSeat.SeatData, pro.Source)

	this.table.SendToSeat(seat.GetSeatData().Number, ID_TableData, &tableData)

	return 0
}

func (this *GameXZDDTable) onCancelTrusteeship(msg *mateProto.MessageMaTe) int32 {
	seatNumber := this.table.OnCancelTrusteeship(msg)
	if seatNumber < 0 {
		return seatNumber
	}

	timerArr := this.table.GameTimer.RemoveBySeatNum(seatNumber)
	// 取消托管后,如果不出牌还的继续操作
	for _, v := range timerArr {
		if v.TimerID == TIMER_PlayPai && this.table.TableRule.TuoGuanTime > 0 {
			this.table.GetBaseQPTable().GameTimer.PutSeatTimer(
				seatNumber,
				TIMER_PlayPai,
				this.table.TableRule.TuoGuanTime*1000, v.DoFunc)
		} else {
			glog.Warning("onCancelTrusteeship() timerID:=", v.TimerID)
		}
	}
	return 0
}

func (this *GameXZDDTable) huCalculateScore(winSeat *XZDDSeat, noticeChangeScore *BroadcastHuNoticeChangeScore) {

	noticeChangeScore.WinnerSeatNum = int32(winSeat.MJSeat.SeatData.Number)

	huRec := qpTable.GameScoreRec{Category: gameMaJiang.ZiMo, BeiShu: 1}
	if winSeat.isZiMo == false {
		huRec.Category = gameMaJiang.DianPao
	}

	diFen := float64(1)
	if winSeat.isZiMo == true && this.gameRule.IsZiMoJiaDi == false {
		winSeat.HuFanShu += 1
		winSeat.MJSeat.HuPaiXing = append(winSeat.MJSeat.HuPaiXing, &gameMaJiang.HuPaiXing{HuPX_ZiMoJiaFan, 1})
		if winSeat.HuFanShu > this.gameRule.FengDingFanShu {
			winSeat.HuFanShu = this.gameRule.FengDingFanShu
		}
	}
	fanShuFen := int64(2)
	if winSeat.HuFanShu == 0 {
		fanShuFen = 1
	} else {
		for i := int64(1); i < winSeat.HuFanShu; i++ {
			fanShuFen *= 2
		}
	}

	huScore := diFen * float64(fanShuFen) * this.gameRule.MultipleFloat64
	if winSeat.isZiMo == true && this.gameRule.IsZiMoJiaDi == true {
		huScore += diFen
		winSeat.MJSeat.HuPaiXing = append(winSeat.MJSeat.HuPaiXing, &gameMaJiang.HuPaiXing{HuPX_ZiMoJiaDi, 1})
	}

	huRec.PaiXing = winSeat.MJSeat.HuPaiXing
	huRec.BeiShu = int(winSeat.HuFanShu)

	giveArr := make([]qpTable.SeatNumber, 0, len(winSeat.WinMap))
	for loseK, _ := range winSeat.WinMap {
		this.table.SeatArr[loseK].(*XZDDSeat).MJSeat.SeatData.RoundScore -= huScore

		giveArr = append(giveArr, qpTable.SeatNumber(loseK))
		huRecED := huRec
		huRecED.Score, huRecED.TargetUID = huScore, []qpTable.SeatNumber{winSeat.MJSeat.SeatData.Number}
		this.table.SeatArr[loseK].(*XZDDSeat).MJSeat.SeatData.PutGameScoreItem(&huRecED, -1)
		noticeChangeScore.LoserSeatArr = append(noticeChangeScore.LoserSeatArr, LoseSeat{loseK, commonDef.Float64ToString(huScore)})

		huRec.Score += huScore
	}

	winSeat.MJSeat.SeatData.RoundScore += huRec.Score
	huRec.TargetUID = giveArr
	winSeat.MJSeat.SeatData.PutGameScoreItem(&huRec, 1)

	noticeChangeScore.WinScore = commonDef.Float64ToString(huRec.Score)
}

func (this *GameXZDDTable) calculateScore() {

	//for k, _ := range this.playPaiLogic.huSeatMap {
	//winSeat := this.table.SeatArr[k].(*XZDDSeat)
	//
	//huRec := qpTable.GameScoreRec{Category: gameMaJiang.ZiMo,BeiShu:1}
	//if winSeat.isZiMo == false {
	//	huRec.Category = gameMaJiang.DianPao
	//}
	//
	//diFen := int64(1)
	//if this.gameRule.IsZiMoJiaDi == true {
	//	diFen += 1
	//} else {
	//	winSeat.HuFanShu += 1
	//	if winSeat.HuFanShu > this.gameRule.FengDingFanShu {
	//		winSeat.HuFanShu = this.gameRule.FengDingFanShu
	//	}
	//}
	//fanShuFen := int64(2)
	//for i := int64(0); i < winSeat.HuFanShu; i++ {
	//	fanShuFen *= 2
	//}
	//huScore := float64(diFen) * float64(fanShuFen) * this.gameRule.MultipleFloat64
	//
	//huRec.PaiXing = winSeat.MJSeat.HuPaiXing
	//huRec.BeiShu = int(winSeat.HuFanShu)
	//
	//for loseK, _ := range winSeat.WinMap {
	//	this.table.SeatArr[loseK].(*XZDDSeat).MJSeat.SeatData.RoundScore -= huScore
	//	winSeat.MJSeat.SeatData.RoundScore += huScore
	//
	//	huRecED := huRec
	//	huRecED.Score = huScore
	//	this.table.SeatArr[loseK].(*XZDDSeat).MJSeat.SeatData.PutGameScoreItem(&huRecED,-1)
	//
	//	huRec.Score += huScore
	//}
	//winSeat.MJSeat.SeatData.RoundScore += winSeat.ZhuanYiGangScore
	//
	//winSeat.MJSeat.SeatData.PutGameScoreItem(&huRec,1)
	//}

	if len(this.playPaiLogic.huSeatMap) < 1 {
		tingArr := []qpTable.SeatNumber{}
		nothingArr := []qpTable.SeatNumber{}
		for _, v := range this.table.SeatArr {
			if v == nil {
				continue
			}
			if len(v.(*XZDDSeat).MJSeat.ShouPai) < 1 {
				continue
			}
			if this.playPaiLogic.HuLogic.isTingPai(v.(*XZDDSeat)) == true {
				tingArr = append(tingArr, v.GetSeatData().Number)
			} else {
				nothingArr = append(nothingArr, v.GetSeatData().Number)
			}
		}

		score := float64(1) * this.gameRule.MultipleFloat64

		for _, v := range tingArr {
			if len(nothingArr) < 1 {
				break
			}
			huRec := qpTable.GameScoreRec{Category: gameMaJiang.ChaJiaoYing, BeiShu: 1}
			huRec.Score, huRec.TargetUID = score*float64(len(nothingArr)), nothingArr
			this.table.SeatArr[v].GetSeatData().PutGameScoreItem(&huRec, 1)

			this.table.SeatArr[v].(*XZDDSeat).MJSeat.SeatData.RoundScore += huRec.Score
		}
		for _, j := range nothingArr {
			if len(tingArr) < 1 {
				break
			}
			huRec := qpTable.GameScoreRec{Category: gameMaJiang.ChaJiaoYing, BeiShu: 1}
			huRec.Score, huRec.TargetUID = score*float64(len(tingArr)), tingArr
			this.table.SeatArr[j].GetSeatData().PutGameScoreItem(&huRec, -1)

			this.table.SeatArr[j].(*XZDDSeat).MJSeat.SeatData.RoundScore -= huRec.Score
		}
	}

	// 喜钱
	xiSeat := qpTable.INVALID_SEAT_NUMBER
	for _, v := range this.table.SeatArr {
		if v == nil {
			continue
		}
		if v.(*XZDDSeat).MJSeat.GetPaiCount(gameMaJiang.Zhong) == 4 {
			xiSeat = v.GetSeatData().Number
			break
		}
	}

	if xiSeat != qpTable.INVALID_SEAT_NUMBER {
		xiScore := 60 * this.gameRule.MultipleFloat64
		winXiSxore := float64(0)

		scoreRec := qpTable.GameScoreRec{Category: gameMaJiang.XiFen, BeiShu: 60}

		loseArr := []qpTable.SeatNumber{}
		for _, v := range this.table.SeatArr {
			if v == nil {
				continue
			}
			xzddSeat := v.(*XZDDSeat)
			if xzddSeat.MJSeat.SeatData.IsContainSeatState(qpTable.SS_Playing) == false {
				continue
			}
			if xzddSeat.MJSeat.SeatData.Number == xiSeat {
				continue
			}
			loseArr = append(loseArr, xzddSeat.MJSeat.SeatData.Number)
			xzddSeat.MJSeat.SeatData.RoundScore -= xiScore

			t := scoreRec
			t.Score, t.TargetUID = xiScore, []qpTable.SeatNumber{xiSeat}
			xzddSeat.MJSeat.SeatData.PutGameScoreItem(&t, -1)

			winXiSxore += xiScore
		}
		this.table.SeatArr[xiSeat].(*XZDDSeat).MJSeat.SeatData.RoundScore += winXiSxore
		scoreRec.Score, scoreRec.TargetUID = winXiSxore, loseArr
		this.table.SeatArr[xiSeat].(*XZDDSeat).MJSeat.SeatData.PutGameScoreItem(&scoreRec, 1)
	}
}

func (this *GameXZDDTable) onDissolveTableVote(pro *mateProto.MessageMaTe) int32 {
	rspCode := this.table.OnMessage(pro)

	if this.table.Status != qpTable.TS_Invalid {
		return rspCode
	}

	this.handleXiaoJieSuan()
	this.handleDaJieSuan()

	return rspCode
}
