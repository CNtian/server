package db

import (
	"fmt"
	"github.com/golang/glog"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"vvService/appDB/collIndex"
	"vvService/dbCollectionDefine"
)

func PutTongZhuItem(mzClubID int32, date int, uidArr []int64, rID primitive.ObjectID) error {

	collName := fmt.Sprintf("%s_%d_%d", dbCollectionDefine.CollTongZhuo, mzClubID, date)

	coll_ := MongoClient.Database(CurDBName).Collection(collName)
	collIndex.FindAndCreate_tong_zhuo(coll_, collName)

	//writeArr := make([]mongo.WriteModel, 0, 10)
	var (
		rsp *mongo.UpdateResult
		err error
	)
	for i := 0; i < len(uidArr); i++ {
		for j := i + 1; j < len(uidArr); j++ {

			opt := options.UpdateOptions{}
			opt.SetUpsert(true)
			rsp, err = coll_.UpdateOne(nil,
				bson.M{"$or": bson.A{bson.M{"uid": uidArr[i], "b_id": uidArr[j]}, bson.M{"uid": uidArr[j], "b_id": uidArr[i]}}},
				bson.M{"$push": bson.M{"r_id": rID}}, &opt)
			if err != nil {
				glog.Warning(err.Error())
			} else {
				if rsp.UpsertedID != nil {
					_, err = coll_.UpdateOne(nil, bson.M{"_id": rsp.UpsertedID},
						bson.M{"$set": bson.M{"uid": uidArr[i], "b_id": uidArr[j]}})
					if err != nil {
						glog.Warning(err.Error())
					}
				}
			}
		}
	}
	return nil
}

//dbWrite := mongo.NewUpdateOneModel()
//dbWrite.SetUpsert(true)
//dbWrite.SetFilter(bson.M{"$or": bson.A{bson.M{"uid": uidArr[i], "b_id": uidArr[j]}, bson.M{"uid": uidArr[j], "b_id": uidArr[i]}}})
//dbWrite.SetUpdate(bson.M{"$push": bson.M{"r_id": rID}})
//writeArr = append(writeArr, dbWrite)
//_, err := coll_.BulkWrite(nil, writeArr)

//func Test(date int, aID, bID int64, rID primitive.ObjectID) error {
//
//	collName := fmt.Sprintf("%s%d", dbCollectionDefine.CollTongZhuo, date)
//
//	coll_ := MongoClient.Database(CurDBName).Collection(collName)
//	collIndex.FindAndCreate_tong_zhuo(coll_, collName)
//
//	glog.Warning(rsp)
//
//	return err
//}
