package models

import (
	"database/sql"
	"errors"
	"path"
	cfg "seckill/web/configs"
	"strings"
	"time"

	"go.etcd.io/etcd/clientv3"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type BaseModel struct {
	ID        uint         `json:"id" gorm:"primarykey"`
	CreatedAt time.Time    `json:"create_time" gorm:"type:TIMESTAMP;default:CURRENT_TIMESTAMP;<-:false"`
	UpdatedAt time.Time    `json:"updated_time" gorm:"type:TIMESTAMP;default:CURRENT_TIMESTAMP"`
	DeletedAt sql.NullTime `json:"-" gorm:"index"`
}

type DBConfigs = cfg.DBConfigs

var (
	orm             *gorm.DB
	err             error
	logs            = cfg.Logs
	EtcdCfgs        = cfg.EtcdCfgs
	etcdClient      *clientv3.Client
	etcdKey         = path.Join(cfg.EtcdSecKeyPrefix, cfg.EtcdSecProductKey)
	etcdConnTimeout = EtcdCfgs.DialTimeout
)

const (
	DateTimeFormatStr = cfg.DateTimeFormatStr
	DateFormatStr     = cfg.DateFormatStr
	TimeFormatStr     = cfg.TimeFormatStr
)

func Init() {
	if err := initDB(); err != nil {
		panic("init orm failed!")
	}
	logs.Debug("init orm succ!")
	if err := initEtcd(); err != nil {
		panic("init etcd failed!")
	}
	logs.Debug("init etcd succ!")
}

func initEtcd() (err error) {
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   EtcdCfgs.Endpoints,
		DialTimeout: EtcdCfgs.DialTimeout,
	})
	if err != nil {
		logs.Errorf("init etcd failed! err:%v", err)
		return
	}
	etcdClient = cli
	return
}

func initDB() (err error) {
	orm, err = getDB(cfg.DBInfos)

	if err != nil {
		logs.Errorf("init orm failed! err:%v", err)
		return
	}

	logs.Debug("conn database succ!")
	return
}

// 可单独到一个文件中
func getDB(cfg *DBConfigs) (db *gorm.DB, err error) {
	switch cfg.Driver {
	case "mysql":
		db, err = MysqlConn(cfg)
	default:
		err = errors.New("Please select a database!")
	}
	return
}

func MysqlConn(dbCfg *DBConfigs) (db *gorm.DB, err error) {
	// dsn := "tq:Tq123456@!@tcp(192.168.1.188:3306)/know?charset=utf8&parseTime=True&loc=Local"
	defaultOpts := []string{"charset=utf8mb4", "parseTime=True", "loc=Local"}
	user := dbCfg.User + ":" + dbCfg.Password
	addr := "tcp(" + dbCfg.Host + ":" + dbCfg.Port + ")/" + dbCfg.Database
	opts := strings.Join(defaultOpts, "&")
	dsn := user + "@" + addr + "?" + opts
	db, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
	return
}
