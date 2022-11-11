package worker

import (
	"encoding/json"
	"fmt"
	"github.com/golang/glog"
	"go.mongodb.org/mongo-driver/mongo"
	"strconv"
	"time"
	"vvService/appDB/db"
	"vvService/appDB/protoDefine"
	commonDef "vvService/commonPackge"
	"vvService/commonPackge/mateProto"
	collClub "vvService/dbCollectionDefine/club"
)

func onTotalManageFee(msg *mateProto.MessageMaTe) {

	msgBody := protoDefine.SS_TotalMangeFee{}
	json.Unmarshal(msg.Data, &msgBody)

	clubIDMap, err := db.GetClubManageFeeClubID(msgBody.Date)
	if err != nil {
		glog.Warning("onTotalManageFee() err. date:=", msgBody.Date, ",err:=", err.Error())
		return
	}

	var (
		clubData       *collClub.DBClubData
		supervisorClub *collClub.DBClubData
	)
	for _, v := range clubIDMap {
		if v.IsPay == true {
			continue
		}

		manageFeeLog := collClub.DBClubManageFeeLog{}
		tempClubID := v.ClubID
		for i := 0; i < 100 && tempClubID != 0; i++ {
			clubMangeFee, ok := clubIDMap[tempClubID]
			if ok == false || clubMangeFee.IsPay == true {
				continue
			}

			clubData, err = loadClubData(tempClubID)
			if err != nil {
				glog.Warning("onTotalManageFee() err. date:=", msgBody.Date, ",clubID:=", tempClubID, ",err:=", err.Error())
				break
			}
			tempClubID = clubData.DirectSupervisor.ClubID
			if tempClubID < 1 {
				break
			}

			clubMangeFee.IsPay = true
			// 避免多次重算 管理费， 每次只算更新的费用
			clubMangeFee.ManageFee = (clubMangeFee.HaoKa*clubData.ManagementFee)/commonDef.SR - clubMangeFee.ManageFee
			clubMangeFee.TempManageFeeCount += clubMangeFee.ManageFee

			supervisorClub, err = loadClubData(clubData.DirectSupervisor.ClubID)
			if err != nil {
				glog.Warning("onTotalManageFee() err. date:=", msgBody.Date, ",clubID:=", clubData.DirectSupervisor.ClubID, ",err:=", err.Error())
				break
			}

			// 写入日志
			manageFeeLog.PayUID = clubData.CreatorID
			manageFeeLog.PayClubID, manageFeeLog.PayClubName = clubData.ClubID, clubData.Name
			manageFeeLog.ConsumeCount, manageFeeLog.ManageFeeScore = clubMangeFee.HaoKa, clubMangeFee.ManageFee

			manageFeeLog.GotUID = supervisorClub.CreatorID
			manageFeeLog.GotClubID, manageFeeLog.GotClubName = supervisorClub.ClubID, supervisorClub.Name

			err = db.UpdateManageFeeLog(&manageFeeLog)
			if err != nil {
				glog.Warning("UpdateManageFeeLog() err. err:=", err.Error(), ",param:=", manageFeeLog)
			}

			if manageFeeLog.PayBeforeClubScore < 0 {
				unusable := unusableInfo{ClubID: manageFeeLog.PayClubID}

				unusableMap := map[int64]*unusableInfo{}
				unusableMap[manageFeeLog.PayUID] = &unusable

				unusable.Score = manageFeeLog.GotCurClubScore - manageFeeLog.GotBeforeClubScore

				putClubPlayerUnusableScore(unusableMap)
				updateClubUnusableScore(unusableMap)
			} else if manageFeeLog.PayCurClubScore < 0 {
				unusable := unusableInfo{ClubID: manageFeeLog.PayClubID}

				unusableMap := map[int64]*unusableInfo{}
				unusableMap[manageFeeLog.PayUID] = &unusable

				unusable.Score = manageFeeLog.GotCurClubScore

				putClubPlayerUnusableScore(unusableMap)
				updateClubUnusableScore(unusableMap)
			}

			if manageFeeLog.GotBeforeClubScore < 0 {
				unusable := unusableInfo{ClubID: manageFeeLog.GotClubID}

				unusableMap := map[int64]*unusableInfo{}
				unusableMap[manageFeeLog.GotUID] = &unusable

				if manageFeeLog.GotCurClubScore >= 0 {
					var tempScore int64
					tempScore, err = db.DeleteClubPlayerUnusableScore(manageFeeLog.GotUID, manageFeeLog.GotClubID)
					if err != nil {
						if err == mongo.ErrNoDocuments {
							return
						}
						glog.Warning("DeleteClubPlayerUnusableScore() err:=", err.Error(), ", uid:=", manageFeeLog.GotUID, ",clubID:=", manageFeeLog.GotClubID)
						return
					}
					tempScore *= -1
					updateClubUnusableScore(unusableMap)
				} else {
					unusable.Score = manageFeeLog.GotCurClubScore - manageFeeLog.GotBeforeClubScore

					putClubPlayerUnusableScore(unusableMap)
					updateClubUnusableScore(unusableMap)
				}
			}
		}
	}

	err = db.UpdateMangeFee(msgBody.Date, clubIDMap)
	if err != nil {
		glog.Warning("UpdateMangeFee() err. date:=", msgBody.Date, ",err:=", err.Error())
	}
}

func onDeleteTotal(msg *mateProto.MessageMaTe) {
	msgDeleteClubTotal := mateProto.SS_DeleteClubTotal{}
	err := json.Unmarshal(msg.Data, &msgDeleteClubTotal)
	if err != nil {
		glog.Warning("onDeleteTotal() err. clubID:=", msgDeleteClubTotal.ClubID, " ,err:=", err.Error())
		return
	}
	nowTT := time.Now()
	dateText := fmt.Sprintf("%d%02d%02d", nowTT.Year(), nowTT.Month(), nowTT.Day())
	dateInt, _ := strconv.Atoi(dateText)

	var haokaCount int64
	haokaCount, err = db.GetClubHaoKaCount(dateInt, msgDeleteClubTotal.ClubID)
	if err != nil {
		glog.Warning("onDeleteTotal() err. clubID:=", msgDeleteClubTotal.ClubID, " ,err:=", err.Error())
		return
	}
	if haokaCount < 1 {
		return
	}

	msgDeleteClubTotal.SubordinateClubID = append(msgDeleteClubTotal.SubordinateClubID, msgDeleteClubTotal.ClubID)
	db.UpdateHaoKa(dateInt, msgDeleteClubTotal.SuperiorClubID, haokaCount, msgDeleteClubTotal.SubordinateClubID)
}
