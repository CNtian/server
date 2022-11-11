package mjXZDDTable

import (
	"encoding/json"
	"github.com/golang/glog"
	"math/rand"
	commonDef "qpGame/commonDefine"
	"qpGame/commonDefine/mateProto"
	"qpGame/commonDefine/mateProto/protoGameBasic"
	"qpGame/db"
	"qpGame/game/gameMaJiang"
	"qpGame/qpTable"
	"qpGame/wrapMQ"
	"strconv"
	"time"
)

const (
	HuPX_PingHu         = 1
	HuPX_KaWuXing       = 2
	HuPX_PengPengHu     = 3
	HuPX_MingSiGui      = 4
	HuPX_LiangDao       = 5
	HuPX_GangShangHua   = 6
	HuPX_GangShangPao   = 7
	HuPX_QiangGangHu    = 8
	HuPX_HaiDiLao       = 9
	HuPX_AnSiGui        = 10
	HuPX_XiaoSanYuan    = 11
	HuPX_QingYiSe       = 12
	HuPX_ShouZhuaYi     = 13 // 单吊
	HuPX_QiXiaoDui      = 14
	HuPX_DaSanYuan      = 15
	HuPX_HaoHuaQiDui    = 16
	HuPX_JiuLianBaoDeng = 17
	HuPX_HaiDiPao       = 18
	HuPX_ShangLou       = 19

	HuPX_JiangYiSe  = 30
	HuPX_YaoJiu     = 31
	HuPX_JiangDui   = 32
	HuPX_Gen        = 33
	HuPX_MengQing   = 34
	HuPX_ZhongZhang = 35
	HuPX_TianHu     = 36
	HuPX_DiHu       = 37
	HuPX_ZiMoJiaDi  = 38
	HuPX_ZiMoJiaFan = 39
)

type XueZhanDaoDiPlayLogic struct {
	PaiMgr   gameMaJiang.MJPaiMgr // 牌的管理器
	HuLogic  xzddHuLogic          // 逻辑
	Table    *qpTable.QPTable
	PlayRule *XZDDPlayRule // 玩法规则

	RoundOverFunc func()

	BankerSeatNum qpTable.SeatNumber // 庄家座位号

	// 小局 待清理 成员
	sRoundHuSeatMap map[int32]*gameMaJiang.MJSeat // 有多少人胡了
	lastGangSeatNum qpTable.SeatNumber            // 上次杠的座位号(记录杠上炮)
	buGangSeatNum   qpTable.SeatNumber            // 补杠座位
	buGangPai       int8                          // 补杠牌
	huSeatMap       map[int32]*gameMaJiang.MJSeat // 已经胡的玩家 key:座位号
	CurPlayPai      int8                          // 最近一次出的牌
	lastPlaySeatNum qpTable.SeatNumber            // 最近一次出牌的座位号
	CurPlaySeatNum  qpTable.SeatNumber            // 当前出牌的座位号
	CurMoPaiSeatNum qpTable.SeatNumber            // 当前摸牌座位号
	CurMoPai        int8                          // 当前摸的牌
	CurPengSeatNum  qpTable.SeatNumber            // 当前碰的座位号
	OperationTime   int64                         // 玩家操作起始点
	dianPaoSeatMap  map[qpTable.SeatNumber][]*groupWinner

	TempHuPlayerData struct {
		winSeat     qpTable.SeatNumber
		dianPaoSeat qpTable.SeatNumber
		f           []func()
	} // 点了胡牌操作

	delayActiveFunc func()
}

// 清空每一小局数据
func (this *XueZhanDaoDiPlayLogic) CleanRoundData() {
	this.sRoundHuSeatMap = nil
	this.lastGangSeatNum = qpTable.INVALID_SEAT_NUMBER
	this.buGangSeatNum = qpTable.INVALID_SEAT_NUMBER
	this.buGangPai = gameMaJiang.InvalidPai
	this.huSeatMap = nil
	this.CurPlayPai = gameMaJiang.InvalidPai
	this.lastPlaySeatNum = qpTable.INVALID_SEAT_NUMBER
	this.CurPlaySeatNum = qpTable.INVALID_SEAT_NUMBER
	this.CurMoPaiSeatNum = qpTable.INVALID_SEAT_NUMBER
	this.CurMoPai = gameMaJiang.InvalidPai
	this.CurPengSeatNum = qpTable.INVALID_SEAT_NUMBER
	this.TempHuPlayerData.f, this.TempHuPlayerData.dianPaoSeat = nil, qpTable.INVALID_SEAT_NUMBER
	this.TempHuPlayerData.winSeat = qpTable.INVALID_SEAT_NUMBER
	this.dianPaoSeatMap = nil

	this.delayActiveFunc = nil
}

func (this *XueZhanDaoDiPlayLogic) isRoundOver() bool {
	if this.Table.GetCurSeatCount()-int32(len(this.huSeatMap))-this.Table.LookerCount < 2 {
		return true
	}
	if this.PaiMgr.GetTheRestOfPaiCount() < 1 {
		return true
	}
	return false
}

func (this *XueZhanDaoDiPlayLogic) GetNextPlayingSeat(cur qpTable.SeatNumber) qpTable.QPSeat {

	for i := cur + 1; int(i) < len(this.Table.SeatArr); i++ {
		if this.Table.SeatArr[i] != nil &&
			this.Table.SeatArr[i].GetSeatData().IsAssignSeatState(qpTable.SS_Playing) == true &&
			this.Table.SeatArr[i].GetSeatData().IsAssignSeatState(SS_HU) == false {
			return this.Table.SeatArr[i]
		}
	}

	for i := 0; qpTable.SeatNumber(i) < cur; i++ {
		if this.Table.SeatArr[i] != nil &&
			this.Table.SeatArr[i].GetSeatData().IsAssignSeatState(qpTable.SS_Playing) == true &&
			this.Table.SeatArr[i].GetSeatData().IsAssignSeatState(SS_HU) == false {
			return this.Table.SeatArr[i]
		}
	}
	return nil
}

// 事件处理
func (this *XueZhanDaoDiPlayLogic) OnMessage(pro *mateProto.MessageMaTe) int32 {

	var rspCode int32 = -1
	switch pro.MessageID {
	case gameMaJiang.ID_Guo:
		rspCode = this.OnGuo(pro)
	case gameMaJiang.ID_Play:
		rspCode = this.OnPlay(pro)
	case gameMaJiang.ID_Peng:
		rspCode = this.OnPeng(pro)
	case gameMaJiang.ID_Gang:
		rspCode = this.OnGang(pro)
	case ID_CustomNextPai:
		return this.onCustomNextPai(pro)
	case ID_GetRemainingPai:
		return this.onGetRemainingPai(pro)
	case ID_PlayerDingQue:
		return this.OnDingQue(pro)
	case ID_PlayerChanePai:
		return this.OnChangedPai(pro)
	case protoGameBasic.ID_ActiveTrusteeship:
		return this.OnActiveTrusteeship(pro)
	default:
		return this.Table.OnMessage(pro)
	}

	return rspCode
}

func (this *XueZhanDaoDiPlayLogic) OnGameStart(pro *mateProto.MessageMaTe) int32 {
	if this.Table.IsAssignTableState(qpTable.TS_WaitingPlayerEnter) == false &&
		this.Table.IsAssignTableState(qpTable.TS_WaitingReady) == false {
		return mateProto.Err_TableStatusNotMatch
	}

	errNumber := this.Table.OnGameStart(pro)
	if errNumber != mateProto.Err_Success {
		return errNumber
	}

	this.Table.CurXRound += 1
	this.Table.OperateRec.SetTableInfo(this.Table.TableNum, this.Table.CurXRound, this.PlayRule.RuleJson, this.Table.TableRule.TableCfgJson)

	this.faShouPai()

	return mateProto.Err_Success
}

func (this *XueZhanDaoDiPlayLogic) faShouPai() {
	// 无庄时
	if this.BankerSeatNum == qpTable.INVALID_SEAT_NUMBER {
		this.BankerSeatNum = qpTable.SeatNumber(rand.Intn(int(this.Table.GetCurSeatCount())))
		if this.BankerSeatNum < 0 || this.Table.SeatArr[this.BankerSeatNum] == nil ||
			this.Table.SeatArr[this.BankerSeatNum].GetSeatData().IsAssignSeatState(qpTable.SS_Playing) == false {
			for _, v := range this.Table.SeatArr {
				if v != nil && v.GetSeatData().IsAssignSeatState(qpTable.SS_Playing) {
					this.BankerSeatNum = v.GetSeatData().Number
					break
				}
			}
		}
	} else {
		tempSeat := this.Table.SeatArr[this.BankerSeatNum]
		if tempSeat.GetSeatData().IsAssignSeatState(qpTable.SS_Looker) == true {
			tempSeat = this.GetNextPlayingSeat(this.BankerSeatNum)
			if tempSeat == nil {
				for _, v := range this.Table.SeatArr {
					if v != nil && v.GetSeatData().IsAssignSeatState(qpTable.SS_Playing) {
						this.BankerSeatNum = v.GetSeatData().Number
						break
					}
				}
			} else {
				this.BankerSeatNum = tempSeat.GetSeatData().Number
			}
		}
	}

	this.PaiMgr.XiPai()

	// 发手牌
	for i, v := range this.Table.SeatArr {
		if v == nil {
			continue
		}
		seat := v.GetSeatData()
		if seat.IsAssignSeatState(qpTable.SS_Looker) == true {
			this.Table.SendGameEventToSeat(qpTable.SeatNumber(i),
				gameMaJiang.ID_FaShouPai,
				&gameMaJiang.SC_FaShouPai{
					SeatNum:       int32(i),
					BankerSeatNum: int32(this.BankerSeatNum)})
			continue
		}
		if seat.IsAssignSeatState(qpTable.SS_Playing) == false {
			continue
		}

		paiArr := this.PaiMgr.GetGroupPai(this.PlayRule.ShouPaiCount)

		mjSeat := v.GetXSeatData(0).(*gameMaJiang.MJSeat)

		seatCardTemp := make([]int8, len(paiArr))
		for i, v := range paiArr {
			mjSeat.PushShouPai(v)
			seatCardTemp[i] = v
		}

		this.Table.OperateRec.PutPlayer(seat)

		this.Table.SendGameEventToSeat(qpTable.SeatNumber(i),
			gameMaJiang.ID_FaShouPai,
			&gameMaJiang.SC_FaShouPai{
				SeatNum:       int32(i),
				Pai:           seatCardTemp,
				BankerSeatNum: int32(this.BankerSeatNum)})
	}

	this.Table.GameTimer.PutTableTimer(TIMER_PlayPai, 2200, func() {
		// 庄家 起手摸牌
		this.qiShouMoPaiOperation(this.BankerSeatNum)
	})
}

func (this *XueZhanDaoDiPlayLogic) OnChangedPai(msg *mateProto.MessageMaTe) int32 {
	if this.Table.IsAssignTableState(TS_ChangePai) == false {
		return mateProto.Err_ActionNotMatchStatus
	}

	seat := this.Table.GetSeatDataByPlayerID(qpTable.PlayerID(msg.SenderID))
	if seat == nil {
		return mateProto.Err_NotFindPlayer
	}

	xzddSeat := seat.(*XZDDSeat)
	if xzddSeat.ChangePai != nil {
		return mateProto.Err_OperationRepeat
	}

	msgChangePai := CS_PlayerChanePai{}
	err := json.Unmarshal(msg.Data, &msgChangePai)
	if err != nil {
		return mateProto.Err_OperationParamErr
	}

	if len(msgChangePai.Value) != this.PlayRule.ChangePai {
		return mateProto.Err_OperationParamErr
	}

	paiType := msgChangePai.Value[0] >> 4
	for _, v := range msgChangePai.Value {
		if xzddSeat.MJSeat.GetPaiCount(v) < 1 {
			return mateProto.Err_PaiNotExist
		}
		if paiType != v>>4 {
			return mateProto.Err_PaiNotExist
		}
	}

	this.Table.OperateRec.PutPlayerStep(int32(seat.GetSeatData().Number), ID_PlayerChanePai, &msgChangePai)

	xzddSeat.ChangePai = msgChangePai.Value

	this.Table.SendToAllPlayer(ID_BroadcastChanePaiFinish, &CS_BroadcastChanePaiFinish{int32(xzddSeat.MJSeat.SeatData.Number)})

	// 是否 所有玩家 确定换牌
	msgBroadcastChangePai := CS_BroadCastChangePaiResult{SeatNumArr: make([]int32, 0, 4)}
	players := int32(0)
	for _, v := range this.Table.SeatArr {
		if v == nil {
			continue
		}
		if v.(*XZDDSeat).MJSeat.SeatData.IsAssignSeatState(qpTable.SS_Playing) == false {
			continue
		}
		if v.(*XZDDSeat).ChangePai != nil {
			players += 1
		}
		msgBroadcastChangePai.SeatNumArr = append(msgBroadcastChangePai.SeatNumArr,
			int32(v.GetSeatData().Number))
	}
	if players != this.Table.GetCurSeatCount()-this.Table.LookerCount {
		return 0
	}

	// 0:对换  1:顺时针  2:逆时针
	changeMode := 0
	changeMode = rand.Intn(3)
	if this.Table.GetCurSeatCount() == 3 && changeMode == 0 {
		changeMode = 1
	}
	msgBroadcastChangePai.Mode = changeMode
	this.Table.BroadCastGameEvent(ID_BroadCastChangePaiResult, &msgBroadcastChangePai)

	this.Table.GameTimer.RemoveByTimeID(TIMER_ChanePai)
	this.Table.DelTableState(TS_ChangePai)

	for _, v := range msgBroadcastChangePai.SeatNumArr {
		xzddSeat := this.Table.SeatArr[v].(*XZDDSeat)
		for _, pai := range xzddSeat.ChangePai {
			xzddSeat.MJSeat.DeleteShouPai(pai)
		}
	}

	// 测试
	//changeMode++
	//if changeMode > 2 {
	//	changeMode = 2
	//}
	//glog.Info("tableID:=", this.Table.TableNum, ",changeMode:=", changeMode)

	if changeMode == 0 {
		if this.Table.GetCurSeatCount() == 2 {
			var aSeat *XZDDSeat
			for _, v := range this.Table.SeatArr {
				if v == nil {
					continue
				}
				temp := v.(*XZDDSeat)
				if temp.MJSeat.SeatData.IsAssignSeatState(qpTable.SS_Playing) == false {
					continue
				}
				if aSeat == nil {
					aSeat = temp
					continue
				}
				aSeat.ChangedPai, temp.ChangedPai = temp.ChangePai, aSeat.ChangePai
				break
			}
		} else if this.Table.GetCurSeatCount() == 4 {
			aSeat := this.Table.SeatArr[0].(*XZDDSeat)
			bSeat := this.Table.SeatArr[2].(*XZDDSeat)
			aSeat.ChangedPai, bSeat.ChangedPai = bSeat.ChangePai, aSeat.ChangePai

			aSeat = this.Table.SeatArr[1].(*XZDDSeat)
			bSeat = this.Table.SeatArr[3].(*XZDDSeat)
			aSeat.ChangedPai, bSeat.ChangedPai = bSeat.ChangePai, aSeat.ChangePai
		}
	} else if changeMode == 1 {
		var lSeat, fSeat *XZDDSeat
		for i := len(this.Table.SeatArr) - 1; i >= 0; i-- {
			if this.Table.SeatArr[i] == nil {
				continue
			}
			temp := this.Table.SeatArr[i].(*XZDDSeat)
			if temp.MJSeat.SeatData.IsAssignSeatState(qpTable.SS_Playing) == false {
				continue
			}

			// 记录第一位
			if fSeat == nil {
				fSeat, lSeat = temp, temp
				continue
			}
			temp.ChangedPai = lSeat.ChangePai
			lSeat = temp
		}
		fSeat.ChangedPai = lSeat.ChangePai
	} else if changeMode == 2 {
		var lSeat, fSeat *XZDDSeat
		for _, v := range this.Table.SeatArr {
			if v == nil {
				continue
			}
			temp := v.(*XZDDSeat)
			if temp.MJSeat.SeatData.IsAssignSeatState(qpTable.SS_Playing) == false {
				continue
			}

			// 记录第一位
			if fSeat == nil {
				fSeat, lSeat = temp, temp
				continue
			}
			temp.ChangedPai = lSeat.ChangePai
			lSeat = temp
		}
		fSeat.ChangedPai = lSeat.ChangePai
	}

	for _, v := range msgBroadcastChangePai.SeatNumArr {
		xzddSeat := this.Table.SeatArr[v].(*XZDDSeat)
		for _, pai := range xzddSeat.ChangedPai {
			xzddSeat.MJSeat.PushShouPai(pai)
		}
		this.Table.SendGameEventToSeat(qpTable.SeatNumber(v), ID_ChangePaiResult,
			&CS_ChangePaiResult{this.Table.SeatArr[v].(*XZDDSeat).ChangedPai})
	}

	if this.PlayRule.WanFa == 4 {

		this.Table.GameTimer.PutTableTimer(TIMER_DelayDingQue, 1500, func() {
			this.Table.AppendTableState(TS_DingQue)
			for _, v := range this.Table.SeatArr {
				if v == nil {
					continue
				}
				if v.GetSeatData().IsAssignSeatState(qpTable.SS_Playing) == false {
					continue
				}
				v.(*XZDDSeat).ReadyDingQue = v.(*XZDDSeat).GetAutoDingQue()
				t := SC_NoticeDingQue{Pai: v.(*XZDDSeat).ReadyDingQue}
				this.Table.SendGameEventToSeat(v.(*XZDDSeat).MJSeat.SeatData.Number, ID_NoticeDingQue, &t)
			}
			this.Table.GameTimer.PutTableTimer(TIMER_DingQue, 10*1000, this.timerDingQue)

			this.OperationTime = time.Now().Unix()
		})
		return 0
	}

	this.Table.GameTimer.PutTableTimer(TIMER_DelayDingQue, 1500, func() {
		this.MoPaiOperation(this.BankerSeatNum)
	})
	return 0
}

// 出牌
func (this *XueZhanDaoDiPlayLogic) OnPlay(pro *mateProto.MessageMaTe) int32 {
	playerID := qpTable.PlayerID(pro.SenderID)

	playPai := gameMaJiang.CS_Play{}
	err := json.Unmarshal(pro.Data, &playPai)
	if err != nil {
		return mateProto.Err_ProtocolDataErr
	}

	playSeat, errCode := this.CheckOperation(playerID, playPai.OperationID)
	if errCode != mateProto.Err_Success {
		return errCode
	}

	seatData := playSeat.(*XZDDSeat)
	mjSeatData := seatData.MJSeat
	if (mjSeatData.OperationItem & gameMaJiang.OPI_PlayPai) != gameMaJiang.OPI_PlayPai {
		return mateProto.Err_OperationNotExist
	}

	if seatData.GetDingQuePaiCount() > 0 &&
		playPai.Pai>>4 != seatData.DingQue {
		return mateProto.Err_PaiXingError
	}

	if mjSeatData.GetPaiCount(playPai.Pai) < 1 {
		return mateProto.Err_PaiNotExist
	}

	if mjSeatData.DeleteShouPai(playPai.Pai) == false {
		return mateProto.Err_PaiNotExist
	}

	mjSeatData.OperationItem = 0
	seatData.MJSeat.SeatData.CleanOperationID()
	playSeat.(*XZDDSeat).PlayPaiCount += 1
	mjSeatData.CurMoPai = gameMaJiang.InvalidPai
	seatData.DGHPlayPaiSeat = qpTable.INVALID_SEAT_NUMBER
	this.Table.OperateRec.PutPlayerStep(int32(seatData.MJSeat.SeatData.Number), gameMaJiang.ID_Play, &playPai)
	mjSeatData.PlayPai = append(mjSeatData.PlayPai, playPai.Pai)
	this.Table.GetBaseQPTable().GameTimer.RemoveBySeatNum(int32(seatData.MJSeat.SeatData.Number))

	this.CurPengSeatNum = qpTable.INVALID_SEAT_NUMBER
	this.CurMoPaiSeatNum = qpTable.INVALID_SEAT_NUMBER
	this.CurMoPai = gameMaJiang.InvalidPai
	this.CurPlaySeatNum = seatData.MJSeat.SeatData.Number
	this.lastPlaySeatNum = seatData.MJSeat.SeatData.Number
	this.CurPlayPai = playPai.Pai
	this.buGangSeatNum = qpTable.INVALID_SEAT_NUMBER
	this.buGangPai = gameMaJiang.InvalidPai
	this.sRoundHuSeatMap = nil
	this.TempHuPlayerData.f, this.TempHuPlayerData.dianPaoSeat = nil, qpTable.INVALID_SEAT_NUMBER
	this.TempHuPlayerData.winSeat = qpTable.INVALID_SEAT_NUMBER

	// 广播 出牌
	this.Table.BroadCastGameEvent(gameMaJiang.ID_BroadcastPlay,
		&gameMaJiang.BroadcastPlay{SeatNumber: int32(seatData.MJSeat.SeatData.Number), Pai: playPai.Pai})

	canOperationSeatArr, isFindHu := this.GetPlayPaiOperation(this.CurPlaySeatNum, this.CurPlayPai)

	if isFindHu == false && this.lastGangSeatNum != qpTable.INVALID_SEAT_NUMBER {
		this.Table.SeatArr[this.lastGangSeatNum].(*XZDDSeat).LastGangScore = 0
		this.lastGangSeatNum = qpTable.INVALID_SEAT_NUMBER
	}

	// 无人 可操作
	if len(canOperationSeatArr) < 1 {
		if this.isRoundOver() == true {
			this.RoundOverFunc()
		} else {
			nextSeat := this.GetNextPlayingSeat(this.CurPlaySeatNum)
			this.MoPaiOperation(nextSeat.GetSeatData().Number)
		}
		return mateProto.Err_Success
	}
	this.OperationTime = time.Now().Unix()

	// 记录 可杠操作
	this.delayActiveFunc = nil

	// 通知 操作
	for i, _ := range canOperationSeatArr {
		// 非亮倒时
		proMsg := gameMaJiang.SC_NoticeOperation{SeatNumber: int32(canOperationSeatArr[i].GetSeatData().Number),
			OperationID: canOperationSeatArr[i].GetSeatData().GetOperationID(),
			Operation:   canOperationSeatArr[i].GetXSeatData(0).(*gameMaJiang.MJSeat).OperationItem,
			Pai:         playPai.Pai,
			GangPai:     canOperationSeatArr[i].GetXSeatData(0).(*gameMaJiang.MJSeat).GangArr,
		}
		this.NoticePlayerOperation(&proMsg)
	}

	return mateProto.Err_Success
}

// 碰
func (this *XueZhanDaoDiPlayLogic) OnPeng(pro *mateProto.MessageMaTe) int32 {
	playerID := qpTable.PlayerID(pro.SenderID)

	operationPeng := gameMaJiang.CS_Peng{}
	err := json.Unmarshal(pro.Data, &operationPeng)
	if err != nil {
		return mateProto.Err_ProtocolDataErr
	}

	pengPaiSeat, errCode := this.CheckOperation(playerID, operationPeng.OperationID)
	if errCode != mateProto.Err_Success {
		return errCode
	}

	seatData := pengPaiSeat.GetSeatData()
	mjSeatData := pengPaiSeat.GetXSeatData(0).(*gameMaJiang.MJSeat)
	if (mjSeatData.OperationItem & gameMaJiang.OPI_PENG) != gameMaJiang.OPI_PENG {
		return mateProto.Err_OperationNotExist
	}
	// 是否是 碰 出的牌
	if operationPeng.Pai != this.CurPlayPai {
		return mateProto.Err_OperationParamErr
	}
	// 手牌 是否存在
	if mjSeatData.GetPaiCount(this.CurPlayPai) < 2 {
		return mateProto.Err_PaiNotExist
	}

	// 是否已经有人胡牌了,我却不想胡
	if len(this.sRoundHuSeatMap) > 0 && (mjSeatData.OperationItem&gameMaJiang.OPI_HU == gameMaJiang.OPI_HU) {
		return mateProto.Err_FindHuPlayer
	}

	// 操作成功
	seatData.CleanOperationID()
	mjSeatData.OperationItem = 0
	this.Table.OperateRec.PutPlayerStep(int32(seatData.Number), gameMaJiang.ID_Peng, &operationPeng)
	this.Table.GetBaseQPTable().GameTimer.RemoveBySeatNum(int32(seatData.Number))

	heightPriority := gameMaJiang.OPI_GANG | gameMaJiang.OPI_HU
	if this.FindPriorityOperation(seatData.Number, heightPriority) == true {
		// 操作寄存起来
		this.delayActiveFunc = func() {
			this.DoPeng(seatData, mjSeatData, &operationPeng)
		}
		pro.MsgBody = &protoGameBasic.JsonResponse{Status: 0}
		this.Table.SendMsgToSeat(seatData.Number, pro)
		return mateProto.Err_Success
	}
	// 已经有人胡牌了
	if len(this.sRoundHuSeatMap) > 0 {
		pro.MsgBody = &protoGameBasic.JsonResponse{Status: 0}
		this.Table.SendMsgToSeat(seatData.Number, pro)

		this.hadHu()
		return mateProto.Err_Success
	}

	this.DoPeng(seatData, mjSeatData, &operationPeng)
	return mateProto.Err_Success
}

func (this *XueZhanDaoDiPlayLogic) DoPeng(seatData *qpTable.SeatData, mjSeatData *gameMaJiang.MJSeat, pengPai *gameMaJiang.CS_Peng) {

	//commonDef.LOG_Info("playerID:=", seatData.Player.ID, ",table number:=", this.Table.GetTableNumber(), ",action:= Peng")

	playPaiMJSeat := this.Table.SeatArr[this.CurPlaySeatNum].GetXSeatData(0).(*gameMaJiang.MJSeat)
	playPaiMJSeat.PlayPai = playPaiMJSeat.PlayPai[:len(playPaiMJSeat.PlayPai)-1]

	this.CleanAllSeatOperation()

	mjSeatData.DeleteShouPai(pengPai.Pai)
	mjSeatData.DeleteShouPai(pengPai.Pai)

	// 记录 操作区域
	operInfo := &gameMaJiang.OperationPaiInfo{OperationPXItem: gameMaJiang.OPX_PENG,
		PlayPaiSeatNumber: int32(this.CurPlaySeatNum),
		PaiArr:            []int8{pengPai.Pai}}
	mjSeatData.OperationPai = append(mjSeatData.OperationPai, operInfo)

	// 广播
	this.Table.BroadCastGameEvent(gameMaJiang.ID_BroadcastPeng,
		&gameMaJiang.BroadcastPeng{
			SeatNumber:  int32(seatData.Number),
			Pai:         pengPai.Pai,
			PlaySeatNum: int32(this.CurPlaySeatNum)})

	oper := gameMaJiang.OPI_PlayPai
	if this.PaiMgr.GetTheRestOfPaiCount() > 0 &&
		this.IsZiMoGang1(this.Table.SeatArr[seatData.Number].(*XZDDSeat)) == true {
		oper |= gameMaJiang.OPI_GANG
	}

	// 通知 出牌
	mjSeatData.SetOperation(oper)

	proMsg := gameMaJiang.SC_NoticeOperation{SeatNumber: int32(seatData.Number),
		OperationID: seatData.GetOperationID(),
		Operation:   mjSeatData.OperationItem,
		Pai:         gameMaJiang.InvalidPai,
		GangPai:     mjSeatData.GangArr,
	}
	this.NoticePlayerOperation(&proMsg)

	this.CurPengSeatNum = seatData.Number
	this.delayActiveFunc = nil
	this.lastGangSeatNum = qpTable.INVALID_SEAT_NUMBER
}

// 杠
func (this *XueZhanDaoDiPlayLogic) OnGang(pro *mateProto.MessageMaTe) int32 {
	playerID := qpTable.PlayerID(pro.SenderID)

	operationGang := gameMaJiang.CS_Gang{}
	err := json.Unmarshal(pro.Data, &operationGang)
	if err != nil {
		return mateProto.Err_ProtocolDataErr
	}

	gangPaiSeat, errCode := this.CheckOperation(playerID, operationGang.OperationID)
	if errCode != mateProto.Err_Success {
		return errCode
	}

	seatData := gangPaiSeat.(*XZDDSeat)
	mjSeatData := seatData.MJSeat
	if (mjSeatData.OperationItem & gameMaJiang.OPI_GANG) != gameMaJiang.OPI_GANG {
		return mateProto.Err_OperationNotExist
	}

	if len(this.sRoundHuSeatMap) > 0 && (mjSeatData.OperationItem&gameMaJiang.OPI_HU == gameMaJiang.OPI_HU) {
		return mateProto.Err_FindHuPlayer
	}

	// 1: 手上3张,别人 打出 1张
	// 2: 自己4张
	// 3： 碰了一次, 自摸1张

	isFind := false
	for _, v := range mjSeatData.GangArr {
		if v == operationGang.Pai {
			isFind = true
			break
		}
	}
	if isFind == false {
		return mateProto.Err_CheckFailed
	}

	var gangType int32 = 0

	// 只能 明杠
	if this.CurPlayPai != gameMaJiang.InvalidPai && this.CurPengSeatNum == qpTable.INVALID_SEAT_NUMBER {
		// 是否是 杠 出的牌
		if operationGang.Pai != this.CurPlayPai {
			return mateProto.Err_OperationParamErr
		}
		// 手牌 是否存在
		if mjSeatData.GetPaiCount(this.CurPlayPai) != 3 {
			return mateProto.Err_PaiNotExist
		}
		gangType = 1
	} else if this.CurMoPaiSeatNum == seatData.MJSeat.SeatData.Number || this.CurPengSeatNum == seatData.MJSeat.SeatData.Number {
		// 手牌 是否存在
		if mjSeatData.GetPaiCount(operationGang.Pai) == 4 {
			gangType = 2
		} else if mjSeatData.GetPaiCount(operationGang.Pai) == 1 {
			for _, v := range mjSeatData.OperationPai {
				if v.OperationPXItem == gameMaJiang.OPX_PENG &&
					v.PaiArr[0] == operationGang.Pai {
					gangType = 3
					break
				}
			}
			if gangType == 0 {
				return mateProto.Err_OperationParamErr
			}
		} else {
			return mateProto.Err_PaiNotExist
		}
	} else {
		return mateProto.Err_OperationParamErr
	}

	// 操作成功
	seatData.MJSeat.SeatData.CleanOperationID()
	mjSeatData.OperationItem = 0
	this.Table.OperateRec.PutPlayerStep(int32(seatData.MJSeat.SeatData.Number), gameMaJiang.ID_Gang, &operationGang)
	this.Table.GetBaseQPTable().GameTimer.RemoveBySeatNum(int32(seatData.MJSeat.SeatData.Number))

	heightPriority := gameMaJiang.OPI_HU
	if gangType == 1 && this.FindPriorityOperation(seatData.MJSeat.SeatData.Number, heightPriority) == true {
		// 操作寄存起来
		this.delayActiveFunc = func() {
			this.DoGang(seatData, &operationGang, gangType)
		}
		pro.MsgBody = &protoGameBasic.JsonResponse{Status: 0}
		this.Table.SendMsgToSeat(seatData.MJSeat.SeatData.Number, pro)
		return mateProto.Err_Success
	}

	// 已经有人胡牌了
	if len(this.sRoundHuSeatMap) > 0 {
		pro.MsgBody = &protoGameBasic.JsonResponse{Status: 0}
		this.Table.SendMsgToSeat(seatData.MJSeat.SeatData.Number, pro)

		this.hadHu()
		return mateProto.Err_Success
	}

	if gangType == 3 {
		// 广播
		brodcastGang := gameMaJiang.BroadcastGang{
			SeatNumber:  int32(seatData.MJSeat.SeatData.Number),
			Type:        gangType,
			PlaySeatNum: int32(this.CurPlaySeatNum),
			Pai:         operationGang.Pai,
		}
		this.Table.BroadCastGameEvent(gameMaJiang.ID_BroadcastGang, &brodcastGang)

		{
			// 删除手牌
			mjSeatData.DeleteShouPai(operationGang.Pai)

			// 更新 操作区域
			for _, v := range mjSeatData.OperationPai {
				if v.OperationPXItem == gameMaJiang.OPX_PENG && v.PaiArr[0] == operationGang.Pai {
					v.OperationPXItem = gameMaJiang.OPX_BU_GANG
					break
				}
			}
		}

		// 是否存在 抢杠胡
		if this.findQiangGangHu(seatData.MJSeat.SeatData.Number, operationGang.Pai) == true {
			// 操作寄存起来
			this.delayActiveFunc = func() {
				this.DoGang(seatData, &operationGang, gangType)
			}
			return mateProto.Err_Success
		}
	}

	this.DoGang(seatData, &operationGang, gangType)

	return mateProto.Err_Success
}

func (this *XueZhanDaoDiPlayLogic) DoGang(seatData *XZDDSeat, gangPai *gameMaJiang.CS_Gang, gangType int32) {

	//commonDef.LOG_Info("playerID:=", seatData.MJSeat.SeatData.Player.ID, ",table number:=", this.Table.GetTableNumber(), ",action:= Gang")

	seatData.MJSeat.GangCount += 1
	this.lastGangSeatNum = seatData.MJSeat.SeatData.Number
	seatData.DGHPlayPaiSeat = qpTable.INVALID_SEAT_NUMBER

	this.CleanAllSeatOperation()

	gangScore := float64(0)

	// 补杠 是直接广播的
	if gangType != 3 {
		brodcastGang := gameMaJiang.BroadcastGang{
			SeatNumber:  int32(seatData.MJSeat.SeatData.Number),
			Type:        gangType,
			PlaySeatNum: int32(this.CurPlaySeatNum),
			Pai:         gangPai.Pai,
		}
		// 广播
		this.Table.BroadCastGameEvent(gameMaJiang.ID_BroadcastGang, &brodcastGang)
	}

	rec := qpTable.GameScoreRec{}

	if gangType == 1 {
		rec.Category = gameMaJiang.MingGang
		playPaiMJSeat := this.Table.SeatArr[this.CurPlaySeatNum].GetXSeatData(0).(*gameMaJiang.MJSeat)
		playPaiMJSeat.PlayPai = playPaiMJSeat.PlayPai[:len(playPaiMJSeat.PlayPai)-1]
		rec.BeiShu = 2

		// 删除手牌
		seatData.MJSeat.DeleteShouPai(gangPai.Pai)
		seatData.MJSeat.DeleteShouPai(gangPai.Pai)
		seatData.MJSeat.DeleteShouPai(gangPai.Pai)

		// 记录 操作区域
		operInfo := &gameMaJiang.OperationPaiInfo{OperationPXItem: gameMaJiang.OPX_MING_GANG,
			PlayPaiSeatNumber: int32(this.CurPlaySeatNum),
			PaiArr:            []int8{gangPai.Pai}}
		seatData.MJSeat.OperationPai = append(seatData.MJSeat.OperationPai, operInfo)

		seatData.DGHPlayPaiSeat = this.CurPlaySeatNum
		gangScore = 2
	} else if gangType == 2 {
		rec.Category = gameMaJiang.AnGang
		rec.BeiShu = 2

		// 删除手牌
		seatData.MJSeat.DeleteShouPai(gangPai.Pai)
		seatData.MJSeat.DeleteShouPai(gangPai.Pai)
		seatData.MJSeat.DeleteShouPai(gangPai.Pai)
		seatData.MJSeat.DeleteShouPai(gangPai.Pai)

		// 记录 操作区域
		operInfo := &gameMaJiang.OperationPaiInfo{OperationPXItem: gameMaJiang.OPX_AN_GANG,
			PaiArr: []int8{gangPai.Pai}}
		seatData.MJSeat.OperationPai = append(seatData.MJSeat.OperationPai, operInfo)

		gangScore = 2
	} else if gangType == 3 {
		rec.Category = gameMaJiang.BuGang
		rec.BeiShu = 1

		if this.CurMoPai == gangPai.Pai {
			gangScore = 1
		}
	}

	gangScore *= this.PlayRule.MultipleFloat64
	rec.Score = gangScore

	// 算 杠分
	if gangType == 1 {
		seatData.LastGangScore = gangScore
		this.changeGangScore2(seatData.MJSeat.SeatData.Number, this.CurPlaySeatNum, &rec)
		if this.PlayRule.IsCaiGua == true {
			rec.Category = gameMaJiang.CaGua
			rec.Score /= 2
			rec.BeiShu = 1
			seatData.LastGangScore +=
				this.changeGangScore(seatData.MJSeat.SeatData.Number, this.CurPlaySeatNum, &rec)
		}
	} else {
		seatData.LastGangScore = this.changeGangScore(seatData.MJSeat.SeatData.Number, qpTable.INVALID_SEAT_NUMBER, &rec)
	}

	// 通知 摸牌
	if this.PaiMgr.GetTheRestOfPaiCount() < 1 {
		this.RoundOverFunc()
	} else {
		this.MoPaiOperation(seatData.MJSeat.SeatData.Number)
	}

	this.delayActiveFunc = nil
}

// 过
func (this *XueZhanDaoDiPlayLogic) OnGuo(pro *mateProto.MessageMaTe) int32 {
	playerID := qpTable.PlayerID(pro.SenderID)

	operationGuo := gameMaJiang.CS_Guo{}
	err := json.Unmarshal(pro.Data, &operationGuo)
	if err != nil {
		return mateProto.Err_ProtocolDataErr
	}

	guoSeat, errCode := this.CheckOperation(playerID, operationGuo.OperationID)
	if errCode != mateProto.Err_Success {
		return errCode
	}

	seatData := guoSeat.GetSeatData()
	mjSeatData := guoSeat.GetXSeatData(0).(*gameMaJiang.MJSeat)

	const operationCode = gameMaJiang.OPI_CHI | gameMaJiang.OPI_PENG | gameMaJiang.OPI_GANG | gameMaJiang.OPI_HU
	if (mjSeatData.OperationItem & operationCode) == 0 {
		return mateProto.Err_OperationNotExist
	}

	if mjSeatData.OperationItem&gameMaJiang.OPI_HU == gameMaJiang.OPI_HU {
		if this.CurPlaySeatNum != qpTable.INVALID_SEAT_NUMBER {
			guoSeat.(*XZDDSeat).IsGuoHu = true
			guoSeat.(*XZDDSeat).GuoHuFengDing = guoSeat.(*XZDDSeat).HuFanShu
		}
	}

	seatData.CleanOperationID()
	mjSeatData.OperationItem = 0
	this.Table.OperateRec.PutPlayerStep(int32(seatData.Number), gameMaJiang.ID_Guo, &operationGuo)
	this.Table.GetBaseQPTable().GameTimer.RemoveBySeatNum(int32(seatData.Number))

	// 碰后 可杠 取消
	if this.CurPengSeatNum == seatData.Number {
		// 通知 出牌
		mjSeatData.SetOperation(gameMaJiang.OPI_PlayPai)

		proMsg := gameMaJiang.SC_NoticeOperation{SeatNumber: int32(seatData.Number),
			OperationID: seatData.GetOperationID(),
			Operation:   mjSeatData.OperationItem,
			Pai:         gameMaJiang.InvalidPai,
			GangPai:     nil,
		}
		this.NoticePlayerOperation(&proMsg)

		mjSeatData.GangArr = nil
		this.CurPengSeatNum = qpTable.INVALID_SEAT_NUMBER
		return mateProto.Err_Success
	}

	// 自摸阶段 取消了操作
	if this.CurMoPaiSeatNum != qpTable.INVALID_SEAT_NUMBER {
		if this.CurMoPaiSeatNum == seatData.Number {
			mjSeatData.SetOperation(gameMaJiang.OPI_PlayPai)

			proMsg := gameMaJiang.SC_NoticeOperation{
				SeatNumber:  int32(seatData.Number),
				OperationID: seatData.GetOperationID(),
				Operation:   mjSeatData.OperationItem,
				Pai:         gameMaJiang.InvalidPai,
				GangPai:     nil,
			}
			this.NoticePlayerOperation(&proMsg)

			this.timerAutoTrusteeship(int32(seatData.Number))

			return mateProto.Err_Success
		}

		// 补杠
		if this.buGangSeatNum != qpTable.INVALID_SEAT_NUMBER {
			isEmpty := true
			for _, v := range this.Table.SeatArr {
				if v == nil {
					continue
				}
				if (v.GetXSeatData(0).(*gameMaJiang.MJSeat).OperationItem & gameMaJiang.OPI_HU) == gameMaJiang.OPI_HU {
					isEmpty = false
					break
				}
			}
			if isEmpty == true {
				if this.delayActiveFunc != nil {
					this.delayActiveFunc()
					return 0
				}
				this.MoPaiOperation(this.CurMoPaiSeatNum)
			}
			return 0
		}

		// 发生错误 todo
		return mateProto.Err_Success
	}

	// 出牌阶段, 是否还有人 未操作
	if this.FindPriorityOperation(seatData.Number, operationCode) == true {
		//if this.FindPriorityOperation(seatData.Number, gameMaJiang.OPI_HU) == true {
		//	return mateProto.Err_Success
		//}
		//if this.delayActiveFunc != nil {
		//	this.delayActiveFunc()
		//	return mateProto.Err_Success
		//}
		pro.Data = nil
		wrapMQ.ReplyToSource(pro, nil)
		return mateProto.Err_Success
	}

	// 出牌阶段, 没有玩家还有操作
	if this.delayActiveFunc != nil {
		this.delayActiveFunc()
		return mateProto.Err_Success
	}

	if this.isRoundOver() == true {
		this.RoundOverFunc()
		return mateProto.Err_Success
	}

	// 出牌阶段, 点炮多人胡牌
	if this.hadHu() > 0 {
		return mateProto.Err_Success
	}

	nextSeat := this.GetNextPlayingSeat(this.CurPlaySeatNum)
	this.MoPaiOperation(nextSeat.GetSeatData().Number)

	return mateProto.Err_Success
}

func (this *XueZhanDaoDiPlayLogic) OnDingQue(msg *mateProto.MessageMaTe) int32 {
	if this.Table.IsAssignTableState(TS_DingQue) == false {
		return mateProto.Err_ActionNotMatchStatus
	}

	seat := this.Table.GetSeatDataByPlayerID(qpTable.PlayerID(msg.SenderID))
	if seat == nil {
		return mateProto.Err_NotFindPlayer
	}

	if this.PlayRule.WanFa != 4 {
		return mateProto.Err_NotMatchTableRule
	}

	xzddSeat := seat.(*XZDDSeat)
	if xzddSeat.DingQue != -1 {
		return mateProto.Err_OperationRepeat
	}

	msgDingQue := CS_PlayerDingQue{}
	err := json.Unmarshal(msg.Data, &msgDingQue)
	if err != nil {
		return mateProto.Err_OperationParamErr
	}

	if msgDingQue.Value < gameMaJiang.MinHuaSe || msgDingQue.Value >= gameMaJiang.MaxHuaSe {
		return mateProto.Err_OperationParamErr
	}

	this.Table.OperateRec.PutPlayerStep(int32(seat.GetSeatData().Number), ID_PlayerDingQue, &msgDingQue)

	seat.(*XZDDSeat).DingQue = msgDingQue.Value

	this.Table.SendToAllPlayer(ID_BroadcastDingQueFinish, &CS_BroadcastDingQueFinish{int32(xzddSeat.MJSeat.SeatData.Number)})

	msgBroadcastDingQue := SC_BroadcastPlayerDingQue{make([]DingQueValue, 0, 4)}
	// 是否 所有玩家 选漂
	players := int32(0)
	for _, v := range this.Table.SeatArr {
		if v == nil {
			continue
		}
		if v.(*XZDDSeat).MJSeat.SeatData.IsAssignSeatState(qpTable.SS_Playing) == false {
			continue
		}
		if v.(*XZDDSeat).DingQue != -1 {
			players += 1
		}
		msgBroadcastDingQue.SeatArr = append(msgBroadcastDingQue.SeatArr,
			DingQueValue{SeatNum: int32(v.GetSeatData().Number), Value: v.(*XZDDSeat).DingQue})
	}
	if players != this.Table.GetCurSeatCount()-this.Table.LookerCount {
		return 0
	}

	this.Table.GameTimer.RemoveByTimeID(TIMER_DingQue)
	this.Table.DelTableState(TS_DingQue)
	this.Table.BroadCastGameEvent(ID_BroadcastPlayerDingQue, &msgBroadcastDingQue)

	this.Table.GameTimer.PutTableTimer(TIMER_PlayPai, 1200, func() {
		this.MoPaiOperation(this.BankerSeatNum)
	})

	return 0
}

func (this *XueZhanDaoDiPlayLogic) findQiangGangHu(gangSeatNum qpTable.SeatNumber, gangPai int8) bool {

	this.buGangSeatNum = gangSeatNum
	this.buGangPai = gangPai

	isFindHu := false
	for _, v := range this.Table.SeatArr {
		if v == nil {
			continue
		}
		if v.GetSeatData().Number == gangSeatNum {
			continue
		}
		mjSeat := v.GetXSeatData(0).(*gameMaJiang.MJSeat)

		var oper gameMaJiang.PlayerMJOperation
		if this.IsDianPaoHu(v.(*XZDDSeat), gangPai) == true {
			oper |= gameMaJiang.OPI_HU
		}
		if oper == 0 {
			continue
		}

		mjSeat.SetOperation(oper)
		isFindHu = true

		// 通知 操作
		proMsg := gameMaJiang.SC_NoticeOperation{
			SeatNumber:  int32(mjSeat.SeatData.Number),
			OperationID: mjSeat.SeatData.GetOperationID(),
			Operation:   mjSeat.OperationItem,
			Pai:         gangPai,
		}
		this.NoticePlayerOperation(&proMsg)
	}

	if isFindHu == true {
		return true
	}

	return false
}

// 清理所有座位操作
func (this *XueZhanDaoDiPlayLogic) CleanAllSeatOperation() {

	for _, v := range this.Table.SeatArr {
		if v == nil {
			continue
		}
		v.GetSeatData().CleanOperationID()
		mjSeat := v.GetXSeatData(0).(*gameMaJiang.MJSeat)
		mjSeat.OperationItem = 0
		this.Table.GameTimer.RemoveBySeatNum(int32(v.GetSeatData().Number))
	}

	this.delayActiveFunc = nil
	this.buGangSeatNum = qpTable.INVALID_SEAT_NUMBER
	this.buGangPai = gameMaJiang.InvalidPai
}

// 是否有更高优先级的操作
func (this *XueZhanDaoDiPlayLogic) FindPriorityOperation(excludeSeatNum qpTable.SeatNumber, oper gameMaJiang.PlayerMJOperation) bool {
	for _, v := range this.Table.SeatArr {
		if v == nil || v.GetSeatData().Number == excludeSeatNum {
			continue
		}
		if (v.GetXSeatData(0).(*gameMaJiang.MJSeat).OperationItem & oper) != 0 {
			return true
		}
	}
	return false
}

func (this *XueZhanDaoDiPlayLogic) moPai(number qpTable.SeatNumber) {
	this.buGangSeatNum = qpTable.INVALID_SEAT_NUMBER
	this.buGangPai = gameMaJiang.InvalidPai

	xzddSeat := this.Table.SeatArr[number].(*XZDDSeat)
	mjSeat := xzddSeat.MJSeat
	xzddSeat.IsGuoHu, xzddSeat.GuoHuFengDing = false, 0
	xzddSeat.MoPaiCount += 1

	var moPai int8
	if mjSeat.CustomNextPai == gameMaJiang.InvalidPai {
		moPai = this.PaiMgr.GetPai()
	} else {
		moPai = this.PaiMgr.(*xzddPaiMgr).GetNextPai(mjSeat.CustomNextPai)
		mjSeat.CustomNextPai = gameMaJiang.InvalidPai
	}
	mjSeat.PushShouPai(moPai)
	mjSeat.CurMoPai = moPai

	// 更新 桌子数据
	this.CurMoPaiSeatNum = number
	this.CurMoPai = moPai
	this.CurPlayPai = gameMaJiang.InvalidPai
	this.CurPlaySeatNum = qpTable.INVALID_SEAT_NUMBER
	this.CurPengSeatNum = qpTable.INVALID_SEAT_NUMBER
	if number != this.lastGangSeatNum && this.lastGangSeatNum != qpTable.INVALID_SEAT_NUMBER {
		this.Table.SeatArr[this.lastGangSeatNum].(*XZDDSeat).LastGangScore = 0
		this.lastGangSeatNum = qpTable.INVALID_SEAT_NUMBER
	}
	this.delayActiveFunc = nil
	this.OperationTime = time.Now().Unix()
	this.sRoundHuSeatMap = nil
	this.TempHuPlayerData.f, this.TempHuPlayerData.dianPaoSeat = nil, qpTable.INVALID_SEAT_NUMBER
	this.TempHuPlayerData.winSeat = qpTable.INVALID_SEAT_NUMBER

	// 广播 摸牌
	broadMoPai := gameMaJiang.BroadcastMoPai{SeatNumber: int32(number),
		CardCount: this.PaiMgr.GetTheRestOfPaiCount()}
	this.Table.BroadCastGameEvent(gameMaJiang.ID_BroadcastMoPai, &broadMoPai)
}

// 摸牌
func (this *XueZhanDaoDiPlayLogic) qiShouMoPaiOperation(number qpTable.SeatNumber) {

	if this.PlayRule.ChangePai != 0 {
		this.Table.AppendTableState(TS_ChangePai)
		for _, v := range this.Table.SeatArr {
			if v == nil {
				continue
			}
			if v.GetSeatData().IsAssignSeatState(qpTable.SS_Playing) == false {
				continue
			}
			v.(*XZDDSeat).ReadyChangePai = v.(*XZDDSeat).GetAutoHuanPai(this.PlayRule.ChangePai)
			t := SC_NoticeChangePai{Pai: v.(*XZDDSeat).ReadyChangePai}
			this.Table.SendGameEventToSeat(v.(*XZDDSeat).MJSeat.SeatData.Number, ID_NoticeChangePai, &t)
		}
		this.OperationTime = time.Now().Unix()

		this.Table.GameTimer.PutTableTimer(TIMER_ChanePai, 10*1000, this.timerChangePai)
		return
	}

	if this.PlayRule.WanFa == 4 {
		this.Table.AppendTableState(TS_DingQue)
		for _, v := range this.Table.SeatArr {
			if v == nil {
				continue
			}
			if v.GetSeatData().IsAssignSeatState(qpTable.SS_Playing) == false {
				continue
			}
			v.(*XZDDSeat).ReadyDingQue = v.(*XZDDSeat).GetAutoDingQue()
			t := SC_NoticeDingQue{Pai: v.(*XZDDSeat).ReadyDingQue}
			this.Table.SendGameEventToSeat(v.(*XZDDSeat).MJSeat.SeatData.Number, ID_NoticeDingQue, &t)
		}
		this.OperationTime = time.Now().Unix()

		this.Table.GameTimer.PutTableTimer(TIMER_DingQue, 10*1000, this.timerDingQue)
		return
	}

	this.MoPaiOperation(number)
}

func (this *XueZhanDaoDiPlayLogic) MoPaiOperation(number qpTable.SeatNumber) {

	this.moPai(number)

	xzddSeat := this.Table.SeatArr[number].(*XZDDSeat)
	oper := this.getZiMoOperation(xzddSeat)
	xzddSeat.MJSeat.SetOperation(oper)

	// 通知 操作
	proMoPai := gameMaJiang.SC_PlayerMoPai{Card: this.CurMoPai,
		OperationID: xzddSeat.MJSeat.SeatData.OperationID,
		Operation:   oper,
		GangArr:     xzddSeat.MJSeat.GangArr}
	this.Table.SendGameEventToSeat(number, gameMaJiang.ID_PlayerMoPai, &proMoPai)

	this.timerAutoTrusteeship(int32(number))
}

// 自摸操作
func (this *XueZhanDaoDiPlayLogic) getZiMoOperation(kwxSeat *XZDDSeat) gameMaJiang.PlayerMJOperation {

	oper := gameMaJiang.OPI_PlayPai | gameMaJiang.OPI_MO_Pai

	if kwxSeat.GetDingQuePaiCount() < 1 && this.IsZiMoHu(kwxSeat) == true {
		oper |= gameMaJiang.OPI_HU
	}
	if this.PaiMgr.GetTheRestOfPaiCount() > 0 &&
		this.IsZiMoGang1(kwxSeat) == true {
		oper |= gameMaJiang.OPI_GANG
	}

	return oper
}

// 玩家出牌,其它人操作
func (this *XueZhanDaoDiPlayLogic) GetPlayPaiOperation(playSeatNum qpTable.SeatNumber, playPai int8) ([]*XZDDSeat, bool) {

	findHu := false
	operSeat := make([]*XZDDSeat, 0)
	for i, v := range this.Table.SeatArr {
		if v == nil {
			continue
		}
		seat := v.GetSeatData()
		if seat.Number == playSeatNum {
			continue
		}
		if seat.IsContainSeatState(SS_HU|qpTable.SS_Looker) == true {
			continue
		}

		mjSeat := v.GetXSeatData(0).(*gameMaJiang.MJSeat)

		var oper gameMaJiang.PlayerMJOperation
		if v.(*XZDDSeat).GetDingQuePaiCount() < 1 && this.IsDianPaoHu(v.(*XZDDSeat), playPai) == true {
			if v.(*XZDDSeat).IsGuoHu == false || v.(*XZDDSeat).HuFanShu > v.(*XZDDSeat).GuoHuFengDing {
				oper |= gameMaJiang.OPI_HU
				findHu = true
			}
		}
		if this.IsPeng(v.(*XZDDSeat), mjSeat.ShouPai, playPai) == true {
			if playPai>>4 != v.(*XZDDSeat).DingQue {
				oper |= gameMaJiang.OPI_PENG
			}
		}
		if this.PaiMgr.GetTheRestOfPaiCount() > 0 &&
			this.IsMingGang(v.(*XZDDSeat), playPai) == true {
			if playPai>>4 != v.(*XZDDSeat).DingQue {
				oper |= gameMaJiang.OPI_GANG
			}
		}

		if oper != 0 {
			mjSeat.SetOperation(oper)
			operSeat = append(operSeat, this.Table.SeatArr[i].(*XZDDSeat))
		}
	}

	return operSeat, findHu
}

// 检查操作
func (this *XueZhanDaoDiPlayLogic) CheckOperation(playerID qpTable.PlayerID, operationID string) (qpTable.QPSeat, int32) {
	seat := this.Table.GetSeatDataByPlayerID(playerID)
	if seat == nil {
		return nil, mateProto.Err_NotFindPlayer
	}
	if len(seat.GetSeatData().OperationID) < 1 {
		return nil, mateProto.Err_OperationIDErr
	}

	if seat.GetSeatData().OperationID != operationID {
		return nil, mateProto.Err_OperationIDErr
	}
	return seat, mateProto.Err_Success
}

// 定时托管 出牌
func (this *XueZhanDaoDiPlayLogic) timerAutoTrusteeship(seatNum int32) {
	//if this.Table.TableRule.TuoGuanTime < 1 {
	//	return
	//}
	//
	//tempTime := this.Table.TableRule.TuoGuanTime
	//if this.Table.SeatArr[seatNum].GetSeatData().IsAssignSeatState(qpTable.SS_Trusteeship) == true {
	//	tempTime = 1
	//}
	tempTime := this.Table.TableRule.TuoGuanTime * 1000

	if this.Table.TableRule.TuoGuanTime < 1 {
		// 是否 主动开启了 托管
		if this.Table.SeatArr[seatNum].GetSeatData().IsAssignSeatState(qpTable.SS_Trusteeship) == false {
			return
		}
		tempTime = 800
	} else {
		if this.Table.SeatArr[seatNum].GetSeatData().IsAssignSeatState(qpTable.SS_Trusteeship) == true {
			tempTime = 800
		}
	}

	this.Table.GetBaseQPTable().GameTimer.PutSeatTimer(
		seatNum,
		TIMER_PlayPai,
		tempTime, func() {
			this.autoTrusteeshipPlayPai(seatNum)
		})
}

func (this *XueZhanDaoDiPlayLogic) timerChangePai() {
	if this.Table.IsAssignTableState(TS_ChangePai) == false {
		return
	}

	for _, v := range this.Table.SeatArr {
		if v == nil {
			continue
		}
		xzddSeat := v.(*XZDDSeat)
		if xzddSeat.MJSeat.SeatData.IsAssignSeatState(qpTable.SS_Playing) == false {
			continue
		}
		if xzddSeat.ChangePai != nil {
			continue
		}

		msg := mateProto.MessageMaTe{
			SenderID:  int64(xzddSeat.MJSeat.SeatData.Player.ID),
			MessageID: ID_PlayerChanePai}
		msg.Data, _ = json.Marshal(&CS_PlayerChanePai{xzddSeat.ReadyChangePai})

		rspCode := this.Table.RootTable.OnMessage(&msg)
		if rspCode != 0 {
			glog.Warning("timerChangePai() rspCode:=", rspCode)
		}
	}
}

func (this *XueZhanDaoDiPlayLogic) timerDingQue() {
	if this.Table.IsAssignTableState(TS_DingQue) == false {
		return
	}

	for _, v := range this.Table.SeatArr {
		if v == nil {
			continue
		}
		xzddSeat := v.(*XZDDSeat)
		if xzddSeat.MJSeat.SeatData.IsAssignSeatState(qpTable.SS_Playing) == false {
			continue
		}
		if xzddSeat.DingQue >= gameMaJiang.MinHuaSe {
			continue
		}

		msg := mateProto.MessageMaTe{MessageID: ID_PlayerDingQue, SenderID: int64(xzddSeat.MJSeat.SeatData.Player.ID)}
		msg.Data, _ = json.Marshal(&CS_PlayerDingQue{xzddSeat.ReadyDingQue})

		rspCode := this.Table.RootTable.OnMessage(&msg)
		if rspCode != 0 {
			glog.Warning("timerDingQue() rspCode:=", rspCode)
		}
	}
}

func (this *XueZhanDaoDiPlayLogic) autoTrusteeshipPlayPai(seatNum int32) {
	seat := this.Table.SeatArr[seatNum]

	if seat.GetSeatData().IsAssignSeatState(qpTable.SS_Trusteeship) == false {
		seat.GetSeatData().AppendState(qpTable.SS_Trusteeship)
		this.Table.NotifyPlayerStateChange(seat.GetSeatData().Number)
	}

	mjSeat := seat.GetXSeatData(0).(*gameMaJiang.MJSeat)

	msg := mateProto.MessageMaTe{SenderID: int64(seat.GetSeatData().Player.ID)}

	if (mjSeat.OperationItem & gameMaJiang.OPI_HU) == gameMaJiang.OPI_HU {
		msgBodyGuo := gameMaJiang.CS_Guo{OperationID: seat.GetSeatData().OperationID}
		msg.Data, _ = json.Marshal(&msgBodyGuo)
		msg.MessageID = gameMaJiang.ID_Guo
	} else if (mjSeat.OperationItem & gameMaJiang.OPI_PENG) == gameMaJiang.OPI_PENG {
		msgBodyGuo := gameMaJiang.CS_Guo{OperationID: seat.GetSeatData().OperationID}
		msg.Data, _ = json.Marshal(&msgBodyGuo)
		msg.MessageID = gameMaJiang.ID_Guo
	} else if (mjSeat.OperationItem & gameMaJiang.OPI_GANG) == gameMaJiang.OPI_GANG {
		msgBodyGuo := gameMaJiang.CS_Guo{OperationID: seat.GetSeatData().OperationID}
		msg.Data, _ = json.Marshal(&msgBodyGuo)
		msg.MessageID = gameMaJiang.ID_Guo
	} else if (mjSeat.OperationItem & gameMaJiang.OPI_PlayPai) == gameMaJiang.OPI_PlayPai {

		chuPai := gameMaJiang.InvalidPai
		shouPaiArr := mjSeat.GetShouPai()
		xzSeat := seat.(*XZDDSeat)
		if xzSeat.DingQue >= 0 && mjSeat.ShouPai[xzSeat.DingQue][0] > 0 {
			for i := gameMaJiang.MinDianShu_1; i <= gameMaJiang.MaxDianShu_9; i++ {
				if mjSeat.ShouPai[xzSeat.DingQue][i] > 0 {
					chuPai = xzSeat.DingQue*0x10 | i
					break
				}
			}
		}

		msgBodyPlay := gameMaJiang.CS_Play{OperationID: seat.GetSeatData().OperationID, Pai: shouPaiArr[0]}

		if this.CurMoPaiSeatNum == qpTable.SeatNumber(seatNum) {
			msgBodyPlay.Pai = this.CurMoPai
		}
		if chuPai != gameMaJiang.InvalidPai {
			msgBodyPlay.Pai = chuPai
		}

		msg.Data, _ = json.Marshal(&msgBodyPlay)
		msg.MessageID = gameMaJiang.ID_Play
	} else {
		glog.Warning("not match operationItem. value:=", mjSeat.OperationItem)
		return
	}

	rspCode := this.Table.RootTable.OnMessage(&msg)
	if rspCode != 0 {
		glog.Warning("autoTrusteeshipPlayPai() rspCode:=", rspCode)
	}
}

func (this *XueZhanDaoDiPlayLogic) IsPeng(seat *XZDDSeat, shouPai [gameMaJiang.MaxHuaSe + 1][gameMaJiang.MaxDianShu_9 + 1]int8, pai int8) bool {
	huaSeIndex := pai >> 4
	if huaSeIndex == seat.DingQue {
		return false
	}
	if huaSeIndex == gameMaJiang.MaxHuaSe {
		return false
	}

	if shouPai[huaSeIndex][pai&0x0F] > 1 {
		return true
	}

	return false
}

func (this *XueZhanDaoDiPlayLogic) IsMingGang(seat *XZDDSeat, pai int8) bool {

	huaSeIndex := pai >> 4
	if huaSeIndex == seat.DingQue {
		return false
	}

	mjSeat := seat.GetXSeatData(0).(*gameMaJiang.MJSeat)
	mjSeat.GangArr = nil
	gangArr := make([]int8, 0, 4)

	if mjSeat.ShouPai[huaSeIndex][pai&0x0F] > 2 {
		gangArr = append(gangArr, pai)
	}

	mjSeat.GangArr = gangArr
	if len(mjSeat.GangArr) > 0 {
		return true
	}
	return false
}

func (this *XueZhanDaoDiPlayLogic) IsZiMoGang1(seat *XZDDSeat) bool {

	mjSeat := seat.GetXSeatData(0).(*gameMaJiang.MJSeat)
	mjSeat.GangArr = nil
	gangArr := make([]int8, 0, 4)

	for i := gameMaJiang.MinHuaSe; i < gameMaJiang.MaxHuaSe; i++ {
		if mjSeat.ShouPai[i][0] < 1 || i == seat.DingQue {
			continue
		}

		for j := gameMaJiang.MinDianShu_1; j <= gameMaJiang.MaxDianShu_9; j++ {
			if mjSeat.ShouPai[i][j] < 4 {
				continue
			}
			gangArr = append(gangArr, i*0x10+j)
		}
	}

	// 补杠
	for _, v := range mjSeat.OperationPai {
		if v.OperationPXItem != gameMaJiang.OPX_PENG {
			continue
		}
		pai := v.PaiArr[0]
		huaSeIndex := uint8(pai) >> 4
		if mjSeat.ShouPai[huaSeIndex][pai&0x0F] > 0 {
			gangArr = append(gangArr, pai)
		}
	}

	mjSeat.GangArr = gangArr
	if len(mjSeat.GangArr) > 0 {
		return true
	}
	return false
}

func (this *XueZhanDaoDiPlayLogic) IsZiMoHu(kwxSeat *XZDDSeat) bool {
	mjSeat := kwxSeat.GetXSeatData(0).(*gameMaJiang.MJSeat)
	if this.HuLogic.IsZiMoHu(kwxSeat, this.CurMoPai) == true {
		mjSeat.HuPaiXing = this.getPaiXing(kwxSeat)
		return true
	}
	return false
}

// isGang:杠后,出牌点炮
func (this *XueZhanDaoDiPlayLogic) IsDianPaoHu(kwxSeat *XZDDSeat, playPai int8) bool {

	mjSeat := kwxSeat.GetXSeatData(0).(*gameMaJiang.MJSeat)
	if this.HuLogic.IsDianPaoHu(kwxSeat, playPai) == false {
		return false
	}

	mjSeat.HuPaiXing = this.getPaiXing(kwxSeat)
	if len(mjSeat.HuPaiXing) == 1 &&
		mjSeat.HuPaiXing[0].PaiXing == HuPX_PingHu &&
		this.PlayRule.IsDianPaoPingHu == false {
		return false
	}

	return true
}

func (this *XueZhanDaoDiPlayLogic) getPaiXing(kwxSeat *XZDDSeat) []*gameMaJiang.HuPaiXing {

	mjSeat := kwxSeat.GetXSeatData(0).(*gameMaJiang.MJSeat)
	mjSeat.HuScore = 1
	paiXingArr := make([]*gameMaJiang.HuPaiXing, 0, 17)
	paiXingMap := make(map[int32]int64) // key: 牌型  番数

	if this.PlayRule.QingYiSeFS > 0 && this.HuLogic.isQingYiSe(&mjSeat.OperationPai) == true {
		paiXingMap[HuPX_QingYiSe] = int64(this.PlayRule.QingYiSeFS)
	}

	if this.HuLogic.is7Dui() == true {
		if this.HuLogic.isHaoHua7Dui() == true {
			paiXingMap[HuPX_HaoHuaQiDui] = 3
		} else {
			paiXingMap[HuPX_QiXiaoDui] = 2
		}
	}

	if this.HuLogic.isPengPengHu(&mjSeat.OperationPai) == true {
		paiXingMap[HuPX_PengPengHu] = 1
		if this.PlayRule.IsPengPengHux2 == true {
			paiXingMap[HuPX_PengPengHu] = 2
		}
	}

	if this.HuLogic.isJiangYiSe(&mjSeat.OperationPai) == true {
		paiXingMap[HuPX_JiangYiSe] = 2
	}

	if this.PlayRule.IsKaWuXing == true && this.HuLogic.isKaWuXing(this.CurMoPai, this.CurPlayPai) == true {
		paiXingMap[HuPX_KaWuXing] = 2
	}

	if this.PlayRule.Is19JiangDui == true && this.HuLogic.isYaoJiu(&mjSeat.OperationPai) == true {
		paiXingMap[HuPX_YaoJiu] = 2
	}

	if this.PlayRule.Is19JiangDui == true && this.HuLogic.isJiangDui(&mjSeat.OperationPai) == true {
		paiXingMap[HuPX_JiangDui] = 2
	}

	if this.lastGangSeatNum != qpTable.INVALID_SEAT_NUMBER &&
		kwxSeat.MJSeat.SeatData.Number != this.lastGangSeatNum &&
		this.CurPlaySeatNum == this.lastGangSeatNum {
		paiXingMap[HuPX_GangShangPao] = 1
	}

	if kwxSeat.MJSeat.SeatData.Number == this.lastGangSeatNum {
		paiXingMap[HuPX_GangShangHua] = 1
	}

	genCount := this.HuLogic.genCount(&mjSeat.OperationPai)
	if genCount > 0 {
		paiXingMap[HuPX_Gen] = genCount
	}

	if kwxSeat.MJSeat.SeatData.Number == this.CurMoPaiSeatNum &&
		this.PaiMgr.GetTheRestOfPaiCount() == 0 {
		paiXingMap[HuPX_HaiDiLao] = 1
	}
	if this.CurPlaySeatNum != qpTable.INVALID_SEAT_NUMBER &&
		this.PaiMgr.GetTheRestOfPaiCount() == 0 {
		paiXingMap[HuPX_HaiDiPao] = 1
	}

	if this.buGangSeatNum != qpTable.INVALID_SEAT_NUMBER &&
		kwxSeat.MJSeat.SeatData.Number != this.buGangSeatNum {
		paiXingMap[HuPX_QiangGangHu] = 1
	}

	if len(kwxSeat.MJSeat.GetShouPai()) <= 2 {
		paiXingMap[HuPX_ShouZhuaYi] = 1
	}

	if this.PlayRule.IsMenQingZhongZhang == true &&
		this.HuLogic.isMenQing(&mjSeat.OperationPai) == true {
		paiXingMap[HuPX_MengQing] = 1
	}
	if this.PlayRule.IsMenQingZhongZhang == true &&
		this.HuLogic.isZhongZhang(&mjSeat.OperationPai) == true {
		paiXingMap[HuPX_ZhongZhang] = 1
	}

	if this.PlayRule.IsTianDiHu == true {
		if this.BankerSeatNum != kwxSeat.MJSeat.SeatData.Number &&
			kwxSeat.MoPaiCount < 2 && kwxSeat.PlayPaiCount == 0 {
			paiXingMap[HuPX_DiHu] = 2
		} else if this.BankerSeatNum == this.CurMoPaiSeatNum &&
			kwxSeat.MoPaiCount == 1 && kwxSeat.PlayPaiCount == 0 {
			paiXingMap[HuPX_TianHu] = 3
		}
	}

	if len(paiXingMap) < 1 {
		paiXingMap[HuPX_PingHu] = 0
	}

	kwxSeat.HuFanShu = 0
	for k, v := range paiXingMap {
		paiXingArr = append(paiXingArr, &gameMaJiang.HuPaiXing{PaiXing: k, FanShu: v})
		kwxSeat.HuFanShu += v
	}

	if kwxSeat.HuFanShu > this.PlayRule.FengDingFanShu {
		kwxSeat.HuFanShu = this.PlayRule.FengDingFanShu
	}

	kwxSeat.FanXingMap = paiXingMap
	return paiXingArr
}

func (this *XueZhanDaoDiPlayLogic) changeGangScore(win, exclude qpTable.SeatNumber, rec *qpTable.GameScoreRec) float64 {
	if rec.Score < 0.1 {
		return 0
	}
	msgBody := protoGameBasic.BroadcastPlayerScoreChanged{
		WinnerSeatNum: int32(win),
		LoserSeatNum:  make([]int32, 0, 4),
		Score:         commonDef.Float64ToString(rec.Score),
		Category:      rec.Category}

	allScore := float64(0)
	giveSeat := []qpTable.SeatNumber{}
	for i, v := range this.Table.SeatArr {
		if v == nil {
			continue
		}
		if v.GetSeatData().IsAssignSeatState(qpTable.SS_Playing) == false {
			continue
		}
		if qpTable.SeatNumber(i) == win || qpTable.SeatNumber(i) == exclude {
			continue
		}
		if v.(*XZDDSeat).MJSeat.SeatData.IsAssignSeatState(SS_HU) == true {
			continue
		}
		giveSeat = append(giveSeat, qpTable.SeatNumber(i))
		rec.TargetUID = []qpTable.SeatNumber{win}
		v.(*XZDDSeat).GangScore -= rec.Score
		v.GetSeatData().RoundScore -= rec.Score
		v.GetSeatData().PutGameScoreItem(rec, -1)
		msgBody.LoserSeatNum = append(msgBody.LoserSeatNum, int32(v.GetSeatData().Number))

		allScore += rec.Score
		this.Table.SeatArr[win].(*XZDDSeat).GangScore += rec.Score
		this.Table.SeatArr[win].GetSeatData().RoundScore += rec.Score
	}
	rec.Score, rec.TargetUID = allScore, giveSeat
	this.Table.SeatArr[win].GetSeatData().PutGameScoreItem(rec, 1)

	this.Table.BroadCastGameEvent(protoGameBasic.ID_PlayerRoundScoreChanged, &msgBody)

	return allScore
}

func (this *XueZhanDaoDiPlayLogic) changeGangScore2(win, lose qpTable.SeatNumber, rec *qpTable.GameScoreRec) {
	if rec.Score < 0.1 {
		return
	}
	rec.TargetUID = []qpTable.SeatNumber{lose}
	this.Table.SeatArr[win].(*XZDDSeat).GangScore += rec.Score
	this.Table.SeatArr[win].GetSeatData().RoundScore += rec.Score
	this.Table.SeatArr[win].GetSeatData().PutGameScoreItem(rec, 1)

	rec.TargetUID = []qpTable.SeatNumber{win}
	this.Table.SeatArr[lose].(*XZDDSeat).GangScore -= rec.Score
	this.Table.SeatArr[lose].GetSeatData().RoundScore -= rec.Score
	this.Table.SeatArr[lose].GetSeatData().PutGameScoreItem(rec, -1)

	msgBody := protoGameBasic.BroadcastPlayerScoreChanged{
		WinnerSeatNum: int32(win),
		LoserSeatNum:  make([]int32, 0, 1),
		Score:         commonDef.Float64ToString(rec.Score),
		Category:      rec.Category}

	msgBody.LoserSeatNum = append(msgBody.LoserSeatNum, int32(lose))

	this.Table.BroadCastGameEvent(protoGameBasic.ID_PlayerRoundScoreChanged, &msgBody)
}

func (this *XueZhanDaoDiPlayLogic) NoticePlayerOperation(msgOperation *gameMaJiang.SC_NoticeOperation) {
	this.Table.SendGameEventToSeat(
		qpTable.SeatNumber(msgOperation.SeatNumber),
		gameMaJiang.ID_NoticeOperation,
		msgOperation)

	this.timerAutoTrusteeship(msgOperation.SeatNumber)
}

func (this *XueZhanDaoDiPlayLogic) onCustomNextPai(pro *mateProto.MessageMaTe) int32 {
	playerID := qpTable.PlayerID(pro.SenderID)

	customNextPai := CS_CustomNextPai{}
	err := json.Unmarshal(pro.Data, &customNextPai)
	if err != nil {
		return mateProto.Err_ProtocolDataErr
	}
	seat := this.Table.GetSeatDataByPlayerID(playerID)
	if seat == nil {
		return mateProto.Err_NotFindPlayer
	}

	powerMap, _ := db.GetPlayerPower(pro.SenderID)
	if powerMap == nil {
		return 0
	}

	if v, ok := powerMap[strconv.Itoa(int(this.Table.GameID))]; ok == false || v == 0 {
		return 0
	}

	if seat.GetSeatData().IsAssignSeatState(qpTable.SS_Playing) == false {
		return mateProto.Err_ActionNotMatchStatus
	}
	mjSeatData := seat.GetXSeatData(0).(*gameMaJiang.MJSeat)
	mjSeatData.CustomNextPai = customNextPai.Pai

	return 0
}

func (this *XueZhanDaoDiPlayLogic) onGetRemainingPai(pro *mateProto.MessageMaTe) int32 {
	playerID := qpTable.PlayerID(pro.SenderID)

	seat := this.Table.GetSeatDataByPlayerID(playerID)
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

	if v, ok := powerMap[strconv.Itoa(int(this.Table.GameID))]; ok == false || v == 0 {
		return 0
	}

	type PaiInfo struct {
		Pai   int8
		Count int8
	}
	arr := make([]PaiInfo, 0, 27)
	for k, v := range this.PaiMgr.(*xzddPaiMgr).GetRemainPai() {
		arr = append(arr, PaiInfo{Pai: k, Count: v})
	}

	wrapMQ.ReplyToSource(pro, arr)

	return 0
}

// :0-没发现  1:处理完  2:等待
func (this *XueZhanDaoDiPlayLogic) hadHu() int {
	if len(this.TempHuPlayerData.f) < 1 {
		return 0
	}
	for _, v := range this.Table.SeatArr {
		if v == nil {
			continue
		}
		if v.(*XZDDSeat).MJSeat.OperationItem&gameMaJiang.OPI_HU == gameMaJiang.OPI_HU {
			return 2
		}
	}

	for _, v := range this.TempHuPlayerData.f {
		v()
	}

	if this.TempHuPlayerData.dianPaoSeat != qpTable.INVALID_SEAT_NUMBER {
		this.findHuJiaoZhuanYi(this.TempHuPlayerData.dianPaoSeat)
	}

	// 自摸，点炮一人胡， 胡的下一个摸牌 ，  一炮多响  点炮人摸牌
	nextPlayer := this.GetNextPlayingSeat(this.TempHuPlayerData.winSeat).GetSeatData().Number
	if this.TempHuPlayerData.dianPaoSeat != qpTable.INVALID_SEAT_NUMBER {
		v, _ := this.dianPaoSeatMap[this.TempHuPlayerData.dianPaoSeat]
		if len(v[len(v)-1].SeatArr) > 1 {
			nextPlayer = this.TempHuPlayerData.dianPaoSeat
		}
	}

	this.huNextStep(nextPlayer)

	return 1
}

func (this *XueZhanDaoDiPlayLogic) findHuJiaoZhuanYi(dianPaoSeatNum qpTable.SeatNumber) {
	if this.PlayRule.IsHuJiaoZhuanYi == true &&
		this.lastGangSeatNum != qpTable.INVALID_SEAT_NUMBER &&
		this.lastGangSeatNum == this.CurPlaySeatNum {

		hjzyMsg := BroadcastHuJiaoZhuanYi{WinSeatArr: make([]LoseSeat, 0, 1)}
		zyRec := qpTable.GameScoreRec{Category: gameMaJiang.HuJiaoZhuanYi, BeiShu: 1, TargetUID: []qpTable.SeatNumber{}}

		var zhuanYi []qpTable.GameScoreRec
		gangSeat := this.Table.SeatArr[this.lastGangSeatNum].(*XZDDSeat)
		stepLen := len(gangSeat.MJSeat.SeatData.GameScoreRecStep)
		if stepLen > 2 && gangSeat.MJSeat.SeatData.GameScoreRecStep[stepLen-2].Category == gameMaJiang.CaGua {
			// 跳过 点炮 记录
			zhuanYi = gangSeat.MJSeat.SeatData.GameScoreRecStep[stepLen-3 : stepLen-1]
		} else if stepLen > 1 {
			zhuanYi = gangSeat.MJSeat.SeatData.GameScoreRecStep[stepLen-2 : stepLen-1]
		}

		groupWinnerArr, _ := this.dianPaoSeatMap[dianPaoSeatNum]
		lastGroup := groupWinnerArr[len(groupWinnerArr)-1]

		gangScore := gangSeat.LastGangScore / float64(len(lastGroup.SeatArr))
		for _, v := range lastGroup.SeatArr {
			hjzyMsg.WinSeatArr = append(hjzyMsg.WinSeatArr, LoseSeat{int32(v), commonDef.Float64ToString(gangScore)})
			this.Table.SeatArr[v].(*XZDDSeat).MJSeat.SeatData.RoundScore += gangScore

			gotRec := qpTable.GameScoreRec{Category: gameMaJiang.HuJiaoZhuanYi, BeiShu: 1, Score: gangScore}
			gotRec.PaiXing, gotRec.TargetUID = zhuanYi, []qpTable.SeatNumber{gangSeat.MJSeat.SeatData.Number}
			this.Table.SeatArr[v].(*XZDDSeat).MJSeat.SeatData.PutGameScoreItem(&gotRec, 1)
		}

		gangSeat.GangScore -= gangSeat.LastGangScore
		gangSeat.MJSeat.SeatData.RoundScore -= gangSeat.LastGangScore

		zyRec.PaiXing, zyRec.Score = zhuanYi, gangSeat.LastGangScore
		zyRec.TargetUID = lastGroup.SeatArr
		gangSeat.MJSeat.SeatData.PutGameScoreItem(&zyRec, -1)

		hjzyMsg.LoseSeatNum = int32(gangSeat.MJSeat.SeatData.Number)
		hjzyMsg.LoseScore = commonDef.Float64ToString(gangSeat.LastGangScore)
		this.Table.BroadCastGameEvent(ID_HuJiaoZhuanYi, &hjzyMsg)
	}
}

func (this *XueZhanDaoDiPlayLogic) huNextStep(nextPlayer qpTable.SeatNumber) {

	if this.isRoundOver() == true {
		this.RoundOverFunc()
		return
	}

	for _, v := range this.Table.SeatArr {
		if v == nil {
			continue
		}
		v.(*XZDDSeat).MJSeat.OperationItem = 0
		v.(*XZDDSeat).MJSeat.SeatData.CleanOperationID()
	}

	this.MoPaiOperation(nextPlayer)
}

// 激活托管
func (this *XueZhanDaoDiPlayLogic) OnActiveTrusteeship(msg *mateProto.MessageMaTe) int32 {
	seatNum := this.Table.OnActiveTrusteeship(msg)
	if seatNum < 0 {
		return seatNum
	}

	this.autoTrusteeshipPlayPai(seatNum)

	return seatNum
}

//func (this *XueZhanDaoDiPlayLogic) findPlayPai(seatNum qpTable.SeatNumber) int8 {
//
//	allLiangDaoMap := make(map[int8]int8)
//
//	for _, v := range this.Table.SeatArr {
//		if v == nil {
//			continue
//		}
//		for k, _ := range v.(*XZDDSeat).LiangDaoMap {
//			allLiangDaoMap[k] = k
//		}
//	}
//
//	mjSeat := this.Table.SeatArr[seatNum].GetXSeatData(0).(*gameMaJiang.MJSeat)
//
//	// 查找手里是否还有别人 不听的牌
//	prePlayPai := int8(0)
//	mjSeat.RangeShouPai(func(shouPai int8) bool {
//		if _, ok1 := allLiangDaoMap[shouPai]; ok1 == false {
//			prePlayPai = shouPai
//			return true
//		}
//		prePlayPai = shouPai
//		return false
//	})
//	return prePlayPai
//}
