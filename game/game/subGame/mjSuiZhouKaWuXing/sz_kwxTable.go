package mjSuiZhouKWXTable

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
	TIMER_XuanPiao = protoGameBasic.KaWuXing
	TIMER_PlayPai  = protoGameBasic.KaWuXing + 1
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
	case ID_PlayerXuanPiao:
		return this.onXuanPiao(msg)
	case protoGameBasic.ID_DissolveTableVote:
		return this.onDissolveTableVote(msg)
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
	huMode := int32(0)
	if len(this.playPaiLogic.huSeatMap) > 0 {
		if this.playPaiLogic.CurMoPaiSeatNum != qpTable.INVALID_SEAT_NUMBER {

			if this.playPaiLogic.buGangSeatNum != qpTable.INVALID_SEAT_NUMBER {
				this.playPaiLogic.dianPaoSeatNum = this.playPaiLogic.buGangSeatNum
				huMode = 2
			} else {
				huMode = 1
			}
			roundOver.Hupai = this.playPaiLogic.CurMoPai
		} else {
			this.playPaiLogic.dianPaoSeatNum = this.playPaiLogic.CurPlaySeatNum
			roundOver.Hupai = this.playPaiLogic.CurPlayPai
			huMode = 2
		}

		if huMode == 2 {
			dianPaoSeat := this.table.SeatArr[this.playPaiLogic.dianPaoSeatNum].(*KWXSeat)
			dianPaoSeat.GetXSeatData(0).(*gameMaJiang.MJSeat).DianPaoCount += 1
		}

	} else {
		// 亮倒分
		for _, liang := range this.playPaiLogic.liangDaoSeatNumMap {
			for _, noliang := range this.playPaiLogic.noLiangDaoSeatNumMap {
				ldScore := 1 * this.gameRule.MultipleFloat64

				liang.LiangDaoScore -= ldScore
				noliang.LiangDaoScore += ldScore

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
		for _, v := range this.playPaiLogic.huSeatMap {
			roundOver.MaPaiArr = this.calculateScore(huMode, v.SeatData.Number)
		}
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

		tempHuMode := int32(0)
		_, ok := this.playPaiLogic.huSeatMap[int32(tempSeat.Number)]
		if ok == true {
			tempHuMode = huMode
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
			HuMode:        tempHuMode,
			ShouPai:       mjSeat.GetShouPai(),
			OperationPai:  mjSeat.OperationPai,
			RoundScore:    commonDef.Float64ToString(tempSeat.RoundScore),
			SeatScore:     commonDef.Float64ToString(tempSeat.SeatScore),
			PiaoScore:     commonDef.Float64ToString(v.(*KWXSeat).OverPiaoScore),
			MaScore:       commonDef.Float64ToString(v.(*KWXSeat).MaScore),
			LiangDaoScore: commonDef.Float64ToString(v.(*KWXSeat).LiangDaoScore),
			GangScore:     commonDef.Float64ToString(v.(*KWXSeat).GangScore),
			KouPai:        v.(*KWXSeat).GetKouPai(),
			IsLiangDao:    isLiangDao,
			Piao:          v.(*KWXSeat).PiaoScore,
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
		if len(this.playPaiLogic.huSeatMap) > 1 {
			this.playPaiLogic.BankerSeatNum = this.playPaiLogic.dianPaoSeatNum
		} else if len(this.playPaiLogic.huSeatMap) > 0 {
			for k, _ := range this.playPaiLogic.huSeatMap {
				this.playPaiLogic.BankerSeatNum = qpTable.SeatNumber(k)
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
			HuMode:        -1,
			ShouPai:       mjSeat.GetShouPai(),
			OperationPai:  mjSeat.OperationPai,
			RoundScore:    commonDef.Float64ToString(tempSeat.RoundScore),
			SeatScore:     commonDef.Float64ToString(tempSeat.SeatScore),
			PiaoScore:     commonDef.Float64ToString(v.(*KWXSeat).OverPiaoScore),
			MaScore:       commonDef.Float64ToString(v.(*KWXSeat).MaScore),
			LiangDaoScore: commonDef.Float64ToString(v.(*KWXSeat).LiangDaoScore),
			GangScore:     commonDef.Float64ToString(v.(*KWXSeat).GangScore),
			KouPai:        v.(*KWXSeat).GetKouPai(),
			IsLiangDao:    isLiangDao,
			Piao:          v.(*KWXSeat).PiaoScore,
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

func (this *GameKWXTable) onXuanPiao(msg *mateProto.MessageMaTe) int32 {
	if this.table.IsAssignTableState(TS_XuanPiao) == false {
		return mateProto.Err_ActionNotMatchStatus
	}

	seat := this.table.GetSeatDataByPlayerID(qpTable.PlayerID(msg.SenderID))
	if seat == nil {
		return mateProto.Err_NotFindPlayer
	}

	if seat.GetSeatData().IsContainSeatState(qpTable.SS_Looker) == true {
		return mateProto.Err_ActionNotMatchStatus
	}

	if seat.(*KWXSeat).PiaoScore >= 0 {
		return mateProto.Err_OperationRepeat
	}

	msgXuanPiao := CS_XuanPiao{}
	err := json.Unmarshal(msg.Data, &msgXuanPiao)
	if err != nil {
		return mateProto.Err_OperationParamErr
	}

	switch msgXuanPiao.Value {
	case 0, 1, 2, 3, 5:
	default:
		return mateProto.Err_OperationParamErr
	}

	this.table.OperateRec.PutPlayerStep(int32(seat.GetSeatData().Number), ID_PlayerXuanPiao, &msgXuanPiao)

	seat.(*KWXSeat).PiaoScore = msgXuanPiao.Value

	this.table.SendToSeat(seat.GetSeatData().Number, ID_PlayerXuanPiao, &msgXuanPiao)

	msgBroadcastXuanPiao := BroadXuanPiao{SeatPiaoScoreArr: make([]SeatPiaoScore, 0, 3)}
	// 是否 所有玩家 选漂
	players := int32(0)
	for _, v := range this.table.SeatArr {
		if v == nil {
			continue
		}
		if v.(*KWXSeat).MJSeat.SeatData.IsAssignSeatState(qpTable.SS_Playing) == false {
			continue
		}
		if v.(*KWXSeat).PiaoScore >= 0 {
			players += 1
		}
		msgBroadcastXuanPiao.SeatPiaoScoreArr = append(msgBroadcastXuanPiao.SeatPiaoScoreArr,
			SeatPiaoScore{SeatNumber: int32(v.GetSeatData().Number), Value: v.(*KWXSeat).PiaoScore})
	}
	if players == this.table.GetCurSeatCount()-this.table.LookerCount {
		this.table.BroadCastGameEvent(ID_BroadcastPlayerXuanPiao, &msgBroadcastXuanPiao)

		this.playPaiLogic.faShouPai()
	}

	return 0
}

func (this *GameKWXTable) OnHu(pro *mateProto.MessageMaTe) int32 {
	playerID := qpTable.PlayerID(pro.SenderID)

	operationHu := gameMaJiang.CS_Hu{}
	err := json.Unmarshal(pro.Data, &operationHu)
	if err != nil {
		return mateProto.Err_ProtocolDataErr
	}

	guoSeat, errCode := this.playPaiLogic.CheckOperation(playerID, operationHu.OperationID)
	if errCode != mateProto.Err_Success {
		return errCode
	}

	seatData := guoSeat.GetSeatData()
	mjSeatData := guoSeat.GetXSeatData(0).(*gameMaJiang.MJSeat)

	if (mjSeatData.OperationItem & gameMaJiang.OPI_HU) == 0 {
		return mateProto.Err_OperationNotExist
	}

	seatData.CleanOperationID()
	mjSeatData.OperationItem = 0
	this.table.GameTimer.RemoveBySeatNum(int32(seatData.Number))

	if this.playPaiLogic.huSeatMap == nil {
		this.playPaiLogic.huSeatMap = make(map[int32]*gameMaJiang.MJSeat)
	}
	this.playPaiLogic.huSeatMap[int32(seatData.Number)] = mjSeatData

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

	this.RoundOver()

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
			ClubScore:    commonDef.Float64ToString(qpSeat.ClubScore),
			SeatScore:    commonDef.Float64ToString(qpSeat.SeatScore),
			RoundScore:   commonDef.Float64ToString(qpSeat.RoundScore),
			ShouPaiCount: mjSeat.ShouPaiCount,
			ChuPai:       mjSeat.PlayPai,
			PiaoScore:    kwxSeat.PiaoScore,
			VoteStatus:   v.GetSeatData().DissolveVote,
			//OperationTime: time.Now().Unix() - v.GetSeatData().OperationStart,
		}

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

	//tableData.AnGangCard = make([]int8, 0)
	//for _, v := range mjSeat.OperationPai {
	//	if v.OperationPXItem == gameMaJiang.OPX_AN_GANG {
	//		tableData.AnGangCard = append(tableData.AnGangCard, v.PaiArr[0])
	//	}
	//}

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

func (this *GameKWXTable) calculateScore(huMode int32, huSeatNum qpTable.SeatNumber) []int8 {

	maPaiArr := make([]int8, 0, 4)

	huSeat := this.table.SeatArr[huSeatNum].(*KWXSeat)

	// 自摸
	if huMode == 1 {
		maSocre := int64(0)
		getMaPaiFunc := func() {
			if this.gameRule.MaShu == 2 {
				for i := 0; i < 5; i++ {
					tempPai := this.playPaiLogic.PaiMgr.GetPai()
					if tempPai == gameMaJiang.InvalidPai {
						break
					}
					maPaiArr = append(maPaiArr, tempPai)
					if (tempPai >> 4) == gameMaJiang.MaxHuaSe {
						maSocre += 10
					} else {
						maSocre += int64(tempPai & 0x0F)
					}
					if (tempPai & 0x0F) != gameMaJiang.MinDianShu_1 {
						break
					}
				}
			} else if this.gameRule.MaShu == 1 {
				tempPai := this.playPaiLogic.PaiMgr.GetPai()
				if tempPai != gameMaJiang.InvalidPai {
					maPaiArr = append(maPaiArr, tempPai)
					if (tempPai >> 4) == gameMaJiang.MaxHuaSe {
						maSocre += 10
					} else {
						maSocre += int64(tempPai & 0x0F)
					}
				}
			} else if this.gameRule.MaShu == 3 {
				tempPai := this.playPaiLogic.PaiMgr.GetPai()
				if tempPai != gameMaJiang.InvalidPai {
					maPaiArr = append(maPaiArr, tempPai)
					if (tempPai >> 4) == gameMaJiang.MaxHuaSe {
						maSocre += 10
					} else {
						if (tempPai & 0x0F) < 6 {
							maSocre += 5
						} else {
							maSocre += 10
						}
					}
				}
			}
		}

		if this.gameRule.MaiMa == 1 && len(huSeat.LiangDaoMap) > 0 {
			getMaPaiFunc()
		} else if this.gameRule.MaiMa == 2 {
			getMaPaiFunc()
		}

		huMJSeat := huSeat.GetXSeatData(0).(*gameMaJiang.MJSeat)
		huMJSeat.HuPaiCount += 1
		huScore := float64(huMJSeat.HuScore) * this.gameRule.MultipleFloat64
		for _, v := range this.table.SeatArr {
			if v == nil {
				continue
			}
			if v.GetSeatData().IsAssignSeatState(qpTable.SS_Looker) == true {
				continue
			}
			if v.GetSeatData().Number == huSeatNum {
				continue
			}
			tempPiaoScore := float64(huSeat.PiaoScore+v.(*KWXSeat).PiaoScore) * this.gameRule.MultipleFloat64
			huSeat.OverPiaoScore += tempPiaoScore
			v.(*KWXSeat).OverPiaoScore -= tempPiaoScore

			tempMaScore := float64(maSocre) * this.gameRule.MultipleFloat64
			huSeat.MaScore += tempMaScore
			v.(*KWXSeat).MaScore -= tempMaScore

			// 胡牌人没亮倒, 其他人 亮倒时(算亮倒x2)
			liangDaoScore := float64(0)
			if this.playPaiLogic.HuLogic.isLiangDao(huSeat) == false &&
				this.playPaiLogic.HuLogic.isLiangDao(v.(*KWXSeat)) == true {

				if huMJSeat.HuScore*2 < int64(this.playPaiLogic.PlayRule.FengDingFanShu) {
					liangDaoScore = huScore
				} else {
					liangDaoScore = (float64(this.playPaiLogic.PlayRule.FengDingFanShu) * this.gameRule.MultipleFloat64) - huScore
				}
			}

			tempScore := huScore + tempPiaoScore + tempMaScore + liangDaoScore

			v.GetSeatData().RoundScore -= tempScore
			huSeat.GetSeatData().RoundScore += tempScore
		}
		return maPaiArr
	}

	// 点炮
	dianPaoSeat := this.table.SeatArr[this.playPaiLogic.dianPaoSeatNum].(*KWXSeat)
	//dianPaoSeat.GetXSeatData(0).(*gameMaJiang.MJSeat).DianPaoCount += 1

	mjSeat := huSeat.GetXSeatData(0).(*gameMaJiang.MJSeat)
	mjSeat.JiePaoCount += 1

	piaoScore := float64(dianPaoSeat.PiaoScore+huSeat.PiaoScore) * this.gameRule.MultipleFloat64

	dianPaoSeat.OverPiaoScore -= piaoScore
	huSeat.OverPiaoScore += piaoScore

	huScore := float64(mjSeat.HuScore) * this.gameRule.MultipleFloat64

	// 胡牌人没亮倒, 其他人 亮倒时(算亮倒x2)
	liangDaoScore := float64(0)
	if this.playPaiLogic.HuLogic.isLiangDao(huSeat) == false &&
		this.playPaiLogic.HuLogic.isLiangDao(dianPaoSeat) == true {

		if mjSeat.HuScore*2 < int64(this.playPaiLogic.PlayRule.FengDingFanShu) {
			liangDaoScore = huScore
		} else {
			liangDaoScore = (float64(this.playPaiLogic.PlayRule.FengDingFanShu) * this.gameRule.MultipleFloat64) - huScore
		}
	}

	winScore := piaoScore + huScore + liangDaoScore

	this.table.ChangeWinner_Loser_RoundScore(
		huSeat.GetSeatData().Number,
		this.playPaiLogic.dianPaoSeatNum,
		winScore,
		false)

	return maPaiArr
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
