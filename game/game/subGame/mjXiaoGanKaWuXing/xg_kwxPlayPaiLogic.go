package mjXiaoGanKWXTable

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
	HuPX_ShouZhuaYi     = 13
	HuPX_QiXiaoDui      = 14
	HuPX_DaSanYuan      = 15
	HuPX_HaoHuaQiDui    = 16
	HuPX_JiuLianBaoDeng = 17
	HuPX_HaiDiPao       = 18
)

type KaWuXingPlayLogic struct {
	PaiMgr   gameMaJiang.MJPaiMgr // 牌的管理器
	HuLogic  kwxHuLogic           // 逻辑
	Table    *qpTable.QPTable
	PlayRule *KWXPlayRule // 玩法规则

	RoundOverFunc func()

	BankerSeatNum qpTable.SeatNumber // 庄家座位号

	// 小局 待清理 成员
	liangDaoSeatNumMap   map[qpTable.SeatNumber]*KWXSeat // 亮倒的座位 key:座位号
	noLiangDaoSeatNumMap map[qpTable.SeatNumber]*KWXSeat // 未亮倒的座位 key:座位号

	llastGangSeatNum qpTable.SeatNumber            // 上次杠的座位号(记录杠上炮)
	dianPaoSeatNum   qpTable.SeatNumber            // 点炮座位号
	buGangSeatNum    qpTable.SeatNumber            // 补杠座位
	huSeatMap        map[int32]*gameMaJiang.MJSeat // 已经胡的玩家 key:座位号
	CurPlayPai       int8                          // 最近一次出的牌
	lastPlaySeatNum  qpTable.SeatNumber            // 最近一次出牌的座位号
	CurPlaySeatNum   qpTable.SeatNumber            // 当前出牌的座位号
	CurMoPaiSeatNum  qpTable.SeatNumber            // 当前摸牌座位号
	CurMoPai         int8                          // 当前摸的牌
	CurPengSeatNum   qpTable.SeatNumber            // 当前碰的座位号
	OperationTime    int64                         // 玩家操作起始点

	delayActiveFunc func()
}

// 清空每一小局数据
func (this *KaWuXingPlayLogic) CleanRoundData() {
	this.liangDaoSeatNumMap = make(map[qpTable.SeatNumber]*KWXSeat)
	this.noLiangDaoSeatNumMap = make(map[qpTable.SeatNumber]*KWXSeat)
	this.llastGangSeatNum = 0
	this.dianPaoSeatNum = qpTable.INVALID_SEAT_NUMBER
	this.buGangSeatNum = qpTable.INVALID_SEAT_NUMBER
	this.huSeatMap = make(map[int32]*gameMaJiang.MJSeat)
	this.CurPlayPai = gameMaJiang.InvalidPai
	this.lastPlaySeatNum = qpTable.INVALID_SEAT_NUMBER
	this.CurPlaySeatNum = qpTable.INVALID_SEAT_NUMBER
	this.CurMoPaiSeatNum = qpTable.INVALID_SEAT_NUMBER
	this.CurMoPai = gameMaJiang.InvalidPai
	this.CurPengSeatNum = qpTable.INVALID_SEAT_NUMBER

	this.delayActiveFunc = nil
}

// 事件处理
func (this *KaWuXingPlayLogic) OnMessage(pro *mateProto.MessageMaTe) int32 {

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
	case ID_LiangDao:
		rspCode = this.onLiang(pro)
	case ID_CustomNextPai:
		this.onCustomNextPai(pro)
	case ID_GetRemainingPai:
		this.onGetRemainingPai(pro)
	default:
		return this.Table.OnMessage(pro)
	}

	return rspCode
}

func (this *KaWuXingPlayLogic) OnGameStart(pro *mateProto.MessageMaTe) int32 {
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

	if this.PlayRule.XuanPiao == 100 {
		this.OperationTime = time.Now().Unix()
		this.Table.AppendTableState(TS_XuanPiao)
		this.Table.BroadCastGameEvent(ID_NoticeXuanPiao, nil)
		this.Table.GameTimer.PutTableTimer(TIMER_XuanPiao, 15*1000, func() {
			for _, v := range this.Table.SeatArr {
				if v == nil || v.(*KWXSeat).PiaoScore >= 0 {
					continue
				}

				xuanPiaoBody := CS_XuanPiao{Value: 0}
				jsonData, _ := json.Marshal(&xuanPiaoBody)
				msgXuanPiao := mateProto.MessageMaTe{
					SenderID:  int64(v.GetSeatData().Player.ID),
					MessageID: ID_PlayerXuanPiao,
					Data:      jsonData,
				}

				this.Table.RootTable.OnMessage(&msgXuanPiao)
			}
		})
		return 0
	}

	this.faShouPai()

	return mateProto.Err_Success
}

func (this *KaWuXingPlayLogic) faShouPai() {

	this.Table.GameTimer.RemoveByTimeID(TIMER_XuanPiao)
	this.Table.DelTableState(TS_XuanPiao)

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
			tempSeat = this.Table.GetNextPlayingSeat(this.BankerSeatNum)
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
		this.noLiangDaoSeatNumMap[seat.Number] = v.(*KWXSeat)

		if this.PlayRule.XuanPiao != 100 {
			v.(*KWXSeat).PiaoScore = this.PlayRule.XuanPiao
		}

		paiArr := this.PaiMgr.GetGroupPai(13)

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
		// 庄家摸牌
		this.MoPaiOperation(this.BankerSeatNum)
	})
}

// 出牌
func (this *KaWuXingPlayLogic) OnPlay(pro *mateProto.MessageMaTe) int32 {
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

	seatData := playSeat.GetSeatData()
	mjSeatData := playSeat.GetXSeatData(0).(*gameMaJiang.MJSeat)
	if (mjSeatData.OperationItem & gameMaJiang.OPI_PlayPai) != gameMaJiang.OPI_PlayPai {
		return mateProto.Err_OperationNotExist
	}

	if mjSeatData.GetPaiCount(playPai.Pai) < 1 {
		return mateProto.Err_PaiNotExist
	}

	// 没亮倒时 不能出 别人听的牌
	if len(playSeat.(*KWXSeat).LiangDaoMap) < 1 {
		allTingPaiMap := make(map[int8]int8)
		for i, v := range this.Table.SeatArr {
			if v == nil {
				continue
			}
			if i == int(seatData.Number) {
				continue
			}
			for k, _ := range v.(*KWXSeat).TingPaiMap {
				allTingPaiMap[k] = k
			}
		}
		// 出的牌是否 是别人的 听牌
		if _, ok := allTingPaiMap[playPai.Pai]; ok == true {
			// 查找手里是否还有别人 不听的牌
			ok = mjSeatData.RangeShouPai(func(shouPai int8) bool {
				if _, ok1 := allTingPaiMap[shouPai]; ok1 == false {
					return true
				}
				return false
			})
			if ok == false {
				return mateProto.Err_CustomPai
			}
		}

		// 扣牌不能出
		if playSeat.(*KWXSeat).KouMap != nil {
			if _, ok := playSeat.(*KWXSeat).KouMap[playPai.Pai]; ok == true {
				if mjSeatData.ShouPai[playPai.Pai>>4][playPai.Pai&0x0F] < 4 {
					glog.Warning("kouPai:=", playPai.Pai, ",count:=", mjSeatData.ShouPai[playPai.Pai>>4][playPai.Pai&0x0F])
					return mateProto.Err_CustomPai
				}
			}
		}
	}

	if mjSeatData.DeleteShouPai(playPai.Pai) == false {
		return mateProto.Err_PaiNotExist
	}

	gangCountBak := mjSeatData.LianGangCount
	// 座位数据 清理
	this.llastGangSeatNum = qpTable.INVALID_SEAT_NUMBER
	if mjSeatData.LianGangCount > 0 {
		this.llastGangSeatNum = seatData.Number
	}
	mjSeatData.OperationItem = 0
	seatData.CleanOperationID()
	mjSeatData.LianGangCount = 0
	mjSeatData.CurMoPai = gameMaJiang.InvalidPai
	this.Table.OperateRec.PutPlayerStep(int32(seatData.Number), gameMaJiang.ID_Play, &playPai)
	mjSeatData.PlayPai = append(mjSeatData.PlayPai, playPai.Pai)
	this.Table.GetBaseQPTable().GameTimer.RemoveBySeatNum(int32(seatData.Number))

	this.CurPengSeatNum = qpTable.INVALID_SEAT_NUMBER
	this.CurMoPaiSeatNum = qpTable.INVALID_SEAT_NUMBER
	this.CurMoPai = gameMaJiang.InvalidPai
	this.CurPlaySeatNum = seatData.Number
	this.lastPlaySeatNum = seatData.Number
	this.CurPlayPai = playPai.Pai
	this.buGangSeatNum = qpTable.INVALID_SEAT_NUMBER

	// 广播 出牌
	this.Table.BroadCastGameEvent(gameMaJiang.ID_BroadcastPlay,
		&gameMaJiang.BroadcastPlay{SeatNumber: int32(seatData.Number), Pai: playPai.Pai})

	canOperationSeatArr := this.GetPlayPaiOperation(this.CurPlaySeatNum, this.CurPlayPai, gangCountBak)
	//playSeat.(*KWXSeat).IsLiangDaoIng = false

	// 无人 可操作
	if canOperationSeatArr == nil || len(canOperationSeatArr) < 1 {

		if this.PaiMgr.GetTheRestOfPaiCount() < 1 {
			this.RoundOverFunc()
		} else {
			nextSeat := this.Table.GetNextPlayingSeat(this.CurPlaySeatNum)
			this.MoPaiOperation(nextSeat.GetSeatData().Number)
		}
		return mateProto.Err_Success
	}
	this.OperationTime = time.Now().Unix()

	// 记录 可杠操作
	this.delayActiveFunc = nil
	liangDaoHuFunc := make([]func(), 0)

	// 通知 操作
	for i, _ := range canOperationSeatArr {
		// 非亮倒时
		if canOperationSeatArr[i].LiangDaoMap == nil {
			proMsg := gameMaJiang.SC_NoticeOperation{SeatNumber: int32(canOperationSeatArr[i].GetSeatData().Number),
				OperationID: canOperationSeatArr[i].GetSeatData().GetOperationID(),
				Operation:   canOperationSeatArr[i].GetXSeatData(0).(*gameMaJiang.MJSeat).OperationItem,
				Pai:         playPai.Pai,
				GangPai:     canOperationSeatArr[i].GetXSeatData(0).(*gameMaJiang.MJSeat).GangArr,
			}
			this.NoticePlayerOperation(&proMsg)
			continue
		}

		if (canOperationSeatArr[i].MJSeat.OperationItem & gameMaJiang.OPI_HU) == gameMaJiang.OPI_HU {
			tempSeat := canOperationSeatArr[i]
			liangDaoHuFunc = append(liangDaoHuFunc, func() {
				msgHu := mateProto.MessageMaTe{SenderID: int64(tempSeat.GetSeatData().Player.ID)}
				msgBodyHu := gameMaJiang.CS_Hu{OperationID: tempSeat.GetSeatData().OperationID}
				msgHu.Data, _ = json.Marshal(&msgBodyHu)
				msgHu.MessageID = gameMaJiang.ID_Hu
				rspCode := this.Table.RootTable.OnMessage(&msgHu)
				if rspCode != 0 {
					glog.Warning("Auto OnHu() rspCode:=", rspCode)
				}
			})
		} else if (canOperationSeatArr[i].MJSeat.OperationItem & gameMaJiang.OPI_GANG) == gameMaJiang.OPI_GANG {
			tempSeat := canOperationSeatArr[i]
			this.delayActiveFunc = func() {
				msgGang := mateProto.MessageMaTe{SenderID: int64(tempSeat.GetSeatData().Player.ID)}
				msgBodyGang := gameMaJiang.CS_Gang{OperationID: tempSeat.GetSeatData().OperationID, Pai: tempSeat.MJSeat.GangArr[0]}
				msgGang.Data, _ = json.Marshal(&msgBodyGang)
				msgGang.MessageID = gameMaJiang.ID_Gang
				rspCode := this.Table.RootTable.OnMessage(&msgGang)
				if rspCode != 0 {
					glog.Warning("Auto GangPaiOperation() rspCode:=", rspCode)
				}
			}
		}
	}

	if len(liangDaoHuFunc) > 0 {
		time.Sleep(700 * time.Millisecond)
		for _, v := range liangDaoHuFunc {
			v()
		}
	}

	if len(liangDaoHuFunc) < 1 && this.delayActiveFunc != nil {
		time.Sleep(700 * time.Millisecond)
		this.delayActiveFunc()
	}

	return mateProto.Err_Success
}

// 碰
func (this *KaWuXingPlayLogic) OnPeng(pro *mateProto.MessageMaTe) int32 {
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
		return mateProto.Err_Success
	}
	// 已经有人胡牌了
	if len(this.huSeatMap) > 0 {
		return mateProto.Err_Success
	}

	this.DoPeng(seatData, mjSeatData, &operationPeng)
	return mateProto.Err_Success
}

func (this *KaWuXingPlayLogic) DoPeng(seatData *qpTable.SeatData, mjSeatData *gameMaJiang.MJSeat, pengPai *gameMaJiang.CS_Peng) {

	//commonDef.LOG_Info("playerID:=", seatData.Player.ID, ",table number:=", this.Table.GetTableNumber(), ",action:= Peng")

	playPaiMJSeat := this.Table.SeatArr[this.CurPlaySeatNum].GetXSeatData(0).(*gameMaJiang.MJSeat)
	playPaiMJSeat.PlayPai = playPaiMJSeat.PlayPai[:len(playPaiMJSeat.PlayPai)-1]

	this.CleanAllSeatOperation()
	mjSeatData.LianGangCount = 0

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
		this.IsZiMoGang1(this.Table.SeatArr[seatData.Number].(*KWXSeat), gameMaJiang.InvalidPai) == true {
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
}

// 杠
func (this *KaWuXingPlayLogic) OnGang(pro *mateProto.MessageMaTe) int32 {
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

	seatData := gangPaiSeat.GetSeatData()
	mjSeatData := gangPaiSeat.GetXSeatData(0).(*gameMaJiang.MJSeat)
	if (mjSeatData.OperationItem & gameMaJiang.OPI_GANG) != gameMaJiang.OPI_GANG {
		return mateProto.Err_OperationNotExist
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
	} else if this.CurMoPaiSeatNum == seatData.Number || this.CurPengSeatNum == seatData.Number {
		// 手牌 是否存在
		if mjSeatData.GetPaiCount(operationGang.Pai) == 4 {
			gangType = 2
		} else if mjSeatData.GetPaiCount(operationGang.Pai) == 1 {
			for _, v := range mjSeatData.OperationPai {
				if v.OperationPXItem == gameMaJiang.OPX_PENG &&
					v.PaiArr[0] == operationGang.Pai &&
					v.PaiArr[0] == this.CurMoPai {
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
	seatData.CleanOperationID()
	mjSeatData.OperationItem = 0
	this.Table.OperateRec.PutPlayerStep(int32(seatData.Number), gameMaJiang.ID_Gang, &operationGang)
	this.Table.GetBaseQPTable().GameTimer.RemoveBySeatNum(int32(seatData.Number))

	heightPriority := gameMaJiang.OPI_HU
	if gangType == 1 && this.FindPriorityOperation(seatData.Number, heightPriority) == true {
		// 操作寄存起来
		this.delayActiveFunc = func() {
			this.DoGang(seatData, mjSeatData, &operationGang, gangType)
		}
		return mateProto.Err_Success
	}
	// 已经有人胡牌了
	if len(this.huSeatMap) > 0 {
		return mateProto.Err_Success
	}

	if gangType == 3 {
		// 广播
		brodcastGang := gameMaJiang.BroadcastGang{
			SeatNumber:  int32(seatData.Number),
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
		if this.findQiangGangHu(seatData.Number, operationGang.Pai) == true {
			// 操作寄存起来
			this.delayActiveFunc = func() {
				this.DoGang(seatData, mjSeatData, &operationGang, gangType)
			}
			return mateProto.Err_Success
		}
	}

	this.DoGang(seatData, mjSeatData, &operationGang, gangType)

	return mateProto.Err_Success
}

func (this *KaWuXingPlayLogic) DoGang(seatData *qpTable.SeatData, mjSeatData *gameMaJiang.MJSeat, gangPai *gameMaJiang.CS_Gang, gangType int32) {

	//commonDef.LOG_Info("playerID:=", seatData.Player.ID, ",table number:=", this.Table.GetTableNumber(), ",action:= Gang")

	kwxSeat := this.Table.SeatArr[seatData.Number].(*KWXSeat)
	kouMap := kwxSeat.KouMap
	delete(kouMap, gangPai.Pai)
	mjSeatData.GangCount += 1

	this.CleanAllSeatOperation()

	gangScore := float64(0)

	// 补杠 是直接广播的
	if gangType != 3 {
		brodcastGang := gameMaJiang.BroadcastGang{
			SeatNumber:  int32(seatData.Number),
			Type:        gangType,
			PlaySeatNum: int32(this.CurPlaySeatNum),
			Pai:         gangPai.Pai,
		}
		// 广播
		this.Table.BroadCastGameEvent(gameMaJiang.ID_BroadcastGang, &brodcastGang)
	}

	if gangType == 1 {
		playPaiMJSeat := this.Table.SeatArr[this.CurPlaySeatNum].GetXSeatData(0).(*gameMaJiang.MJSeat)
		playPaiMJSeat.PlayPai = playPaiMJSeat.PlayPai[:len(playPaiMJSeat.PlayPai)-1]

		// 删除手牌
		mjSeatData.DeleteShouPai(gangPai.Pai)
		mjSeatData.DeleteShouPai(gangPai.Pai)
		mjSeatData.DeleteShouPai(gangPai.Pai)

		// 记录 操作区域
		operInfo := &gameMaJiang.OperationPaiInfo{OperationPXItem: gameMaJiang.OPX_MING_GANG,
			PlayPaiSeatNumber: int32(this.CurPlaySeatNum),
			PaiArr:            []int8{gangPai.Pai}}
		mjSeatData.OperationPai = append(mjSeatData.OperationPai, operInfo)

		gangScore = 2
		for i := int32(0); i < mjSeatData.LianGangCount; i++ {
			gangScore *= 2
		}
	} else if gangType == 2 {
		// 删除手牌
		mjSeatData.DeleteShouPai(gangPai.Pai)
		mjSeatData.DeleteShouPai(gangPai.Pai)
		mjSeatData.DeleteShouPai(gangPai.Pai)
		mjSeatData.DeleteShouPai(gangPai.Pai)

		// 记录 操作区域
		operInfo := &gameMaJiang.OperationPaiInfo{OperationPXItem: gameMaJiang.OPX_AN_GANG,
			PaiArr: []int8{gangPai.Pai}}
		mjSeatData.OperationPai = append(mjSeatData.OperationPai, operInfo)

		gangScore = 2
		for i := int32(0); i < mjSeatData.LianGangCount; i++ {
			gangScore *= 2
		}
	} else if gangType == 3 {
		gangScore = 1
		for i := int32(0); i < mjSeatData.LianGangCount; i++ {
			gangScore *= 2
		}
	}

	gangScore *= this.PlayRule.MultipleFloat64

	// 算 杠分
	if gangType == 1 {
		this.changeGangScore2(seatData.Number, this.CurPlaySeatNum, gangScore)
	} else {
		this.changeGangScore(seatData.Number, gangScore)
	}

	// 连杠次数
	mjSeatData.LianGangCount += 1

	// 通知 摸牌
	if this.PaiMgr.GetTheRestOfPaiCount() < 1 {
		this.RoundOverFunc()
	} else {
		this.MoPaiOperation(seatData.Number)
	}

	this.delayActiveFunc = nil
}

// 过
func (this *KaWuXingPlayLogic) OnGuo(pro *mateProto.MessageMaTe) int32 {
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
			mjSeatData.IsGuoHu = true
		}
	}

	seatData.CleanOperationID()
	mjSeatData.OperationItem = 0
	this.Table.OperateRec.PutPlayerStep(int32(seatData.Number), gameMaJiang.ID_Guo, &operationGuo)
	this.Table.GetBaseQPTable().GameTimer.RemoveBySeatNum(int32(seatData.Number))

	// 是否有人已经胡了
	if len(this.huSeatMap) > 0 {
		for _, v := range this.Table.SeatArr {
			if v == nil {
				continue
			}
			// 等待其他人操作
			tempMJSeat := v.GetXSeatData(0).(*gameMaJiang.MJSeat)
			if (tempMJSeat.OperationItem & gameMaJiang.OPI_HU) == gameMaJiang.OPI_HU {
				return mateProto.Err_Success
			}
		}
		this.RoundOverFunc()
		return mateProto.Err_Success
	}

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

			if guoSeat.(*KWXSeat).LiangDaoMap != nil {
				msgPlayPai := mateProto.MessageMaTe{SenderID: int64(seatData.GetSeatData().Player.ID)}
				msgBodyPlay := gameMaJiang.CS_Play{
					OperationID: seatData.GetSeatData().OperationID,
					Pai:         this.CurMoPai}

				msgPlayPai.Data, _ = json.Marshal(&msgBodyPlay)
				msgPlayPai.MessageID = gameMaJiang.ID_Play

				rspCode := this.Table.RootTable.OnMessage(&msgPlayPai)
				if rspCode != 0 {
					glog.Warning("MoPaiOperation() rspCode:=", rspCode)
				}
			} else {
				this.timerAutoTrusteeship(int32(seatData.Number))
			}

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

	if this.PaiMgr.GetTheRestOfPaiCount() < 1 {
		this.RoundOverFunc()
		return mateProto.Err_Success
	}

	nextSeat := this.Table.GetNextPlayingSeat(this.CurPlaySeatNum)
	this.MoPaiOperation(nextSeat.GetSeatData().Number)

	return mateProto.Err_Success
}

func (this *KaWuXingPlayLogic) findQiangGangHu(gangSeatNum qpTable.SeatNumber, gangPai int8) bool {

	this.buGangSeatNum = gangSeatNum
	liangDaoHuFunc := make([]func(), 0)
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
		if this.IsDianPaoHu(v.(*KWXSeat), gangPai, true, false) == true {
			oper |= gameMaJiang.OPI_HU
		}
		if oper == 0 {
			continue
		}

		mjSeat.SetOperation(oper)
		isFindHu = true

		if v.(*KWXSeat).LiangDaoMap == nil {
			// 通知 操作
			proMsg := gameMaJiang.SC_NoticeOperation{
				SeatNumber:  int32(v.GetSeatData().Number),
				OperationID: v.GetSeatData().GetOperationID(),
				Operation:   v.GetXSeatData(0).(*gameMaJiang.MJSeat).OperationItem,
				Pai:         gangPai,
			}
			this.NoticePlayerOperation(&proMsg)
		} else {
			tempSeat := v
			liangDaoHuFunc = append(liangDaoHuFunc, func() {
				msgHu := mateProto.MessageMaTe{SenderID: int64(tempSeat.GetSeatData().Player.ID)}
				msgBodyHu := gameMaJiang.CS_Hu{OperationID: tempSeat.GetSeatData().OperationID}
				msgHu.Data, _ = json.Marshal(&msgBodyHu)
				msgHu.MessageID = gameMaJiang.ID_Hu
				rspCode := this.Table.RootTable.OnMessage(&msgHu)
				if rspCode != 0 {
					glog.Warning("Auto OnHu() rspCode:=", rspCode)
				}
			})
		}
	}

	if isFindHu == true {
		time.Sleep(700 * time.Millisecond)
		for _, v := range liangDaoHuFunc {
			v()
		}
		return true
	}
	return false
}

// 清理所有座位操作
func (this *KaWuXingPlayLogic) CleanAllSeatOperation() {

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
}

// 是否有更高优先级的操作
func (this *KaWuXingPlayLogic) FindPriorityOperation(excludeSeatNum qpTable.SeatNumber, oper gameMaJiang.PlayerMJOperation) bool {
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

// 摸牌
func (this *KaWuXingPlayLogic) MoPaiOperation(number qpTable.SeatNumber) {

	this.buGangSeatNum = qpTable.INVALID_SEAT_NUMBER

	seat := this.Table.SeatArr[number]
	mjSeat := seat.GetXSeatData(0).(*gameMaJiang.MJSeat)

	var moPai int8
	if mjSeat.CustomNextPai == gameMaJiang.InvalidPai {
		moPai = this.PaiMgr.GetPai()
	} else {
		moPai = this.PaiMgr.(*kwxPaiMgr).GetNextPai(mjSeat.CustomNextPai)
		mjSeat.CustomNextPai = gameMaJiang.InvalidPai
	}
	mjSeat.PushShouPai(moPai)
	mjSeat.CurMoPai = moPai
	mjSeat.IsGuoHu = false

	// 更新 桌子数据
	this.CurMoPaiSeatNum = number
	this.CurMoPai = moPai
	this.CurPlayPai = gameMaJiang.InvalidPai
	this.CurPlaySeatNum = qpTable.INVALID_SEAT_NUMBER
	this.CurPengSeatNum = qpTable.INVALID_SEAT_NUMBER
	this.delayActiveFunc = nil
	this.OperationTime = time.Now().Unix()

	oper := this.getZiMoOperation(seat.(*KWXSeat))
	mjSeat.SetOperation(oper)

	// 广播 摸牌
	broadMoPai := gameMaJiang.BroadcastMoPai{SeatNumber: int32(number),
		CardCount: this.PaiMgr.GetTheRestOfPaiCount()}
	this.Table.BroadCastGameEvent(gameMaJiang.ID_BroadcastMoPai, &broadMoPai)

	if seat.(*KWXSeat).LiangDaoMap == nil {
		// 通知 操作
		proMoPai := gameMaJiang.SC_PlayerMoPai{Card: moPai,
			OperationID: seat.GetSeatData().OperationID,
			Operation:   oper,
			GangArr:     mjSeat.GangArr}
		this.Table.SendGameEventToSeat(number, gameMaJiang.ID_PlayerMoPai, &proMoPai)

		this.timerAutoTrusteeship(int32(number))
		return
	}

	// 亮倒时
	if (oper & gameMaJiang.OPI_HU) == gameMaJiang.OPI_HU {
		// 通知 胡
		proMoPai := gameMaJiang.SC_PlayerMoPai{Card: moPai,
			OperationID: seat.GetSeatData().OperationID,
			Operation:   0,
			GangArr:     mjSeat.GangArr}
		this.Table.SendGameEventToSeat(number, gameMaJiang.ID_PlayerMoPai, &proMoPai)

		this.Table.GameTimer.PutTableTimer(TIMER_PlayPai, 700, func() {
			msgHu := mateProto.MessageMaTe{SenderID: int64(seat.GetSeatData().Player.ID)}
			msgBodyHu := gameMaJiang.CS_Hu{OperationID: seat.GetSeatData().OperationID}
			msgHu.Data, _ = json.Marshal(&msgBodyHu)
			msgHu.MessageID = gameMaJiang.ID_Hu
			rspCode := this.Table.RootTable.OnMessage(&msgHu)
			if rspCode != 0 {
				glog.Warning("HuPaiOperation() rspCode:=", rspCode)
			}
		})
		return
	} else if (oper & gameMaJiang.OPI_GANG) == gameMaJiang.OPI_GANG {
		// 通知 摸牌
		proMoPai := gameMaJiang.SC_PlayerMoPai{
			Card:        moPai,
			OperationID: seat.GetSeatData().OperationID,
			Operation:   gameMaJiang.OPI_MO_Pai,
			GangArr:     mjSeat.GangArr}
		this.Table.SendGameEventToSeat(number, gameMaJiang.ID_PlayerMoPai, &proMoPai)

		this.Table.GameTimer.PutTableTimer(TIMER_PlayPai, 700, func() {
			msgGang := mateProto.MessageMaTe{SenderID: int64(seat.GetSeatData().Player.ID)}
			msgBodyGang := gameMaJiang.CS_Gang{OperationID: seat.GetSeatData().OperationID, Pai: mjSeat.GangArr[0]}
			msgGang.Data, _ = json.Marshal(&msgBodyGang)
			msgGang.MessageID = gameMaJiang.ID_Gang
			rspCode := this.Table.RootTable.OnMessage(&msgGang)
			if rspCode != 0 {
				glog.Warning("GangPaiOperation() rspCode:=", rspCode)
			}
		})

		return
	} else {
		// 通知 摸牌
		proMoPai := gameMaJiang.SC_PlayerMoPai{Card: moPai,
			OperationID: seat.GetSeatData().OperationID,
			Operation:   oper,
			GangArr:     mjSeat.GangArr}
		this.Table.SendGameEventToSeat(number, gameMaJiang.ID_PlayerMoPai, &proMoPai)

		this.Table.GameTimer.PutTableTimer(TIMER_PlayPai, 700, func() {
			msgPlayPai := mateProto.MessageMaTe{SenderID: int64(seat.GetSeatData().Player.ID)}
			msgBodyPlay := gameMaJiang.CS_Play{
				OperationID: seat.GetSeatData().OperationID,
				Pai:         moPai}

			msgPlayPai.Data, _ = json.Marshal(&msgBodyPlay)
			msgPlayPai.MessageID = gameMaJiang.ID_Play

			rspCode := this.Table.RootTable.OnMessage(&msgPlayPai)
			if rspCode != 0 {
				glog.Warning("MoPaiOperation() rspCode:=", rspCode)
			}
		})
		return
	}
}

// 自摸操作
func (this *KaWuXingPlayLogic) getZiMoOperation(kwxSeat *KWXSeat) gameMaJiang.PlayerMJOperation {

	oper := gameMaJiang.OPI_PlayPai | gameMaJiang.OPI_MO_Pai

	if this.IsZiMoHu(kwxSeat) == true {
		oper |= gameMaJiang.OPI_HU
	}
	if this.PaiMgr.GetTheRestOfPaiCount() > 0 &&
		this.IsZiMoGang1(kwxSeat, this.CurMoPai) == true {
		oper |= gameMaJiang.OPI_GANG
	}

	return oper
}

// 玩家出牌,其它人操作
func (this *KaWuXingPlayLogic) GetPlayPaiOperation(playSeatNum qpTable.SeatNumber, playPai int8, gangCount int32) []*KWXSeat {

	// 出牌者 上次 是否 杠操作
	playPaiSeatIsGang := false
	if gangCount > 0 {
		playPaiSeatIsGang = true
	}

	operSeat := make([]*KWXSeat, 0)
	for i, v := range this.Table.SeatArr {
		if v == nil {
			continue
		}
		if v.GetSeatData().Number == playSeatNum {
			continue
		}
		if v.GetSeatData().IsContainSeatState(qpTable.SS_Looker) == true {
			continue
		}
		mjSeat := v.GetXSeatData(0).(*gameMaJiang.MJSeat)

		var oper gameMaJiang.PlayerMJOperation
		if this.IsDianPaoHu(v.(*KWXSeat), playPai, false, playPaiSeatIsGang) == true {
			oper |= gameMaJiang.OPI_HU
		}
		if mjSeat.IsGuoHu == false && v.(*KWXSeat).LiangDaoMap == nil && this.IsPeng(mjSeat.ShouPai, playPai) == true {
			oper |= gameMaJiang.OPI_PENG
		}
		if this.PaiMgr.GetTheRestOfPaiCount() > 0 &&
			this.IsMingGang(v.(*KWXSeat), playPai) == true {
			oper |= gameMaJiang.OPI_GANG
		}

		if oper != 0 {
			mjSeat.SetOperation(oper)
			operSeat = append(operSeat, this.Table.SeatArr[i].(*KWXSeat))
		}
	}

	//nextSeat := this.Table.GetNextPlayingSeat(playSeatNum)
	//mjSeat := nextSeat.GetXSeatData(0).(*gameMaJiang.MJSeat)
	//if this.Logic.IsChi(mjSeat.ShouPai, playPai) == true {
	//	mjSeat.PutOperation(gameMaJiang.OPI_CHI)
	//	operSeat = append(operSeat, mjSeat)
	//}

	return operSeat
}

// 检查操作
func (this *KaWuXingPlayLogic) CheckOperation(playerID qpTable.PlayerID, operationID string) (qpTable.QPSeat, int32) {
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

// 定时托管
func (this *KaWuXingPlayLogic) timerAutoTrusteeship(seatNum int32) {
	if this.Table.TableRule.TuoGuanTime < 1 {
		return
	}

	tempTime := this.Table.TableRule.TuoGuanTime
	if this.Table.SeatArr[seatNum].GetSeatData().IsAssignSeatState(qpTable.SS_Trusteeship) == true {
		tempTime = 1
	}
	this.Table.GetBaseQPTable().GameTimer.PutSeatTimer(
		seatNum,
		TIMER_PlayPai,
		tempTime*1000, func() {
			this.autoTrusteeshipPlayPai(seatNum)
		})
}

func (this *KaWuXingPlayLogic) autoTrusteeshipPlayPai(seatNum int32) {
	seat := this.Table.SeatArr[seatNum]

	if seat.GetSeatData().IsAssignSeatState(qpTable.SS_Trusteeship) == false {
		seat.GetSeatData().AppendState(qpTable.SS_Trusteeship)
		this.Table.NotifyPlayerStateChange(seat.GetSeatData().Number)
	}

	mjSeat := seat.GetXSeatData(0).(*gameMaJiang.MJSeat)

	msg := mateProto.MessageMaTe{SenderID: int64(seat.GetSeatData().Player.ID)}

	if (mjSeat.OperationItem & gameMaJiang.OPI_HU) == gameMaJiang.OPI_HU {
		//msgBodyHu := gameMaJiang.CS_Hu{OperationID: seat.GetSeatData().OperationID}
		//msg.Data, _ = json.Marshal(&msgBodyHu)
		//msg.MessageID = gameMaJiang.ID_Hu
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

		shouPaiArr := mjSeat.GetShouPai()

		msgBodyPlay := gameMaJiang.CS_Play{OperationID: seat.GetSeatData().OperationID, Pai: shouPaiArr[0]}
		if this.CurMoPaiSeatNum == qpTable.SeatNumber(seatNum) {
			msgBodyPlay.Pai = this.CurMoPai
		}

		// 没有亮倒时
		if seat.(*KWXSeat).LiangDaoMap == nil {
			tingPaiMap := make(map[int8]int8)
			for _, v := range this.Table.SeatArr {
				if v == nil {
					continue
				}
				for k, _ := range v.(*KWXSeat).TingPaiMap {
					tingPaiMap[k] = k
				}
			}
			if _, ok := tingPaiMap[msgBodyPlay.Pai]; ok == true {
				for _, v := range shouPaiArr {
					if _, ok := tingPaiMap[v]; ok == false {
						msgBodyPlay.Pai = v
						break
					}
				}
			}
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

func (this *KaWuXingPlayLogic) IsPeng(shouPai [gameMaJiang.MaxHuaSe + 1][gameMaJiang.MaxDianShu_9 + 1]int8, pai int8) bool {
	huaSeIndex := uint8(pai) >> 4
	if shouPai[huaSeIndex][pai&0x0F] > 1 {
		return true
	}

	return false
}

func (this *KaWuXingPlayLogic) IsMingGang(seat *KWXSeat, pai int8) bool {

	mjSeat := seat.GetXSeatData(0).(*gameMaJiang.MJSeat)
	mjSeat.GangArr = nil
	gangArr := make([]int8, 0, 4)

	if seat.LiangDaoMap != nil {
		// 亮倒的牌 不能杠
		if _, ok := seat.LiangDaoMap[pai]; ok == true {
			return false
		}
		// 扣 的牌 才能杠
		if _, ok := seat.KouMap[pai]; ok == true {
			gangArr = append(gangArr, pai)
		}
	} else {
		huaSeIndex := uint8(pai) >> 4
		if mjSeat.ShouPai[huaSeIndex][pai&0x0F] > 2 {
			gangArr = append(gangArr, pai)
		}
	}

	mjSeat.GangArr = gangArr
	if len(mjSeat.GangArr) > 0 {
		return true
	}
	return false
}

func (this *KaWuXingPlayLogic) IsZiMoGang1(seat *KWXSeat, moPai int8) bool {

	mjSeat := seat.GetXSeatData(0).(*gameMaJiang.MJSeat)
	mjSeat.GangArr = nil
	gangArr := make([]int8, 0, 4)

	for i := gameMaJiang.MinHuaSe; i <= gameMaJiang.MaxHuaSe; i++ {
		if mjSeat.ShouPai[i][0] < 1 {
			continue
		}
		for j := gameMaJiang.MinDianShu_1; j <= gameMaJiang.MaxDianShu_9; j++ {
			if mjSeat.ShouPai[i][j] < 4 {
				continue
			}

			if seat.LiangDaoMap != nil {
				// 是否是摸的牌
				if this.CurMoPai != i*0x10|j {
					continue
				}
				// 亮倒的牌 不能杠
				if _, ok := seat.LiangDaoMap[i*0x10|j]; ok == true {
					continue
				}
				// 扣 的牌 才能杠
				if _, ok := seat.KouMap[i*0x10+j]; ok == false {
					continue
				}
			}
			gangArr = append(gangArr, i*0x10+j)
		}
	}

	// 补杠 过后不补
	for _, v := range mjSeat.OperationPai {
		if v.OperationPXItem != gameMaJiang.OPX_PENG {
			continue
		}
		pai := v.PaiArr[0]
		huaSeIndex := uint8(pai) >> 4
		if mjSeat.ShouPai[huaSeIndex][pai&0x0F] > 0 && moPai == pai {
			gangArr = append(gangArr, pai)
		}
	}

	mjSeat.GangArr = gangArr
	if len(mjSeat.GangArr) > 0 {
		return true
	}
	return false
}

func (this *KaWuXingPlayLogic) IsZiMoHu(kwxSeat *KWXSeat) bool {
	mjSeat := kwxSeat.GetXSeatData(0).(*gameMaJiang.MJSeat)
	if this.HuLogic.IsZiMoHu(kwxSeat, this.CurMoPai) == true {
		mjSeat.HuPaiXing = this.getPaiXing(kwxSeat)
		return true
	}
	return false
}

// isGang:杠后,出牌点炮
func (this *KaWuXingPlayLogic) IsDianPaoHu(kwxSeat *KWXSeat, playPai int8, isQiangGang, isGang bool) bool {

	mjSeat := kwxSeat.GetXSeatData(0).(*gameMaJiang.MJSeat)
	if this.HuLogic.IsDianPaoHu(kwxSeat, playPai) == false {
		return false
	}
	isLiangDao := false
	if kwxSeat.LiangDaoMap != nil { //&& kwxSeat.IsLiangDaoIng == false {
		isLiangDao = true
	} else if this.CurPlaySeatNum != qpTable.INVALID_SEAT_NUMBER {
		playSeat := this.Table.SeatArr[this.CurPlaySeatNum].(*KWXSeat)
		if playSeat.LiangDaoMap != nil { //&& playSeat.IsLiangDaoIng == false {
			isLiangDao = true
		}
	}

	mjSeat.HuPaiXing = this.getPaiXing(kwxSeat)
	if mjSeat.HuScore > 1 || isQiangGang == true || isLiangDao == true || isGang == true {
		return true
	}
	// 海底炮
	if this.PaiMgr.GetTheRestOfPaiCount() == 0 {
		return true
	}
	return false
}

func (this *KaWuXingPlayLogic) getPaiXing(kwxSeat *KWXSeat) []*gameMaJiang.HuPaiXing {

	mjSeat := kwxSeat.GetXSeatData(0).(*gameMaJiang.MJSeat)
	mjSeat.HuScore = 1
	paiXingArr := make([]*gameMaJiang.HuPaiXing, 0, 17)

	if this.HuLogic.isLiangDao(kwxSeat) == true {
		tempPXFS := &gameMaJiang.HuPaiXing{PaiXing: HuPX_LiangDao, FanShu: 2}
		paiXingArr = append(paiXingArr, tempPXFS)
		mjSeat.HuScore *= tempPXFS.FanShu
	}

	isHaoHua7Dui := false
	if this.HuLogic.is7Dui() == true {
		tempPXFS := &gameMaJiang.HuPaiXing{PaiXing: HuPX_QiXiaoDui, FanShu: 4}
		if this.HuLogic.isHaoHua7Dui() == true {
			isHaoHua7Dui = true
			tempPXFS.PaiXing, tempPXFS.FanShu = HuPX_HaoHuaQiDui, 8
		}
		paiXingArr = append(paiXingArr, tempPXFS)
		mjSeat.HuScore *= tempPXFS.FanShu
	}

	if this.HuLogic.isKaWuXing(this.CurMoPai, this.CurPlayPai) == true {
		tempPXFS := &gameMaJiang.HuPaiXing{PaiXing: HuPX_KaWuXing, FanShu: 2}
		if this.PlayRule.IsKaWuXingx4 == true {
			tempPXFS.FanShu = 4
		}
		paiXingArr = append(paiXingArr, tempPXFS)
		mjSeat.HuScore *= tempPXFS.FanShu
	}

	isShouZhuaYi := false
	if this.HuLogic.isShouZhuaYi(kwxSeat) == true {
		isShouZhuaYi = true
		tempPXFS := &gameMaJiang.HuPaiXing{PaiXing: HuPX_ShouZhuaYi, FanShu: 4}
		paiXingArr = append(paiXingArr, tempPXFS)
		mjSeat.HuScore *= tempPXFS.FanShu
	}
	if this.CurPlaySeatNum != qpTable.INVALID_SEAT_NUMBER {
		if this.HuLogic.isHaiDiLao(this.PaiMgr.GetTheRestOfPaiCount()) == true {
			tempPXFS := &gameMaJiang.HuPaiXing{PaiXing: HuPX_HaiDiPao, FanShu: 2}
			paiXingArr = append(paiXingArr, tempPXFS)
			mjSeat.HuScore *= tempPXFS.FanShu
		}
	} else {
		if this.HuLogic.isHaiDiLao(this.PaiMgr.GetTheRestOfPaiCount()) == true {
			tempPXFS := &gameMaJiang.HuPaiXing{PaiXing: HuPX_HaiDiLao, FanShu: 2}
			paiXingArr = append(paiXingArr, tempPXFS)
			mjSeat.HuScore *= tempPXFS.FanShu
		}
	}

	if isShouZhuaYi == false && this.HuLogic.isPengPengHu(&mjSeat.OperationPai) == true {
		tempPXFS := &gameMaJiang.HuPaiXing{PaiXing: HuPX_PengPengHu, FanShu: 2}
		if this.PlayRule.IsPengPengHux4 == true {
			tempPXFS.FanShu = 4
		}
		paiXingArr = append(paiXingArr, tempPXFS)
		mjSeat.HuScore *= tempPXFS.FanShu
	}

	is9LianBaoDeng := false
	if this.HuLogic.is9LianBaoDeng() == true {
		tempPXFS := &gameMaJiang.HuPaiXing{PaiXing: HuPX_JiuLianBaoDeng, FanShu: 8}
		paiXingArr = append(paiXingArr, tempPXFS)
		mjSeat.HuScore *= tempPXFS.FanShu
		is9LianBaoDeng = true
	}

	isQingYiSe := false
	if this.HuLogic.isQingYiSe(&mjSeat.OperationPai) == true {
		isQingYiSe = true
		if is9LianBaoDeng == false {
			tempPXFS := &gameMaJiang.HuPaiXing{PaiXing: HuPX_QingYiSe, FanShu: 4}
			paiXingArr = append(paiXingArr, tempPXFS)
			mjSeat.HuScore *= tempPXFS.FanShu
		}
	}

	if isHaoHua7Dui == false &&
		is9LianBaoDeng == false &&
		this.HuLogic.isMingSiGui(&mjSeat.OperationPai, isQingYiSe) == true {
		tempPXFS := &gameMaJiang.HuPaiXing{PaiXing: HuPX_MingSiGui, FanShu: 2}
		paiXingArr = append(paiXingArr, tempPXFS)
		mjSeat.HuScore *= tempPXFS.FanShu
	}
	if isHaoHua7Dui == false && is9LianBaoDeng == false && this.HuLogic.isAnSiGui(isQingYiSe) == true {
		tempPXFS := &gameMaJiang.HuPaiXing{PaiXing: HuPX_AnSiGui, FanShu: 4}
		paiXingArr = append(paiXingArr, tempPXFS)
		mjSeat.HuScore *= tempPXFS.FanShu
	}

	if this.HuLogic.isGangShangHua(mjSeat) == true {
		tempPXFS := &gameMaJiang.HuPaiXing{PaiXing: HuPX_GangShangHua, FanShu: 2}
		if this.PlayRule.IsGangShangHuax4 == true {
			tempPXFS.FanShu = 4
		}
		paiXingArr = append(paiXingArr, tempPXFS)
		mjSeat.HuScore *= tempPXFS.FanShu
	}
	if this.HuLogic.isGangShangPao(mjSeat.GetSeatData().Number, this.llastGangSeatNum, this.CurPlaySeatNum) == true {
		tempPXFS := &gameMaJiang.HuPaiXing{PaiXing: HuPX_GangShangPao, FanShu: 2}
		paiXingArr = append(paiXingArr, tempPXFS)
		mjSeat.HuScore *= tempPXFS.FanShu
	}
	if this.HuLogic.isQiangGangHu(mjSeat.GetSeatData().Number, this.buGangSeatNum) == true {
		tempPXFS := &gameMaJiang.HuPaiXing{PaiXing: HuPX_QiangGangHu, FanShu: 2}
		paiXingArr = append(paiXingArr, tempPXFS)
		mjSeat.HuScore *= tempPXFS.FanShu
	}

	if da, xiao := this.HuLogic.isDaXiaoSanYuan(&mjSeat.OperationPai); da == true || xiao == true {
		var tempPXFS *gameMaJiang.HuPaiXing
		if da == true {
			tempPXFS = &gameMaJiang.HuPaiXing{PaiXing: HuPX_DaSanYuan, FanShu: 8}
		} else {
			tempPXFS = &gameMaJiang.HuPaiXing{PaiXing: HuPX_XiaoSanYuan, FanShu: 4}
		}
		paiXingArr = append(paiXingArr, tempPXFS)
		mjSeat.HuScore *= tempPXFS.FanShu
	}

	if len(paiXingArr) < 1 {
		paiXingArr = append(paiXingArr, &gameMaJiang.HuPaiXing{PaiXing: HuPX_PingHu, FanShu: 1})
	}

	if mjSeat.HuScore > int64(this.PlayRule.FengDingFanShu) {
		mjSeat.HuScore = int64(this.PlayRule.FengDingFanShu)
	}

	if this.PlayRule.IsShuKang == true {
		kwxSeat.KanShu = this.HuLogic.shuKang(&mjSeat.OperationPai)
	}

	return paiXingArr
}

func (this *KaWuXingPlayLogic) onLiang(pro *mateProto.MessageMaTe) int32 {
	playerID := qpTable.PlayerID(pro.SenderID)

	msgLiangDao := CS_LiangDao{}
	err := json.Unmarshal(pro.Data, &msgLiangDao)
	if err != nil {
		return mateProto.Err_ProtocolDataErr
	}

	if len(msgLiangDao.LiangDaoPaiArr) < 1 || len(msgLiangDao.TingPai) < 1 {
		return mateProto.Err_OperationParamErr
	}

	seat := this.Table.GetSeatDataByPlayerID(playerID)
	if seat == nil {
		return mateProto.Err_NotFindPlayer
	}
	mjSeat := seat.GetXSeatData(0).(*gameMaJiang.MJSeat)

	if this.PlayRule.IsLess12NotLiang == true && this.PaiMgr.GetTheRestOfPaiCount() <= 12 {
		return mateProto.Err_ShouPaiCount
	}

	// 打出牌后,是否 听牌
	{
		shouPaiBak := mjSeat.ShouPai
		if msgLiangDao.PlayPai != 0 {
			if mjSeat.GetPaiCount(msgLiangDao.PlayPai) > 0 {
				shouPaiBak[msgLiangDao.PlayPai>>4][0] -= 1
				shouPaiBak[msgLiangDao.PlayPai>>4][msgLiangDao.PlayPai&0x0F] -= 1
			} else {
				return mateProto.Err_PaiNotExist
			}

			// 所有人的  亮倒牌
			allTingPaiMap := make(map[int8]int8)
			for i, v := range this.Table.SeatArr {
				if v == nil {
					continue
				}
				if i == int(seat.GetSeatData().Number) {
					continue
				}
				for k, _ := range v.(*KWXSeat).TingPaiMap {
					allTingPaiMap[k] = k
				}
			}
			// 出的牌是否 是别人的 听牌
			if _, ok := allTingPaiMap[msgLiangDao.PlayPai]; ok == true {
				// 查找手里是否还有别人 不听的牌
				ok = mjSeat.RangeShouPai(func(shouPai int8) bool {
					if _, ok1 := allTingPaiMap[shouPai]; ok1 == false {
						return true
					}
					return false
				})
				if ok == false {
					return mateProto.Err_CustomPai
				}
			}
		}

		// 验证 听牌
		{
			// 扣牌去掉
			for _, v := range msgLiangDao.KouPai {
				shouPaiBak[v>>4][0] -= 3
				shouPaiBak[v>>4][v&0x0F] -= 3
			}

			tingPaiMap := this.HuLogic.isTingPai(shouPaiBak)

			if len(tingPaiMap) < 1 || len(tingPaiMap) != len(msgLiangDao.TingPai) {
				glog.Warning("table num :=", this.Table.TableNum, ",tingPaiMap:=", tingPaiMap)
				return mateProto.Err_CheckFailed
			}
			for _, v := range msgLiangDao.TingPai {
				if _, ok := tingPaiMap[v]; ok == false {
					glog.Warning("table num :=", this.Table.TableNum, ",tingPaiMap:=", tingPaiMap)
					return mateProto.Err_CheckFailed
				}
				delete(tingPaiMap, v)
			}
		}
	}

	// 统计牌
	countPaiMap := make(map[int8]int8)
	// 亮倒
	tempLianDaoMap := make(map[int8]int8)
	for _, v := range msgLiangDao.LiangDaoPaiArr {
		if _, ok := tempLianDaoMap[v]; ok == false {
			tempLianDaoMap[v] = 1
		} else {
			tempLianDaoMap[v] += 1
		}

		if _, ok := countPaiMap[v]; ok == false {
			countPaiMap[v] = 1
		} else {
			countPaiMap[v] += 1
		}
	}
	for _, v := range msgLiangDao.KouPai {
		if _, ok := countPaiMap[v]; ok == false {
			countPaiMap[v] = 3
		} else {
			countPaiMap[v] += 3
		}
	}

	// 手牌是否足够
	for k, v := range countPaiMap {
		if mjSeat.GetPaiCount(k) < v {
			return mateProto.Err_PaiNotExist
		}
	}

	delete(this.noLiangDaoSeatNumMap, seat.GetSeatData().Number)
	this.liangDaoSeatNumMap[seat.GetSeatData().Number] = seat.(*KWXSeat)

	seat.(*KWXSeat).LiangDaoMap = tempLianDaoMap
	// 听倒
	tempTingPaiMap := make(map[int8]int8)
	for _, v := range msgLiangDao.TingPai {
		tempTingPaiMap[v] = 1
	}
	// 扣牌
	tempKouPaiMap := make(map[int8]int8)
	for _, v := range msgLiangDao.KouPai {
		tempKouPaiMap[v] = 1
	}

	seat.(*KWXSeat).TingPaiMap = tempTingPaiMap
	seat.(*KWXSeat).KouMap = tempKouPaiMap
	//seat.(*KWXSeat).IsLiangDaoIng = true

	this.Table.OperateRec.PutPlayerStep(int32(seat.GetSeatData().Number), pro.MessageID, &msgLiangDao)
	this.Table.BroadCastGameEvent(ID_BroadcastLiangDao, &CS_BroadcastLiangDao{
		SeatNumber: int32(seat.GetSeatData().Number),
		PaiArr:     msgLiangDao.LiangDaoPaiArr,
		TingPai:    msgLiangDao.TingPai,
	})

	if msgLiangDao.PlayPai != 0 {
		msgPlay := mateProto.MessageMaTe{SenderID: int64(seat.GetSeatData().Player.ID)}
		msgBodyPlay := gameMaJiang.CS_Play{
			OperationID: seat.GetSeatData().OperationID,
			Pai:         msgLiangDao.PlayPai}

		msgPlay.MessageID = gameMaJiang.ID_Play
		msgPlay.Data, _ = json.Marshal(&msgBodyPlay)
		this.OnPlay(&msgPlay)
	}

	return 0
}

func (this *KaWuXingPlayLogic) changeGangScore(win qpTable.SeatNumber, score float64) {
	msgBody := protoGameBasic.BroadcastPlayerScoreChanged{
		WinnerSeatNum: int32(win),
		LoserSeatNum:  make([]int32, 0, 4),
		Score:         commonDef.Float64ToString(score)}

	for i, v := range this.Table.SeatArr {
		if v == nil {
			continue
		}
		if v.GetSeatData().IsAssignSeatState(qpTable.SS_Playing) == false {
			continue
		}
		if qpTable.SeatNumber(i) == win {
			continue
		}
		v.(*KWXSeat).GangScore -= score
		v.GetSeatData().RoundScore -= score
		msgBody.LoserSeatNum = append(msgBody.LoserSeatNum, int32(v.GetSeatData().Number))

		this.Table.SeatArr[win].(*KWXSeat).GangScore += score
		this.Table.SeatArr[win].GetSeatData().RoundScore += score
	}
	this.Table.BroadCastGameEvent(protoGameBasic.ID_PlayerRoundScoreChanged, &msgBody)
}

func (this *KaWuXingPlayLogic) changeGangScore2(win, lose qpTable.SeatNumber, score float64) {
	this.Table.SeatArr[win].(*KWXSeat).GangScore += score
	this.Table.SeatArr[win].GetSeatData().RoundScore += score

	this.Table.SeatArr[lose].(*KWXSeat).GangScore -= score
	this.Table.SeatArr[lose].GetSeatData().RoundScore -= score

	msgBody := protoGameBasic.BroadcastPlayerScoreChanged{
		WinnerSeatNum: int32(win),
		LoserSeatNum:  make([]int32, 0, 1),
		Score:         commonDef.Float64ToString(score)}

	msgBody.LoserSeatNum = append(msgBody.LoserSeatNum, int32(lose))

	this.Table.BroadCastGameEvent(protoGameBasic.ID_PlayerRoundScoreChanged, &msgBody)
}

func (this *KaWuXingPlayLogic) NoticePlayerOperation(msgOperation *gameMaJiang.SC_NoticeOperation) {
	this.Table.SendGameEventToSeat(
		qpTable.SeatNumber(msgOperation.SeatNumber),
		gameMaJiang.ID_NoticeOperation,
		msgOperation)

	this.timerAutoTrusteeship(msgOperation.SeatNumber)
}

func (this *KaWuXingPlayLogic) onCustomNextPai(pro *mateProto.MessageMaTe) int32 {
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

func (this *KaWuXingPlayLogic) onGetRemainingPai(pro *mateProto.MessageMaTe) int32 {
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
	for k, v := range this.PaiMgr.(*kwxPaiMgr).GetRemainPai() {
		arr = append(arr, PaiInfo{Pai: k, Count: v})
	}

	wrapMQ.ReplyToSource(pro, arr)

	return 0
}

//func (this *KaWuXingPlayLogic) findPlayPai(seatNum qpTable.SeatNumber) int8 {
//
//	allLiangDaoMap := make(map[int8]int8)
//
//	for _, v := range this.Table.SeatArr {
//		if v == nil {
//			continue
//		}
//		for k, _ := range v.(*KWXSeat).LiangDaoMap {
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
