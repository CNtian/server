package tableSnapshot

import (
	"time"
	"vvService/appClub/clubCommon"
	"vvService/appClub/localConfig"
	clubProto "vvService/appClub/protoDefine"
	commonDef "vvService/commonPackge"
	"vvService/commonPackge/mateProto"
)

var (
	msgChan         = make(chan *mateProto.MessageMaTe, 1024)
	SelfPostEvent   SelfPostEvents
	postEventToClub clubCommon.PostEvent

	// 800305 shao hua
	//testMZClubID = int32(0) //int32(807279)
)

type SelfPostEvents struct {
}

func (this *SelfPostEvents) PostMaTeEvent(msg *mateProto.MessageMaTe) {
	msgChan <- msg
}

func SetClubEvent(e clubCommon.PostEvent) {
	postEventToClub = e
}

func HandleRequest() {

	if localConfig.GetConfig().ID == "club" {
		return
	}

	commonDef.Wait.Add(1)
	defer commonDef.Wait.Done()

	handleFunc := func(msg *mateProto.MessageMaTe) {
		switch msg.MessageID {
		case clubProto.ID_TablePutNew:
			onPutNewTable(msg)
		case clubProto.ID_TablePutPlayer:
			onPutPlayer(msg)
		case clubProto.ID_TableDelPlayer:
			onDeletePlayer(msg)
		case clubProto.ID_TableStatusChanged:
			onTablePlaying(msg)
		case clubProto.ID_TableDelete:
			onDeleteTable(msg)
		//case clubProto.ID_GetPlayerClubInfo:
		//	onGetPlayerClubInfo(msg)
		case clubProto.ID_TableGet, clubProto.ID_PerSeconGetTables:
			onGetTable(msg)
		case clubProto.ID_PushAllTable:
			onPushServiceIDAllTable(msg)
		case clubProto.ID_DeleteServiceIDTable:
			onDeleteServiceIDTable(msg)
		case clubProto.ID_QuickStart:
			onQuickStart(msg)
		case clubProto.ID_DeleteClubPlay:
			onDeleteClubPlay(msg)
		case clubProto.ID_GetClubTableCount:
			onGetClubTableCount(msg)
		default:
		}
	}

	tt := time.After(time.Second)
	for commonDef.IsRun {
		select {
		case msg := <-msgChan:
			handleFunc(msg)
		case <-tt:
			tableCount := 0
			for _, v := range clubMap {
				tableCount += makeTableJson(v)
			}
			// 告知俱乐部当前桌子总数
			postEventToClub.PostMaTeEvent(&mateProto.MessageMaTe{MessageID: mateProto.ID_CurrentTableCount,
				SenderID: int64(tableCount)})

			tt = time.After(time.Second)
		}
	}
}
