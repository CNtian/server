package collIndex

import (
	"github.com/golang/glog"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"time"
	commonDB "vvService/commonPackge/db"
)

var (
	collIndexMap = make(map[string]time.Time) // key:索引名称  v:记录时间
)

func FindAndCreate_game_record(c *mongo.Collection, collName string) {
	if _, ok := collIndexMap[collName]; ok == true {
		return
	}

	ok, err := commonDB.GetCurIndexes(c, "index_"+collName)
	if err != nil {
		glog.Warning("GetCurIndexes() err:=", err.Error(), ",data:=", collName)
	}
	if ok == true {
		collIndexMap[collName] = time.Now()
		return
	}

	indexName := "index_" + collName
	indexModel := mongo.IndexModel{
		Keys:    bson.D{{"players.clubID", 1}, {"players.uid", 1}, {"create_time", -1}},
		Options: &options.IndexOptions{Name: &indexName}}

	err = commonDB.CreateIndex(c, []mongo.IndexModel{indexModel})
	if err != nil {
		glog.Warning("CreateIndex() err:=", err.Error(), ",data:=", collName)
	} else {
		collIndexMap[collName] = time.Now()
	}
}

func FindAndCreate_club_score(c *mongo.Collection, collName string) {
	if _, ok := collIndexMap[collName]; ok == true {
		return
	}

	ok, err := commonDB.GetCurIndexes(c, "index_"+collName)
	if err != nil {
		glog.Warning("GetCurIndexes() err:=", err.Error(), ",data:=", collName)
	}
	if ok == true {
		collIndexMap[collName] = time.Now()
		return
	}

	if len(collIndexMap) > 1000 {
		tt := time.Now()
		for k, v_ := range collIndexMap {
			if tt.Sub(v_) > time.Hour*24*7 {
				delete(collIndexMap, k)
			}
		}
	}

	indexName := "index_" + collName
	indexModel := mongo.IndexModel{
		Keys:    bson.D{{"clubID", 1}, {"playerID", 1}, {"create_time", -1}},
		Options: &options.IndexOptions{Name: &indexName}}

	err = commonDB.CreateIndex(c, []mongo.IndexModel{indexModel})
	if err != nil {
		glog.Warning("CreateIndex() err:=", err.Error(), ",data:=", collName)
	} else {
		collIndexMap[collName] = time.Now()
	}
}

func FindAndCreate_tong_zhuo(c *mongo.Collection, collName string) {
	if _, ok := collIndexMap[collName]; ok == true {
		return
	}

	ok, err := commonDB.GetCurIndexes(c, "index_"+collName)
	if err != nil {
		glog.Warning("GetCurIndexes() err:=", err.Error(), ",data:=", collName)
	}
	if ok == true {
		collIndexMap[collName] = time.Now()
		return
	}

	if len(collIndexMap) > 1000 {
		tt := time.Now()
		for k, v_ := range collIndexMap {
			if tt.Sub(v_) > time.Hour*24*7 {
				delete(collIndexMap, k)
			}
		}
	}

	indexName := "index_" + collName
	indexModel := mongo.IndexModel{
		Keys:    bson.D{{"uid", 1}, {"b_id", 1}},
		Options: &options.IndexOptions{Name: &indexName}}

	err = commonDB.CreateIndex(c, []mongo.IndexModel{indexModel})
	if err != nil {
		glog.Warning("CreateIndex() err:=", err.Error(), ",data:=", collName)
	} else {
		collIndexMap[collName] = time.Now()
	}
}
