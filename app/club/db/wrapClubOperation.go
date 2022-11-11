package db

import (
	"context"
	"fmt"
	"github.com/golang/glog"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/writeconcern"
	"time"
	"vvService/commonPackge/errorCodeDef"
	collClub "vvService/dbCollectionDefine/club"
	collPlayer "vvService/dbCollectionDefine/playerInfo"
)

func SetClubState(clubID int32, statusType int32, value bool) (int32, error) {
	coll := mongoDBClient.Database(databaseName).Collection(collClub.CollClubInfo)
	ctx := context.Background()

	filter := bson.M{"club_id": clubID}
	update := bson.M{}
	if statusType == 1 {
		update = bson.M{"$set": bson.M{"open": value, "close_timestamp": time.Now()}}
	} else if statusType == 2 {
		update = bson.M{"$set": bson.M{"frozen": value}}
	} else {
		return errorCodeDef.Err_Param, nil
	}

	updateResult, err := coll.UpdateOne(ctx, filter, update)
	if err != nil {
		return 0, err
	}
	if updateResult.MatchedCount < 1 && updateResult.ModifiedCount < 1 {
		return errorCodeDef.Err_Not_Find_Item, fmt.Errorf("not match uid:=%d,clubID:=%d", clubID, clubID)
	}
	return 0, nil
}

func SetClubStocking(clubIDArr []int32, value bool) (int32, error) {
	coll := mongoDBClient.Database(databaseName).Collection(collClub.CollClubInfo)
	ctx := context.Background()

	filter := bson.M{"club_id": bson.M{"$in": clubIDArr}}
	update := bson.M{"$set": bson.M{"stocking": value}}

	updateResult, err := coll.UpdateMany(ctx, filter, update)
	if err != nil {
		return 0, err
	}
	if updateResult.MatchedCount < 1 && updateResult.ModifiedCount < 1 {
		return errorCodeDef.Err_Not_Find_Item, fmt.Errorf("not match uid:=,clubID:=%v", clubIDArr)
	}
	return 0, nil
}

func SetClubSeal(clubID int32, value bool) (int32, error) {
	coll := mongoDBClient.Database(databaseName).Collection(collClub.CollClubInfo)
	ctx := context.Background()

	filter := bson.M{"club_id": clubID}
	update := bson.M{"$set": bson.M{"seal": value}}

	updateResult, err := coll.UpdateOne(ctx, filter, update)
	if err != nil {
		return 0, err
	}
	if updateResult.MatchedCount < 1 && updateResult.ModifiedCount < 1 {
		return errorCodeDef.Err_Not_Find_Item, fmt.Errorf("not match uid:=%d,clubID:=%d", clubID, clubID)
	}
	return 0, nil
}

func SetClubBaoDi(clubID int32, baoDi int64) (int32, error) {
	coll := mongoDBClient.Database(databaseName).Collection(collClub.CollClubInfo)
	ctx := context.Background()

	filter := bson.M{"club_id": clubID}
	updateResult, err := coll.UpdateOne(ctx, filter,
		bson.M{"$set": bson.M{"bao_di": baoDi}})
	if err != nil {
		return 0, err
	}
	if updateResult.MatchedCount < 1 && updateResult.ModifiedCount < 1 {
		return errorCodeDef.Err_Not_Find_Item, fmt.Errorf("not match uid:=%d,clubID:=%d", clubID, clubID)
	}
	return 0, nil
}

func SetClubPercentage(clubID, percentage int32) (int32, error) {
	coll := mongoDBClient.Database(databaseName).Collection(collClub.CollClubInfo)
	ctx := context.Background()

	filter := bson.M{"club_id": clubID}
	updateResult, err := coll.UpdateOne(ctx, filter,
		bson.M{"$set": bson.M{"d_superior.per": percentage}})
	if err != nil {
		return 0, err
	}
	if updateResult.MatchedCount < 1 && updateResult.ModifiedCount < 1 {
		return errorCodeDef.Err_Not_Find_Item, fmt.Errorf("not match uid:=%d,clubID:=%d", clubID, clubID)
	}
	return 0, nil
}

func SetClubManageFee(clubID int32, mangeFee int64) (int32, error) {
	coll := mongoDBClient.Database(databaseName).Collection(collClub.CollClubInfo)
	ctx := context.Background()

	filter := bson.M{"club_id": clubID}
	updateResult, err := coll.UpdateOne(ctx, filter,
		bson.M{"$set": bson.M{"manageFee": mangeFee}})
	if err != nil {
		return 0, err
	}
	if updateResult.MatchedCount < 1 && updateResult.ModifiedCount < 1 {
		return errorCodeDef.Err_Not_Find_Item, fmt.Errorf("not match uid:=%d,clubID:=%d", clubID, clubID)
	}
	return 0, nil
}

func SetClubNotice(clubID int32, text string) (int32, error) {
	coll := mongoDBClient.Database(databaseName).Collection(collClub.CollClubInfo)
	ctx := context.Background()

	filter := bson.M{"club_id": clubID}
	updateResult, err := coll.UpdateOne(ctx, filter,
		bson.M{"$set": bson.M{"notice": text}, "$inc": bson.M{"ver_num": 1}})
	if err != nil {
		return 0, err
	}
	if updateResult.MatchedCount < 1 && updateResult.ModifiedCount < 1 {
		return errorCodeDef.Err_Not_Find_Item, fmt.Errorf("not match uid:=%d,clubID:=%d", clubID, clubID)
	}
	return 0, nil
}

func SetMZNotice(clubIDArr []int32, text string) (int32, error) {
	coll := mongoDBClient.Database(databaseName).Collection(collClub.CollClubInfo)
	ctx := context.Background()

	filter := bson.M{"club_id": bson.M{"$in": clubIDArr}}
	updateResult, err := coll.UpdateMany(ctx, filter,
		bson.M{"$set": bson.M{"mz_notice": text}, "$inc": bson.M{"ver_num": 1}})
	if err != nil {
		return 0, err
	}
	if updateResult.MatchedCount < 1 && updateResult.ModifiedCount < 1 {
		return errorCodeDef.Err_Not_Find_Item, fmt.Errorf("not match uid:=, clubID:=%d", clubIDArr)
	}
	return 0, nil
}

//func SetGongXianWay(clubID int32, value int) (int32, error) {
//	coll := mongoDBClient.Database(databaseName).Collection(collClub.CollClubInfo)
//	ctx := context.Background()
//
//	filter := bson.M{"club_id": clubID}
//	updateResult, err := coll.UpdateOne(ctx, filter,
//		bson.M{"$set": bson.M{"gx_way": value}, "$inc": bson.M{"ver_num": 1}})
//	if err != nil {
//		return 0, err
//	}
//	if updateResult.MatchedCount < 1 && updateResult.ModifiedCount < 1 {
//		return errorCodeDef.Err_Not_Find_Item, fmt.Errorf("not match uid:=%d,clubID:=%d", clubID, clubID)
//	}
//	return 0, nil
//}

func SetShowScoreWater(clubIDArr []int32, value bool) (int32, error) {
	coll := mongoDBClient.Database(databaseName).Collection(collClub.CollClubInfo)
	ctx := context.Background()

	filter := bson.M{"club_id": bson.M{"$in": clubIDArr}}
	updateResult, err := coll.UpdateMany(ctx, filter,
		bson.M{"$set": bson.M{"score_water": value}, "$inc": bson.M{"ver_num": 1}})
	if err != nil {
		return 0, err
	}
	if updateResult.MatchedCount < 1 && updateResult.ModifiedCount < 1 {
		return errorCodeDef.Err_Not_Find_Item, fmt.Errorf("not match")
	}
	return 0, nil
}

func UpdateClubShowRankList(clubIDArr []int32, value bool) (int32, error) {
	coll := mongoDBClient.Database(databaseName).Collection(collClub.CollClubInfo)
	ctx := context.Background()

	filter := bson.M{"club_id": bson.M{"$in": clubIDArr}}
	updateResult, err := coll.UpdateMany(ctx, filter,
		bson.M{"$set": bson.M{"show_rankList": value}, "$inc": bson.M{"ver_num": 1}})
	if err != nil {
		return 0, err
	}
	if updateResult.MatchedCount < 1 && updateResult.ModifiedCount < 1 {
		return errorCodeDef.Err_Not_Find_Item, fmt.Errorf("not match")
	}
	return 0, nil
}

func UpdateClubMaxTongZhuoCount(clubIDArr []int32, value int32) (int32, error) {
	coll := mongoDBClient.Database(databaseName).Collection(collClub.CollClubInfo)
	ctx := context.Background()

	filter := bson.M{"club_id": bson.M{"$in": clubIDArr}}
	updateResult, err := coll.UpdateMany(ctx, filter,
		bson.M{"$set": bson.M{"tz_count": value}, "$inc": bson.M{"ver_num": 1}})
	if err != nil {
		return 0, err
	}
	if updateResult.MatchedCount < 1 && updateResult.ModifiedCount < 1 {
		return errorCodeDef.Err_Not_Find_Item, fmt.Errorf("not match")
	}
	return 0, nil
}

func UpdateClubBiLiWay(clubIDArr []int32, value int) (int32, error) {
	coll := mongoDBClient.Database(databaseName).Collection(collClub.CollClubInfo)
	ctx := context.Background()

	filter := bson.M{"club_id": bson.M{"$in": clubIDArr}}
	updateResult, err := coll.UpdateMany(ctx, filter,
		bson.M{"$set": bson.M{"bl_way": value}, "$inc": bson.M{"ver_num": 1}})
	if err != nil {
		return 0, err
	}
	if updateResult.MatchedCount < 1 && updateResult.ModifiedCount < 1 {
		return errorCodeDef.Err_Not_Find_Item, fmt.Errorf("not match")
	}
	return 0, nil
}

func UpdateClubPlayerGongXianWay(clubIDArr []int32, value int) (int32, error) {
	coll := mongoDBClient.Database(databaseName).Collection(collClub.CollClubInfo)
	ctx := context.Background()

	filter := bson.M{"club_id": bson.M{"$in": clubIDArr}}
	updateResult, err := coll.UpdateMany(ctx, filter,
		bson.M{"$set": bson.M{"gx_way": value}, "$inc": bson.M{"ver_num": 1}})
	if err != nil {
		return 0, err
	}
	if updateResult.MatchedCount < 1 && updateResult.ModifiedCount < 1 {
		return errorCodeDef.Err_Not_Find_Item, fmt.Errorf("not match")
	}
	return 0, nil
}

func UpdateClubShowBaoMingFee(clubIDArr []int32, value bool) (int32, error) {
	coll := mongoDBClient.Database(databaseName).Collection(collClub.CollClubInfo)
	ctx := context.Background()

	filter := bson.M{"club_id": bson.M{"$in": clubIDArr}}
	updateResult, err := coll.UpdateMany(ctx, filter,
		bson.M{"$set": bson.M{"bm_fee": value}, "$inc": bson.M{"ver_num": 1}})
	if err != nil {
		return 0, err
	}
	if updateResult.MatchedCount < 1 && updateResult.ModifiedCount < 1 {
		return errorCodeDef.Err_Not_Find_Item, fmt.Errorf("not match")
	}
	return 0, nil
}

func SetClubName(clubIDArr []int32, text string) (int32, error) {
	coll := mongoDBClient.Database(databaseName).Collection(collClub.CollClubInfo)
	ctx := context.Background()

	filter := bson.M{"club_id": bson.M{"$in": clubIDArr}}
	updateResult, err := coll.UpdateMany(ctx, filter,
		bson.M{"$set": bson.M{"name": text}, "$inc": bson.M{"ver_num": 1}})
	if err != nil {
		return 0, err
	}
	if updateResult.MatchedCount < 1 && updateResult.ModifiedCount < 1 {
		return errorCodeDef.Err_Not_Find_Item, fmt.Errorf("not match uid:=,clubID:=%d", clubIDArr)
	}
	return 0, nil
}

func SetClubKickOutMember(clubID int32, value bool) (int32, error) {
	coll := mongoDBClient.Database(databaseName).Collection(collClub.CollClubInfo)
	ctx := context.Background()

	filter := bson.M{"club_id": clubID}
	updateResult, err := coll.UpdateOne(ctx, filter,
		bson.M{"$set": bson.M{"kt_member": value}})
	if err != nil {
		return 0, err
	}
	if updateResult.MatchedCount < 1 && updateResult.ModifiedCount < 1 {
		return errorCodeDef.Err_Not_Find_Item, fmt.Errorf("not match uid:=%d,clubID:=%d", clubID, clubID)
	}
	return 0, nil
}

func SetClubKickOutLeague(clubID int32, value bool) (int32, error) {
	coll := mongoDBClient.Database(databaseName).Collection(collClub.CollClubInfo)
	ctx := context.Background()

	filter := bson.M{"club_id": clubID}
	updateResult, err := coll.UpdateOne(ctx, filter,
		bson.M{"$set": bson.M{"kt_League": value}})
	if err != nil {
		return 0, err
	}
	if updateResult.MatchedCount < 1 && updateResult.ModifiedCount < 1 {
		return errorCodeDef.Err_Not_Find_Item, fmt.Errorf("not match uid:=%d,clubID:=%d", clubID, clubID)
	}
	return 0, nil
}

func SetClubMemberFreeExit(clubID int32, value bool) (int32, error) {
	coll := mongoDBClient.Database(databaseName).Collection(collClub.CollClubInfo)
	ctx := context.Background()

	filter := bson.M{"club_id": clubID}
	updateResult, err := coll.UpdateOne(ctx, filter,
		bson.M{"$set": bson.M{"free_quit": value}})
	if err != nil {
		return 0, err
	}
	if updateResult.MatchedCount < 1 && updateResult.ModifiedCount < 1 {
		return errorCodeDef.Err_Not_Find_Item, fmt.Errorf("not match uid:=%d,clubID:=%d", clubID, clubID)
	}
	return 0, nil
}

// (所有上级俱乐部ID,直属上级,被解除的俱乐部，被解除俱乐部的所有下级(不包括被解除的))
func DiscardCombination(superiorClubIDArr []int32, direClubID, removeClubID int32, subordinateClubIDArr []int32) (int32, error) {

	ctx := context.Background()

	tempAllSubordinateClubID := subordinateClubIDArr
	tempAllSubordinateClubID = append(tempAllSubordinateClubID, removeClubID)

	wcMajority := writeconcern.New(writeconcern.WMajority(), writeconcern.WTimeout(3*time.Second))
	wcMajorityCollectionOpts := options.Collection().SetWriteConcern(wcMajority)
	collClub := mongoDBClient.Database(databaseName).Collection(collClub.CollClubInfo, wcMajorityCollectionOpts)

	session, err := mongoDBClient.StartSession()
	if err != nil {
		return 0, err
	}
	defer session.EndSession(ctx)

	_, err = session.WithTransaction(ctx, func(sctx mongo.SessionContext) (interface{}, error) {

		// 更新自己
		updateRes, err := collClub.UpdateOne(sctx, bson.M{"club_id": removeClubID},
			bson.M{"$set": bson.M{"d_superior.club_id": 0, "d_superior.per": 100, "manageFee": 0}})
		if err != nil {
			return nil, err
		}

		if updateRes.ModifiedCount < 1 {
			return nil, fmt.Errorf("not find match clubID:=%d", removeClubID)
		}

		// 更新 直属上级
		updateRes, err = collClub.UpdateOne(sctx, bson.M{"club_id": direClubID},
			bson.M{"$pull": bson.M{"d_subordinate": bson.M{"club_id": removeClubID}}})
		if updateRes.ModifiedCount < 1 {
			return nil, fmt.Errorf("not find match clubID:=%d", direClubID)
		}

		// 更新所有上级
		for _, v := range superiorClubIDArr {
			updateRes, err = collClub.UpdateOne(sctx, bson.M{"club_id": v},
				bson.M{"$pull": bson.M{"subordinates": bson.M{"$in": tempAllSubordinateClubID}}})
			if updateRes.ModifiedCount < 1 {
				return nil, fmt.Errorf("not find match clubID:=%d", v)
			}
		}

		// 更新 移除合并 俱乐部 的 所有下级俱乐部 的百分比
		//_, err = collClub.UpdateMany(sctx, bson.M{"club_id": bson.M{"$in": subordinateClubIDArr}},
		//	bson.M{"$set": bson.M{"d_superior.per": 0}})
		//if err != nil {
		//	return nil, err
		//}

		return nil, nil
	})

	return 0, err
}

// (所有上级俱乐部ID,直属上级,被解除的俱乐部，被解除俱乐部的所有下级(不包括被解除的))
func DiscardCombination_(superiorClubIDArr []int32, direClubID, removeClubID int32, subordinateClubIDArr []int32) (int32, error) {

	glog.Warning("superiorClubIDArr:=", superiorClubIDArr, ",direClubID:=", direClubID, ",removeClubID:=", removeClubID, ",subordinateClubIDArr:=", subordinateClubIDArr)

	ctx := context.Background()

	tempAllSubordinateClubID := subordinateClubIDArr
	tempAllSubordinateClubID = append(tempAllSubordinateClubID, removeClubID)

	wcMajority := writeconcern.New(writeconcern.WMajority(), writeconcern.WTimeout(3*time.Second))
	wcMajorityCollectionOpts := options.Collection().SetWriteConcern(wcMajority)
	collClub := mongoDBClient.Database(databaseName).Collection(collClub.CollClubInfo, wcMajorityCollectionOpts)
	collPlayer_ := mongoDBClient.Database(databaseName).Collection(collPlayer.CollPlayerInfo, wcMajorityCollectionOpts)

	session, err := mongoDBClient.StartSession()
	if err != nil {
		return 0, err
	}
	defer session.EndSession(ctx)

	_, err = session.WithTransaction(ctx, func(sctx mongo.SessionContext) (interface{}, error) {

		// 更新 俱乐部的 创建者(移除关系)
		_, err = collClub.UpdateMany(sctx, bson.M{"club_id": bson.M{"$in": tempAllSubordinateClubID}},
			bson.M{"$mul": bson.M{"creator_id": -1}})
		if err != nil {
			return nil, err
		}

		// 更新 玩家的所属俱乐部(移除关系)
		updateRes, err := collPlayer_.UpdateMany(sctx, bson.M{},
			bson.M{"$pull": bson.M{"club_info": bson.M{"clubID": bson.M{"$in": tempAllSubordinateClubID}}}})
		if err != nil {
			return nil, err
		}

		if updateRes.ModifiedCount < 1 {
			return nil, fmt.Errorf(".not find match clubID:=%d", removeClubID)
		}

		// 更新 直属上级 的 直属下级
		updateRes, err = collClub.UpdateOne(sctx, bson.M{"club_id": direClubID},
			bson.M{"$pull": bson.M{"d_subordinate": bson.M{"club_id": removeClubID}}})
		if updateRes.ModifiedCount < 1 {
			return nil, fmt.Errorf("..not find match clubID:=%d", direClubID)
		}

		// 更新所有上级的 子级
		for _, v := range superiorClubIDArr {
			updateRes, err = collClub.UpdateOne(sctx, bson.M{"club_id": v},
				bson.M{"$pull": bson.M{"subordinates": bson.M{"$in": tempAllSubordinateClubID}}})
			if updateRes.ModifiedCount < 1 {
				return nil, fmt.Errorf("...not find match clubID:=%d", v)
			}
		}

		// 更新 移除合并 俱乐部 的 所有下级俱乐部 的百分比
		//_, err = collClub.UpdateMany(sctx, bson.M{"club_id": bson.M{"$in": subordinateClubIDArr}},
		//	bson.M{"$set": bson.M{"d_superior.per": 0}})
		//if err != nil {
		//	return nil, err
		//}

		return nil, nil
	})

	return 0, err
}

//批量修改
func BatchSetPercentage(updateClubMap map[int32]int32) error {

	ctx := context.Background()

	wcMajority := writeconcern.New(writeconcern.WMajority(), writeconcern.WTimeout(3*time.Second))
	wcMajorityCollectionOpts := options.Collection().SetWriteConcern(wcMajority)
	collClub := mongoDBClient.Database(databaseName).Collection(collClub.CollClubInfo, wcMajorityCollectionOpts)

	session, err := mongoDBClient.StartSession()
	if err != nil {
		return err
	}
	defer session.EndSession(ctx)

	_, err = session.WithTransaction(ctx, func(sctx mongo.SessionContext) (interface{}, error) {

		for k, v := range updateClubMap {
			updateResult, err := collClub.UpdateOne(sctx, bson.M{"club_id": k},
				bson.M{"$set": bson.M{"d_superior.per": v}})
			if err != nil {
				return 0, err
			}
			if updateResult.MatchedCount < 1 && updateResult.ModifiedCount < 1 {
				return errorCodeDef.Err_Not_Find_Item, fmt.Errorf("not match clubID:=%d", k)
			}
		}

		return nil, nil
	})

	return err
}

/*
// 俱乐部移除合并
func (this *ClubDBMongo) ClubOperationRemoveMerge(uid int64, clubID, removeClubID int32, removeSubordinateClubID []int32) error {

	tempAllSubordinateClubID := removeSubordinateClubID
	tempAllSubordinateClubID = append(tempAllSubordinateClubID, removeClubID)

	wcMajority := writeconcern.New(writeconcern.WMajority(), writeconcern.WTimeout(1*time.Second))
	wcMajorityCollectionOpts := options.Collection().SetWriteConcern(wcMajority)
	collClub := this.DB.Database(base.DatabaseName).Collection(base.CollClub, wcMajorityCollectionOpts)

	session, err := this.DB.StartSession()
	if err != nil {
		return err
	}
	defer session.EndSession(this.Ctx)

	_, err = session.WithTransaction(this.Ctx, func(sctx mongo.SessionContext) (interface{}, error) {

		updateRes, err := collClub.UpdateOne(sctx, bson.M{"club_id": removeClubID, "direct_supervisor.club_id": clubID},
			bson.M{"$set": bson.M{"direct_supervisor.club_id": 0, "direct_supervisor.percentage": 100}})
		if err != nil {
			return nil, err
		}

		if updateRes.ModifiedCount < 1 {
			return nil, fmt.Errorf("not find match clubID:=%d", clubID)
		}

		var singleRes *mongo.SingleResult
		opt := options.FindOneAndUpdateOptions{}
		opt.SetProjection(bson.M{"direct_supervisor": 1})
		singleRes = collClub.FindOneAndUpdate(sctx, bson.M{"club_id": clubID, "direct_subordinate.club_id": removeClubID},
			bson.M{"$pull": bson.M{"direct_subordinate": bson.M{"club_id": removeClubID}, "subordinates": bson.M{"$in": tempAllSubordinateClubID}}}, &opt)

		var clubData *base.DBClub
		err = singleRes.Decode(&clubData)
		if err != nil {
			return nil, err
		}

		for clubData.DirectSupervisor.ClubID > 0 {
			singleRes = collClub.FindOneAndUpdate(sctx, bson.M{"club_id": clubData.DirectSupervisor.ClubID},
				bson.M{"$pull": bson.M{"subordinates": bson.M{"$in": tempAllSubordinateClubID}}}, &opt)
			err = singleRes.Decode(&clubData)
			if err != nil {
				return nil, err
			}
		}

		// 更新 移除合并 俱乐部 的 所有下级俱乐部 的百分比
		_, err = collClub.UpdateMany(sctx, bson.M{"club_id": bson.M{"$in": removeSubordinateClubID}},
			bson.M{"$set": bson.M{"direct_supervisor.percentage": 0}})
		if err != nil {
			return nil, err
		}

		return nil, nil
	})

	return err
}
*/
