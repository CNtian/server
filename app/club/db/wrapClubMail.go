package db

import (
	"context"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
	"vvService/commonPackge/errorCodeDef"
	collClub "vvService/dbCollectionDefine/club"
	collPlayer "vvService/dbCollectionDefine/playerInfo"
)

const clubMailCount = 100

// 申请加入
func ApplyJoinClub(clubID int32, uid int64, head, nick string) (int32, error) {
	collClubMail := mongoDBClient.Database(databaseName).Collection(collClub.CollClubMail)
	ctx := context.Background()

	// 是否已经申请过了
	filter := bson.M{"club_id": clubID, "data.apply_id": uid, "category": collClub.MailApplyJoinClub, "status": collClub.MailStatusUnread}
	res, err := collClubMail.CountDocuments(ctx, filter)
	if err != nil {
		return 0, err
	}
	if res > 0 {
		return errorCodeDef.ErrClubRepeatOperation, fmt.Errorf("don't repeat apply")
	}

	applyData := collClub.DBApplyJoinClub{
		ApplyID:          uid,
		ApplyAccountName: nick,
		ApplyHeadUrl:     head,
	}

	// 写入 申请记录
	ctx = context.Background()
	mail := collClub.NewDBClubMail(clubID, collClub.MailApplyJoinClub, &applyData)
	_, err = collClubMail.InsertOne(ctx, &mail)
	return 0, err
}

// 申请退出
func ApplyExitClub(clubID int32, uid int64) (int32, error) {
	collClubMail := mongoDBClient.Database(databaseName).Collection(collClub.CollClubMail)
	ctx := context.Background()

	// 是否已经申请过了
	filter := bson.M{"club_id": clubID, "data.apply_id": uid, "category": collClub.MailApplyExitClub, "status": collClub.MailStatusUnread}
	res, err := collClubMail.CountDocuments(ctx, filter)
	if err != nil {
		return 0, err
	}
	if res > 0 {
		return errorCodeDef.ErrClubRepeatOperation, fmt.Errorf("don't repeat apply")
	}

	applyData := collClub.DBApplyExitClub{ApplyID: uid}

	// 写入 申请记录
	ctx = context.Background()
	mail := collClub.NewDBClubMail(clubID, collClub.MailApplyExitClub, &applyData)
	_, err = collClubMail.InsertOne(ctx, &mail)
	return 0, err
}

// 获取申请加入列表
func GetClubMail(clubID, status int32) ([]*collClub.DBClubMail, error) {
	collClubMail := mongoDBClient.Database(databaseName).Collection(collClub.CollClubMail)
	ctx := context.Background()

	opt := options.FindOptions{}
	opt.SetLimit(clubMailCount)
	opt.SetSort(bson.M{"create_time": -1})

	filter := bson.M{"club_id": clubID}
	if status == 1 {
		filter["data.pass"] = 0
	}

	cur, err := collClubMail.Find(ctx, filter, &opt)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	arr := make([]*collClub.DBClubMail, 0, 30)
	for cur.Next(ctx) {
		mailItem := collClub.DBClubMail{}

		rawValue := cur.Current.Lookup("_id")
		mailItem.ID = rawValue.ObjectID()

		rawValue = cur.Current.Lookup("create_time")
		mailItem.CreateTime = rawValue.Time()

		rawValue = cur.Current.Lookup("status")
		mailItem.Status = collClub.MailStatus(rawValue.Int32())

		rawValue = cur.Current.Lookup("category")
		mailItem.Category = collClub.MailType(rawValue.AsInt32())

		mailData := cur.Current.Lookup("data")
		switch mailItem.Category {
		case collClub.MailMergeClub:
			temp := &collClub.DBApplyMergeClub{}
			mailData.Unmarshal(temp)
			mailItem.Data = temp
		case collClub.MailApplyJoinClub:
			temp := &collClub.DBApplyJoinClub{}
			mailData.Unmarshal(temp)
			mailItem.Data = temp
		case collClub.MailKickOutLeague:
			temp := &collClub.DBKickOutLeague{}
			mailData.Unmarshal(temp)
			mailItem.Data = temp
		case collClub.MailApplyExitLeague:
			temp := &collClub.DBApplyExitLeague{}
			mailData.Unmarshal(temp)
			mailItem.Data = temp
		case collClub.MailApplyExitClub:
			temp := &collClub.DBApplyExitClub{}
			mailData.Unmarshal(temp)
			mailItem.Data = temp
		default:
		}

		arr = append(arr, &mailItem)
	}
	return arr, nil
}

// 是否有未处理邮件
func CheckNewMail(clubID int32) (int64, error) {
	collClubMail := mongoDBClient.Database(databaseName).Collection(collClub.CollClubMail)
	ctx := context.Background()

	opt := options.CountOptions{}
	opt.SetLimit(clubMailCount)

	filter := bson.M{"club_id": clubID, "data.pass": 0}

	cur, err := collClubMail.CountDocuments(ctx, filter, &opt)
	if err != nil {
		return 0, err
	}
	return cur, nil
}

// 操作加入俱乐部
func CheckApplyJoin(id primitive.ObjectID, clubID int32, applyUID int64, isPass bool, operID int64, operName string) (int32, error) {
	collClubMail := mongoDBClient.Database(databaseName).Collection(collClub.CollClubMail)
	ctx := context.Background()

	var value int8
	if isPass == true {
		value = 1
	} else {
		value = 2
	}

	filter := bson.M{"_id": id, "club_id": clubID, "data.apply_id": applyUID, "data.pass": 0}
	update := bson.M{"$set": bson.M{"status": collClub.MailStatusRead, "data.pass": value,
		"data.ok_id": operID, "data.ok_nick": operName}}
	res, err := collClubMail.UpdateOne(ctx, filter, update)
	if err != nil {
		return 0, err
	}
	if res.ModifiedCount < 1 {
		return errorCodeDef.Err_Not_Find_Item, fmt.Errorf("modifyCount<1")
	}

	if isPass == false {
		return 0, nil
	}

	return DragIntoClub(clubID, applyUID)
}

// 操作加入俱乐部
func CheckApplyExitClub(id primitive.ObjectID, clubID int32, applyUID int64, isPass bool, operID int64, operName string) (int32, error) {
	collClubMail := mongoDBClient.Database(databaseName).Collection(collClub.CollClubMail)
	ctx := context.Background()

	var value int8
	if isPass == true {
		value = 1
	} else {
		value = 2
	}

	filter := bson.M{"_id": id, "club_id": clubID, "data.apply_id": applyUID, "data.pass": 0}
	update := bson.M{"$set": bson.M{"status": collClub.MailStatusRead, "data.pass": value,
		"data.ok_id": operID, "data.ok_nick": operName}}
	res, err := collClubMail.UpdateOne(ctx, filter, update)
	if err != nil {
		return 0, err
	}
	if res.ModifiedCount < 1 {
		return errorCodeDef.Err_Not_Find_Item, fmt.Errorf("modifyCount<1")
	}

	return 0, nil
}

// 申请合并至某俱乐部
func ApplyMergeClub(applicant *collClub.DBApplyMergeClub, targetClubID int32) (int32, error) {
	collClubMail := mongoDBClient.Database(databaseName).Collection(collClub.CollClubMail)
	ctx := context.Background()

	// 是否已经申请过了
	filter := bson.M{"club_id": targetClubID, "data.apply_club_id": applicant.ApplyClubID, "category": collClub.MailMergeClub, "status": collClub.MailStatusUnread}
	res, err := collClubMail.CountDocuments(ctx, filter)
	if err != nil {
		return 0, err
	}
	if res > 0 {
		return errorCodeDef.ErrClubRepeatOperation, nil
	}

	// 写入 申请记录
	ctx = context.Background()
	mail := collClub.NewDBClubMail(targetClubID, collClub.MailMergeClub, applicant)
	_, err = collClubMail.InsertOne(ctx, &mail)
	return 0, err
}

// 申请退出联盟
func ApplyExitLeague(applicant *collClub.DBApplyExitLeague, targetClubID int32) (int32, error) {
	collClubMail := mongoDBClient.Database(databaseName).Collection(collClub.CollClubMail)
	ctx := context.Background()

	// 是否已经申请过了
	filter := bson.M{"data.target_clubID": targetClubID, "data.initiator_clubID": applicant.InitiatorClubID, "category": collClub.MailApplyExitLeague, "status": collClub.MailStatusUnread}
	res, err := collClubMail.CountDocuments(ctx, filter)
	if err != nil {
		return 0, err
	}
	if res > 0 {
		return errorCodeDef.ErrClubRepeatOperation, nil
	}

	// 写入 申请记录
	ctx = context.Background()
	mail := collClub.NewDBClubMail(targetClubID, collClub.MailApplyExitLeague, applicant)
	_, err = collClubMail.InsertOne(ctx, &mail)
	return 0, err
}

// 申请踢出联盟
func ApplyKickOutLeague(applicant *collClub.DBKickOutLeague, targetClubID int32) (int32, error) {
	collClubMail := mongoDBClient.Database(databaseName).Collection(collClub.CollClubMail)
	ctx := context.Background()

	// 是否已经申请过了
	filter := bson.M{"data.target_clubID": targetClubID, "data.initiator_clubID": applicant.InitiatorClubID, "category": collClub.MailKickOutLeague, "status": collClub.MailStatusUnread}
	res, err := collClubMail.CountDocuments(ctx, filter)
	if err != nil {
		return 0, err
	}
	if res > 0 {
		return errorCodeDef.ErrClubRepeatOperation, nil
	}

	// 写入 申请记录
	ctx = context.Background()
	mail := collClub.NewDBClubMail(targetClubID, collClub.MailKickOutLeague, applicant)
	_, err = collClubMail.InsertOne(ctx, &mail)
	return 0, err
}

func CheckMergeClub(id primitive.ObjectID, applyClubID, targetClubID int32, pass bool, operID int64, operName string, clubLevel int) (int32, error) {

	collClubMail := mongoDBClient.Database(databaseName).Collection(collClub.CollClubMail)
	ctx := context.Background()

	var value int8
	if pass == true {
		value = 1
	} else {
		value = 2
	}

	filter := bson.M{"_id": id, "club_id": targetClubID, "data.apply_club_id": applyClubID, "data.pass": 0}
	update := bson.M{"$set": bson.M{"data.pass": value, "status": collClub.MailStatusRead,
		"data.ok_id": operID, "data.ok_nick": operName}}
	res, err := collClubMail.UpdateOne(ctx, filter, update)
	if err != nil {
		return 0, err
	}
	if res.ModifiedCount < 1 {
		return errorCodeDef.Err_Not_Find_Item, nil
	}

	if pass == false {
		return 0, nil
	}

	return MergeClub(applyClubID, targetClubID)
}

// 操作 退出联盟俱乐部
func CheckExitLeague(id primitive.ObjectID, isPass bool, operID int64, operName string) (int32, *collClub.DBClubMail, error) {
	collClubMail := mongoDBClient.Database(databaseName).Collection(collClub.CollClubMail)
	ctx := context.Background()

	var value int8
	if isPass == true {
		value = 1
	} else {
		value = 2
	}

	filter := bson.M{"_id": id, "data.pass": 0}
	update := bson.M{"$set": bson.M{"status": collClub.MailStatusRead, "data.pass": value,
		"data.ok_id": operID, "data.ok_nick": operName}}
	res := collClubMail.FindOneAndUpdate(ctx, filter, update)
	if res.Err() != nil {
		return 0, nil, res.Err()
	}
	rawData, err := res.DecodeBytes()
	if err != nil {
		return errorCodeDef.Err_Not_Find_Item, nil, err
	}
	category := rawData.Lookup("category")

	clubMail := collClub.DBClubMail{Category: collClub.MailType(category.Int32())}

	if category.Int32() == int32(collClub.MailKickOutLeague) {
		kickOutData := rawData.Lookup("data")
		temp := collClub.DBKickOutLeague{}
		err = kickOutData.Unmarshal(&temp)
		if err != nil {
			return 0, nil, err
		}
		clubMail.Data = &temp
		return 0, &clubMail, nil
	} else if category.Int32() == int32(collClub.MailApplyExitLeague) {
		kickOutData := rawData.Lookup("data")
		temp := collClub.DBApplyExitLeague{}
		err = kickOutData.Unmarshal(&temp)
		if err != nil {
			return 0, nil, err
		}
		clubMail.Data = &temp
		return 0, &clubMail, nil
	}

	return errorCodeDef.Err_Not_Find_Item, nil, nil
}

func MailOperationFailed(id primitive.ObjectID) {
	collClubMail := mongoDBClient.Database(databaseName).Collection(collClub.CollClubMail)

	collClubMail.UpdateOne(context.Background(), bson.M{"_id": id}, bson.M{"$set": bson.M{"data.pass": 2}})
	//res, err := collClubMail.UpdateOne(context.Background(), bson.M{"_id": id}, bson.M{"$set": bson.M{"data.pass": 2}})
	//if err != nil {
	//	glog.Warning(err.Error())
	//}
	//glog.Warning(res.MatchedCount, ",")
}

// 邀请加入俱乐部
func InviteJoinClub(mzClubID, clubID int32, clubName string, inviterID, uid int64) (int32, primitive.ObjectID, error) {
	collPlayerMail := mongoDBClient.Database(databaseName).Collection(collPlayer.CollPlayerEmail)

	// 是否已经申请过了
	filter := bson.M{"uid": uid, "data.club_id": clubID, "category": collPlayer.EmailInviteJoinClub, "status": collClub.MailStatusUnread}
	res, err := collPlayerMail.CountDocuments(nil, filter)
	if err != nil {
		return 0, primitive.NilObjectID, err
	}
	if res > 0 {
		return errorCodeDef.ErrClubRepeatOperation, primitive.NilObjectID, nil
	}

	email := collPlayer.NewPlayerEmail(uid, collPlayer.EmailInviteJoinClub, &collPlayer.ItemEmailInviteToClub{
		MZClubID:  mzClubID,
		ClubID:    clubID,
		ClubName:  clubName,
		InviterID: inviterID,
	})

	// 写入 申请记录
	insertR, err := collPlayerMail.InsertOne(nil, &email)
	if err != nil {
		return errorCodeDef.Err_Failed, primitive.NilObjectID, err
	}
	return 0, insertR.InsertedID.(primitive.ObjectID), err
}

// 邀请成为盟主
func InviteToMengZhu(inviter, playerID int64) (int32, primitive.ObjectID, error) {
	collPlayerMail := mongoDBClient.Database(databaseName).Collection(collPlayer.CollPlayerEmail)

	// 是否已经申请过了
	filter := bson.M{"uid": playerID, "data.uid": inviter, "category": collPlayer.EmailInviteToMengZhu, "status": collClub.MailStatusUnread}
	res, err := collPlayerMail.CountDocuments(nil, filter)
	if err != nil {
		return 0, primitive.NilObjectID, err
	}
	if res > 0 {
		return errorCodeDef.ErrClubRepeatOperation, primitive.NilObjectID, nil
	}

	_, _, nick, _ := LoadPlayerHead(inviter)

	email := collPlayer.NewPlayerEmail(playerID, collPlayer.EmailInviteToMengZhu, &collPlayer.ItemEmailInviteToMengZhu{
		Uid:  inviter,
		Name: nick,
	})

	// 写入 申请记录
	insertR, err := collPlayerMail.InsertOne(nil, &email)
	if err != nil {
		return errorCodeDef.Err_Failed, primitive.NilObjectID, err
	}
	return 0, insertR.InsertedID.(primitive.ObjectID), err
}

// 获取个人邮件
func GetMyEmail(emailID primitive.ObjectID, email *collPlayer.DBPlayerEmail) error {
	collPlayerMail := mongoDBClient.Database(databaseName).Collection(collPlayer.CollPlayerEmail)

	single := collPlayerMail.FindOne(nil, bson.M{"_id": emailID})
	if single.Err() != nil {
		return single.Err()
	}
	raw, err := single.DecodeBytes()
	if err != nil {
		return err
	}
	email.ID = raw.Lookup("_id").ObjectID()
	email.CreateTime = raw.Lookup("create_time").Time()
	email.UserID = raw.Lookup("uid").AsInt64()
	email.Category = collPlayer.MailType(raw.Lookup("category").AsInt32())
	email.Status = collPlayer.MailStatus(raw.Lookup("status").AsInt32())
	logData := raw.Lookup("data")

	switch email.Category {
	case collPlayer.EmailInviteJoinClub:
		temp := &collPlayer.ItemEmailInviteToClub{}
		err = logData.Unmarshal(temp)
		if err != nil {
			return fmt.Errorf("unmarshal error,%d", email.Category)
		}
		email.Data = temp
		return nil
	case collPlayer.EmailInviteToMengZhu:
		temp := &collPlayer.ItemEmailInviteToMengZhu{}
		err = logData.Unmarshal(temp)
		if err != nil {
			return fmt.Errorf("unmarshal error,%d", email.Category)
		}
		email.Data = temp
	default:
		return fmt.Errorf("not find category")
	}

	return nil
}

func UpdateMyEmailStatus(emailID primitive.ObjectID, isDel bool) error {
	collPlayerMail := mongoDBClient.Database(databaseName).Collection(collPlayer.CollPlayerEmail)

	if isDel == true {
		collPlayerMail.DeleteOne(nil, bson.M{"_id": emailID})
	} else {
		collPlayerMail.UpdateOne(nil, bson.M{"_id": emailID}, bson.M{"$set": bson.M{"status": collPlayer.MailStatusRead}})
	}

	return nil
}
