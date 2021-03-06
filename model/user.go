package model

import (
	"database/sql"
	"errors"
	"time"

	"github.com/didi/gendry/builder"
	"github.com/didi/gendry/scanner"
)

/*
	This code is generated by gendry
*/

// User is a mapping object for user table in mysql
type User struct {
	ID        int       `json:"id"`
	Username  string    `json:"username" validate:"required,min=4,max=20" label:"用户名"`
	Password  string    `json:"password" validate:"required,min=6,max=30" label:"密码"`
	Img       string    `json:"img"`
	Birth     time.Time `json:"birth"`
	Gender    int8      `json:"gender"`
	Bio       string    `json:"bio"`
	About     string    `json:"about"`
	Coins     int       `json:"coins"`
	CreatedAt time.Time `json:"created_at"`
	UpdateAt  time.Time `json:"update_at"`
}

type EditUser struct {
	Username string `json:"username" binding:"required"`
	Img      string `json:"img"`
	Birth    int64  `json:"birth"`
	Gender   int    `json:"gender"`
	Bio      string `json:"bio"`
	About    string `json:"about"`
}

type AdviserInfoForUser struct {
	AdviserName string      `json:"adviser_name"`
	Img         string      `json:"img"`
	Bio         string      `json:"bio"`
	Services    interface{} `json:"services"`
	About       string      `json:"about"`
}

type AdviserInfo struct {
	Adviser  *Adviser   `json:"adviser"`
	Services []*Service `json:"services"`
	Reviews  []*Comment `json:"reviews"`
}

type AdviserList struct {
	AdviserName string `json:"adviser_name"`
	Img         string `json:"img"`
	Bio         string `json:"bio"`
}

//GetOne gets one record from table user by condition "where"
func GetOneUser(db *sql.DB, where map[string]interface{}) (*User, error) {
	if nil == db {
		return nil, errors.New("sql.DB object couldn't be nil")
	}
	cond, vals, err := builder.BuildSelect("user", where, nil)
	if nil != err {
		return nil, err
	}
	row, err := db.Query(cond, vals...)
	if nil != err || nil == row {
		return nil, err
	}
	defer row.Close()
	var res *User
	err = scanner.Scan(row, &res)
	return res, err
}

//GetMulti gets multiple records from table user by condition "where"
func GetMultiUser(db *sql.DB, where map[string]interface{}) ([]*User, error) {
	if nil == db {
		return nil, errors.New("sql.DB object couldn't be nil")
	}
	cond, vals, err := builder.BuildSelect("user", where, nil)
	if nil != err {
		return nil, err
	}
	row, err := db.Query(cond, vals...)
	if nil != err || nil == row {
		return nil, err
	}
	defer row.Close()
	var res []*User
	err = scanner.Scan(row, &res)
	return res, err
}

//Insert inserts an array of data into table user
func InsertUser(db *sql.DB, data []map[string]interface{}) (int64, error) {
	if nil == db {
		return 0, errors.New("sql.DB object couldn't be nil")
	}
	cond, vals, err := builder.BuildInsert("user", data)
	if nil != err {
		return 0, err
	}
	result, err := db.Exec(cond, vals...)
	if nil != err || nil == result {
		return 0, err
	}
	return result.LastInsertId()
}

//Update updates the table user
func UpdateUser(db *sql.DB, where, data map[string]interface{}) (int64, error) {
	if nil == db {
		return 0, errors.New("sql.DB object couldn't be nil")
	}
	cond, vals, err := builder.BuildUpdate("user", where, data)
	if nil != err {
		return 0, err
	}
	result, err := db.Exec(cond, vals...)
	if nil != err {
		return 0, err
	}
	return result.RowsAffected()
}

// Delete deletes matched records in user
func DeleteUser(db *sql.DB, where, data map[string]interface{}) (int64, error) {
	if nil == db {
		return 0, errors.New("sql.DB object couldn't be nil")
	}
	cond, vals, err := builder.BuildDelete("user", where)
	if nil != err {
		return 0, err
	}
	result, err := db.Exec(cond, vals...)
	if nil != err {
		return 0, err
	}
	return result.RowsAffected()
}
