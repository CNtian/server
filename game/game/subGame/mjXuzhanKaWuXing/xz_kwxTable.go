package mj_XueZhan_KWXTable

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
	TIMER_XuanPiao   = protoGameBasic.KaWuXing
	TIMER_PlayPai    = protoGameBasic.KaWuXing + 1
	TIMER_FaPai      = protoGameBasic.KaWuXing + 2
	TIMER_HuPaiMoPai = protoGameBasic.KaWuXing + 3
)

const TS_XuanPiao qpTable.TableStatus = 32 // 自定义状态起始值 选漂

type GameKWXTable struct {
	table        *qpTable.QPTable
	playPaiLogic KaWuXingPlayLogic
	gameRule     KWXPlayRule
}

// 清空每一小局数据
func (this *GameKWXTable) CleanRoundData() {

	this.playPaiLogic.CleanRoundData()

	this.table.CleanRoundData()
}

func (this *GameKWXTable) GetStatus() int32 {
	return int32(this.table.Status)
}

func (this *GameKWXTable) ParseTableOptConfig(playCfg string) (rspCode int32, err error) {

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

func (this *GameKWXTable) GetBaseQPTable() *qpTable.QPTable {
	return this.table
}

func (this *GameKWXTable) GetMaxRound() int32 {
	return this.gameRule.MaxRoundCount
}

// 事件处理
func (this *GameKWXTable) OnMessage(msg *mateProto.MessageMaTe) int32 {

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
	case protoGameBasic.ID_ForceDissolveTable:
		return this.onForceDissolveTable(msg)
	default:
		return this.playPaiLogic.OnMessage(msg)
	}

	return mateProto.Err_Success
}

func (this *GameKWXTable) onGameStart() int32 {
	rspCode := this.playPaiLogic.OnGameStart(nil)
	if rspCode != 0 {
		return rspCode
	}

	return 0
}

// 游戏结束
func (this *GameKWXTable) RoundOver() {

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

	// 0:没有胡 1:自摸 2:点炮
	isHued := false
	for _, v := range this.table.SeatArr {
		if v == nil {
			continue
		}
		s := v.(*KWXSeat)
		if len(s.HuPaiArr) > 0 {
			isHued = true
			break
		}
	}
	if isHued == false {
		// 亮倒分
		for _, liang := range this.playPaiLogic.liangDaoSeatNumMap {
			for _, noliang := range this.playPaiLogic.noLiangDaoSeatNumMap {
				ldScore := 1 * this.gameRule.MultipleFloat64

				liang.GetSeatData().RoundScore -= ldScore
				noliang.GetSeatData().RoundScore += ldScore
			}
		}

		if len(this.playPaiLogic.noLiangDaoSeatNumMap) > 0 &&
			len(this.playPaiLogic.liangDaoSeatNumMap) > len(this.playPaiLogic.noLiangDaoSeatNumMap) {
			for k, _ := range this.playPaiLogic.noLiangDaoSeatNumMap {
				this.playPaiLogic.BankerSeatNum = k
			}
		} else if len(this.playPaiLogic.liangDaoSeatNumMap) > 0 &&
			len(this.playPaiLogic.noLiangDaoSeatNumMap) > len(this.playPaiLogic.liangDaoSeatNumMap) {
			for k, _ := range this.playPaiLogic.liangDaoSeatNumMap {
				this.playPaiLogic.BankerSeatNum = k
			}
		} else if len(this.playPaiLogic.liangDaoSeatNumMap) > len(this.playPaiLogic.noLiangDaoSeatNumMap) {
			for k, _ := range this.playPaiLogic.liangDaoSeatNumMap {
				this.playPaiLogic.BankerSeatNum = k
			}
		}
	}

	// 游戏分
	{
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

	nextBanker := qpTable.INVALID_SEAT_NUMBER
	huCount := 0
	huTime := int64(0)

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
		if v.(*KWXSeat).huCount > huCount {
			nextBanker = v.(*KWXSeat).MJSeat.SeatData.Number
			huCount = v.(*KWXSeat).huCount
			huTime = v.(*KWXSeat).firstHuTime
		} else if v.(*KWXSeat).huCount == huCount && v.(*KWXSeat).firstHuTime > huTime {
			nextBanker = v.(*KWXSeat).MJSeat.SeatData.Number
			huCount = v.(*KWXSeat).huCount
			huTime = v.(*KWXSeat).firstHuTime
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
		isLiangDao := false
		if v.(*KWXSeat).LiangDaoMap != nil {
			isLiangDao = true
		}
		tempHuSeat := HuPaiSeat{
			UID:           int64(tempSeat.Player.ID),
			NickName:      tempSeat.Player.Nick,
			Head:          tempSeat.Player.Head,
			SeatNumber:    int32(tempSeat.Number),
			ShouPai:       mjSeat.GetShouPai(),
			OperationPai:  mjSeat.OperationPai,
			RoundScore:    commonDef.Float64ToString(tempSeat.RoundScore),
			SeatScore:     commonDef.Float64ToString(tempSeat.SeatScore),
			KouPai:        v.(*KWXSeat).GetKouPai(),
			IsLiangDao:    isLiangDao,
			GameScoreStep: tempSeat.GameScoreRecStep,
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
		if nextBanker != qpTable.INVALID_SEAT_NUMBER {
			this.playPaiLogic.BankerSeatNum = nextBanker
		} else {

		}
		if this.table.SeatArr[this.playPaiLogic.BankerSeatNum].GetSeatData().IsAssignSeatState(qpTable.SS_Looker) == true {
			seat := this.table.GetNextValidSeat(this.playPaiLogic.BankerSeatNum)
			this.playPaiLogic.BankerSeatNum = seat.GetSeatData().Number
		}
	}

	this.CleanRoundData()

	this.table.TableRule.TimerAutoReady()
}

func (this *GameKWXTable) handleXiaoJieSuan() {
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
		isLiangDao := false
		if v.(*KWXSeat).LiangDaoMap != nil {
			isLiangDao = true
		}
		tempHuSeat := HuPaiSeat{
			UID:           int64(tempSeat.Player.ID),
			NickName:      tempSeat.Player.Nick,
			Head:          tempSeat.Player.Head,
			SeatNumber:    int32(tempSeat.Number),
			ShouPai:       mjSeat.GetShouPai(),
			OperationPai:  mjSeat.OperationPai,
			RoundScore:    commonDef.Float64ToString(tempSeat.RoundScore),
			SeatScore:     commonDef.Float64ToString(tempSeat.SeatScore),
			KouPai:        v.(*KWXSeat).GetKouPai(),
			IsLiangDao:    isLiangDao,
			GameScoreStep: tempSeat.GameScoreRecStep,
		}

		roundOver.SeatDataArr = append(roundOver.SeatDataArr, &tempHuSeat)
	}

	this.table.BroadCastGameEvent(ID_RoundGameOver, &roundOver)
}

func (this *GameKWXTable) handleDaJieSuan() {

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

func (this *GameKWXTable) onTableExpire(pro *mateProto.MessageMaTe) int32 {

	this.table.DissolveType = qpTable.DT_LiveTimeout

	this.handleXiaoJieSuan()
	this.handleDaJieSuan()

	return this.table.OnMessage(pro)
}

func (this *GameKWXTable) onReady(pro *mateProto.MessageMaTe) int32 {
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

func (this *GameKWXTable) onPlayerLeave(pro *mateProto.MessageMaTe) int32 {

	if this.table.IsAssignTableState(qpTable.TS_WaitingPlayerEnter) == false {
		return mateProto.Err_TableStatusNotMatch
	}
	return this.table.OnLeave(pro)
}

func (this *GameKWXTable) onPrivateJoinTable(msg *mateProto.MessageMaTe) int32 {

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

//var (
//	gpsArr = []float64{30.480164, 114.412486, 30.480382, 114.419195, 30.476092, 114.414443}
//	index  = 0
//)

func (this *GameKWXTable) onClubJoinTable(msg *mateProto.MessageMaTe) int32 {

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

	//{
	//	clubJoinTable.Latitude = gpsArr[index]
	//	clubJoinTable.Longitude = gpsArr[index+1]
	//	index += 2
	//	if index >= len(gpsArr) {
	//		index = 0
	//	}
	//}

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

func (this *GameKWXTable) OnHu(pro *mateProto.MessageMaTe) int32 {
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

	seatData := seat.GetSeatData()
	mjSeatData := seat.GetXSeatData(0).(*gameMaJiang.MJSeat)

	if (mjSeatData.OperationItem & gameMaJiang.OPI_HU) == 0 {
		return mateProto.Err_OperationNotExist
	}

	seatData.CleanOperationID()
	mjSeatData.OperationItem = 0
	this.table.GameTimer.RemoveBySeatNum(int32(seatData.Number))

	// 有人胡了
	this.playPaiLogic.isHued = true

	scHu := SC_Hu{HuSeat: int32(seatData.Number),
		PlayPai: this.playPaiLogic.CurPlayPai}
	huMode := int32(0)
	if this.playPaiLogic.CurMoPaiSeatNum != qpTable.INVALID_SEAT_NUMBER {
		if this.playPaiLogic.buGangSeatNum != qpTable.INVALID_SEAT_NUMBER {
			this.playPaiLogic.dianPaoSeatNum = this.playPaiLogic.buGangSeatNum
			scHu.Category, huMode = 3, 2
			scHu.DianPaoSeat = int32(this.playPaiLogic.dianPaoSeatNum)
		} else {
			mjSeatData.DeleteShouPai(this.playPaiLogic.CurMoPai)
			scHu.Category, huMode = 1, 1
		}
		scHu.HuPai = this.playPaiLogic.CurMoPai
	} else {
		this.playPaiLogic.dianPaoSeatNum = this.playPaiLogic.CurPlaySeatNum
		scHu.HuPai = this.playPaiLogic.CurPlayPai
		scHu.Category, huMode = 2, 2
		scHu.DianPaoSeat = int32(this.playPaiLogic.dianPaoSeatNum)
	}

	if this.playPaiLogic.CurMoPaiSeatNum == seatData.Number {
		this.calculateScore(huMode, seatData.Number)
	} else {
		this.calculateScore(huMode, seatData.Number)
	}

	this.table.BroadCastGameEvent(gameMaJiang.ID_Hu, &scHu)

	seat.(*KWXSeat).HuPaiArr = append(seat.(*KWXSeat).HuPaiArr, scHu.HuPai)
	seat.(*KWXSeat).huCount += 1
	if seat.(*KWXSeat).firstHuTime == 0 {
		seat.(*KWXSeat).firstHuTime = time.Now().Unix()
	}
	if seat.(*KWXSeat).LiangDaoMap == nil {
		seatData.AppendState(SS_Suspend)
		this.playPaiLogic.plays -= 1
	}

	for _, v := range this.table.SeatArr {
		if v == nil {
			continue
		}
		// 等待其他人操作
		tempMJSeat := v.GetXSeatData(0).(*gameMaJiang.MJSeat)
		if (tempMJSeat.OperationItem & gameMaJiang.OPI_HU) == gameMaJiang.OPI_HU {
			return mateProto.Err_Success
		}
	}

	this.playPaiLogic.isHued = false

	if this.playPaiLogic.isGameOver() == true {
		this.RoundOver()
		return mateProto.Err_Success
	}

	this.playPaiLogic.CleanAllSeatOperation()

	this.table.GameTimer.PutTableTimer(TIMER_HuPaiMoPai, 2*1000, func() {
		// 点炮的人，摸牌再出
		if this.playPaiLogic.dianPaoSeatNum != qpTable.INVALID_SEAT_NUMBER {
			this.playPaiLogic.MoPaiOperation(this.playPaiLogic.dianPaoSeatNum)
		} else {
			nextSeat := this.playPaiLogic.getNextPlayingSeat(seatData.Number)
			this.playPaiLogic.MoPaiOperation(nextSeat.GetSeatData().Number)
		}
	})

	return mateProto.Err_Success
}

func (this *GameKWXTable) onGetTableData(pro *mateProto.MessageMaTe) int32 {

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
		CurPlaySeatNumber:   int32(this.playPaiLogic.lastPlaySeatNum),
		CurMoPaiSeatNumber:  int32(this.playPaiLogic.CurMoPaiSeatNum),
		CurPengSeatNumber:   int32(this.playPaiLogic.CurPengSeatNum),
		RemainCardCount:     this.playPaiLogic.PaiMgr.GetTheRestOfPaiCount(),
		ClubScore:           commonDef.Float64ToString(seat.GetSeatData().ClubScore),
		DissolveID:          int32(this.table.DissolveSeatNum),
		LaunchDissolveTime:  nowTT - this.table.LaunchDissolveTime,
		PlayerOperationTime: nowTT - this.playPaiLogic.OperationTime,
		FirstRoundReadTime:  nowTT - this.table.FirstRoundReadTime,
	}

	var selfSeat *KWXSeat
	tableData.SeatData = make([]*MsgSeatData, 0)
	for _, v := range this.table.SeatArr {
		if v == nil {
			continue
		}

		kwxSeat := v.(*KWXSeat)
		mjSeat := kwxSeat.MJSeat
		qpSeat := mjSeat.SeatData

		if int64(qpSeat.Player.ID) == pro.SenderID {
			selfSeat = kwxSeat
		}

		proSeat := &MsgSeatData{UID: int64(qpSeat.Player.ID),
			Nick:         qpSeat.Player.Nick,
			HeadURL:      qpSeat.Player.Head,
			Sex:          qpSeat.Player.Sex,
			IP:           qpSeat.Player.IP,
			SeatNumber:   int32(qpSeat.Number),
			SeatStatus:   uint32(qpSeat.Status),
			ClubID:       kwxSeat.MJSeat.SeatData.ClubID,
			ClubScore:    commonDef.Float64ToString(qpSeat.ClubScore),
			SeatScore:    commonDef.Float64ToString(qpSeat.SeatScore),
			RoundScore:   commonDef.Float64ToString(qpSeat.RoundScore),
			ShouPaiCount: mjSeat.ShouPaiCount,
			ChuPai:       mjSeat.PlayPai,
			HuPai:        v.(*KWXSeat).HuPaiArr,
			VoteStatus:   v.GetSeatData().DissolveVote,
			//OperationTime: time.Now().Unix() - v.GetSeatData().OperationStart,
		}

		proSeat.OperationPai = make([]*gameMaJiang.OperationPaiInfo, 0)
		for _, v := range mjSeat.OperationPai {
			switch v.OperationPXItem {
			default:
				proSeat.OperationPai = append(proSeat.OperationPai, v)
			}
		}
		proSeat.LiangDaoPaiArr = kwxSeat.GetLiangDaoPai()
		proSeat.TingPaiArr = kwxSeat.GetTingPai()
		if qpSeat.Lng > 0.1 && qpSeat.Lat > 0.1 {
			proSeat.IsGPS = true
		}

		tableData.SeatData = append(tableData.SeatData, proSeat)
	}

	mjSeat := selfSeat.MJSeat

	tableData.CurMoPai = mjSeat.CurMoPai
	tableData.ShouPai = mjSeat.GetShouPai()
	tableData.OperationID = mjSeat.SeatData.OperationID
	tableData.OperationItem = uint32(mjSeat.OperationItem)
	tableData.GangArr = mjSeat.GangArr
	tableData.KouPaiArr = selfSeat.GetKouPai()

	tableData.GameRuleText = this.gameRule.RuleJson

	this.table.SendToSeat(seat.GetSeatData().Number, ID_TableData, &tableData)

	return 0
}

func (this *GameKWXTable) onCancelTrusteeship(msg *mateProto.MessageMaTe) int32 {
	seatNumber := this.table.OnCancelTrusteeship(msg)
	if seatNumber < 0 {
		return seatNumber
	}

	timerArr := this.table.GameTimer.RemoveBySeatNum(seatNumber)
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

func (this *GameKWXTable) calculateScore(huMode int32, huSeatNum qpTable.SeatNumber) float64 {

	huSeat := this.table.SeatArr[huSeatNum].(*KWXSeat)

	// 自摸
	if huMode == 1 {
		huMJSeat := huSeat.GetXSeatData(0).(*gameMaJiang.MJSeat)
		huMJSeat.HuPaiCount += 1

		huRec := qpTable.GameScoreRec{Category: gameMaJiang.ZiMo, BeiShu: int(huMJSeat.HuScore)}
		huRec.PaiXing = huMJSeat.HuPaiXing
		winScore := float64(0)

		huScore := float64(huMJSeat.HuScore) * this.gameRule.MultipleFloat64

		for _, v := range this.table.SeatArr {
			if v == nil {
				continue
			}
			s := v.(*KWXSeat)
			if s.MJSeat.SeatData.IsAssignSeatState(qpTable.SS_Looker) == true {
				continue
			}
			if s.MJSeat.SeatData.Number == huSeatNum {
				continue
			}
			if s.MJSeat.SeatData.IsAssignSeatState(SS_Suspend) == true {
				continue
			}
			rec := qpTable.GameScoreRec{Category: gameMaJiang.ZiMo, BeiShu: int(huMJSeat.HuScore)}
			rec.PaiXing = huMJSeat.HuPaiXing

			// 胡牌人没亮倒, 其他人 亮倒时(算亮倒x2)
			liangDaoScore := float64(0)
			if this.playPaiLogic.HuLogic.isLiangDao(huSeat) == false &&
				this.playPaiLogic.HuLogic.isLiangDao(v.(*KWXSeat)) == true {

				if huMJSeat.HuScore*2 < this.playPaiLogic.PlayRule.FengDingFanShu {
					liangDaoScore = huScore
					rec.BeiShu *= 2
				} else {
					liangDaoScore = (float64(this.playPaiLogic.PlayRule.FengDingFanShu) * this.gameRule.MultipleFloat64) - huScore
				}
			}

			tempScore := huScore + liangDaoScore

			s.MJSeat.SeatData.RoundScore -= tempScore
			rec.Score -= tempScore
			s.MJSeat.SeatData.PutGameScoreItem(&rec, 1)

			huSeat.GetSeatData().RoundScore += tempScore
			winScore += tempScore
		}
		huRec.Score = winScore
		huSeat.MJSeat.SeatData.PutGameScoreItem(&huRec, 1)
		return 0
	}

	huRec := qpTable.GameScoreRec{Category: gameMaJiang.DianPao}
	dianPaoRec := qpTable.GameScoreRec{Category: gameMaJiang.DianPao}

	// 点炮
	dianPaoSeat := this.table.SeatArr[this.playPaiLogic.dianPaoSeatNum].(*KWXSeat)

	mjSeat := huSeat.GetXSeatData(0).(*gameMaJiang.MJSeat)
	mjSeat.JiePaoCount += 1

	huScore := float64(mjSeat.HuScore) * this.gameRule.MultipleFloat64
	beiShu := int(mjSeat.HuScore)

	// 胡牌人没亮倒, 其他人 亮倒时(算亮倒x2)
	liangDaoScore := float64(0)
	if this.playPaiLogic.HuLogic.isLiangDao(huSeat) == false &&
		this.playPaiLogic.HuLogic.isLiangDao(dianPaoSeat) == true {

		if mjSeat.HuScore*2 < this.playPaiLogic.PlayRule.FengDingFanShu {
			liangDaoScore = huScore
			beiShu *= 2
		} else {
			liangDaoScore = (float64(this.playPaiLogic.PlayRule.FengDingFanShu) * this.gameRule.MultipleFloat64) - huScore
		}
	}

	winScore := huScore + liangDaoScore
	huRec.BeiShu, dianPaoRec.BeiShu = beiShu, beiShu
	huRec.Score, dianPaoRec.Score = winScore, -winScore
	huRec.PaiXing, dianPaoRec.PaiXing = mjSeat.HuPaiXing, mjSeat.HuPaiXing

	dianPaoSeat.MJSeat.SeatData.PutGameScoreItem(&dianPaoRec, 1)
	huSeat.MJSeat.SeatData.PutGameScoreItem(&huRec, 1)

	this.table.ChangeWinner_Loser_RoundScore(
		huSeat.GetSeatData().Number,
		this.playPaiLogic.dianPaoSeatNum,
		winScore,
		false)

	return 0
}

func (this *GameKWXTable) onDissolveTableVote(pro *mateProto.MessageMaTe) int32 {
	rspCode := this.table.OnMessage(pro)

	if this.table.Status != qpTable.TS_Invalid {
		return rspCode
	}

	this.handleXiaoJieSuan()
	this.handleDaJieSuan()

	return rspCode
}

func (this *GameKWXTable) onForceDissolveTable(pro *mateProto.MessageMaTe) int32 {
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
