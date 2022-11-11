package db

import (
	"context"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
	collClub "vvService/dbCollectionDefine/club"
)

func CheckMinScore(clubID int32) (bool, error) {

	coll := mongoDBClient.Database(databaseName).Collection(collClub.CollClubInfo)
	ctx := context.Background()

	count, err := coll.CountDocuments(ctx, bson.M{"club_id": clubID, "$expr": bson.M{"$gt": bson.A{"$score_count", "$bao_di"}}})
	if err != nil {
		return false, err
	}
	if count > 0 {
		return true, err
	}
	return false, err
}

func CheckClubTotalScore(clubIDArr []int32) (int64, error) {
	coll := mongoDBClient.Database(databaseName).Collection(collClub.CollClubInfo)
	ctx := context.Background()

	return coll.CountDocuments(ctx, bson.M{"club_id": bson.M{"$in": clubIDArr}, "$expr": bson.M{"$lte": bson.A{"$score_count", "$bao_di"}}})

	//opt := options.FindOptions{}
	//opt.SetProjection(bson.M{"club_id": 1, "score_count": 1, "bao_di": 1})
	//cur, err := coll.Find(ctx, bson.M{"club_id": bson.M{"$in": clubIDArr}}, &opt)
	//if err != nil {
	//	return 0, err
	//}
	//defer cur.Close(ctx)
	//
	//var club collClub.DBClubData
	//for cur.Next(ctx) {
	//	err = cur.Decode(&club)
	//	if err != nil {
	//		return 0, err
	//	}
	//
	//	if club.ClubScoreCount <= club.BaoDiScore {
	//		return club.ClubID, nil
	//	}
	//}
}

func GetClubSealStatus(clubIDArr []int32, rv *map[int32]bool) error {
	coll := mongoDBClient.Database(databaseName).Collection(collClub.CollClubInfo)
	ctx := context.Background()

	opt := options.FindOptions{}
	opt.SetProjection(bson.M{"seal": 1, "club_id": 1, "_id": 0})

	cur, err := coll.Find(ctx, bson.M{"club_id": bson.M{"$in": clubIDArr}}, &opt)
	if err != nil {
		return nil
	}
	defer cur.Close(ctx)

	type ClubSealStatusArr struct {
		ClubID int32 `bson:"club_id"`
		Value  bool  `bson:"seal"`
	}
	t := []ClubSealStatusArr{}
	err = cur.All(ctx, &t)
	if err != nil {
		return nil
	}

	for _, v := range t {
		(*rv)[v.ClubID] = v.Value
	}
	return nil
}

func PutActivityAwardList(v *collClub.DBActivityAwardList) error {
	collAcAward := mongoDBClient.Database(databaseName).Collection(collClub.CollActivityAwardList)

	_, err := collAcAward.InsertOne(nil, v)
	return err
}

func GetActivityAwardList(clubID int32, awardArr *[]collClub.DBActivityAwardList) error {
	collAcAward := mongoDBClient.Database(databaseName).Collection(collClub.CollActivityAwardList)

	ctx := context.Background()
	cur, err := collAcAward.Find(ctx, bson.M{"clubID": clubID})
	if err != nil {
		return err
	}
	defer cur.Close(ctx)

	return cur.All(ctx, awardArr)
}

func GetMyActivityAward(uid int64, clubID int32, category int32, award *collClub.DBActivityAwardList) error {
	collAcAward := mongoDBClient.Database(databaseName).Collection(collClub.CollActivityAwardList)

	ctx := context.Background()
	cur, err := collAcAward.Find(ctx, bson.M{"clubID": clubID, "uid": uid, "category": category})
	if err != nil {
		return err
	}
	defer cur.Close(ctx)

	cur.Next(ctx)

	return cur.Decode(award)
}

func DelMengZHuActivityAward(clubID int32) error {
	collAcAward := mongoDBClient.Database(databaseName).Collection(collClub.CollActivityAwardList)

	_, err := collAcAward.DeleteMany(nil, bson.M{"clubID": clubID})
	return err
}
