package gameMaJiang

import (
	"encoding/json"
	"qpGame/commonDefine/mateProto"
	"qpGame/qpTable"
)

type MJGameRule struct {
	PaiMgr MJPaiMgr     // 牌的管理器
	Logic  MJBaseAction // 逻辑
	Table  *qpTable.QPTable

	RoundOverFunc func()

	// 小局 待清理 成员
	BankerSeatNum   qpTable.SeatNumber // 庄家座位号
	CurPlayPai      int8               // 最近一次出的牌
	CurPlaySeatNum  qpTable.SeatNumber // 最近一次出牌的座位号
	CurMoPaiSeatNum qpTable.SeatNumber // 当前摸牌座位号
	CurMoPai        int8               // 当前摸的牌

	huSeatArr []*MJSeat // 已经胡的玩家
	chiFunc   func() func()
	pengFunc  func() func()
	gangFunc  func() func()
}

// 清空每一小局数据
func (this *MJGameRule) CleanRoundData() {

	this.BankerSeatNum = qpTable.INVALID_SEAT_NUMBER
	this.CurPlayPai = InvalidPai
	this.CurPlaySeatNum = qpTable.INVALID_SEAT_NUMBER
	this.CurMoPaiSeatNum = qpTable.INVALID_SEAT_NUMBER
	this.CurMoPai = InvalidPai

	this.huSeatArr = make([]*MJSeat, 0)
	this.chiFunc = nil
	this.pengFunc = nil
	this.gangFunc = nil
}

// 事件处理
func (this *MJGameRule) OnMessage(pro *mateProto.MessageMaTe) int32 {

	var rspCode int32 = -1
	switch pro.MessageID {
	case ID_Guo:
		rspCode = this.OnGuo(pro)
	case ID_Play:
		rspCode = this.OnPlay(pro)
	case ID_Chi:
		rspCode = this.OnChi(pro)
	case ID_Peng:
		rspCode = this.OnPeng(pro)
	case ID_Gang:
		rspCode = this.OnGang(pro)
	default:
		return this.Table.OnMessage(pro)
	}

	return rspCode
}

func (this *MJGameRule) OnGameStart(pro *mateProto.MessageMaTe) int32 {
	if this.Table.IsAssignTableState(qpTable.TS_WaitingPlayerEnter) == false &&
		this.Table.IsAssignTableState(qpTable.TS_WaitingReady) == false {
		return mateProto.Err_TableStatusNotMatch
	}

	errNumber := this.Table.OnGameStart(pro)
	if errNumber != mateProto.Err_Success {
		return errNumber
	}

	// 无庄时
	if this.BankerSeatNum == qpTable.INVALID_SEAT_NUMBER {
		for _, v := range this.Table.SeatArr {
			if v != nil && v.GetSeatData().IsAssignSeatState(qpTable.SS_Playing) {
				this.BankerSeatNum = v.GetSeatData().Number
				break
			}
		}
	}

	this.PaiMgr.XiPai()

	// 发手牌
	for i, v := range this.Table.SeatArr {
		if v == nil {
			continue
		}
		if v.GetSeatData().IsAssignSeatState(qpTable.SS_Playing) == false {
			continue
		}
		paiArr := this.PaiMgr.GetGroupPai(13)

		mjSeat := v.GetXSeatData(0).(*MJSeat)

		seatCardTemp := make([]int8, len(paiArr))
		for i, v := range paiArr {
			mjSeat.PushShouPai(v)
			seatCardTemp[i] = v
		}

		this.Table.OperateRec.PutPlayer(v.GetSeatData())

		this.Table.SendGameEventToSeat(qpTable.SeatNumber(i),
			ID_FaShouPai,
			&SC_FaShouPai{SeatNum: int32(i), Pai: seatCardTemp, BankerSeatNum: int32(this.BankerSeatNum)})
	}

	this.Logic.SetRule(false, InvalidPai)

	// 庄家摸牌
	this.MoPaiOperation(this.BankerSeatNum)

	return mateProto.Err_Success
}

// 出牌
func (this *MJGameRule) OnPlay(pro *mateProto.MessageMaTe) int32 {
	playerID := qpTable.PlayerID(pro.SenderID)

	playPai := CS_Play{}
	err := json.Unmarshal(pro.Data, &playPai)
	if err != nil {
		return mateProto.Err_ProtocolDataErr
	}

	playSeat, errCode := this.CheckOperation(playerID, playPai.OperationID)
	if errCode != mateProto.Err_Success {
		return errCode
	}

	seatData := playSeat.GetSeatData()
	mjSeatData := playSeat.GetXSeatData(0).(*MJSeat)
	if (mjSeatData.OperationItem & OPI_PlayPai) != OPI_PlayPai {
		return mateProto.Err_OperationNotExist
	}

	if mjSeatData.DeleteShouPai(playPai.Pai) == false {
		return mateProto.Err_PaiNotExist
	}

	// 座位数据 清理
	mjSeatData.OperationItem = 0
	seatData.CleanOperationID()
	this.Table.OperateRec.PutPlayerStep(int32(seatData.Number), ID_Play, &playPai)
	mjSeatData.PlayPai = append(mjSeatData.PlayPai, playPai.Pai)

	this.CurMoPaiSeatNum = qpTable.INVALID_SEAT_NUMBER
	this.CurMoPai = InvalidPai
	this.CurPlaySeatNum = seatData.Number
	this.CurPlayPai = playPai.Pai

	// 广播 出牌
	this.Table.BroadCastGameEvent(ID_BroadcastPlay,
		&BroadcastPlay{SeatNumber: int32(seatData.Number), Pai: playPai.Pai})

	canOperationSeatArr := this.GetPlayPaiOperation(this.CurPlaySeatNum, this.CurPlayPai)
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

	// 通知 操作
	for _, v := range canOperationSeatArr {
		proMsg := SC_NoticeOperation{SeatNumber: int32(v.GetSeatData().Number),
			OperationID: v.GetSeatData().GetOperationID(),
			Operation:   v.GetXSeatData(0).(*MJSeat).OperationItem,
			Pai:         playPai.Pai,
		}
		this.Table.SendGameEventToSeat(v.GetSeatData().Number,
			ID_NoticeOperation,
			&proMsg)
	}

	return mateProto.Err_Success
}

// 吃
func (this *MJGameRule) OnChi(pro *mateProto.MessageMaTe) int32 {
	playerID := qpTable.PlayerID(pro.SenderID)

	operationChi := CS_Chi{}
	err := json.Unmarshal(pro.Data, &operationChi)
	if err != nil {
		return mateProto.Err_ProtocolDataErr
	}

	chiPaiSeat, errCode := this.CheckOperation(playerID, operationChi.OperationID)
	if errCode != mateProto.Err_Success {
		return errCode
	}

	seatData := chiPaiSeat.GetSeatData()
	mjSeatData := chiPaiSeat.GetXSeatData(0).(*MJSeat)
	if (mjSeatData.OperationItem & OPI_CHI) != OPI_CHI {
		return mateProto.Err_OperationNotExist
	}
	if len(operationChi.Pai) != 3 {
		return mateProto.Err_OperationParamErr
	}
	// 类型
	if (uint8(operationChi.Pai[0])&0xF0) != (uint8(operationChi.Pai[1])&0xF0) ||
		(uint8(operationChi.Pai[0])&0xF0) != (uint8(operationChi.Pai[2])&0xF0) {
		return mateProto.Err_OperationParamErr
	}

	// 如果字牌能吃,需要 另加判断
	if ((operationChi.Pai[1]&0x0F)-(operationChi.Pai[0]&0x0F)) != 1 ||
		((operationChi.Pai[2]&0x0F)-(operationChi.Pai[1]&0x0F)) != 1 {
		return mateProto.Err_OperationParamErr
	}

	isMatchPlayPai := false
	// 手牌 是否存在
	for _, v := range operationChi.Pai {
		if v == this.CurPlayPai {
			isMatchPlayPai = true
		} else if mjSeatData.GetPaiCount(v) < 1 {
			return mateProto.Err_PaiNotExist
		}
	}
	// 是否是 吃 出的牌
	if isMatchPlayPai == false {
		return mateProto.Err_OperationParamErr
	}

	// 操作成功
	seatData.CleanOperationID()
	mjSeatData.OperationItem = 0
	this.Table.OperateRec.PutPlayerStep(int32(seatData.Number), ID_Chi, &operationChi)

	heightPriority := OPI_PENG | OPI_GANG | OPI_HU
	if this.FindPriorityOperation(seatData.Number, heightPriority) == true {
		// 操作寄存起来
		this.chiFunc = func() func() {
			return func() {
				this.DoChi(seatData, mjSeatData, &operationChi)
			}
		}
		return mateProto.Err_Success
	}

	// 已经有人胡牌了
	if len(this.huSeatArr) > 0 {
		return mateProto.Err_Success
	}

	this.DoChi(seatData, mjSeatData, &operationChi)

	return mateProto.Err_Success
}

func (this *MJGameRule) DoChi(seatData *qpTable.SeatData, mjSeatData *MJSeat, chiPai *CS_Chi) {

	//commonDef.LOG_Info("playerID:=", seatData.Player.ID, ",table number:=", this.Table.GetTableNumber(), ",action:= Chi")

	playPaiMJSeat := this.Table.SeatArr[this.CurPlaySeatNum].GetXSeatData(0).(*MJSeat)
	playPaiMJSeat.PlayPai = playPaiMJSeat.PlayPai[:len(playPaiMJSeat.PlayPai)-1]

	this.CleanAllSeatOperation()

	for _, v := range chiPai.Pai {
		if v != this.CurPlayPai {
			mjSeatData.DeleteShouPai(v)
		}
	}
	// 记录 操作区域
	operInfo := &OperationPaiInfo{OperationPXItem: OPX_CHI,
		PlayPaiSeatNumber: int32(this.CurPlaySeatNum),
		PaiArr:            chiPai.Pai}
	mjSeatData.OperationPai = append(mjSeatData.OperationPai, operInfo)

	this.Table.BroadCastGameEvent(ID_BroadcastChi,
		&BroadcastChi{SeatNumber: int32(seatData.Number), Pai: chiPai.Pai, ChiPai: this.CurPlayPai})

	// 通知 出牌
	mjSeatData.SetOperation(OPI_PlayPai)
	proMsg := SC_NoticeOperation{SeatNumber: int32(seatData.Number),
		OperationID: seatData.GetOperationID(),
		Operation:   mjSeatData.OperationItem,
		Pai:         InvalidPai,
	}
	this.Table.SendGameEventToSeat(seatData.Number, ID_NoticeOperation, &proMsg)
}

// 碰
func (this *MJGameRule) OnPeng(pro *mateProto.MessageMaTe) int32 {
	playerID := qpTable.PlayerID(pro.SenderID)

	operationPeng := CS_Peng{}
	err := json.Unmarshal(pro.Data, &operationPeng)
	if err != nil {
		return mateProto.Err_ProtocolDataErr
	}

	pengPaiSeat, errCode := this.CheckOperation(playerID, operationPeng.OperationID)
	if errCode != mateProto.Err_Success {
		return errCode
	}

	seatData := pengPaiSeat.GetSeatData()
	mjSeatData := pengPaiSeat.GetXSeatData(0).(*MJSeat)
	if (mjSeatData.OperationItem & OPI_PENG) != OPI_PENG {
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
	this.Table.OperateRec.PutPlayerStep(int32(seatData.Number), ID_Peng, &operationPeng)

	heightPriority := OPI_GANG | OPI_HU
	if this.FindPriorityOperation(seatData.Number, heightPriority) == true {
		// 操作寄存起来
		this.pengFunc = func() func() {
			return func() {
				this.DoPeng(seatData, mjSeatData, &operationPeng)
			}
		}
		return mateProto.Err_Success
	}
	// 已经有人胡牌了
	if len(this.huSeatArr) > 0 {
		return mateProto.Err_Success
	}

	this.DoPeng(seatData, mjSeatData, &operationPeng)
	return mateProto.Err_Success
}

func (this *MJGameRule) DoPeng(seatData *qpTable.SeatData, mjSeatData *MJSeat, pengPai *CS_Peng) {

	//commonDef.LOG_Info("playerID:=", seatData.Player.ID, ",table number:=", this.Table.GetTableNumber(), ",action:= Chi")

	playPaiMJSeat := this.Table.SeatArr[this.CurPlaySeatNum].GetXSeatData(0).(*MJSeat)
	playPaiMJSeat.PlayPai = playPaiMJSeat.PlayPai[:len(playPaiMJSeat.PlayPai)-1]

	this.CleanAllSeatOperation()

	mjSeatData.DeleteShouPai(pengPai.Pai)
	mjSeatData.DeleteShouPai(pengPai.Pai)

	// 记录 操作区域
	operInfo := &OperationPaiInfo{OperationPXItem: OPX_PENG,
		PlayPaiSeatNumber: int32(this.CurPlaySeatNum),
		PaiArr:            []int8{pengPai.Pai}}
	mjSeatData.OperationPai = append(mjSeatData.OperationPai, operInfo)

	// 广播
	this.Table.BroadCastGameEvent(ID_BroadcastPeng,
		&BroadcastPeng{SeatNumber: int32(seatData.Number), Pai: pengPai.Pai})

	// 通知 出牌
	mjSeatData.SetOperation(OPI_PlayPai)

	proMsg := SC_NoticeOperation{SeatNumber: int32(seatData.Number),
		OperationID: seatData.GetOperationID(),
		Operation:   mjSeatData.OperationItem,
		Pai:         InvalidPai,
	}
	this.Table.SendGameEventToSeat(seatData.Number, ID_NoticeOperation, &proMsg)
}

// 杠
func (this *MJGameRule) OnGang(pro *mateProto.MessageMaTe) int32 {
	playerID := qpTable.PlayerID(pro.SenderID)

	operationGang := CS_Gang{}
	err := json.Unmarshal(pro.Data, &operationGang)
	if err != nil {
		return mateProto.Err_ProtocolDataErr
	}

	gangPaiSeat, errCode := this.CheckOperation(playerID, operationGang.OperationID)
	if errCode != mateProto.Err_Success {
		return errCode
	}

	seatData := gangPaiSeat.GetSeatData()
	mjSeatData := gangPaiSeat.GetXSeatData(0).(*MJSeat)
	if (mjSeatData.OperationItem & OPI_GANG) != OPI_GANG {
		return mateProto.Err_OperationNotExist
	}

	// 1: 手上3张,别人 打出 1张
	// 2: 自己4张
	// 3： 碰了一次, 自摸1张

	var gangType int32 = 0

	// 只能 明杠
	if this.CurPlayPai != InvalidPai {
		// 是否是 杠 出的牌
		if operationGang.Pai != this.CurPlayPai {
			return mateProto.Err_OperationParamErr
		}
		// 手牌 是否存在
		if mjSeatData.GetPaiCount(this.CurPlayPai) != 3 {
			return mateProto.Err_PaiNotExist
		}
		gangType = 1
	} else if this.CurMoPaiSeatNum == seatData.Number {
		// 手牌 是否存在
		if mjSeatData.GetPaiCount(operationGang.Pai) == 4 {
			gangType = 2
		} else if mjSeatData.GetPaiCount(operationGang.Pai) == 1 {
			for _, v := range mjSeatData.OperationPai {
				if v.OperationPXItem == OPX_PENG && v.PaiArr[0] == operationGang.Pai {
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
	this.Table.OperateRec.PutPlayerStep(int32(seatData.Number), ID_Gang, &operationGang)

	heightPriority := OPI_HU
	if this.FindPriorityOperation(seatData.Number, heightPriority) == true {
		// 操作寄存起来
		this.gangFunc = func() func() {
			return func() {
				this.DoGang(seatData, mjSeatData, &operationGang, gangType)
			}
		}
		return mateProto.Err_Success
	}
	// 已经有人胡牌了
	if len(this.huSeatArr) > 0 {
		return mateProto.Err_Success
	}

	this.DoGang(seatData, mjSeatData, &operationGang, gangType)

	return mateProto.Err_Success
}

func (this *MJGameRule) DoGang(seatData *qpTable.SeatData, mjSeatData *MJSeat, gangPai *CS_Gang, gangType int32) {

	//commonDef.LOG_Info("playerID:=", seatData.Player.ID, ",table number:=", this.Table.GetTableNumber(), ",action:= Gang")

	this.CleanAllSeatOperation()

	brodcastGang := BroadcastGang{SeatNumber: int32(seatData.Number), Type: gangType}

	if gangType == 1 {
		// 删除手牌
		mjSeatData.DeleteShouPai(gangPai.Pai)
		mjSeatData.DeleteShouPai(gangPai.Pai)
		mjSeatData.DeleteShouPai(gangPai.Pai)

		// 记录 操作区域
		operInfo := &OperationPaiInfo{OperationPXItem: OPX_MING_GANG,
			PlayPaiSeatNumber: int32(this.CurPlaySeatNum),
			PaiArr:            []int8{gangPai.Pai}}
		mjSeatData.OperationPai = append(mjSeatData.OperationPai, operInfo)

		brodcastGang.Pai = gangPai.Pai
	} else if gangType == 2 {
		// 删除手牌
		mjSeatData.DeleteShouPai(gangPai.Pai)
		mjSeatData.DeleteShouPai(gangPai.Pai)
		mjSeatData.DeleteShouPai(gangPai.Pai)
		mjSeatData.DeleteShouPai(gangPai.Pai)

		// 记录 操作区域
		operInfo := &OperationPaiInfo{OperationPXItem: OPX_AN_GANG,
			PaiArr: []int8{gangPai.Pai}}
		mjSeatData.OperationPai = append(mjSeatData.OperationPai, operInfo)

		brodcastGang.Pai = gangPai.Pai
	} else if gangType == 3 {
		// 删除手牌
		mjSeatData.DeleteShouPai(gangPai.Pai)

		// 更新 操作区域
		for _, v := range mjSeatData.OperationPai {
			if v.OperationPXItem == OPX_PENG && v.PaiArr[0] == gangPai.Pai {
				v.OperationPXItem = OPX_BU_GANG
				break
			}
		}

		brodcastGang.Pai = gangPai.Pai
	}

	// 广播
	this.Table.BroadCastGameEvent(ID_BroadcastGang, &brodcastGang)

	// 通知 摸牌
	if this.PaiMgr.GetTheRestOfPaiCount() < 1 {
		this.RoundOverFunc()
	} else {
		this.MoPaiOperation(seatData.Number)
	}
}

// 胡
func (this *MJGameRule) OnHu(pro *mateProto.MessageMaTe) int32 {
	playerID := qpTable.PlayerID(pro.SenderID)

	operationHu := CS_Hu{}
	err := json.Unmarshal(pro.Data, &operationHu)
	if err != nil {
		return mateProto.Err_ProtocolDataErr
	}

	seat, errCode := this.CheckOperation(playerID, operationHu.OperationID)
	if errCode != mateProto.Err_Success {
		return errCode
	}

	mjSeatData := seat.GetXSeatData(0).(*MJSeat)

	if (mjSeatData.OperationItem & OPI_HU) == 0 {
		return mateProto.Err_OperationNotExist
	}

	seat.GetSeatData().CleanOperationID()
	mjSeatData.OperationItem = 0

	return mateProto.Err_Success
}

// 过
func (this *MJGameRule) OnGuo(pro *mateProto.MessageMaTe) int32 {
	playerID := qpTable.PlayerID(pro.SenderID)

	operationGuo := CS_Guo{}
	err := json.Unmarshal(pro.Data, &operationGuo)
	if err != nil {
		return mateProto.Err_ProtocolDataErr
	}

	guoSeat, errCode := this.CheckOperation(playerID, operationGuo.OperationID)
	if errCode != mateProto.Err_Success {
		return errCode
	}

	seatData := guoSeat.GetSeatData()
	mjSeatData := guoSeat.GetXSeatData(0).(*MJSeat)

	const operationCode = OPI_CHI | OPI_PENG | OPI_GANG | OPI_HU
	if (mjSeatData.OperationItem & operationCode) == 0 {
		return mateProto.Err_OperationNotExist
	}

	seatData.CleanOperationID()
	mjSeatData.OperationItem = 0
	this.Table.OperateRec.PutPlayerStep(int32(seatData.Number), ID_Guo, &operationGuo)

	// 自摸阶段 取消了操作
	if this.CurMoPaiSeatNum != qpTable.INVALID_SEAT_NUMBER {
		if this.CurMoPaiSeatNum == seatData.Number {
			mjSeatData.SetOperation(OPI_PlayPai)

			proMsg := SC_NoticeOperation{
				SeatNumber:  int32(seatData.Number),
				OperationID: seatData.GetOperationID(),
				Operation:   mjSeatData.OperationItem,
				Pai:         InvalidPai,
			}
			this.Table.SendGameEventToSeat(seatData.Number, ID_NoticeOperation, &proMsg)
			return mateProto.Err_Success
		}
		// 发生错误 todo
		return mateProto.Err_Success
	}

	// 出牌阶段, 是否还有人 未操作
	if this.FindPriorityOperation(seatData.Number, operationCode) == true {
		if this.FindPriorityOperation(seatData.Number, OPI_HU) == true {
			return mateProto.Err_Success
		}
		if this.gangFunc != nil {
			this.gangFunc()
			return mateProto.Err_Success
		}
		if this.pengFunc != nil {
			this.pengFunc()
			return mateProto.Err_Success
		}
		if this.chiFunc != nil {
			this.chiFunc()
			return mateProto.Err_Success
		}
		return mateProto.Err_Success
	}

	// 出牌阶段, 没有玩家还有操作
	if this.gangFunc != nil {
		this.gangFunc()
		return mateProto.Err_Success
	}
	if this.pengFunc != nil {
		this.pengFunc()
		return mateProto.Err_Success
	}
	if this.chiFunc != nil {
		this.chiFunc()
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

// 清理所有座位操作
func (this *MJGameRule) CleanAllSeatOperation() {

	for _, v := range this.Table.SeatArr {
		if v == nil {
			continue
		}
		v.GetSeatData().CleanOperationID()
		mjSeat := v.GetXSeatData(0).(*MJSeat)
		mjSeat.OperationItem = 0
	}

	this.chiFunc = nil
	this.pengFunc = nil
	this.gangFunc = nil
}

// 是否有更高优先级的操作
func (this *MJGameRule) FindPriorityOperation(excludeSeatNum qpTable.SeatNumber, oper PlayerMJOperation) bool {
	for _, v := range this.Table.SeatArr {
		if v == nil || v.GetSeatData().Number == excludeSeatNum {
			continue
		}
		if (v.GetXSeatData(0).(*MJSeat).OperationItem & oper) != 0 {
			return true
		}
	}
	return false
}

// 摸牌
func (this *MJGameRule) MoPaiOperation(number qpTable.SeatNumber) {

	seat := this.Table.SeatArr[number]
	mjSeat := seat.GetXSeatData(0).(*MJSeat)

	moPai := this.PaiMgr.GetPai()
	mjSeat.PushShouPai(moPai)

	oper := this.getZiMoOperation(mjSeat)
	mjSeat.SetOperation(oper)

	// 通知 摸牌
	proMoPai := SC_PlayerMoPai{Card: moPai, OperationID: seat.GetSeatData().OperationID, Operation: oper, GangArr: mjSeat.GangArr}
	this.Table.SendGameEventToSeat(number, ID_PlayerMoPai, &proMoPai)

	// 广播 摸牌
	broadMoPai := BroadcastMoPai{SeatNumber: int32(number),
		CardCount: this.PaiMgr.GetTheRestOfPaiCount()}
	this.Table.BroadCastGameEvent(ID_BroadcastMoPai, &broadMoPai)

	// 更新 桌子数据
	this.CurMoPaiSeatNum = number
	this.CurMoPai = moPai
	this.CurPlayPai = InvalidPai
	this.CurPlaySeatNum = qpTable.INVALID_SEAT_NUMBER
}

// 自摸操作
func (this *MJGameRule) getZiMoOperation(seat *MJSeat) PlayerMJOperation {

	oper := OPI_PlayPai | OPI_MO_Pai
	if this.Logic.IsHu(&seat.ShouPai, InvalidPai) == true {
		oper |= OPI_HU
	}
	if this.Logic.IsZiMoGang(seat.ShouPai, seat.OperationPai) == true {
		oper |= OPI_GANG
	}

	return oper
}

// 玩家出牌,其它人操作
func (this *MJGameRule) GetPlayPaiOperation(playSeatNum qpTable.SeatNumber, playPai int8) []*MJSeat {

	operSeat := make([]*MJSeat, 0)
	for _, v := range this.Table.SeatArr {
		if v == nil {
			continue
		}
		if v.GetSeatData().Number == playSeatNum {
			continue
		}
		mjSeat := v.GetXSeatData(0).(*MJSeat)

		var oper PlayerMJOperation
		if this.Logic.IsHu(&mjSeat.ShouPai, playPai) == true {
			oper |= OPI_HU
		}
		if this.Logic.IsPeng(mjSeat.ShouPai, playPai) == true {
			oper |= OPI_PENG
		}
		if this.Logic.IsMingGang(mjSeat.ShouPai, mjSeat.OperationPai, playPai) == true {
			oper |= OPI_GANG
		}

		if oper != 0 {
			mjSeat.SetOperation(oper)
			operSeat = append(operSeat, mjSeat)
		}
	}

	nextSeat := this.Table.GetNextPlayingSeat(playSeatNum)
	mjSeat := nextSeat.GetXSeatData(0).(*MJSeat)
	if this.Logic.IsChi(mjSeat.ShouPai, playPai) == true {
		mjSeat.PutOperation(OPI_CHI)
		operSeat = append(operSeat, mjSeat)
	}

	return operSeat
}

// 检查操作
func (this *MJGameRule) CheckOperation(playerID qpTable.PlayerID, operationID string) (qpTable.QPSeat, int32) {
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
