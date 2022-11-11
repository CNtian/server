package management

import (
	"encoding/json"
	"fmt"
	"github.com/golang/glog"
	"qpGame/commonDefine/mateProto"
	"qpGame/commonDefine/mateProto/protoGameBasic"
	"qpGame/db"
	"qpGame/game/tableFactory"
	"qpGame/localConfig"
	"qpGame/qpTable"
	"qpGame/wrapMQ"
	"strconv"
	"time"
)

func createTable(uid int64, privateCreateParam *protoGameBasic.CS_PrivateCreateGameTable, clubCreateParam *protoGameBasic.CS_ClubCreateTable) (int32, *rootTable, int32) {
	// 校正 玩家 所在 游戏服务
	rootab := getPlayer(uid)
	if rootab != nil {
		return 0, nil, mateProto.Err_AlreadyPlaying
	} else {
		db.RemovePlayerGameIntro(uid)
	}

	// 服务器状态
	if serviceStatus == SS_NotCreatTable || serviceStatus == SS_Stop {
		glog.Warning("serviceStatus := ", serviceStatus)
		return 0, nil, mateProto.Err_ServiceStatus
	}

	// 当前桌子数量 是否超过了 桌子号 数量
	if curTableCount >= GetTableNumberCount() {
		return 0, nil, mateProto.Err_NotFindIdleTabNumber
	}

	tableNum := GetIdleTableNumber()
	if _, ok := tableMap.Load(tableNum); ok == true {
		return 0, nil, mateProto.Err_NotFindIdleTabNumber
	}

	var (
		err              error
		gameID, MZClubID int32
		playConfig       string
		tableConfig      string
		clubPlayID       int64
		isStop3, isStop4 bool // 是否禁止3人局
	)
	if privateCreateParam != nil {
		gameID = privateCreateParam.GameID
		playConfig = privateCreateParam.PlayConfig
		tableConfig = privateCreateParam.TableConfig
	} else if clubCreateParam != nil {
		gameID = clubCreateParam.GameID
		MZClubID = clubCreateParam.MZClubID
		playConfig = clubCreateParam.PlayConfig
		tableConfig = clubCreateParam.TableConfig
		clubPlayID = clubCreateParam.ClubPlayID
		isStop3 = clubCreateParam.IsStop3Players
		isStop4 = clubCreateParam.IsStop4Players
	} else {
		return 0, nil, mateProto.Err_CheckFailed
	}

	gameTable, resCode, desc := tableFactory.NewGameTable(tableNum,
		gameID,
		playConfig,
		tableConfig)
	if resCode != 0 {
		glog.Warning("uid:=", uid, ",onPrivateCreateTable() resCode:=", resCode, ",desc:=", desc)
		return 0, nil, mateProto.Err_CreateTableParam
	}

	baseTable := gameTable.GetBaseQPTable()
	if privateCreateParam != nil {
		resCode, err = db.ChangePlayerRoomCardCount(uid, -baseTable.Consumables)
	} else if clubCreateParam != nil {
		//resCode, err = db.ChangePlayerDiamondCount(clubCreateParam.PayUID, -baseTable.Consumables)
		resCode, err = db.ChangePlayerRoomCardCount(clubCreateParam.PayUID, -baseTable.Consumables)
	} else {
		return 0, nil, mateProto.Err_CreateTableParam
	}
	if err != nil {
		if resCode != 0 {
			return 0, nil, resCode
		}
		return 0, nil, mateProto.Err_Failed
	}
	if resCode != 0 {
		return 0, nil, resCode
	}

	if isStop3 == true && gameTable.GetBaseQPTable().MaxPlayers == 3 {
		return baseTable.Consumables, nil, mateProto.ErrClubStop3Player
	}
	if isStop4 == true && gameTable.GetBaseQPTable().MaxPlayers == 4 {
		return baseTable.Consumables, nil, mateProto.ErrClubStop3Player
	}

	err = db.StoreTableInfo(tableNum, MZClubID, clubPlayID, gameTable.GetBaseQPTable().MaxPlayers, baseTable.GameID, playConfig, tableConfig)
	if err != nil {
		glog.Warning("uid:=", uid, ",StoreTableInfo() err:=", err.Error())
		return baseTable.Consumables, nil, mateProto.Err_SystemError
	}

	rootGameTable := &rootTable{
		gameTable: gameTable,
		msgChan:   make(chan *mateProto.MessageMaTe, 1024),
	}

	return baseTable.Consumables, rootGameTable, 0
}

// 创建 私人 桌子
func onPrivateCreateTable(msg *mateProto.MessageMaTe) (int32, *rootTable) {

	msgCreateTable := protoGameBasic.CS_PrivateCreateGameTable{}
	err := json.Unmarshal(msg.Data, &msgCreateTable)
	if err != nil {
		glog.Warning("onPrivateCreateTable() err. err:=", err.Error(), ",data:=", string(msg.Data))
		return mateProto.Err_ProtocolDataErr, nil
	}
	msg.MsgBody = &msgCreateTable

	var (
		payCount         int32
		rootGameTable    *rootTable
		rspCode, joinRsp int32 = 0, -1
	)

	defer func() {
		if joinRsp >= 0 {
			return
		}
		if payCount > 0 {
			_, err = db.ChangePlayerRoomCardCount(msg.SenderID, payCount)
			if err != nil {
				glog.Warning("ChangePlayerRoomCardCount() err. err:=", err.Error(), ",data:=", msg.SenderID)
			}
		}
		if rootGameTable != nil {
			rootGameTable.ReleaseResource()
		}
	}()

	payCount, rootGameTable, rspCode =
		createTable(msg.SenderID, &msgCreateTable, nil)
	if rspCode != 0 {
		return rspCode, nil
	}

	// 加入桌子
	msgJoinTable := protoGameBasic.CS_PrivateJoinGameTable{
		TableNumber: rootGameTable.gameTable.GetBaseQPTable().TableNum,
		IP:          msgCreateTable.IP,
		Longitude:   msgCreateTable.Longitude,
		Latitude:    msgCreateTable.Latitude,
	}
	joinTableData, _ := json.Marshal(&msgJoinTable)

	joinRsp = rootGameTable.OnMessage(&mateProto.MessageMaTe{
		Source:    msg.Source,
		SenderID:  msg.SenderID,
		MessageID: protoGameBasic.ID_PrivateJoinGameTable,
		MsgBody:   &msgJoinTable,
		Data:      joinTableData,
	})

	if joinRsp >= 0 {
		baseTable := rootGameTable.GetBaseQPTable()
		baseTable.PayUID = msg.SenderID
		rootGameTable.StartWork(baseTable.TableNum)
		return mateProto.Err_Success, rootGameTable
	}

	return joinRsp, nil
}

func onPrivateJoinGameTable(msg *mateProto.MessageMaTe) int32 {
	// 校正 玩家 所在 游戏服务
	rootab := getPlayer(msg.SenderID)
	if rootab != nil {
		return mateProto.Err_AlreadyPlaying
	} else {
		db.RemovePlayerGameIntro(msg.SenderID)
	}

	if serviceStatus == SS_NotJoinTable || serviceStatus == SS_Stop {
		glog.Warning("serviceStatus := ", serviceStatus)
		return mateProto.Err_ServiceStatus
	}

	msgJoinTable := protoGameBasic.CS_PrivateJoinGameTable{}
	err := json.Unmarshal(msg.Data, &msgJoinTable)
	if err != nil {
		return mateProto.Err_ProtocolDataErr
	}
	msg.MsgBody = &msgJoinTable

	v, ok := tableMap.Load(msgJoinTable.TableNumber)
	if ok == false {
		return mateProto.Err_NotFindTable
	}

	v.(*rootTable).msgChan <- msg

	return 0
}

// 创建俱乐部 桌子
func onCreateClubTable(msg *mateProto.MessageMaTe) (int32, *rootTable) {

	msgCreateTable := protoGameBasic.CS_ClubCreateTable{}
	err := json.Unmarshal(msg.Data, &msgCreateTable)
	if err != nil {
		glog.Warning("onCreateClubTable() err. err:=", err.Error(), ",data:=", string(msg.Data))
		return mateProto.Err_ProtocolDataErr, nil
	}
	msg.MsgBody = &msgCreateTable

	msgSource := msg.Source
	defer func() {
		msg.Source = msgSource
		msg.MessageID = protoGameBasic.ID_ClubCreateTable
		msg.MsgBody = &msgCreateTable
	}()

	var clubRule qpTable.DBClubRule
	if len(msgCreateTable.ClubConfig) < 1 {
		return mateProto.Err_ClubRule, nil
	} else if err = json.Unmarshal([]byte(msgCreateTable.ClubConfig), &clubRule); err != nil {
		return mateProto.Err_ClubRule, nil
	}
	clubRule.OverScoreInt, _ = strconv.ParseFloat(clubRule.OverScoreText, 64)

	var (
		payCount         int32
		rootGameTable    *rootTable
		rspCode, joinRsp int32 = 0, -1
	)

	defer func() {
		if joinRsp >= 0 {
			return
		}
		if payCount > 0 {
			//db.ChangePlayerDiamondCount(msgCreateTable.PayUID, payCount)
			_, err = db.ChangePlayerRoomCardCount(msgCreateTable.PayUID, payCount)
			if err != nil {
				glog.Warning("ChangePlayerRoomCardCount() err. err:=", err.Error(), ",data:=", msgCreateTable.PayUID)
			}
		}
		if rootGameTable != nil {
			rootGameTable.ReleaseResource()
		}
	}()

	payCount, rootGameTable, rspCode =
		createTable(msg.SenderID, nil, &msgCreateTable)
	if rspCode != 0 {
		return rspCode, nil
	}

	baseTable := rootGameTable.gameTable.GetBaseQPTable()
	baseTable.MZClubID = msgCreateTable.MZClubID
	baseTable.ClubPlayID = msgCreateTable.ClubPlayID
	baseTable.ClubRule = &clubRule
	baseTable.PayUID = msgCreateTable.PayUID
	baseTable.ClubRuleText = msgCreateTable.ClubConfig
	baseTable.MaxTZCount = msgCreateTable.MaxTZCount
	baseTable.RobotJoinReady = msgCreateTable.RobotJoinReady
	baseTable.RobotJoinPlaying = msgCreateTable.RobotJoinPlaying
	baseTable.RobotInviteTimer = msgCreateTable.RobotInviteTimer

	y, m, d := time.Now().Date()
	baseTable.TZKEY = fmt.Sprintf("%s%d%02d%02d", db.HKEY_MaxTZCount, y, m, d)

	// 加入桌子
	msgJoinTableData, _ := json.Marshal(&protoGameBasic.CS_ClubJoinTable{
		TableNumber:     baseTable.TableNum,
		IP:              msgCreateTable.IP,
		Longitude:       msgCreateTable.Longitude,
		Latitude:        msgCreateTable.Latitude,
		ClubID:          msgCreateTable.PlayerClubID,
		PlayerClubScore: msgCreateTable.PlayerClubScore})

	joinRsp = rootGameTable.OnMessage(&mateProto.MessageMaTe{
		Source:    msg.Source,
		SenderID:  msg.SenderID,
		MessageID: protoGameBasic.ID_ClubJoinTable,
		Data:      msgJoinTableData,
	})

	if joinRsp >= 0 {
		rootGameTable.StartWork(baseTable.TableNum)

		// 转发给 俱乐部
		msg.To, msg.MessageID = msg.MZID, protoGameBasic.ID_TablePutNew
		msgBody := protoGameBasic.SS_PutNewTable{
			ClubID:      msgCreateTable.MZClubID,
			TableNumber: baseTable.TableNum,
			ClubPlayID:  msgCreateTable.ClubPlayID,
			GameID:      msgCreateTable.GameID,
			MaxPlayers:  baseTable.MaxPlayers,
			UID:         msg.SenderID,
			CreateTime:  baseTable.CreateTime,
			ServiceID:   localConfig.GetConfig().ID,
		}

		err = wrapMQ.SendMsgTo(msg, &msgBody)
		if err != nil {
			glog.Warning("PutNewTable() err. err :=", err.Error())
		}
		rootGameTable.noticeClub = true

		return mateProto.Err_Success, rootGameTable
	}
	return joinRsp, nil
}

//
func onJoinClubTable(msg *mateProto.MessageMaTe) int32 {
	// 校正 玩家 所在 游戏服务
	rootab := getPlayer(msg.SenderID)
	if rootab != nil {
		return mateProto.Err_AlreadyPlaying
	} else {
		db.RemovePlayerGameIntro(msg.SenderID)
	}

	if serviceStatus == SS_NotJoinTable || serviceStatus == SS_Stop {
		glog.Warning("serviceStatus := ", serviceStatus)
		return mateProto.Err_ServiceStatus
	}

	msgJoinTable := protoGameBasic.CS_ClubJoinTable{}
	err := json.Unmarshal(msg.Data, &msgJoinTable)
	if err != nil {
		return mateProto.Err_ProtocolDataErr
	}
	msg.MsgBody = &msgJoinTable

	v, ok := tableMap.Load(msgJoinTable.TableNumber)
	if ok == false {
		return mateProto.Err_NotFindTable
	}

	v.(*rootTable).msgChan <- msg

	return 0
}

func onForceDissolveTable(msg *mateProto.MessageMaTe) int32 {

	msgBody := protoGameBasic.CS_ForceDissolveTable{}
	err := json.Unmarshal(msg.Data, &msgBody)
	if err != nil {
		return mateProto.Err_ProtocolDataErr
	}
	msg.MsgBody = &msgBody

	v, ok := tableMap.Load(msgBody.TableID)
	if ok == false {
		return mateProto.Err_NotFindTable
	}

	v.(*rootTable).msgChan <- msg

	return 0
}

func onGetTableCount(msg *mateProto.MessageMaTe) int32 {

	return 0
}
