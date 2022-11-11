package pokerYaAnD14

import (
	"encoding/json"
	"fmt"
	"github.com/golang/glog"
	"math/rand"
	commonDef "qpGame/commonDefine"
	"qpGame/commonDefine/mateProto"
	"qpGame/commonDefine/mateProto/protoGameBasic"
	"qpGame/db"
	"qpGame/game/gameMaJiang"
	"qpGame/qpTable"
	"qpGame/wrapMQ"
	"sort"
	"strconv"
	"time"
)

const (
	HuPX_TouPao      = 2  // 头炮
	HuPx_TianHu      = 3  // 天胡
	HuPX_QiangGangHu = 8  // 抢杠胡
	HuPX_HaiDiLao    = 9  // 海底劳
	HuPX_HaiDiPao    = 10 // 海底炮

	HuPX_XiaoDui    = 14 // 小队
	HuPX_LongDui    = 15 //  龙对
	HuPX_HaoHuaDui  = 16 // 豪华对
	HuPX_RuanSu     = 17 // 软素
	HuPX_QuanSu     = 18 // 全素
	HuPX_QuanRen    = 19 // 全人
	HuPX_MoPaiHu    = 20 // 偷牌胡
	HuPX_LiangGang  = 21 // 连杠
	HuPX_W_SJinSChu = 22 // 王 双进双出
	HuPX_4_SJinSChu = 23 // 4个一样的 双进双出
	HuPX_Bao        = 24 // 报

	HuPX_Jin20        = 25 // 金20
	HuPX_Hong50       = 26 // 红50
	HuPX_Hei50        = 27 // 黑50
	HuPX_JinGouDiao   = 28 // 金钩钓
	HuPX_AllHei       = 29 // 全黑
	HuPX_AllHong      = 30 // 全红
	HuPX_DiHu         = 31 // 地胡
	HuPX_ZiMo         = 32 // 自摸
	HuPX_GangShangPao = 33 // 杠上炮

	HuPX_LoseLiuJuChaJiao = 34 // 流局分(流局后，检查是否有玩家听牌)
	HuPX_WinLiuJuChaJiao  = 35 // 流局分(流局后，检查是否有玩家听牌)
)
const (
	timerAutoPlayPai = protoGameBasic.PaoDeKuai
	timerAutoBao     = protoGameBasic.PaoDeKuai + 1
	timerFanPai      = protoGameBasic.PaoDeKuai + 2
	timerHuPai       = protoGameBasic.PaoDeKuai + 3
	timerBaoGang     = protoGameBasic.PaoDeKuai + 4
	timerBaoGangPlay = protoGameBasic.PaoDeKuai + 5
)

const TS_BaoPai qpTable.TableStatus = 32 // 自定义状态起始值 报牌

type PokerDou14Table struct {
	PaiMgr   *Dou14PaiBaseMgr // 牌的管理器
	logic    dou14Logic
	table    *qpTable.QPTable
	gameRule Dou14Rule

	bankerSeatNo qpTable.SeatNumber

	huSeatNo         qpTable.SeatNumber
	dianPaoNo        qpTable.SeatNumber
	curPointToSeatNo qpTable.SeatNumber
	baoJiaoCount     int32
	huPai            int8
	// 小局 待清理 成员
	curGangSeatNo   qpTable.SeatNumber
	curMoPaiSeatNo  qpTable.SeatNumber
	curPlaySeatNo   qpTable.SeatNumber // 当前出牌座位号
	curFanPaiSeatNo qpTable.SeatNumber
	curPlayPai      int8
	curFanChuPai    int8               // 翻出的牌
	lastGangSeatNo  qpTable.SeatNumber //上次杠的座位号

	delay_ struct {
		seatNo    qpTable.SeatNumber
		operItem  D14Operation
		delayFunc func() // 延迟操作
	}

	OperationTime int64 // 玩家操作起始点
}

// 清空每一小局数据
func (this *PokerDou14Table) CleanRoundData() {
	this.table.CleanRoundData()

	this.baoJiaoCount = 0
	this.huPai = InvalidPai
	this.huSeatNo, this.dianPaoNo = qpTable.INVALID_SEAT_NUMBER, qpTable.INVALID_SEAT_NUMBER
	this.curPointToSeatNo = qpTable.INVALID_SEAT_NUMBER
	this.curGangSeatNo = qpTable.INVALID_SEAT_NUMBER
	this.curMoPaiSeatNo = qpTable.INVALID_SEAT_NUMBER
	this.lastGangSeatNo = qpTable.INVALID_SEAT_NUMBER
	this.curFanPaiSeatNo, this.curFanChuPai = qpTable.INVALID_SEAT_NUMBER, InvalidPai
	this.curPlaySeatNo, this.curPlayPai = qpTable.INVALID_SEAT_NUMBER, InvalidPai
	this.delay_.seatNo = qpTable.INVALID_SEAT_NUMBER
	this.delay_.operItem, this.delay_.delayFunc = 0, nil
}

func (this *PokerDou14Table) cleanTableRec() {
	this.curGangSeatNo = qpTable.INVALID_SEAT_NUMBER
	this.curFanPaiSeatNo, this.curFanChuPai = qpTable.INVALID_SEAT_NUMBER, InvalidPai
	this.curMoPaiSeatNo = qpTable.INVALID_SEAT_NUMBER
	this.curPlaySeatNo, this.curPlayPai = qpTable.INVALID_SEAT_NUMBER, InvalidPai
	this.delay_.seatNo = qpTable.INVALID_SEAT_NUMBER
	this.delay_.operItem, this.delay_.delayFunc = 0, nil

	this.OperationTime = time.Now().Unix()
}

func (this *PokerDou14Table) SetTableNumber(tabNumber int32) {
	this.table.TableNum = tabNumber
}

func (this *PokerDou14Table) GetStatus() int32 {
	return int32(this.table.Status)
}

func (this *PokerDou14Table) ParseTableOptConfig(gameRuleCfg string) (rspCode int32, err error) {

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

func (this *PokerDou14Table) GetMaxRound() int32 {
	return this.gameRule.MaxRoundCount
}

func (this *PokerDou14Table) OnMessage(msg *mateProto.MessageMaTe) int32 {

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

	case ID_TouMo:
		return this.OnTouMo(msg)
	case ID_Bao:
		return this.OnBao(msg)
	case ID_Play:
		return this.OnPlayPai(msg)
	case ID_Chi:
		return this.OnChi(msg)
	case ID_Peng:
		return this.OnPeng(msg)
	case ID_Gang:
		return this.OnGang(msg)
	case ID_Hu:
		return this.OnHu(msg)
	case ID_Guo:
		return this.OnGuo(msg)

	case protoGameBasic.ID_ActiveTrusteeship:
		return this.OnActiveTrusteeship(msg)
	case protoGameBasic.ID_DissolveTableVote:
		return this.onDissolveTableVote(msg)
	case protoGameBasic.ID_ForceDissolveTable:
		return this.onForceDissolveTable(msg)
	case ID_CustomShouPai:
		return this.onCustomShouPai(msg)
	case ID_CustomMoPai:
		return this.onCustomMoPai(msg)
	case ID_GetSurplus:
		return this.onGetSurplus(msg)
	default:
		return this.table.OnMessage(msg)
	}
}

func (this *PokerDou14Table) GetBaseQPTable() *qpTable.QPTable {
	return this.table
}

func (this *PokerDou14Table) onPrivateJoinTable(msg *mateProto.MessageMaTe) int32 {

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
		this.table.SeatArr[rspCode].(*PokerDou14Seat).paiLogic.rule = &this.gameRule
		seatData := this.table.SeatArr[rspCode].GetSeatData()
		seatData.Player.IP = joinTable.IP
		seatData.Lat, seatData.Lng = joinTable.Latitude, joinTable.Longitude
	}

	// 还原成 原始ID
	msg.MessageID = protoGameBasic.ID_PrivateJoinGameTable

	return rspCode
}

func (this *PokerDou14Table) onClubJoinTable(msg *mateProto.MessageMaTe) int32 {

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

		this.table.SeatArr[rspCode].(*PokerDou14Seat).paiLogic.rule = &this.gameRule
	}

	return rspCode
}

func (this *PokerDou14Table) onPlayerLeave(pro *mateProto.MessageMaTe) int32 {

	if this.table.IsAssignTableState(qpTable.TS_WaitingPlayerEnter) == false {
		return mateProto.Err_TableStatusNotMatch
	}
	return this.table.OnLeave(pro)
}

func (this *PokerDou14Table) onReady(pro *mateProto.MessageMaTe) int32 {
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

func (this *PokerDou14Table) onGameStart(pro *mateProto.MessageMaTe) int32 {
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

	this.PaiMgr.XiPai(this.gameRule.WangHua)

	// 第一局随机庄
	if this.table.CurXRound == 1 {
		this.bankerSeatNo = qpTable.INVALID_SEAT_NUMBER
		_tSeatNo := rand.Intn(int(this.gameRule.MaxPlayer))
		for i := _tSeatNo; i < len(this.table.SeatArr); i++ {
			if this.table.SeatArr[_tSeatNo] != nil &&
				this.table.SeatArr[_tSeatNo].GetSeatData().IsAssignSeatState(qpTable.SS_Playing) == true {
				this.bankerSeatNo = qpTable.SeatNumber(i)
				break
			}
		}
		if this.bankerSeatNo == qpTable.INVALID_SEAT_NUMBER {
			for i := 0; i < _tSeatNo; i++ {
				if this.table.SeatArr[_tSeatNo] != nil &&
					this.table.SeatArr[_tSeatNo].GetSeatData().IsAssignSeatState(qpTable.SS_Playing) == true {
					this.bankerSeatNo = qpTable.SeatNumber(i)
					break
				}
			}
		}
		// 测试
		this.bankerSeatNo = 0
	}
	if this.bankerSeatNo == qpTable.INVALID_SEAT_NUMBER {
		glog.Warning("not find banker.")
		return 0
	}

	noticeShouPaiInfo := SC_NoticeShouPaiInfo{
		SeatShouPaiCount: make([]int32, 4),
		BankerSeat:       int32(this.bankerSeatNo),
		XiaoSeat:         int32(qpTable.INVALID_SEAT_NUMBER)}

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

		d14Seat := v.(*PokerDou14Seat)
		faPaiCount := int32(7)
		if i == int(this.bankerSeatNo) {
			faPaiCount = 8
		}

		paiArr := this.PaiMgr.FaPai(faPaiCount, d14Seat.reserveShouPai)
		d14Seat.reserveShouPai = nil

		for _, v := range paiArr {
			d14Seat.PushShouPai(v)
		}

		this.table.SendGameEventToSeat(qpTable.SeatNumber(i),
			SC_FaShouPai,
			&MsgFaShouPai{SeatNumber: int32(i), Pai: paiArr})

		noticeShouPaiInfo.SeatShouPaiCount[i] = faPaiCount
	}

	this.PaiMgr.FaPaiOver()
	noticeShouPaiInfo.SurplusPai = this.PaiMgr.GetTheRestOfPaiCount()
	this.table.BroadCastGameEvent(ID_NoticeShouPaiInfo, &noticeShouPaiInfo)
	this.table.AppendTableState(TS_BaoPai)

	// 是否 有玩家 偷,报,杠,胡
	if this.baoFindNextTouMoBao(this.bankerSeatNo, true) == true {
		return mateProto.Err_Success
	}

	// 没人可 偷,报
	this.table.BroadCastGameEvent(B_CurOperationSeatNumber, &MsgBroadcastOperation{this.bankerSeatNo})
	this.NoticeOperation(this.bankerSeatNo, &MsgNoticeOperation{OperationItem: OPI_PlayPai})

	return mateProto.Err_Success
}

func (this *PokerDou14Table) OnTouMo(pro *mateProto.MessageMaTe) int32 {
	if this.table.IsAssignTableState(qpTable.TS_Playing) == false {
		return mateProto.Err_ActionNotMatchStatus
	}

	msgTouMoPai := CS_TouMo{}
	err := json.Unmarshal(pro.Data, &msgTouMoPai)
	if err != nil {
		return mateProto.Err_ProtocolDataErr
	}

	seat, errCode := this.checkOperation(pro.SenderID, msgTouMoPai.OperationID, OPI_TouMo)
	if errCode != mateProto.Err_Success {
		return errCode
	}
	dou14Seat := seat.(*PokerDou14Seat)

	if this.PaiMgr.GetTheRestOfPaiCount() < 1 {
		this.RoundOverFun()
		return 0
	}

	isAnTou, ok := dou14Seat.checkTouPai(msgTouMoPai.Tou, this.curFanChuPai)
	if ok == false {
		errCode = mateProto.Err_OperationParamErr
		return errCode
	}

	if this.curFanChuPai != InvalidPai {
		msgTouMoPai.Tou = SortPai{this.curFanChuPai}
	}

	if len(msgTouMoPai.Tou) < 1 {
		glog.Warning("tou empty....")
		errCode = mateProto.Err_OperationParamErr
		return errCode
	}

	this.table.GameTimer.RemoveBySeatNum(int32(dou14Seat.seatData.Number))
	broadcastTouMo := &SC_BroadcastTouMo{
		SeatNo: int32(dou14Seat.seatData.Number),
		Tou:    msgTouMoPai.Tou,
		IsFan:  this.curFanChuPai,
	}
	if isAnTou {
		noticeSelf := *broadcastTouMo
		this.table.SendGameEventToSeat(dou14Seat.seatData.Number, ID_NoticeAnTou, &noticeSelf)
		broadcastTouMo.Tou = []int8{InvalidPai, InvalidPai, InvalidPai}
	}
	this.table.BroadCastGameEvent(ID_BroadcastTouMo, &broadcastTouMo)

	this.clearAllPlayerOperation()
	this.cleanTableRec()
	this.curPointToSeatNo = dou14Seat.seatData.Number
	this.curMoPaiSeatNo = dou14Seat.seatData.Number
	dou14Seat.PutTou(msgTouMoPai.Tou, this.gameRule.Dian7, isAnTou)
	dou14Seat.lastIsTouPai = true
	dou14Seat.moPaiCount += 1
	for _, v := range msgTouMoPai.Tou {
		dou14Seat.DeleteShouPai(v)
	}
	pai := this.PaiMgr.MoPai(dou14Seat.reserveMoPai)
	dou14Seat.PushShouPai(pai)

	this.table.BroadCastGameEvent(ID_BroadcastMoPai, &SC_BroadcastMoPai{SeatNo: int32(dou14Seat.seatData.Number),
		PaiC: 1, SurplusPai: this.PaiMgr.GetTheRestOfPaiCount()})
	this.table.SendGameEventToSeat(dou14Seat.seatData.Number, ID_NoticeMoPai, &SC_NoticeMoPai{[]int8{pai}})

	// 是否是  报 阶段
	if this.table.IsAssignTableState(TS_BaoPai) {
		if this.baoFindNextTouMoBao(dou14Seat.seatData.Number, true) == false {
			// 没人偷摸,报
			this.table.BroadCastGameEvent(B_CurOperationSeatNumber, &MsgBroadcastOperation{this.bankerSeatNo})
			this.NoticeOperation(this.bankerSeatNo, &MsgNoticeOperation{OperationItem: OPI_PlayPai})
		}
		return 0
	}

	// 翻出大小王,赖子, 直接偷
	huanPai := dou14Seat.IsExistHuanPai()
	if huanPai != InvalidPai {

		notice := MsgNoticeOperation{OperationItem: OPI_TouMo, AllCanGangPai: nil, IsFan: InvalidPai}
		this.RecOperation(dou14Seat.seatData.Number, &notice)

		this.table.GetBaseQPTable().GameTimer.PutSeatTimer(
			int32(dou14Seat.seatData.Number),
			timerAutoPlayPai,
			700, func() {
				msg := mateProto.MessageMaTe{SenderID: int64(dou14Seat.seatData.Player.ID), MessageID: ID_TouMo}
				msg.Data, _ = json.Marshal(&CS_TouMo{OperationID: dou14Seat.seatData.OperationID, Tou: SortPai{huanPai}})
				rspCode := this.table.RootTable.OnMessage(&msg)
				if rspCode != 0 {
					glog.Warning("autoPlayPai() rspCode:=", rspCode)
				}
			})

		return 0
	}

	// 常规游戏进行中
	operationItem := OPI_PlayPai

	// 胡
	if this.isHuPai(dou14Seat, InvalidPai) == true {
		operationItem |= OPI_HU
	}
	// 杠
	isBaoGang := false
	dou14Seat.hasGang(InvalidPai, InvalidPai, qpTable.INVALID_SEAT_NUMBER, &isBaoGang)
	if len(dou14Seat.canGang) > 0 {
		operationItem |= OPI_GANG
	}

	if dou14Seat.seatData.IsAssignSeatState(SS_Bao) {
		if operationItem&OPI_HU == OPI_HU {
			this.RecOperation(dou14Seat.seatData.Number, &MsgNoticeOperation{OperationItem: operationItem})
			this.table.GameTimer.PutSeatTimer(int32(dou14Seat.seatData.Number), timerHuPai, 800, func() {
				this.autoTrusteeshipPlayPai(dou14Seat.seatData.Number, false)
			})
			return 0
		}
		if operationItem&OPI_GANG == OPI_GANG && isBaoGang {
			this.RecOperation(dou14Seat.seatData.Number, &MsgNoticeOperation{OperationItem: operationItem})
			this.table.GameTimer.PutSeatTimer(int32(dou14Seat.seatData.Number), timerBaoGang, 800, func() {
				this.autoGang(dou14Seat.seatData.Number)
			})
			return 0
		}
		this.RecOperation(dou14Seat.seatData.Number, &MsgNoticeOperation{OperationItem: operationItem})
		this.table.GameTimer.PutSeatTimer(int32(dou14Seat.seatData.Number), timerAutoPlayPai, 800, func() {
			this.autoBaoGangPlay(dou14Seat.seatData.Number, pai)
		})
		return 0
	}

	// 偷
	if dou14Seat.hasTouMo(InvalidPai) == true {
		operationItem |= OPI_TouMo
	}

	if operationItem != 0 {
		notice := MsgNoticeOperation{OperationItem: operationItem, AllCanGangPai: dou14Seat.canGang}
		this.NoticeOperation(dou14Seat.seatData.Number, &notice)
	}

	return 0
}

func (this *PokerDou14Table) OnBao(pro *mateProto.MessageMaTe) int32 {
	if this.table.IsAssignTableState(qpTable.TS_Playing|TS_BaoPai) == false {
		return mateProto.Err_ActionNotMatchStatus
	}

	bao_ := CS_Bao{}
	err := json.Unmarshal(pro.Data, &bao_)
	if err != nil {
		return mateProto.Err_ProtocolDataErr
	}

	seat, errCode := this.checkOperation(pro.SenderID, bao_.OperationID, OPI_Bao)
	if errCode != mateProto.Err_Success {
		return errCode
	}
	dou14Seat := seat.(*PokerDou14Seat)
	dou14Seat.seatData.AppendState(SS_Bao)
	this.baoJiaoCount++
	this.clearAllPlayerOperation()

	this.table.GameTimer.RemoveBySeatNum(int32(dou14Seat.seatData.Number))
	this.table.BroadCastGameEvent(ID_BroadcastTouBao, &SC_BroadcastTouBao{
		SeatNo: int32(dou14Seat.seatData.Number),
	})

	// 是否还有其他玩家 在 报 阶段  可操作
	if this.baoFindNextTouMoBao(dou14Seat.seatData.Number, false) == true {
		return 0
	}

	// 没人偷摸,报
	this.table.BroadCastGameEvent(B_CurOperationSeatNumber, &MsgBroadcastOperation{this.bankerSeatNo})
	this.NoticeOperation(this.bankerSeatNo, &MsgNoticeOperation{OperationItem: OPI_PlayPai})

	return 0
}

func (this *PokerDou14Table) baoFindNextTouMoBao(seatNo qpTable.SeatNumber, isFindSelf bool) bool {

	this.OperationTime = time.Now().Unix()

	if isFindSelf == false {
		next := this.GetNextPlayingSeat(seatNo, true)
		if next == nil {
			return false
		}
		if next.seatData.Number == this.bankerSeatNo {
			return false
		}
		seatNo = next.seatData.Number
	}

	for i := 0; i < 4; i++ {
		d14Seat := this.table.SeatArr[seatNo].(*PokerDou14Seat)
		operationItem := D14Operation(0)

		if d14Seat.IsExistHuanPai() != InvalidPai {
			operationItem |= OPI_TouMo
		} else {
			// 偷摸
			if d14Seat.hasTouMo(InvalidPai) == true {
				operationItem |= OPI_TouMo
			}
			// 胡
			if this.isHuPai(d14Seat, InvalidPai) == true {
				operationItem |= OPI_HU
			}
			// 听牌
			if d14Seat.seatData.Number != this.bankerSeatNo && this.logic.HasBao(d14Seat, this.isHuPai) == true {
				operationItem |= OPI_Bao
			} // else if d14Seat.seatData.Number == this.bankerSeatNo && this.logic.BankerHasBao(d14Seat, this.isHuPai) {
			//	operationItem |= OPI_Bao
			//}

			// 杠
			isBaoGang := false
			d14Seat.hasGang(InvalidPai, InvalidPai, qpTable.INVALID_SEAT_NUMBER, &isBaoGang)
			if len(d14Seat.canGang) > 0 {
				operationItem |= OPI_GANG
			}
		}

		this.curPointToSeatNo = d14Seat.seatData.Number
		this.table.BroadCastGameEvent(B_CurOperationSeatNumber, &MsgBroadcastOperation{SeatNumber: seatNo})

		if operationItem != 0 {
			notice := MsgNoticeOperation{OperationItem: operationItem, AllCanGangPai: d14Seat.canGang}
			this.NoticeOperation(seatNo, &notice)
			return true
		}

		d14Seat.baoJiaoed = true
		time.Sleep(time.Millisecond * 800)
		this.table.BroadCastGameEvent(ID_Bao_Guo, &SC_BroadcastBao_Guo{SeatNo: seatNo})
		nextSeat := this.GetNextPlayingSeat(seatNo, true)
		if nextSeat == nil {
			return false
		}
		if nextSeat.baoJiaoed {
			return false
		}
		seatNo = nextSeat.GetSeatData().Number
		if seatNo == this.bankerSeatNo {
			break
		}
	}
	return false
}

func (this *PokerDou14Table) OnPlayPai(pro *mateProto.MessageMaTe) int32 {

	if this.table.IsAssignTableState(qpTable.TS_Playing) == false {
		return mateProto.Err_ActionNotMatchStatus
	}
	if this.table.IsAssignTableState(TS_BaoPai) == true {
		return mateProto.Err_ActionNotMatchStatus
	}

	operPlayPai := CS_PlayPai{}
	err := json.Unmarshal(pro.Data, &operPlayPai)
	if err != nil {
		return mateProto.Err_ProtocolDataErr
	}

	seat, errCode := this.checkOperation(pro.SenderID, operPlayPai.OperationID, OPI_PlayPai)
	if errCode != mateProto.Err_Success {
		return errCode
	}

	dou14Seat := seat.(*PokerDou14Seat)

	if dou14Seat.IsExist(operPlayPai.ChuPai) == false {
		return mateProto.Err_PaiNotExist
	}

	// 某些牌 不能出
	switch operPlayPai.ChuPai {
	case DaWang, XiaoWang, LaiZi:
		return mateProto.Err_OperationParamErr
	default:
	}

	// 庄家报叫 后， 打出牌后，是否 继续听
	if dou14Seat.seatData.Number == this.bankerSeatNo && dou14Seat.seatData.IsAssignSeatState(SS_Bao) {
		if this.logic.CheckBankerBao(dou14Seat, operPlayPai.ChuPai, this.isHuPai) == false {
			return mateProto.Err_OperationParamErr
		}
	}

	lastGangNo_ := this.lastGangSeatNo
	this.clearAllPlayerOperation()
	dou14Seat.DeleteShouPai(operPlayPai.ChuPai)
	dou14Seat.playPai = append(dou14Seat.playPai, operPlayPai.ChuPai)
	dou14Seat.chuPaiCount += 1
	this.cleanTableRec()
	this.curPlaySeatNo = dou14Seat.seatData.Number
	this.curPlayPai = operPlayPai.ChuPai
	this.curPointToSeatNo = dou14Seat.seatData.Number
	this.lastGangSeatNo = lastGangNo_

	// 广播出牌
	this.table.GameTimer.RemoveBySeatNum(int32(dou14Seat.seatData.Number))
	broadPlayerPlayMsg := MsgBroadcastPlayerPai{
		SeatNum: int32(dou14Seat.seatData.Number),
		ChuPai:  operPlayPai.ChuPai}
	this.table.BroadCastGameEvent(B_PlayerPlay, &broadPlayerPlayMsg)

	// 移走 自动出牌定时器
	//this.table.GameTimer.RemoveByTimeID(timerAutoPlayPai)

	// 查找要牌的玩家
	if this.findWhoGetPlayPai() == true {
		return 0
	}

	if this.baoJiaoCount == this.table.MaxPlayers-this.table.LookerCount {
		this.RoundOverFun()
		return 0
	}

	// 没人要,就翻牌
	time.Sleep(time.Millisecond * 800)
	this.fanPai()

	return mateProto.Err_Success
}

func (this *PokerDou14Table) OnChi(pro *mateProto.MessageMaTe) int32 {

	if this.table.IsAssignTableState(qpTable.TS_Playing) == false {
		return mateProto.Err_ActionNotMatchStatus
	}
	if this.table.IsAssignTableState(TS_BaoPai) == true {
		return mateProto.Err_ActionNotMatchStatus
	}

	msgChiPai := CS_Chi{}
	err := json.Unmarshal(pro.Data, &msgChiPai)
	if err != nil {
		return mateProto.Err_ProtocolDataErr
	}

	seat, errCode := this.checkOperation(pro.SenderID, msgChiPai.OperationID, OPI_CHI)
	if errCode != mateProto.Err_Success {
		return errCode
	}

	pai := InvalidPai
	if this.curPlaySeatNo != qpTable.INVALID_SEAT_NUMBER {
		pai = this.curPlayPai
	} else if this.curFanPaiSeatNo != qpTable.INVALID_SEAT_NUMBER {
		pai = this.curFanChuPai
	} else {
		glog.Warning("not match . ", this.table.TableNum)
		return 0
	}

	dou14Seat := seat.(*PokerDou14Seat)
	useLaiZi, isOk := dou14Seat.checkChi(msgChiPai.Pai, pai)
	if isOk == false {
		return mateProto.Err_PaiXingError
	}

	this.table.GameTimer.RemoveBySeatNum(int32(dou14Seat.seatData.Number))

	executeCache, highPriority :=
		this.findPriorityPlayer(dou14Seat, useLaiZi, OPI_CHI, func() { this.doChi(dou14Seat, msgChiPai.Pai) })
	if highPriority == true {
		dou14Seat.seatData.CleanOperationID()
		dou14Seat.curOperationItem = 0
		pro.MsgBody = &protoGameBasic.JsonResponse{Status: 0}
		this.table.SendMsgToSeat(dou14Seat.seatData.Number, pro)
		return 0
	}
	if executeCache == true {
		this.delay_.delayFunc()
		return 0
	}

	this.doChi(dou14Seat, msgChiPai.Pai)

	return 0
}

func (this *PokerDou14Table) doChi(dou14Seat *PokerDou14Seat, chiPai int8) {

	chiArr := []int8{chiPai, 0}
	if this.curFanPaiSeatNo != qpTable.INVALID_SEAT_NUMBER {
		chiArr[1] = this.curFanChuPai
	} else {
		chiArr[1] = this.curPlayPai
	}
	dou14Seat.PutChi(chiArr)
	dou14Seat.DeleteShouPai(chiPai)
	this.clearAllPlayerOperation()

	this.table.BroadCastGameEvent(ID_BroadcastChi, &SC_BroadcastChi{
		SeatNo: int32(dou14Seat.seatData.Number),
		Pai:    chiArr,
	})

	this.NoticeOperation(dou14Seat.seatData.Number, &MsgNoticeOperation{OperationItem: OPI_PlayPai})

	this.cleanTableRec()
	this.curPointToSeatNo = dou14Seat.seatData.Number
}

func (this *PokerDou14Table) OnPeng(pro *mateProto.MessageMaTe) int32 {

	if this.table.IsAssignTableState(qpTable.TS_Playing) == false {
		return mateProto.Err_ActionNotMatchStatus
	}
	if this.table.IsAssignTableState(TS_BaoPai) == true {
		return mateProto.Err_ActionNotMatchStatus
	}

	msgPeng := CS_Peng{}
	err := json.Unmarshal(pro.Data, &msgPeng)
	if err != nil {
		return mateProto.Err_ProtocolDataErr
	}

	seat, errCode := this.checkOperation(pro.SenderID, msgPeng.OperationID, OPI_PENG)
	if errCode != mateProto.Err_Success {
		return errCode
	}

	pai := InvalidPai
	if this.curPlaySeatNo != qpTable.INVALID_SEAT_NUMBER {
		pai = this.curPlayPai
	} else if this.curFanPaiSeatNo != qpTable.INVALID_SEAT_NUMBER {
		pai = this.curFanChuPai
	} else {
		glog.Warning("not match . ", this.table.TableNum)
		return 0
	}

	dou14Seat := seat.(*PokerDou14Seat)
	useLaiZi, isOk := dou14Seat.checkPeng(msgPeng.PaiArr, pai)
	if isOk == false {
		return mateProto.Err_PaiXingError
	}

	this.table.GameTimer.RemoveBySeatNum(int32(dou14Seat.seatData.Number))

	if pai != InvalidPai {
		msgPeng.PaiArr = append(msgPeng.PaiArr, pai)
	}
	executeCache, highPriority := this.findPriorityPlayer(dou14Seat, useLaiZi, OPI_PENG, func() { this.doPeng(dou14Seat, msgPeng.PaiArr) })
	if highPriority {
		dou14Seat.seatData.CleanOperationID()
		dou14Seat.curOperationItem = 0
		pro.MsgBody = &protoGameBasic.JsonResponse{Status: 0}
		this.table.SendMsgToSeat(dou14Seat.seatData.Number, pro)
		return 0
	}
	if executeCache == true {
		this.delay_.delayFunc()
		return 0
	}

	this.doPeng(dou14Seat, msgPeng.PaiArr)
	return 0
}

func (this *PokerDou14Table) doPeng(dou14Seat *PokerDou14Seat, paiArr []int8) {

	dou14Seat.PutPeng(paiArr)
	this.clearAllPlayerOperation()
	for _, v := range paiArr {
		dou14Seat.DeleteShouPai(v)
	}

	this.table.BroadCastGameEvent(ID_BroadcastPeng, &SC_BroadcastPeng{
		SeatNo: int32(dou14Seat.seatData.Number),
		PaiArr: paiArr,
	})

	if this.PaiMgr.GetTheRestOfPaiCount() < 1 {
		this.RoundOverFun()
		return
	}

	this.cleanTableRec()
	this.curPointToSeatNo = dou14Seat.seatData.Number
	this.curMoPaiSeatNo = dou14Seat.seatData.Number
	pai := this.PaiMgr.MoPai(dou14Seat.reserveMoPai)
	dou14Seat.PushShouPai(pai)
	dou14Seat.lastIsTouPai = true

	time.Sleep(time.Millisecond * 800)
	this.table.BroadCastGameEvent(ID_BroadcastMoPai, &SC_BroadcastMoPai{SeatNo: int32(dou14Seat.seatData.Number),
		PaiC: 1, SurplusPai: this.PaiMgr.GetTheRestOfPaiCount()})
	this.table.SendGameEventToSeat(dou14Seat.seatData.Number, ID_NoticeMoPai, &SC_NoticeMoPai{[]int8{pai}})

	operationItem := D14Operation(0)
	huanPai := dou14Seat.IsExistHuanPai()
	if huanPai != InvalidPai {

		notice := MsgNoticeOperation{OperationItem: OPI_TouMo, AllCanGangPai: nil, IsFan: InvalidPai}
		this.RecOperation(dou14Seat.seatData.Number, &notice)

		this.table.GetBaseQPTable().GameTimer.PutSeatTimer(
			int32(dou14Seat.seatData.Number),
			timerAutoPlayPai,
			700, func() {
				msg := mateProto.MessageMaTe{SenderID: int64(dou14Seat.seatData.Player.ID), MessageID: ID_TouMo}
				msg.Data, _ = json.Marshal(&CS_TouMo{OperationID: dou14Seat.seatData.OperationID, Tou: SortPai{huanPai}})
				rspCode := this.table.RootTable.OnMessage(&msg)
				if rspCode != 0 {
					glog.Warning("autoPlayPai() rspCode:=", rspCode)
				}
			})
		return
	}

	if dou14Seat.hasTouMo(InvalidPai) {
		operationItem |= OPI_TouMo
	}
	if this.isHuPai(dou14Seat, InvalidPai) {
		operationItem |= OPI_HU
	}
	isBaoGang := false
	dou14Seat.hasGang(InvalidPai, InvalidPai, qpTable.INVALID_SEAT_NUMBER, &isBaoGang)
	if len(dou14Seat.canGang) > 0 {
		operationItem |= OPI_GANG
	}

	operationItem |= OPI_PlayPai

	notice := MsgNoticeOperation{OperationItem: operationItem, AllCanGangPai: dou14Seat.canGang}
	this.NoticeOperation(dou14Seat.seatData.Number, &notice)
}

func (this *PokerDou14Table) OnGang(pro *mateProto.MessageMaTe) int32 {

	if this.table.IsAssignTableState(qpTable.TS_Playing) == false {
		return mateProto.Err_ActionNotMatchStatus
	}

	msgGang := CS_Gang{}
	err := json.Unmarshal(pro.Data, &msgGang)
	if err != nil {
		return mateProto.Err_ProtocolDataErr
	}

	seat, errCode := this.checkOperation(pro.SenderID, msgGang.OperationID, OPI_GANG)
	if errCode != mateProto.Err_Success {
		return errCode
	}

	dou14Seat := seat.(*PokerDou14Seat)

	dou14Seat._gangInfo = gangInfo{PengIndex: -1, TouIndex: -1, GangIndex: -1, FanPai: InvalidPai, ChuPai: InvalidPai}

	if this.curPlaySeatNo != qpTable.INVALID_SEAT_NUMBER {
		dou14Seat._gangInfo.ChuPai = this.curPlayPai
	} else if this.curFanPaiSeatNo != qpTable.INVALID_SEAT_NUMBER {
		dou14Seat._gangInfo.FanPai = this.curFanChuPai
	} else if this.curMoPaiSeatNo == dou14Seat.seatData.Number {
	} else if this.table.IsAssignTableState(TS_BaoPai) && this.curPointToSeatNo == dou14Seat.seatData.Number {
	} else {
		glog.Warning("not match . ", this.table.TableNum)
		return 0
	}

	ok := dou14Seat.checkGang(msgGang.PaiArr, &dou14Seat._gangInfo)
	if ok == false {
		return mateProto.Err_PaiXingError
	}

	isBaoStatus := this.table.IsAssignTableState(TS_BaoPai)
	if isBaoStatus == true {
		this.doGang(dou14Seat, &dou14Seat._gangInfo, &msgGang)
		return 0
	}

	if len(dou14Seat._gangInfo.gangPaiArr) != 4 {
		glog.Warning("not fang .", this.table.TableNum)
		return 0
	}

	isFindQiangGang := false
	if (dou14Seat._gangInfo.PengIndex >= 0 || dou14Seat._gangInfo.TouIndex >= 0) && this.curMoPaiSeatNo == dou14Seat.seatData.Number {
		isFindQiangGang = true
	}
	// 去掉 抢杠胡
	isFindQiangGang = false

	this.table.GameTimer.RemoveBySeatNum(int32(dou14Seat.seatData.Number))

	if isFindQiangGang {
		this.curGangSeatNo = dou14Seat.seatData.Number
	}
	executeCache, highPriority :=
		this.findPriorityPlayer(dou14Seat, false, OPI_GANG, func() { this.doGang(dou14Seat, &dou14Seat._gangInfo, &msgGang) })
	if highPriority {
		dou14Seat.seatData.CleanOperationID()
		dou14Seat.curOperationItem = 0
		pro.MsgBody = &protoGameBasic.JsonResponse{Status: 0}
		this.table.SendMsgToSeat(dou14Seat.seatData.Number, pro)
		return 0
	}
	if executeCache == true {
		this.delay_.delayFunc()
		return 0
	}
	// 是否能抢杠胡
	if isFindQiangGang {
		isFindQiangGang = false
		gangPai_ := dou14Seat._gangInfo.gangPaiArr[0]
		if len(msgGang.PaiArr) == 1 &&
			(dou14Seat._gangInfo.gangPaiArr[0]&0x0F) == (msgGang.PaiArr[0]&0x0F) {
			gangPai_ = msgGang.PaiArr[0]
		}

		dou14Seat._gangInfo.isQiangGangHu = true
		{
			if dou14Seat._gangInfo.PengIndex >= 0 {
				dou14Seat._gangInfo.ptgBak_ = dou14Seat.pengPai[dou14Seat._gangInfo.PengIndex]
				dou14Seat.pengPai = append(dou14Seat.pengPai[:dou14Seat._gangInfo.PengIndex], dou14Seat.pengPai[dou14Seat._gangInfo.PengIndex+1:]...) // 从碰中删掉
			} else if dou14Seat._gangInfo.TouIndex >= 0 {
				dou14Seat._gangInfo.ptgBak_ = dou14Seat.touPai[dou14Seat._gangInfo.TouIndex]
				dou14Seat.touPai = append(dou14Seat.touPai[:dou14Seat._gangInfo.TouIndex], dou14Seat.touPai[dou14Seat._gangInfo.TouIndex+1:]...) // 从偷中删掉
			} else if dou14Seat._gangInfo.GangIndex >= 0 {
				dou14Seat._gangInfo.ptgBak_ = dou14Seat.gangPai[dou14Seat._gangInfo.GangIndex]
				dou14Seat.gangPai = append(dou14Seat.gangPai[:dou14Seat._gangInfo.GangIndex], dou14Seat.gangPai[dou14Seat._gangInfo.GangIndex+1:]...) // 从杠中删掉
			}
			var anTouPai int8
			dou14Seat.PutGang(dou14Seat._gangInfo.gangPaiArr, &anTouPai)

			this.clearAllPlayerOperation()
			for _, v := range dou14Seat._gangInfo.gangPaiArr {
				dou14Seat.DeleteShouPai(v)
			}

			if anTouPai != InvalidPai && len(msgGang.PaiArr) > 0 {
				anTouPai = msgGang.PaiArr[0]
			}
			this.table.BroadCastGameEvent(ID_BroadcastGang, &SC_BroadcastGang{
				SeatNo:   int32(dou14Seat.seatData.Number),
				PaiArr:   dou14Seat._gangInfo.gangPaiArr,
				GangType: dou14Seat._gangInfo.Category,
				AnTouPai: anTouPai,
			})
		}

		for _, v := range this.table.SeatArr {
			if v == nil {
				continue
			}
			t_ := v.(*PokerDou14Seat)
			if t_.seatData.Number == dou14Seat.seatData.Number {
				continue
			}
			if this.isHuPai(t_, gangPai_) {
				isFindQiangGang = true

				if t_.seatData.IsAssignSeatState(SS_Bao) {
					this.RecOperation(t_.seatData.Number, &MsgNoticeOperation{OperationItem: OPI_HU})
					this.table.GameTimer.PutSeatTimer(int32(t_.seatData.Number), timerHuPai, 800, func() {
						this.autoTrusteeshipPlayPai(t_.seatData.Number, false)
					})
				} else {
					this.NoticeOperation(t_.seatData.Number, &MsgNoticeOperation{OperationItem: OPI_HU, AllCanGangPai: nil})
				}
			}
		}
		if isFindQiangGang {
			dou14Seat.seatData.CleanOperationID()
			dou14Seat.curOperationItem = 0
			pro.MsgBody = &protoGameBasic.JsonResponse{Status: 0}
			this.table.SendMsgToSeat(dou14Seat.seatData.Number, pro)

			this.delay_.seatNo = dou14Seat.seatData.Number
			this.delay_.operItem = OPI_GANG
			this.delay_.delayFunc = func() { this.doGang(dou14Seat, &dou14Seat._gangInfo, &msgGang) }
			return 0
		}
	}

	this.doGang(dou14Seat, &dou14Seat._gangInfo, &msgGang)
	return 0
}

func (this *PokerDou14Table) doGang(dou14Seat *PokerDou14Seat, info *gangInfo, msgGangInfo *CS_Gang) {

	bakGang := dou14Seat.lianGang

	if info.isQiangGangHu == false {
		var prePaiArr SortPai
		if info.PengIndex >= 0 {
			prePaiArr = dou14Seat.pengPai[info.PengIndex]
			dou14Seat.pengPai = append(dou14Seat.pengPai[:info.PengIndex], dou14Seat.pengPai[info.PengIndex+1:]...) // 从碰中删掉
		} else if info.TouIndex >= 0 {
			prePaiArr = dou14Seat.touPai[info.TouIndex]
			dou14Seat.touPai = append(dou14Seat.touPai[:info.TouIndex], dou14Seat.touPai[info.TouIndex+1:]...) // 从偷中删掉
		} else if info.GangIndex >= 0 {
			prePaiArr = dou14Seat.gangPai[info.GangIndex]
			dou14Seat.gangPai = append(dou14Seat.gangPai[:info.GangIndex], dou14Seat.gangPai[info.GangIndex+1:]...) // 从杠中删掉
		}

		redV, blackV := 0, 0
		for _, v := range prePaiArr {
			addDianShu(&redV, &blackV, v)
		}
		dou14Seat.hongDian -= redV
		dou14Seat.heiDian -= blackV

		var anTouPai int8
		dou14Seat.PutGang(info.gangPaiArr, &anTouPai)

		this.clearAllPlayerOperation()
		for _, v := range info.gangPaiArr {
			dou14Seat.DeleteShouPai(v)
		}

		if anTouPai != InvalidPai {
			if len(msgGangInfo.PaiArr) > 0 {
				anTouPai = msgGangInfo.PaiArr[0]
			} else if this.curFanChuPai != InvalidPai {
				anTouPai = this.curFanChuPai
			} else if this.curPlayPai != InvalidPai {
				anTouPai = this.curFanChuPai
			}
		}
		this.table.BroadCastGameEvent(ID_BroadcastGang, &SC_BroadcastGang{
			SeatNo:   int32(dou14Seat.seatData.Number),
			PaiArr:   info.gangPaiArr,
			GangType: info.Category,
			AnTouPai: anTouPai,
		})
	}

	moPaiCount := int32(1)
	if info.useShouPai >= 3 {
		moPaiCount = 2
	}

	if this.PaiMgr.GetTheRestOfPaiCount() < moPaiCount {
		this.RoundOverFun()
		return
	}
	dou14Seat.lianGang = 1 + bakGang

	this.cleanTableRec()
	this.curPointToSeatNo = dou14Seat.seatData.Number
	this.curMoPaiSeatNo = dou14Seat.seatData.Number
	moPaiArr := make([]int8, 0, 2)
	for i := int32(0); i < moPaiCount; i++ {
		pai := this.PaiMgr.MoPai(dou14Seat.reserveMoPai)
		dou14Seat.PushShouPai(pai)
		moPaiArr = append(moPaiArr, pai)
	}

	time.Sleep(time.Millisecond * 800)
	this.table.BroadCastGameEvent(ID_BroadcastMoPai, &SC_BroadcastMoPai{SeatNo: int32(dou14Seat.seatData.Number),
		PaiC: len(moPaiArr), SurplusPai: this.PaiMgr.GetTheRestOfPaiCount()})
	this.table.SendGameEventToSeat(dou14Seat.seatData.Number, ID_NoticeMoPai, &SC_NoticeMoPai{Pai: moPaiArr})

	// 是否是  报 阶段
	if this.table.IsAssignTableState(TS_BaoPai) {
		if this.baoFindNextTouMoBao(dou14Seat.seatData.Number, true) == false {
			// 没人偷摸,报
			this.table.BroadCastGameEvent(B_CurOperationSeatNumber, &MsgBroadcastOperation{this.bankerSeatNo})
			this.NoticeOperation(this.bankerSeatNo, &MsgNoticeOperation{OperationItem: OPI_PlayPai})
		}
		return
	}

	operationItem := D14Operation(0)
	huanPai := dou14Seat.IsExistHuanPai()
	if huanPai != InvalidPai {
		notice := MsgNoticeOperation{OperationItem: OPI_TouMo, AllCanGangPai: nil, IsFan: InvalidPai}
		this.RecOperation(dou14Seat.seatData.Number, &notice)

		this.table.GetBaseQPTable().GameTimer.PutSeatTimer(
			int32(dou14Seat.seatData.Number),
			timerAutoPlayPai,
			700, func() {
				msg := mateProto.MessageMaTe{SenderID: int64(dou14Seat.seatData.Player.ID), MessageID: ID_TouMo}
				msg.Data, _ = json.Marshal(&CS_TouMo{OperationID: dou14Seat.seatData.OperationID, Tou: SortPai{huanPai}})
				rspCode := this.table.RootTable.OnMessage(&msg)
				if rspCode != 0 {
					glog.Warning("autoPlayPai() rspCode:=", rspCode)
				}
			})
		return
	}

	if dou14Seat.seatData.IsAssignSeatState(SS_Bao) {
		this.RecOperation(dou14Seat.seatData.Number, &MsgNoticeOperation{OperationItem: OPI_PlayPai})
		this.table.GetBaseQPTable().GameTimer.PutSeatTimer(
			int32(dou14Seat.seatData.Number),
			timerBaoGangPlay,
			700, func() {
				this.autoBaoGangPlay(dou14Seat.seatData.Number, moPaiArr[0])
			})
		return
	}

	if dou14Seat.hasTouMo(InvalidPai) {
		operationItem |= OPI_TouMo
	}
	if this.isHuPai(dou14Seat, InvalidPai) {
		operationItem |= OPI_HU
	}
	isBaoGang := false
	dou14Seat.hasGang(InvalidPai, InvalidPai, qpTable.INVALID_SEAT_NUMBER, &isBaoGang)
	if len(dou14Seat.canGang) > 0 {
		operationItem |= OPI_GANG
	}

	operationItem |= OPI_PlayPai

	notice := MsgNoticeOperation{OperationItem: operationItem, AllCanGangPai: dou14Seat.canGang}
	this.NoticeOperation(dou14Seat.seatData.Number, &notice)

	this.lastGangSeatNo = dou14Seat.seatData.Number
}

func (this *PokerDou14Table) OnHu(pro *mateProto.MessageMaTe) int32 {

	if this.table.IsAssignTableState(qpTable.TS_Playing) == false {
		return mateProto.Err_ActionNotMatchStatus
	}

	msgHu := CS_Hu{}
	err := json.Unmarshal(pro.Data, &msgHu)
	if err != nil {
		return mateProto.Err_ProtocolDataErr
	}

	seat, errCode := this.checkOperation(pro.SenderID, msgHu.OperationID, OPI_HU)
	if errCode != mateProto.Err_Success {
		return errCode
	}

	dou14Seat := seat.(*PokerDou14Seat)

	this.table.GameTimer.RemoveBySeatNum(int32(dou14Seat.seatData.Number))

	executeCache, highPriority :=
		this.findPriorityPlayer(dou14Seat, false, OPI_HU, func() { this.doHu(dou14Seat) })
	if highPriority {
		dou14Seat.seatData.CleanOperationID()
		dou14Seat.curOperationItem = 0
		pro.MsgBody = &protoGameBasic.JsonResponse{Status: 0}
		this.table.SendMsgToSeat(dou14Seat.seatData.Number, pro)
		return 0
	}
	if executeCache == true {
		this.delay_.delayFunc()
		return 0
	}

	this.doHu(dou14Seat)

	return 0
}

func (this *PokerDou14Table) doHu(winSeat *PokerDou14Seat) {

	if this.curGangSeatNo != qpTable.INVALID_SEAT_NUMBER {
		gangSeat_ := this.table.SeatArr[this.curGangSeatNo].(*PokerDou14Seat)

		gangSeat_.gangPai = gangSeat_.gangPai[:len(gangSeat_.gangPai)-1]
		if gangSeat_._gangInfo.PengIndex >= 0 {
			gangSeat_.pengPai = append(gangSeat_.pengPai, gangSeat_._gangInfo.ptgBak_) // 恢复
		} else if gangSeat_._gangInfo.TouIndex >= 0 {
			gangSeat_.touPai = append(gangSeat_.touPai, gangSeat_._gangInfo.ptgBak_) // 恢复
		} else if gangSeat_._gangInfo.GangIndex >= 0 {
			gangSeat_.gangPai = append(gangSeat_.gangPai, gangSeat_._gangInfo.ptgBak_) // 恢复
		}
	}

	this.table.BroadCastGameEvent(ID_BroadcastHu, &SC_BroadcastHu{SeatNo: int32(winSeat.seatData.Number), Pai: 0})

	this.dianPaoNo = qpTable.INVALID_SEAT_NUMBER
	if this.curPlaySeatNo != qpTable.INVALID_SEAT_NUMBER {
		this.dianPaoNo = this.curPlaySeatNo
	} else if this.curGangSeatNo != qpTable.INVALID_SEAT_NUMBER {
		this.dianPaoNo = this.curGangSeatNo
	}
	if this.dianPaoNo != qpTable.INVALID_SEAT_NUMBER {
		this.table.SeatArr[this.dianPaoNo].(*PokerDou14Seat).dianPaoCount++
	} else {
		winSeat.ziMoCount++
	}
	winSeat.hupaiCount++

	this.huSeatNo = winSeat.seatData.Number
	tHuScore := float64(winSeat.gameScore) * this.gameRule.MultipleFloat64

	for _, v := range this.table.SeatArr {
		if v == nil {
			continue
		}
		t_ := v.(*PokerDou14Seat)
		if t_.seatData.Number == this.huSeatNo {
			continue
		}

		tempScore_ := tHuScore
		if t_.seatData.IsAssignSeatState(SS_Bao) {
			tRec_ := qpTable.GameScoreRec{Score: tHuScore}
			tRec_.PaiXing = []gameMaJiang.HuPaiXing{{PaiXing: HuPX_Bao, FanShu: 1}}
			t_.seatData.PutGameScoreItem(&tRec_, 1)
			tempScore_ += float64(2) * this.gameRule.MultipleFloat64
		}

		if this.dianPaoNo != qpTable.INVALID_SEAT_NUMBER {
			t_ = this.table.SeatArr[this.dianPaoNo].(*PokerDou14Seat)
		}

		if t_.seatData.SeatScore-tempScore_ <= t_.seatData.ClubScore*-1 {
			tempScore_ = t_.seatData.ClubScore + t_.seatData.SeatScore
		}

		t_.seatData.RoundScore -= tempScore_
		t_.seatData.SeatScore -= tempScore_

		winSeat.seatData.RoundScore += tempScore_
	}
	winSeat.seatData.SeatScore += winSeat.seatData.RoundScore

	winSeat.seatData.PutGameScoreItem(&qpTable.GameScoreRec{PaiXing: winSeat.huPaiXin}, 1)

	this.RoundOverFun()

	this.bankerSeatNo = winSeat.seatData.Number
}

func (this *PokerDou14Table) OnGuo(pro *mateProto.MessageMaTe) int32 {

	if this.table.IsAssignTableState(qpTable.TS_Playing) == false {
		return mateProto.Err_ActionNotMatchStatus
	}

	msgGuo := CS_Guo{}
	err := json.Unmarshal(pro.Data, &msgGuo)
	if err != nil {
		return mateProto.Err_ProtocolDataErr
	}

	seat, errCode := this.checkOperation(pro.SenderID, msgGuo.OperationID, 0)
	if errCode != mateProto.Err_Success {
		return errCode
	}

	d14Seat := seat.(*PokerDou14Seat)

	if d14Seat.seatData.IsAssignSeatState(SS_Bao) &&
		(d14Seat.curOperationItem&OPI_HU) == OPI_HU {
		return mateProto.Err_ActionNotMatchStatus
	}

	if d14Seat.IsExistHuanPai() != InvalidPai {
		return mateProto.Err_ActionNotMatchStatus
	}

	this.table.GameTimer.RemoveBySeatNum(int32(d14Seat.seatData.Number))

	pro.MsgBody = &protoGameBasic.JsonResponse{Status: 0}
	this.table.SendMsgToSeat(d14Seat.seatData.Number, pro)

	if d14Seat.curOperationItem&OPI_PlayPai == OPI_PlayPai {
		this.NoticeOperation(d14Seat.seatData.Number, &MsgNoticeOperation{OperationItem: OPI_PlayPai})
		return 0
	}

	this.table.OperateRec.PutPlayerStep(int32(d14Seat.seatData.Number), ID_Guo, nil)
	d14Seat.seatData.CleanOperationID()
	d14Seat.curOperationItem = 0
	d14Seat.lianGang = 0
	d14Seat.lastIsTouPai = false

	// 是否还有玩家未操作
	if this.findNotOperationPlayer() == true {
		return 0
	}

	if this.table.IsAssignTableState(TS_BaoPai) == true {
		d14Seat.baoJiaoed = true
		this.table.BroadCastGameEvent(ID_Bao_Guo, &SC_BroadcastBao_Guo{d14Seat.seatData.Number})
		if this.baoFindNextTouMoBao(d14Seat.seatData.Number, false) == false {
			// 没人可 偷,报
			this.table.BroadCastGameEvent(B_CurOperationSeatNumber, &MsgBroadcastOperation{this.bankerSeatNo})
			this.NoticeOperation(this.bankerSeatNo, &MsgNoticeOperation{OperationItem: OPI_PlayPai})
		}
		return 0
	}

	// 是否已经有玩家操作
	if this.delay_.seatNo != qpTable.INVALID_SEAT_NUMBER {
		this.delay_.delayFunc()
		return 0
	}

	// 自己翻的7，不操作, 查找比人操作
	//if this.curFanPaiSeatNo == d14Seat.seatData.Number && this.curFanChuPai&0x0F == 7 {
	//	chiSeat := this.GetNextChiSeat(this.curFanPaiSeatNo)
	//	isEmpty := true
	//	seatNo := this.curFanPaiSeatNo
	//	for i := 0; i < 4; i++ {
	//		nextSeat := this.GetNextPlayingSeat(seatNo)
	//		if nextSeat == nil {
	//			break
	//		}
	//		if nextSeat.seatData.Number == this.curFanPaiSeatNo {
	//			break
	//		}
	//		seatNo = nextSeat.seatData.Number
	//
	//		tSeat_ := this.table.SeatArr[seatNo].(*PokerDou14Seat)
	//
	//		operationItem := this.checkOperationFunc(tSeat_, chiSeat)
	//		if operationItem != 0 {
	//			isEmpty = false
	//			notice := MsgNoticeOperation{OperationItem: operationItem, AllCanGangPai: tSeat_.canGang, IsFan: this.curFanChuPai}
	//			this.NoticeOperation(seatNo, &notice)
	//		}
	//	}
	//	if isEmpty == false {
	//		return 0
	//	}
	//}

	this.clearAllPlayerOperation()
	this.fanPai()

	return 0
}

func (this *PokerDou14Table) findWhoGetPlayPai() bool {

	chiSeat := this.GetNextChiSeat(this.curPlaySeatNo)

	isFind := false
	for i, v := range this.table.SeatArr {
		if v == nil || i == int(this.curPlaySeatNo) {
			continue
		}
		dou14Seat := v.(*PokerDou14Seat)

		operationItem := D14Operation(0)
		if this.isHuPai(dou14Seat, this.curPlayPai) {
			operationItem |= OPI_HU
		}
		isBaoGang := false
		dou14Seat.hasGang(InvalidPai, this.curPlayPai, qpTable.INVALID_SEAT_NUMBER, &isBaoGang)
		if len(dou14Seat.canGang) > 0 {
			operationItem |= OPI_GANG
		}

		// 报的玩家, 不能有其它操作
		if dou14Seat.seatData.IsAssignSeatState(SS_Bao) == true {
			if (operationItem & OPI_HU) == OPI_HU {
				isFind = true
				this.RecOperation(dou14Seat.seatData.Number, &MsgNoticeOperation{OperationItem: OPI_HU})
				this.table.GameTimer.PutSeatTimer(int32(dou14Seat.seatData.Number), timerHuPai, 800, func() {
					this.autoTrusteeshipPlayPai(dou14Seat.seatData.Number, false)
				})
			} else if (operationItem&OPI_GANG) == OPI_GANG && isBaoGang {
				this.RecOperation(dou14Seat.seatData.Number, &MsgNoticeOperation{OperationItem: operationItem})
				this.table.GameTimer.PutSeatTimer(int32(dou14Seat.seatData.Number), timerBaoGang, 800, func() {
					this.autoGang(dou14Seat.seatData.Number)
				})
			}

		} else {
			isChi := false
			if chiSeat != nil && chiSeat.seatData.Number == dou14Seat.seatData.Number {
				isChi = true
			}

			if dou14Seat.hasPeng(this.curPlayPai) {
				operationItem |= OPI_PENG
			}
			if isChi && dou14Seat.hasChi(this.curPlayPai) {
				operationItem |= OPI_CHI
			}

			if operationItem != 0 {
				isFind = true
				notice := MsgNoticeOperation{OperationItem: operationItem, AllCanGangPai: dou14Seat.canGang, IsPlay: this.curPlayPai}
				this.NoticeOperation(qpTable.SeatNumber(i), &notice)
			}
		}
	}

	return isFind
}

func (this *PokerDou14Table) fanPai() {
	// 游戏结束
	if this.PaiMgr.GetTheRestOfPaiCount() < 1 {
		this.RoundOverFun()
		return
	}

	// 翻出的牌没用，放入 出牌区
	if this.curFanPaiSeatNo != qpTable.INVALID_SEAT_NUMBER && this.curFanChuPai != InvalidPai {
		t_ := this.table.SeatArr[this.curFanPaiSeatNo].(*PokerDou14Seat)
		t_.playPai = append(t_.playPai, this.curFanChuPai)
		this.table.BroadCastGameEvent(ID_FanPaiPutPlay, &SC_BroadcastFanPaiPutPlay{})
	}

	var fanPaiSeat *PokerDou14Seat
	if this.curPlaySeatNo != qpTable.INVALID_SEAT_NUMBER {
		fanPaiSeat = this.GetNextPlayingSeat(this.curPlaySeatNo, false)
		if fanPaiSeat == nil {
			fanPaiSeat = this.table.SeatArr[this.curPlaySeatNo].(*PokerDou14Seat)
		}
	} else if this.curFanPaiSeatNo != qpTable.INVALID_SEAT_NUMBER {
		fanPaiSeat = this.GetNextPlayingSeat(this.curFanPaiSeatNo, false)
		if fanPaiSeat == nil {
			fanPaiSeat = this.table.SeatArr[this.curFanPaiSeatNo].(*PokerDou14Seat)
		}
	} else if this.curMoPaiSeatNo != qpTable.INVALID_SEAT_NUMBER {
		fanPaiSeat = this.GetNextPlayingSeat(this.curMoPaiSeatNo, false)
		if fanPaiSeat == nil {
			fanPaiSeat = this.table.SeatArr[this.curMoPaiSeatNo].(*PokerDou14Seat)
		}
	} else {
		glog.Warning("fanPai --- not match....", this.table.TableNum)
		return
	}

	this.cleanTableRec()
	this.clearAllPlayerOperation()
	this.curFanPaiSeatNo = fanPaiSeat.seatData.Number
	this.curFanChuPai = this.PaiMgr.MoPai(fanPaiSeat.reserveMoPai)
	fanPaiSeat.reserveMoPai = InvalidPai
	this.curPointToSeatNo = fanPaiSeat.seatData.Number

	this.table.BroadCastGameEvent(B_CurOperationSeatNumber, &MsgBroadcastOperation{this.curFanPaiSeatNo})

	// 玩家翻牌
	this.table.BroadCastGameEvent(ID_BroadcastFanPai, &SC_BroadcastFanPai{
		SeatNo:     int32(this.curFanPaiSeatNo),
		Pai:        this.curFanChuPai,
		SurplusPai: this.PaiMgr.GetTheRestOfPaiCount(),
	})

	// 翻出大小王, 直接偷
	if this.curFanChuPai == DaWang || this.curFanChuPai == XiaoWang || this.curFanChuPai == LaiZi ||
		(this.curFanChuPai&0x0F) == 7 {

		notice := MsgNoticeOperation{OperationItem: OPI_TouMo, AllCanGangPai: nil, IsFan: this.curFanChuPai}
		this.RecOperation(fanPaiSeat.seatData.Number, &notice)

		this.table.GetBaseQPTable().GameTimer.PutSeatTimer(
			int32(fanPaiSeat.seatData.Number),
			timerAutoPlayPai,
			700, func() {
				msg := mateProto.MessageMaTe{SenderID: int64(fanPaiSeat.seatData.Player.ID), MessageID: ID_TouMo}
				msg.Data, _ = json.Marshal(&CS_TouMo{OperationID: fanPaiSeat.seatData.OperationID, Tou: SortPai{this.curFanChuPai}})
				rspCode := this.table.RootTable.OnMessage(&msg)
				if rspCode != 0 {
					glog.Warning("autoPlayPai() rspCode:=", rspCode)
				}
			})

		return
	}

	chiSeat := this.GetNextChiSeat(this.curFanPaiSeatNo)

	isEmpty := true
	seatNo := this.curFanPaiSeatNo
	for i := 0; i < 4; i++ {
		tSeat_ := this.table.SeatArr[seatNo].(*PokerDou14Seat)

		operationItem, isNotice := this.checkOperationFunc(tSeat_, chiSeat)
		if operationItem != 0 {
			isEmpty = false
			if isNotice {
				notice := MsgNoticeOperation{OperationItem: operationItem, AllCanGangPai: tSeat_.canGang, IsFan: this.curFanChuPai}
				this.NoticeOperation(seatNo, &notice)
			}
		}

		// 翻的7 只有自己操作
		if this.curFanChuPai&0x0F == 7 && seatNo == this.curFanPaiSeatNo {
			break
		}

		nextSeat := this.GetNextPlayingSeat(seatNo, false)
		if nextSeat == nil {
			break
		}
		if nextSeat.seatData.Number == this.curFanPaiSeatNo {
			break
		}
		seatNo = nextSeat.seatData.Number
	}

	if isEmpty == true {
		this.table.GameTimer.PutTableTimer(timerFanPai, 1000, this.fanPai)
	}
}

// ():操作项,是否需要通知
func (this *PokerDou14Table) checkOperationFunc(seat_ *PokerDou14Seat, chiSeat *PokerDou14Seat) (D14Operation, bool) {
	operationItem := D14Operation(0)

	if this.isHuPai(seat_, this.curFanChuPai) {
		operationItem |= OPI_HU
	}

	// 报的玩家, 不能有其它操作
	if seat_.seatData.IsAssignSeatState(SS_Bao) == true {
		if (operationItem & OPI_HU) == OPI_HU {
			this.RecOperation(seat_.seatData.Number, &MsgNoticeOperation{OperationItem: operationItem})
			this.table.GameTimer.PutSeatTimer(int32(seat_.seatData.Number), timerHuPai, 800, func() {
				this.autoTrusteeshipPlayPai(seat_.seatData.Number, false)
			})
		} else {
			isBaoGang := false
			seat_.hasGang(this.curFanChuPai, InvalidPai, this.curFanPaiSeatNo, &isBaoGang)
			if len(seat_.canGang) > 0 && isBaoGang {
				operationItem |= OPI_GANG

				this.RecOperation(seat_.seatData.Number, &MsgNoticeOperation{OperationItem: operationItem})
				this.table.GameTimer.PutSeatTimer(int32(seat_.seatData.Number), timerBaoGang, 800, func() {
					this.autoGang(seat_.seatData.Number)
				})
			}
		}

		return operationItem, false
	} else {
		isChi := false
		if chiSeat != nil && chiSeat.seatData.Number == seat_.seatData.Number {
			isChi = true
		}

		if seat_.seatData.Number == this.curFanPaiSeatNo {
			if seat_.hasTouMo(this.curFanChuPai) {
				operationItem |= OPI_TouMo
			}
			isChi = true
		}
		if seat_.hasPeng(this.curFanChuPai) {
			operationItem |= OPI_PENG
		}
		if isChi && seat_.hasChi(this.curFanChuPai) {
			operationItem |= OPI_CHI
		}

		isBaoGang := false
		seat_.hasGang(this.curFanChuPai, InvalidPai, this.curFanPaiSeatNo, &isBaoGang)
		if len(seat_.canGang) > 0 {
			operationItem |= OPI_GANG
		}
	}
	return operationItem, true
}

// 是否还有其他玩家未操作
func (this *PokerDou14Table) findNotOperationPlayer() bool {

	opi := OPI_CHI | OPI_PENG | OPI_GANG | OPI_HU
	for _, v := range this.table.SeatArr {
		if v == nil {
			continue
		}
		dou14Seat := v.(*PokerDou14Seat)
		if dou14Seat.curOperationItem&opi != 0 {
			return true
		}
	}

	return false
}

func (this *PokerDou14Table) RoundOverFun() {

	if this.gameRule.LiuJuScore && this.huSeatNo == qpTable.INVALID_SEAT_NUMBER {
		this.checkLiuJuHu()
	}

	roundOver := BroadcastRoundOver{
		TableNumber:   this.GetBaseQPTable().TableNum,
		Timestamp:     time.Now().Unix(),
		HuSeatNo:      int32(this.huSeatNo),
		DianPaoSeatNo: this.dianPaoNo,
		HuPai:         this.huPai,
	}
	msgRoundOver := mateProto.MessageMaTe{MsgBody: &roundOver}

	roundSeatScoreArr := make([]*RoundSeatScore, 0, 3)
	recPlayerGameScoreArr := make([]*protoGameBasic.PlayerGameScore, 0, 3)

	for _, v := range this.table.SeatArr {
		if v == nil {
			continue
		}
		seat := v.(*PokerDou14Seat)
		if seat.seatData.IsAssignSeatState(qpTable.SS_Looker) == true {
			continue
		}
		if seat.seatData.IsAssignSeatState(qpTable.SS_Playing) == false {
			continue
		}
		seat.seatData.RoundOverMsg = &msgRoundOver

		temp := protoGameBasic.PlayerGameScore{
			UID:    int64(seat.seatData.Player.ID),
			Nick:   seat.seatData.Player.Nick,
			ClubID: seat.seatData.ClubID,
			SScore: commonDef.Float64Mul1000ToService(seat.seatData.RoundScore),
		}
		recPlayerGameScoreArr = append(recPlayerGameScoreArr, &temp)

		tempTouPai := seat.touPai
		tempTouPai = append(tempTouPai, seat.huanPai...)
		roundSeatScoreArr = append(roundSeatScoreArr, &RoundSeatScore{
			UID:        int64(seat.seatData.Player.ID),
			NickName:   seat.seatData.Player.Nick,
			Head:       seat.seatData.Player.Head,
			SeatNumber: int32(seat.seatData.Number),
			//DiFen:         seat.diScore,
			WinScore:      seat.gameScore,
			ChiPai:        seat.chiPai,
			PengPai:       seat.pengPai,
			GangPai:       seat.gangPai,
			PlayCard:      seat.playPai,
			TouPai:        tempTouPai,
			ShouPai:       seat.GetAllPai(),
			GameScore:     commonDef.Float64ToString(seat.seatData.RoundScore),
			SeatScore:     commonDef.Float64ToString(seat.seatData.SeatScore),
			GameScoreStep: seat.seatData.GameScoreRecStep})
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

	if this.table.LookerCount > 0 {
		this.table.DissolveType = qpTable.DT_ScoreLess
		this.handleDaJieSuan()
		return
	}

	this.CleanRoundData()
	this.table.TableRule.TimerAutoReady()
}

func (this *PokerDou14Table) handleXiaoJieSuan() {
	if this.table.CurXRound < 1 {
		return
	}

	roundOver := BroadcastRoundOver{
		TableNumber: this.GetBaseQPTable().TableNum,
		Timestamp:   time.Now().Unix(),
	}

	roundSeatScoreArr := make([]*RoundSeatScore, 0, 4)

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

		dou14Seat := v.(*PokerDou14Seat)
		tempTouPai := dou14Seat.touPai
		tempTouPai = append(tempTouPai, dou14Seat.huanPai...)

		roundSeatScoreArr = append(roundSeatScoreArr, &RoundSeatScore{
			UID:        int64(seat.Player.ID),
			NickName:   seat.Player.Nick,
			Head:       seat.Player.Head,
			SeatNumber: int32(dou14Seat.seatData.Number),

			ChiPai:   dou14Seat.chiPai,
			PengPai:  dou14Seat.pengPai,
			GangPai:  dou14Seat.gangPai,
			PlayCard: dou14Seat.playPai,
			TouPai:   tempTouPai,
			ShouPai:  dou14Seat.GetAllPai(),

			GameScoreStep: seat.GameScoreRecStep,

			GameScore: commonDef.Float64ToString(seat.RoundScore),
			SeatScore: commonDef.Float64ToString(seat.SeatScore),
		})

		roundOver.SurplusPaiArr = this.PaiMgr.GetSurplusPai()
		roundOver.SeatData = roundSeatScoreArr
	}
	this.GetBaseQPTable().BroadCastGameEvent(ID_RoundOver, &roundOver)
}

// 大结算
func (this *PokerDou14Table) handleDaJieSuan() {
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

		d14Seat := v.(*PokerDou14Seat)

		tempSeat := &GameOverSeatData{
			ClubID:       d14Seat.seatData.ClubID,
			UID:          int64(d14Seat.seatData.Player.ID),
			Nick:         d14Seat.seatData.Player.Nick,
			Head:         d14Seat.seatData.Player.Head,
			HupaiCount:   d14Seat.hupaiCount,
			DianPaoCount: d14Seat.dianPaoCount,
			ZiMoCount:    d14Seat.ziMoCount,
			SeatScore:    commonDef.Float64ToString(d14Seat.seatData.SeatScore),
			SeatScoreInt: commonDef.Float64Mul1000ToService(d14Seat.seatData.SeatScore),
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

func (this *PokerDou14Table) onTableExpire(pro *mateProto.MessageMaTe) int32 {

	this.table.DissolveType = qpTable.DT_LiveTimeout

	this.handleXiaoJieSuan()

	this.handleDaJieSuan()

	return this.table.OnMessage(pro)
}

func (this *PokerDou14Table) onTableData(pro *mateProto.MessageMaTe) int32 {

	seat := this.table.GetSeatDataByPlayerID(qpTable.PlayerID(pro.SenderID))
	if seat == nil {
		return mateProto.Err_NotFindPlayer
	}

	xiaoJiaNo := qpTable.INVALID_SEAT_NUMBER
	d14SeatArr := make([]*PdkSeatData, 0, 4)

	nowTT := time.Now().Unix()

	for _, v := range this.table.SeatArr {
		if v == nil {
			continue
		}

		dou14Seat := v.(*PokerDou14Seat)
		tempTouPai := make([]SortPai, 0, len(dou14Seat.touPai))

		if dou14Seat.seatData.Number != seat.GetSeatData().Number {
			for i, _ := range dou14Seat.touPai {
				if _, ok := dou14Seat.anTouMap[(dou14Seat.touPai[i][0])&0x0F]; ok == true {
					tempTouPai = append(tempTouPai, SortPai{InvalidPai, InvalidPai, InvalidPai})
				} else {
					tempTouPai = append(tempTouPai, dou14Seat.touPai[i])
				}
			}
		} else {
			tempTouPai = dou14Seat.touPai
		}

		tempTouPai = append(tempTouPai, dou14Seat.huanPai...)

		tempPDKSeat := PdkSeatData{
			UID:           int64(dou14Seat.seatData.Player.ID),
			Nick:          dou14Seat.seatData.Player.Nick,
			HeadURL:       dou14Seat.seatData.Player.Head,
			IP:            dou14Seat.seatData.Player.IP,
			Sex:           dou14Seat.seatData.Player.Sex,
			SeatNumber:    int32(dou14Seat.seatData.Number),
			SeatStatus:    uint32(dou14Seat.seatData.Status),
			SeatScore:     commonDef.Float64ToString(dou14Seat.seatData.SeatScore),
			ClubID:        dou14Seat.seatData.ClubID,
			ClubScore:     commonDef.Float64ToString(dou14Seat.seatData.ClubScore),
			RoundScore:    commonDef.Float64ToString(dou14Seat.seatData.RoundScore),
			ShouPaiCount:  dou14Seat.GetShouPaiCount(),
			VoteStatus:    v.GetSeatData().DissolveVote,
			OperationTime: nowTT - v.GetSeatData().OperationStart,

			ChiPai:   dou14Seat.chiPai,
			PengPai:  dou14Seat.pengPai,
			GangPai:  dou14Seat.gangPai,
			PlayCard: dou14Seat.playPai,
			TouPai:   tempTouPai,
		}

		if dou14Seat.seatData.Lng > 0.1 && dou14Seat.seatData.Lat > 0.1 {
			tempPDKSeat.IsGPS = true
		}

		d14SeatArr = append(d14SeatArr, &tempPDKSeat)
	}

	dou14Seat := seat.(*PokerDou14Seat)

	tableData := MsgTableData{
		MZCID:            this.table.MZClubID,
		TableNumber:      this.table.TableNum,
		TableStatus:      uint32(this.table.Status),
		RoundCount:       this.table.CurXRound,
		TableRuleText:    this.table.TableRule.TableCfgJson,
		ClubRuleText:     this.table.ClubRuleText,
		BankerNo:         int32(this.bankerSeatNo),
		XiaoJiaNo:        int32(xiaoJiaNo),
		MoPaiNo:          int32(this.curMoPaiSeatNo),
		ChuPaiNo:         int32(this.curPlaySeatNo),
		ChuPai:           this.curPlayPai,
		FanPaiNo:         int32(this.curFanPaiSeatNo),
		FanPai:           this.curFanChuPai,
		CurPointToSeatNo: int32(this.curPointToSeatNo),

		SurplusPaiCount:     this.PaiMgr.GetTheRestOfPaiCount(),
		SeatData:            d14SeatArr,
		ShouPai:             dou14Seat.GetAllPai(),
		OperationID:         dou14Seat.seatData.OperationID,
		OperationItem:       int32(dou14Seat.curOperationItem),
		CanGang:             dou14Seat.canGang,
		GameRuleText:        this.gameRule.RuleJson,
		ClubScore:           commonDef.Float64ToString(dou14Seat.seatData.ClubScore),
		DissolveID:          int32(this.table.DissolveSeatNum),
		LaunchDissolveTime:  nowTT - this.table.LaunchDissolveTime,
		PlayerOperationTime: nowTT - this.OperationTime,
	}

	this.table.UpdatePlayerSource(dou14Seat.seatData, pro.Source)

	this.table.SendToSeat(dou14Seat.seatData.Number, CS_TableData, tableData)

	return mateProto.Err_Success
}

func (this *PokerDou14Table) NoticeOperation(seatNumber qpTable.SeatNumber, notice *MsgNoticeOperation) {

	// 庄家出牌
	if seatNumber == this.bankerSeatNo && notice.OperationItem == OPI_PlayPai {
		this.table.DelTableState(TS_BaoPai)
	}

	d14Seat := this.table.SeatArr[seatNumber].(*PokerDou14Seat)
	d14Seat.SetOperationItem(notice.OperationItem)

	// 通知操作
	notice.OperationID = d14Seat.seatData.OperationID
	this.table.SendGameEventToSeat(seatNumber, SC_NoticeOperation, &notice)

	this.timerAutoTrusteeship(seatNumber)
}

func (this *PokerDou14Table) RecOperation(seatNumber qpTable.SeatNumber, notice *MsgNoticeOperation) {

	// 庄家出牌
	if seatNumber == this.bankerSeatNo && notice.OperationItem == OPI_PlayPai {
		this.table.DelTableState(TS_BaoPai)
	}

	d14Seat := this.table.SeatArr[seatNumber].(*PokerDou14Seat)
	d14Seat.SetOperationItem(notice.OperationItem)

	// 通知操作
	notice.OperationID = d14Seat.seatData.OperationID
	//this.table.SendGameEventToSeat(seatNumber, SC_NoticeOperation, &notice)
}

func (this *PokerDou14Table) clearAllPlayerOperation() {

	for _, v := range this.table.SeatArr {
		if v == nil {
			continue
		}
		dou14Seat := v.(*PokerDou14Seat)
		dou14Seat.seatData.CleanOperationID()
		dou14Seat.curOperationItem = 0
		dou14Seat.lastIsTouPai = false
		dou14Seat.lianGang = 0
	}
	this.lastGangSeatNo = qpTable.INVALID_SEAT_NUMBER
}

func heiHong(r, b *bool, pai int8) {
	hua := pai & 0x70
	if hua == HongTao || hua == FangKuai {
		*b = false
	} else {
		*r = false
	}
}

func (this *PokerDou14Table) isHuPai(dou14Seat *PokerDou14Seat, fanChuPai int8) bool {

	this.logic.isHuPai(dou14Seat.shouPaiMap, fanChuPai)
	if this.logic._huDuiZi.duiZi_ != 4 && this.logic._hu14.isHu == false {
		dou14Seat.gameScore, dou14Seat.huPaiXin = 0, nil
		return false
	}
	redDian, blackDian := 0, 0
	isAllHongPai, isAllHeiPai := true, true

	if this.logic._huDuiZi.duiZi_ == 4 {
		redDian, blackDian = this.logic._huDuiZi.hongDian, this.logic._huDuiZi.heiDian
		isAllHongPai, isAllHeiPai = this.logic._huDuiZi.isAllHongPai, this.logic._huDuiZi.isAllHeiPai
	} else if this.logic._hu14.isHu == true {
		redDian, blackDian = this.logic._hu14.hongDian, this.logic._hu14.heiDian
		isAllHongPai, isAllHeiPai = this.logic._hu14.isAllHongPai, this.logic._hu14.isAllHeiPai
	}

	redDian += dou14Seat.hongDian
	blackDian += dou14Seat.heiDian

	//huDianShu := redDian + blackDian

	paiXingArr := make([]gameMaJiang.HuPaiXing, 0, 3)
	fanCount := int64(1)

	{
		// 手牌
		for _, v := range dou14Seat.shouPaiMap {
			for k, _ := range v {
				heiHong(&isAllHongPai, &isAllHeiPai, k)
			}
		}

		if fanChuPai != InvalidPai {
			heiHong(&isAllHongPai, &isAllHeiPai, fanChuPai)
		}

		//吃
		for _, v := range dou14Seat.chiPai {
			for _, pai_ := range v {
				heiHong(&isAllHongPai, &isAllHeiPai, pai_)
			}
		}

		// 碰
		if len(dou14Seat.pengPai) > 0 {
			isAllHongPai = false
			isAllHeiPai = false
			fanCount += int64(len(dou14Seat.pengPai))
		}

		// 杠
		if len(dou14Seat.gangPai) > 0 {
			isAllHongPai = false
			isAllHeiPai = false
			fanCount += int64(len(dou14Seat.gangPai)) * 2
		}

		// 偷
		if len(dou14Seat.touPai) > 0 {
			isAllHongPai = false
			isAllHeiPai = false
			fanCount += int64(len(dou14Seat.touPai))
		}
		// 换
		if len(dou14Seat.huanPai) > 0 {
			fanCount += int64(len(dou14Seat.huanPai))
		}
	}

	if this.logic._huDuiZi.duiZi_ == 4 {
		huDuiZi := false
		if this.gameRule.IsAnyHu || redDian >= 20 {
			huDuiZi = true
		} else if isAllHeiPai || isAllHongPai || blackDian >= 50 {
			huDuiZi = true
		}
		if huDuiZi {
			if this.logic._huDuiZi.longDui {
				paiXingArr = append(paiXingArr, gameMaJiang.HuPaiXing{PaiXing: HuPX_LongDui, FanShu: 6})
				fanCount += paiXingArr[len(paiXingArr)-1].FanShu
			} else {
				paiXingArr = append(paiXingArr, gameMaJiang.HuPaiXing{PaiXing: HuPX_XiaoDui, FanShu: 3})
				fanCount += paiXingArr[len(paiXingArr)-1].FanShu
			}
		}

		if len(paiXingArr) < 1 {
			dou14Seat.gameScore, dou14Seat.huPaiXin = 0, nil
			return false
		}
	}
	// 全黑
	if isAllHeiPai {
		paiXingArr = append(paiXingArr, gameMaJiang.HuPaiXing{PaiXing: HuPX_AllHei, FanShu: 3})
		fanCount += paiXingArr[len(paiXingArr)-1].FanShu
	}

	// 全红
	if isAllHongPai {
		paiXingArr = append(paiXingArr, gameMaJiang.HuPaiXing{PaiXing: HuPX_AllHong, FanShu: 2})
		fanCount += paiXingArr[len(paiXingArr)-1].FanShu
	}

	// 黑50
	if blackDian >= 50 {
		paiXingArr = append(paiXingArr, gameMaJiang.HuPaiXing{PaiXing: HuPX_Hei50, FanShu: 4})
		fanCount += paiXingArr[len(paiXingArr)-1].FanShu
	}

	if redDian < 20 && len(paiXingArr) < 1 {
		dou14Seat.gameScore, dou14Seat.huPaiXin = 0, nil
		return false
	}

	// 金20
	if redDian == 20 {
		paiXingArr = append(paiXingArr, gameMaJiang.HuPaiXing{PaiXing: HuPX_Jin20, FanShu: 1})
		fanCount += paiXingArr[len(paiXingArr)-1].FanShu
	}

	// 金钩钓
	if dou14Seat.GetShouPaiCount() == 1 {
		paiXingArr = append(paiXingArr, gameMaJiang.HuPaiXing{PaiXing: HuPX_JinGouDiao, FanShu: 1})
		fanCount += paiXingArr[len(paiXingArr)-1].FanShu
	}

	// 报叫
	if dou14Seat.seatData.IsAssignSeatState(SS_Bao) {
		paiXingArr = append(paiXingArr, gameMaJiang.HuPaiXing{PaiXing: HuPX_Bao, FanShu: 2})
		fanCount += paiXingArr[len(paiXingArr)-1].FanShu
	}

	// 天胡
	if dou14Seat.seatData.Number == this.bankerSeatNo {
		if dou14Seat.chuPaiCount < 1 &&
			dou14Seat.moPaiCount < 1 &&
			len(dou14Seat.touPai) < 1 {
			paiXingArr = append(paiXingArr, gameMaJiang.HuPaiXing{PaiXing: HuPx_TianHu, FanShu: 3})
			fanCount += paiXingArr[len(paiXingArr)-1].FanShu
		}
	}
	// 头炮/ 地胡
	if this.curPlaySeatNo == this.bankerSeatNo &&
		this.table.SeatArr[this.bankerSeatNo].(*PokerDou14Seat).chuPaiCount == 1 {
		paiXingArr = append(paiXingArr, gameMaJiang.HuPaiXing{PaiXing: HuPX_DiHu, FanShu: 3})
		fanCount += paiXingArr[len(paiXingArr)-1].FanShu
	}

	// 红50
	if redDian >= 50 {
		paiXingArr = append(paiXingArr, gameMaJiang.HuPaiXing{PaiXing: HuPX_Hong50, FanShu: 4})
		fanCount += paiXingArr[len(paiXingArr)-1].FanShu
	}

	// 偷后胡牌(杠上花)
	if dou14Seat.lastIsTouPai {
		paiXingArr = append(paiXingArr, gameMaJiang.HuPaiXing{PaiXing: HuPX_MoPaiHu, FanShu: 1})
		fanCount += paiXingArr[len(paiXingArr)-1].FanShu
	}
	// 杠后出牌 胡
	if this.lastGangSeatNo != qpTable.INVALID_SEAT_NUMBER {
		paiXingArr = append(paiXingArr, gameMaJiang.HuPaiXing{PaiXing: HuPX_GangShangPao, FanShu: 1})
		fanCount += paiXingArr[len(paiXingArr)-1].FanShu
	}
	// 抢杠胡
	if this.curGangSeatNo != qpTable.INVALID_SEAT_NUMBER {
		paiXingArr = append(paiXingArr, gameMaJiang.HuPaiXing{PaiXing: HuPX_QiangGangHu, FanShu: 1})
		fanCount += paiXingArr[len(paiXingArr)-1].FanShu
	}

	// 自摸
	if this.curFanPaiSeatNo == dou14Seat.seatData.Number ||
		this.curMoPaiSeatNo == dou14Seat.seatData.Number {
		paiXingArr = append(paiXingArr, gameMaJiang.HuPaiXing{PaiXing: HuPX_ZiMo, FanShu: 1})
		fanCount += paiXingArr[len(paiXingArr)-1].FanShu
	}

	dou14Seat.gameScore = int(fanCount)
	dou14Seat.huPaiXin = nil
	dou14Seat.huPaiXin = append(dou14Seat.huPaiXin, paiXingArr...)

	this.huPai = fanChuPai

	return true
}

// 检查操作
func (this *PokerDou14Table) checkOperation(playerID int64, operationID string, operItem D14Operation) (qpTable.QPSeat, int32) {
	seat := this.table.GetSeatDataByPlayerID(qpTable.PlayerID(playerID))
	if seat == nil {
		return nil, mateProto.Err_NotFindPlayer
	}
	dou14Seat := seat.(*PokerDou14Seat)
	if len(dou14Seat.seatData.OperationID) < 1 {
		return nil, mateProto.Err_OperationIDErr
	}

	if dou14Seat.seatData.OperationID != operationID {
		return nil, mateProto.Err_OperationIDErr
	}

	if operItem == 0 {
		if dou14Seat.curOperationItem == 0 {
			return nil, mateProto.Err_NotYouOperation
		}
	} else if dou14Seat.curOperationItem&operItem != operItem {
		return nil, mateProto.Err_NotYouOperation
	}

	return seat, mateProto.Err_Success
}

func (this *PokerDou14Table) timerAutoTrusteeship(seatNum qpTable.SeatNumber) {

	tempTime := this.table.TableRule.TuoGuanTime * 1000

	if this.table.TableRule.TuoGuanTime < 1 {
		// 是否 主动开启了 托管
		if this.table.SeatArr[seatNum].GetSeatData().IsAssignSeatState(qpTable.SS_Trusteeship) == false {
			return
		}
		tempTime = 800
	} else {
		if this.table.SeatArr[seatNum].GetSeatData().IsAssignSeatState(qpTable.SS_Trusteeship) == true {
			tempTime = 800
		}
	}

	this.table.GetBaseQPTable().GameTimer.PutSeatTimer(
		int32(seatNum),
		timerAutoPlayPai,
		tempTime, func() {
			this.autoTrusteeshipPlayPai(seatNum, true)
		})
}

func (this *PokerDou14Table) autoTrusteeshipPlayPai(seatNum qpTable.SeatNumber, changedStatus bool) {
	if this.table.IsAssignTableState(qpTable.TS_Playing) == false {
		return
	}

	dou14Seat := this.table.SeatArr[seatNum].(*PokerDou14Seat)

	if dou14Seat.curOperationItem < 1 {
		return
	}

	if changedStatus && dou14Seat.seatData.IsAssignSeatState(qpTable.SS_Trusteeship) == false {
		dou14Seat.seatData.AppendState(qpTable.SS_Trusteeship)
		this.table.NotifyPlayerStateChange(dou14Seat.seatData.Number)
	}

	msg := mateProto.MessageMaTe{SenderID: int64(dou14Seat.seatData.Player.ID)}

	if dou14Seat.curOperationItem&OPI_HU == OPI_HU {
		hu_ := CS_Hu{OperationID: dou14Seat.seatData.OperationID}
		msg.Data, _ = json.Marshal(&hu_)
		msg.MessageID = ID_Hu
	} else if dou14Seat.curOperationItem&OPI_PlayPai == OPI_PlayPai {
		playPai_ := CS_PlayPai{OperationID: dou14Seat.seatData.OperationID}

		for k, v := range dou14Seat.shouPaiMap {
			if k == 0x0E {
				playPai_.ChuPai = XiaoWang
				continue
			}
			if k == 0x0F {
				playPai_.ChuPai = DaWang
				continue
			}
			if k == 0 {
				playPai_.ChuPai = LaiZi
				continue
			}
			if len(v) < 1 {
				continue
			}
			for pai, _ := range v {
				playPai_.ChuPai = pai
				break
			}
			break
		}

		msg.Data, _ = json.Marshal(&playPai_)
		msg.MessageID = ID_Play
	} else {
		if dou14Seat.curOperationItem&OPI_TouMo == OPI_TouMo {
			huanPai := dou14Seat.IsExistHuanPai()
			if huanPai != InvalidPai {
				touMo_ := CS_TouMo{OperationID: dou14Seat.seatData.OperationID}
				touMo_.Tou = SortPai{huanPai}
				msg.Data, _ = json.Marshal(&touMo_)
				msg.MessageID = ID_TouMo
			}
		}

		if len(msg.Data) < 1 {
			guo_ := CS_Guo{OperationID: dou14Seat.seatData.OperationID}
			msg.Data, _ = json.Marshal(&guo_)
			msg.MessageID = ID_Guo
		}
	}

	rspCode := this.table.RootTable.OnMessage(&msg)
	if rspCode != 0 {
		glog.Warning("autoPlayPai() rspCode:=", rspCode)
	}
}

//func (this *PokerDou14Table) autoHuPai(seatNum qpTable.SeatNumber) {
//	if this.table.IsAssignTableState(qpTable.TS_Playing) == false {
//		return
//	}
//
//	dou14Seat := this.table.SeatArr[seatNum].(*PokerDou14Seat)
//
//	if dou14Seat.curOperationItem < 1 {
//		return
//	}
//
//	msg := mateProto.MessageMaTe{SenderID: int64(dou14Seat.seatData.Player.ID)}
//
//	rspCode := this.table.RootTable.OnMessage(&msg)
//	if rspCode != 0 {
//		glog.Warning("autoHuPai() rspCode:=", rspCode)
//	}
//}

func (this *PokerDou14Table) autoGang(seatNum qpTable.SeatNumber) {
	if this.table.IsAssignTableState(qpTable.TS_Playing) == false {
		return
	}

	dou14Seat := this.table.SeatArr[seatNum].(*PokerDou14Seat)

	if dou14Seat.curOperationItem < 1 {
		return
	}

	msg := mateProto.MessageMaTe{SenderID: int64(dou14Seat.seatData.Player.ID)}

	msg.Data, _ = json.Marshal(&CS_Gang{OperationID: dou14Seat.seatData.OperationID})
	msg.MessageID = ID_Gang

	rspCode := this.table.RootTable.OnMessage(&msg)
	if rspCode != 0 {
		glog.Warning("autoGangPai() rspCode:=", rspCode)
	}
}

func (this *PokerDou14Table) autoBaoGangPlay(seatNum qpTable.SeatNumber, playPai int8) {
	if this.table.IsAssignTableState(qpTable.TS_Playing) == false {
		return
	}

	dou14Seat := this.table.SeatArr[seatNum].(*PokerDou14Seat)

	msg := mateProto.MessageMaTe{SenderID: int64(dou14Seat.seatData.Player.ID)}

	msg.Data, _ = json.Marshal(&CS_PlayPai{OperationID: dou14Seat.seatData.OperationID, ChuPai: playPai})
	msg.MessageID = ID_Play

	rspCode := this.table.RootTable.OnMessage(&msg)
	if rspCode != 0 {
		glog.Warning("autoBaoGangPlay() rspCode:=", rspCode)
	}
}

// :是否需要执行暂存操作, 是否能操作
func (this *PokerDou14Table) findPriorityPlayer(dou14Seat *PokerDou14Seat, isLaiZi bool, curOperation D14Operation, f func()) (bool, bool) {

	findHigh := false
	switch curOperation {
	case OPI_CHI:
		// 先找 更高优先级的
		height := OPI_PENG | OPI_GANG | OPI_HU
		// 是否是翻牌者 自己吃
		if this.curFanPaiSeatNo != qpTable.INVALID_SEAT_NUMBER &&
			dou14Seat.seatData.Number != this.curFanPaiSeatNo {
			height |= OPI_CHI
		} else if this.curPlaySeatNo != qpTable.INVALID_SEAT_NUMBER {
			// 是否是 出牌的 下下一个人
			playNextSeat := this.GetNextPlayingSeat(this.curPlaySeatNo, false)
			if playNextSeat != nil {
				playNextNextSeat := this.GetNextPlayingSeat(playNextSeat.seatData.Number, false)
				if playNextNextSeat != nil && playNextNextSeat.seatData.Number == dou14Seat.seatData.Number {
					height |= OPI_CHI
				}
			}
		}
		for _, v := range this.table.SeatArr {
			if v == nil {
				continue
			}
			tempSeat_ := v.(*PokerDou14Seat)
			if tempSeat_.seatData.Number == dou14Seat.seatData.Number {
				continue
			}

			if tempSeat_.curOperationItem&height != 0 {
				findHigh = true
				break
			}
		}

		isReWrite := false

		// 对比已经操作的项 ,是否比我高
		if curOperation > this.delay_.operItem {
			isReWrite = true
		} else if curOperation == this.delay_.operItem &&
			height&curOperation == 0 {
			// 同级别操作,是否是最高的
			isReWrite = true
		}

		if isReWrite {
			// 记录|覆盖
			this.delay_.seatNo = dou14Seat.seatData.Number
			this.delay_.operItem = curOperation
			this.delay_.delayFunc = f
		}

		// 没有更高的优先级  &&  能覆盖掉记录
		if findHigh == false && isReWrite == true {
			return false, false
		}

		// 没有更高优先级 && 又不能覆盖掉暂存 && 暂存中已有回调
		if findHigh == false && isReWrite == false && this.delay_.seatNo != qpTable.INVALID_SEAT_NUMBER {
			return true, false
		}

		return false, true
	case OPI_PENG:
		height := OPI_GANG | OPI_HU
		if isLaiZi == true {
			height |= OPI_PENG
		}
		// 是否是翻牌者 自己碰
		if this.curFanPaiSeatNo != qpTable.INVALID_SEAT_NUMBER &&
			dou14Seat.seatData.Number != this.curFanPaiSeatNo {
			height |= OPI_PENG
		}

		for _, v := range this.table.SeatArr {
			if v == nil {
				continue
			}
			tempSeat_ := v.(*PokerDou14Seat)
			if tempSeat_.seatData.Number == dou14Seat.seatData.Number {
				continue
			}

			if tempSeat_.curOperationItem&height != 0 {
				findHigh = true
				break
			}
		}

		isReWrite := false

		// 对比已经操作的项 ,是否比我高
		if curOperation > this.delay_.operItem {
			isReWrite = true
		} else if curOperation == this.delay_.operItem &&
			height&curOperation == 0 {
			// 同级别操作,是否是最高的
			isReWrite = true
		}

		if isReWrite {
			// 记录|覆盖
			this.delay_.seatNo = dou14Seat.seatData.Number
			this.delay_.operItem = curOperation
			this.delay_.delayFunc = f
		}

		// 没有更高的优先级  &&  能覆盖掉记录
		if findHigh == false && isReWrite == true {
			return false, false
		}

		// 没有更高优先级 && 又不能覆盖掉暂存 && 暂存中已有回调
		if findHigh == false && isReWrite == false && this.delay_.seatNo != qpTable.INVALID_SEAT_NUMBER {
			return true, false
		}

		return false, true
	case OPI_GANG:
		// 自己摸的牌
		if this.curFanPaiSeatNo == qpTable.INVALID_SEAT_NUMBER &&
			this.curPlaySeatNo == qpTable.INVALID_SEAT_NUMBER {
			return false, false
		}

		height := OPI_HU

		for _, v := range this.table.SeatArr {
			if v == nil {
				continue
			}
			tempSeat_ := v.(*PokerDou14Seat)
			if tempSeat_.seatData.Number == dou14Seat.seatData.Number {
				continue
			}

			if tempSeat_.curOperationItem&height != 0 {
				findHigh = true
				break
			}
		}

		isReWrite := false

		// 对比已经操作的项 ,是否比我高
		if curOperation > this.delay_.operItem {
			isReWrite = true
		} else if curOperation == this.delay_.operItem &&
			height&curOperation == 0 {
			// 同级别操作,是否是最高的
			isReWrite = true
		}

		if isReWrite {
			// 记录|覆盖
			this.delay_.seatNo = dou14Seat.seatData.Number
			this.delay_.operItem = curOperation
			this.delay_.delayFunc = f
		}

		// 没有更高的优先级  &&  能覆盖掉记录
		if findHigh == false && isReWrite == true {
			return false, false
		}

		// 没有更高优先级 && 又不能覆盖掉暂存 && 暂存中已有回调
		if findHigh == false && isReWrite == false && this.delay_.seatNo != qpTable.INVALID_SEAT_NUMBER {
			return true, false
		}

		return false, true
	case OPI_HU:
		lastSeatNo := dou14Seat.seatData.Number
		if this.curFanPaiSeatNo != qpTable.INVALID_SEAT_NUMBER {
			lastSeatNo = this.curFanPaiSeatNo
		} else if this.curPlaySeatNo != qpTable.INVALID_SEAT_NUMBER {
			lastSeatNo = this.curPlaySeatNo
		} else if this.curMoPaiSeatNo == dou14Seat.seatData.Number {
			return false, false
		} else if this.curGangSeatNo != qpTable.INVALID_SEAT_NUMBER {
			lastSeatNo = this.curGangSeatNo
		} else if this.table.IsAssignTableState(TS_BaoPai) && this.curPointToSeatNo == dou14Seat.seatData.Number {
			return false, false
		} else {
			glog.Warning("findPriorityPlayer ", this.table.TableNum)
			return false, true
		}

		//isFindGang := false
		//// 抢杠胡  只能在翻牌的时候  杠.
		//if this.curFanPaiSeatNo != qpTable.INVALID_SEAT_NUMBER {
		//	for _, v := range this.table.SeatArr {
		//		if v == nil {
		//			continue
		//		}
		//		ts := v.(*PokerDou14Seat)
		//		if ts.seatData.Number == this.curFanPaiSeatNo ||
		//			ts.seatData.Number == dou14Seat.seatData.Number {
		//			continue
		//		}
		//		if ts.curOperationItem&OPI_GANG != 0 {
		//			isFindGang = true
		//			break
		//		}
		//	}
		//}

		higherHu := false // 是否有更高优先级的玩家 已经点胡
		// 按优先顺序查找
		next := dou14Seat.seatData.Number
		for i := 0; i < 4; i++ {
			if next == lastSeatNo {
				break
			}
			t_ := this.GetPreviousPlayingSeat(next)
			if t_ == nil {
				return false, false
			}
			ts := t_.(*PokerDou14Seat)
			if ts.curOperationItem&OPI_HU != 0 {
				findHigh = true
				break
			}
			if this.delay_.seatNo == ts.seatData.Number && this.delay_.operItem == OPI_HU {
				higherHu = true
			}
			next = ts.seatData.Number
		}

		isReWrite := false

		// 对比已经操作的项 ,是否比我高
		if curOperation > this.delay_.operItem {
			isReWrite = true
		} else if curOperation == this.delay_.operItem {
			if higherHu == false {
				isReWrite = true
			}
		}

		if isReWrite {
			// 记录|覆盖
			this.delay_.seatNo = dou14Seat.seatData.Number
			this.delay_.operItem = curOperation
			this.delay_.delayFunc = f
		}

		//// 等待杠的玩家操作
		//if isFindGang {
		//	return false, true
		//}

		// 没有更高的优先级  &&  能覆盖掉记录
		if findHigh == false && isReWrite == true {
			return false, false
		}

		// 没有更高优先级 && 又不能覆盖掉暂存 && 暂存中已有回调
		if findHigh == false && isReWrite == false && this.delay_.seatNo != qpTable.INVALID_SEAT_NUMBER {
			return true, false
		}

		return false, true
	default:
	}

	glog.Warning("not match ....", this.table.TableNum)

	return false, true
}

// 激活托管
func (this *PokerDou14Table) OnActiveTrusteeship(msg *mateProto.MessageMaTe) int32 {
	seatNum := this.table.OnActiveTrusteeship(msg)
	if seatNum < 0 {
		return seatNum
	}
	dou14Seat := this.table.SeatArr[seatNum].(*PokerDou14Seat)
	if dou14Seat.seatData.IsAssignSeatState(qpTable.SS_Trusteeship) == false {
		dou14Seat.seatData.AppendState(qpTable.SS_Trusteeship)
		this.table.NotifyPlayerStateChange(dou14Seat.seatData.Number)
	}
	if dou14Seat.curOperationItem == 0 {
		return 0
	}

	this.timerAutoTrusteeship(qpTable.SeatNumber(seatNum))

	return seatNum
}

// 获取正在玩的下一位玩家
func (this *PokerDou14Table) GetNextPlayingSeat(cur qpTable.SeatNumber, notBao bool) *PokerDou14Seat {

	for i := cur + 1; int(i) < len(this.table.SeatArr); i++ {
		if this.table.SeatArr[i] == nil {
			continue

		}
		if this.table.SeatArr[i].GetSeatData().IsAssignSeatState(qpTable.SS_Playing) {
			if notBao {
				if this.table.SeatArr[i].GetSeatData().IsAssignSeatState(SS_Bao) == false {
					return this.table.SeatArr[i].(*PokerDou14Seat)
				}
			} else {
				return this.table.SeatArr[i].(*PokerDou14Seat)
			}
		}
	}

	for i := 0; qpTable.SeatNumber(i) < cur; i++ {
		if this.table.SeatArr[i] == nil {
			continue

		}
		if this.table.SeatArr[i].GetSeatData().IsAssignSeatState(qpTable.SS_Playing) {
			if notBao {
				if this.table.SeatArr[i].GetSeatData().IsAssignSeatState(SS_Bao) == false {
					return this.table.SeatArr[i].(*PokerDou14Seat)
				}
			} else {
				return this.table.SeatArr[i].(*PokerDou14Seat)
			}
		}
	}
	return nil
}

// 获取正在玩的上一位玩家
func (this *PokerDou14Table) GetPreviousPlayingSeat(cur qpTable.SeatNumber) qpTable.QPSeat {

	for i := cur - 1; i >= 0; i-- {
		if this.table.SeatArr[i] == nil {
			continue

		}
		if this.table.SeatArr[i].GetSeatData().IsAssignSeatState(qpTable.SS_Playing) == true {
			return this.table.SeatArr[i]
		}
	}

	for i := len(this.table.SeatArr) - 1; int(cur) < i; i-- {
		if this.table.SeatArr[i] == nil {
			continue

		}
		if this.table.SeatArr[i].GetSeatData().IsAssignSeatState(qpTable.SS_Playing) == true {
			return this.table.SeatArr[i]
		}
	}
	return nil
}

// 获取能吃的玩家
func (this *PokerDou14Table) GetNextChiSeat(cur qpTable.SeatNumber) *PokerDou14Seat {

	if this.curFanPaiSeatNo == cur {
		next := this.GetNextPlayingSeat(cur, false)
		if next != nil && next.seatData.IsAssignSeatState(SS_Bao) == false {
			return next
		}
		return nil
	}

	for i := cur + 1; int(i) < len(this.table.SeatArr); i++ {
		if this.table.SeatArr[i] == nil {
			continue
		}
		v := this.table.SeatArr[i].(*PokerDou14Seat)
		if v.seatData.IsAssignSeatState(qpTable.SS_Playing) == true {
			if v.seatData.IsAssignSeatState(SS_Bao) == false {
				return v
			}
		}
	}

	for i := 0; qpTable.SeatNumber(i) < cur; i++ {
		if this.table.SeatArr[i] == nil {
			continue

		}
		v := this.table.SeatArr[i].(*PokerDou14Seat)
		if v.seatData.IsAssignSeatState(qpTable.SS_Playing) == true {
			if v.seatData.IsAssignSeatState(SS_Bao) == false {
				return v
			}
		}
	}
	return nil
}

func (this *PokerDou14Table) onCancelTrusteeship(msg *mateProto.MessageMaTe) int32 {
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

func (this *PokerDou14Table) onCustomShouPai(pro *mateProto.MessageMaTe) int32 {
	msgCustomShouPai := CS_CustomShouPai{}
	err := json.Unmarshal(pro.Data, &msgCustomShouPai)
	if err != nil {
		return mateProto.Err_ProtocolDataErr
	}

	if len(msgCustomShouPai.ShouPai) > 8 {
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

	pai, rsp := this.PaiMgr.ReserveShouPai(msgCustomShouPai.ShouPai)
	if rsp != 0 {
		rspMsg.Status = mateProto.Err_CustomPai
		if rsp == -1 {
			rspMsg.Detail = fmt.Sprintf("used pai:=%d", pai)
		} else if rsp == -2 {
			rspMsg.Detail = fmt.Sprintf("invalid pai:=%d", pai)
		}
	} else {
		seat.(*PokerDou14Seat).reserveShouPai = msgCustomShouPai.ShouPai
	}

	this.table.SendToSeat(seat.GetSeatData().Number, pro.MessageID, &rspMsg)

	return 0
}

func (this *PokerDou14Table) onCustomMoPai(pro *mateProto.MessageMaTe) int32 {
	msgCustomMoPai := CS_CustomMoPai{}
	err := json.Unmarshal(pro.Data, &msgCustomMoPai)
	if err != nil {
		return mateProto.Err_ProtocolDataErr
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

	rsp := this.PaiMgr.ReserveMoPai(msgCustomMoPai.Pai)
	if rsp != 0 {
		rspMsg.Status = mateProto.Err_CustomPai
	} else {
		seat.(*PokerDou14Seat).reserveMoPai = msgCustomMoPai.Pai
	}

	this.table.SendToSeat(seat.GetSeatData().Number, pro.MessageID, &rspMsg)

	return 0
}

func (this *PokerDou14Table) onGetSurplus(pro *mateProto.MessageMaTe) int32 {

	if this.table.IsAssignTableState(qpTable.TS_Playing) == false {
		return mateProto.Err_ActionNotMatchStatus
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

	rspMsg := protoGameBasic.JsonResponse{Data: this.PaiMgr.GetSurplusPai()}
	this.table.SendToSeat(seat.GetSeatData().Number, pro.MessageID, &rspMsg)

	return 0
}

func (this *PokerDou14Table) onDissolveTableVote(pro *mateProto.MessageMaTe) int32 {
	rspCode := this.table.OnMessage(pro)

	if this.table.Status != qpTable.TS_Invalid {
		return rspCode
	}

	this.handleXiaoJieSuan()
	this.handleDaJieSuan()

	return rspCode
}

func (this *PokerDou14Table) onForceDissolveTable(pro *mateProto.MessageMaTe) int32 {
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

func (this *PokerDou14Table) checkLiuJuHu() {
	winPlayerMap := make(map[qpTable.SeatNumber]int64)

	this.cleanTableRec()

	for _, v := range this.table.SeatArr {
		if v == nil {
			continue
		}
		d14Seat := v.(*PokerDou14Seat)
		if d14Seat.seatData.IsAssignSeatState(qpTable.SS_Playing) == false {
			continue
		}

		maxGameScore := 0
		var paiXing []gameMaJiang.HuPaiXing
		for dianShu := MinDianShu; dianShu <= MaxDianShu; dianShu++ {
			if this.isHuPai(d14Seat, FangKuai|dianShu) {
				if d14Seat.gameScore > maxGameScore {
					maxGameScore = d14Seat.gameScore
					paiXing = d14Seat.huPaiXin
				}
			}
		}
		if maxGameScore > 0 {
			d14Seat.gameScore = maxGameScore
			d14Seat.huPaiXin = paiXing
			winPlayerMap[d14Seat.seatData.Number] = 0
		}
	}

	for _, v := range this.table.SeatArr {
		if v == nil {
			continue
		}
		loseSeat := v.(*PokerDou14Seat)
		if loseSeat.seatData.IsAssignSeatState(qpTable.SS_Playing) == false {
			continue
		}

		if _, ok := winPlayerMap[loseSeat.seatData.Number]; ok == true {
			continue
		}

		loseSeat.huPaiXin = append(loseSeat.huPaiXin, gameMaJiang.HuPaiXing{PaiXing: HuPX_LoseLiuJuChaJiao})
		loseSeat.seatData.PutGameScoreItem(&qpTable.GameScoreRec{PaiXing: loseSeat.huPaiXin}, 1)

		for k, _ := range winPlayerMap {
			winSeat := this.table.SeatArr[k].(*PokerDou14Seat)
			winSeat.huPaiXin = append(winSeat.huPaiXin, gameMaJiang.HuPaiXing{PaiXing: HuPX_WinLiuJuChaJiao})
			winSeat.seatData.PutGameScoreItem(&qpTable.GameScoreRec{PaiXing: winSeat.huPaiXin}, 1)

			tHuScore := float64(winSeat.gameScore) * this.gameRule.MultipleFloat64
			tempScore_ := tHuScore

			if loseSeat.seatData.SeatScore-tempScore_ <= loseSeat.seatData.ClubScore*-1 {
				tempScore_ = loseSeat.seatData.ClubScore + loseSeat.seatData.SeatScore
			}

			loseSeat.seatData.RoundScore -= tempScore_
			loseSeat.seatData.SeatScore -= tempScore_

			winSeat.seatData.RoundScore += tempScore_
			winSeat.seatData.SeatScore += tempScore_
		}
	}
}
