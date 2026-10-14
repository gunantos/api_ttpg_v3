package models

import "time"

type TtpgUser struct {
	ID             uint       `gorm:"primaryKey;column:id;type:int(11);autoIncrement" json:"id"`
	AccessID       int        `gorm:"column:access_id;type:int(11);not null;default:2" json:"access_id"`
	PoID           *string    `gorm:"column:po_id;size:50" json:"po_id"`
	Username       string     `gorm:"column:username;size:150;not null;uniqueIndex" json:"username"`
	Password       string     `gorm:"column:password;size:250;not null" json:"-"`
	PublicKey      string     `gorm:"column:public_key;size:150;not null" json:"public_key"`
	SecretKey      string     `gorm:"column:secret_key;size:150;not null" json:"-"`
	DevelopmentKey string     `gorm:"column:development_key;size:150;not null" json:"development_key"`
	Level          *int       `gorm:"column:level;type:int(11)" json:"level"`
	AllAccess      string     `gorm:"column:all_access;size:1;not null;default:n" json:"all_access"`
	Allowed        *string    `gorm:"column:allowed;size:250" json:"allowed"`
	IsBlock        int        `gorm:"column:is_block;type:int(4);not null;default:0" json:"is_block"`
	IsApproved     *int       `gorm:"column:is_approved;type:int(11)" json:"is_approved"`
	CreateDate     time.Time  `gorm:"column:create_date" json:"create_date"`
	UpdateTime     *time.Time `gorm:"column:update_date" json:"update_date"`
}

func (TtpgUser) TableName() string { return "user" }

func (u *TtpgUser) AllowedOrEmpty() string {
	if u.Allowed == nil {
		return ""
	}
	return *u.Allowed
}

func (u *TtpgUser) PoIDOrEmpty() string {
	if u.PoID == nil {
		return ""
	}
	return *u.PoID
}

type TtpgAccess struct {
	ID         int       `gorm:"primaryKey;column:id;type:int(11);autoIncrement" json:"id"`
	UserID     int       `gorm:"column:user_id;type:int(11);not null;uniqueIndex" json:"user_id"`
	IDAPI      string    `gorm:"column:id_api;size:250;not null" json:"id_api"`
	Block      string    `gorm:"column:block;size:1;not null;default:n" json:"block"`
	Allow      string    `gorm:"column:allow;size:1;not null;default:n" json:"allow"`
	CreateTime time.Time `gorm:"column:create_time" json:"create_time"`
	UpdateTime time.Time `gorm:"column:update_time" json:"update_time"`
}

func (TtpgAccess) TableName() string { return "access" }

type TtpgApiURL struct {
	ID            int       `gorm:"primaryKey;column:id;type:int(11);autoIncrement" json:"id"`
	Authorization string    `gorm:"column:authorization;size:20;not null;default:api_key" json:"authorization"`
	Ajax          string    `gorm:"column:ajax;size:1;not null;default:0" json:"ajax"`
	Controller    string    `gorm:"column:controller;size:250;not null" json:"controller"`
	Method        string    `gorm:"column:method;size:10;not null;default:get" json:"method"`
	IgnoreLimits  int       `gorm:"column:ignore_limits;type:int(11);not null;default:0" json:"ignore_limits"`
	Help          *string   `gorm:"column:help" json:"help"`
	CreateDate    time.Time `gorm:"column:create_date" json:"create_date"`
	UpdateTime    time.Time `gorm:"column:update_date" json:"update_date"`
}

func (TtpgApiURL) TableName() string { return "api_url" }

type TtpgLog struct {
	ID           int       `gorm:"primaryKey;column:id;type:int(11);autoIncrement" json:"id"`
	URI          string    `gorm:"column:uri;size:250;not null" json:"uri"`
	Method       string    `gorm:"column:method;size:6;not null" json:"method"`
	Params       *string   `gorm:"column:params" json:"params"`
	APIKey       string    `gorm:"column:api_key;size:40;not null" json:"api_key"`
	IPAddress    string    `gorm:"column:ip_address;size:45;not null" json:"ip_address"`
	Time         int       `gorm:"column:time;type:int(11);not null" json:"time"`
	Rtime        *float64  `gorm:"column:rtime" json:"rtime"`
	Authorized   string    `gorm:"column:authorized;size:1;not null" json:"authorized"`
	ResponseCode int       `gorm:"column:response_code;type:smallint(3);not null" json:"response_code"`
	IDUser       int       `gorm:"column:id_user;type:int(11);not null" json:"id_user"`
	PoID         *string   `gorm:"column:po_id;size:50" json:"po_id"`
	AccessBy     string    `gorm:"column:access_by;size:20;not null" json:"access_by"`
	Date         time.Time `gorm:"column:date" json:"date"`
}

func (TtpgLog) TableName() string { return "logs" }
