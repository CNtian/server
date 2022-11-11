package db

import (
	"context"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"vvService/commonPackge/errorCodeDef"
	collClub "vvService/dbCollectionDefine/club"
)

func PutPlayerToMutexGroup(clubID int32, groupID primitive.ObjectID, player []int64) (int32, error) {
	coll := mongoDBClient.Database(databaseName).Collection(collClub.CollClubInfo)
	ctx := context.Background()

	var (
		updateRes *mongo.UpdateResult
		err       error
	)
	if groupID.IsZero() == true {
		updateRes, err = coll.UpdateOne(ctx,
			bson.M{"club_id": clubID},
			bson.M{"$push": bson.M{"member_mutex": collClub.DBMemberMutexGroup{ID: primitive.NewObjectID(), Player: player}}})
	} else {
		if len(player) == 0 {
			updateRes, err = coll.UpdateOne(ctx,
				bson.M{"club_id": clubID},
				bson.M{"$pull": bson.M{"member_mutex": bson.M{"id": groupID}}})
		} else {
			updateRes, err = coll.UpdateOne(ctx,
				bson.M{"club_id": clubID, "member_mutex.id": groupID},
				bson.M{"$set": bson.M{"member_mutex.$": collClub.DBMemberMutexGroup{ID: groupID, Player: player}}})
		}
	}
	if err != nil {
		return 0, err
	}
	if updateRes != nil && updateRes.MatchedCount < 1 {
		return errorCodeDef.Err_Not_Find_Item, nil
	}
	if updateRes != nil && updateRes.ModifiedCount < 1 {
		return errorCodeDef.ErrClubOperationInvalid, nil
	}

	return 0, nil
}

func DeletePlayerToMutexGroup(clubID int32, groupID primitive.ObjectID) (int32, error) {
	coll := mongoDBClient.Database(databaseName).Collection(collClub.CollClubInfo)
	ctx := context.Background()

	updateRes, err := coll.UpdateOne(ctx,
		bson.M{"club_id": clubID},
		bson.M{"$pull": bson.M{"member_mutex": bson.M{"id": groupID}}})

	if err != nil {
		return 0, err
	}
	if updateRes.MatchedCount < 1 {
		return errorCodeDef.Err_Not_Find_Item, nil
	}
	if updateRes.ModifiedCount < 1 {
		return errorCodeDef.ErrClubOperationInvalid, nil
	}

	return 0, err
}
