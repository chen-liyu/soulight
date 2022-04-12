package api

import (
	"fmt"
	"soulight/model"
	"soulight/response"
	"soulight/utils"
	"soulight/utils/errmsg"
	"strconv"
	"time"

	"github.com/didi/gendry/builder"
	"github.com/didi/gendry/scanner"
	"github.com/gin-gonic/gin"
	"github.com/robfig/cron/v3"
)

//创建订单
func OrderCreate(c *gin.Context) {
	var order model.Order
	//1.绑定参数
	if err := c.ShouldBindJSON(&order); err != nil {
		response.SendResponse(c, errmsg.INVALID_PARAMS)
		return
	}
	//2.检查用户是否有足够的金币
	u, _ := c.Get("user")
	user, _ := u.(*model.User)
	if user.Coins < order.Cost {
		response.SendResponse(c, errmsg.ERROR_COINS_NOT_ENOUGH)
		return
	}
	//3.初始化订单uuid、状态和创建时间等
	order.UserID = user.ID
	order.OrderID = utils.GenerateUUID()
	order.Status = 0
	order.OrderTime = time.Now()
	order.Reply = ""
	order.Rate = 0
	//4.将订单写入数据库并修改用户金币
	order_map := map[string]interface{}{
		"order_id":     order.OrderID,
		"user_id":      order.UserID,
		"adviser_id":   order.AdviserID,
		"situation":    order.Situation,
		"question":     order.Question,
		"reply":        order.Reply,
		"cost":         order.Cost,
		"status":       order.Status,
		"service_type": order.ServiceType,
		"order_time":   order.OrderTime,
		"rate":         order.Rate,
	}
	var data []map[string]interface{}
	data = append(data, order_map)
	conn, err := model.Db.Begin()
	if err != nil {
		response.SendResponse(c, errmsg.ERROR_DATABASE)
		return
	}
	cond, vals, _ := builder.BuildInsert("orders", data)
	if _, err := conn.Exec(cond, vals...); err != nil {
		conn.Rollback()
		response.SendResponse(c, errmsg.ERROR_DATABASE)
		return
	}
	if _, err := conn.Exec("update user set coins=coins-? where id=?", order.Cost, user.ID); err != nil {
		conn.Rollback()
		response.SendResponse(c, errmsg.ERROR_DATABASE)
		return
	}
	conn.Commit()
	//5.开启定时任务，若24小时未回复，则订单过期，金币归还用户
	exp_time := order.OrderTime.Add(24 * time.Hour)
	spec := utils.GetCronSpec(exp_time)
	var entry_id cron.EntryID
	order_id := order.OrderID
	user_id := user.ID
	entry_id, _ = model.Cron.AddFunc(spec, func() {
		var err error
		var o *model.Order
		if o, err = model.GetOneOrder(model.Db, map[string]interface{}{"order_id": order_id}); err != nil {
			return
		}
		if o.Status != 1 {
			//开启事务，修改订单状态为过期，并归还用户金币
			conn, _ := model.Db.Begin()
			if _, err := conn.Exec("update orders set status=2 where order_id=?", order_id); err != nil {
				fmt.Println(err)
				conn.Rollback()
				return
			}
			if _, err := conn.Exec("update user set coins=coins+? where id=?", o.Cost, user_id); err != nil {
				fmt.Println(err)
				conn.Rollback()
				return
			}
			conn.Commit()
		}
		model.Cron.Remove(entry_id)
	})
	response.SendResponse(c, errmsg.SUCCSE, order)
}

//订单列表
func OrderList(c *gin.Context) {
	//1.绑定参数
	service_type, _ := strconv.Atoi(c.DefaultQuery("service_type", "-1"))
	adviser_id := c.GetInt("id")
	//2.查询订单
	var cond string
	var vals []interface{}
	var err error
	if service_type != -1 {
		cond, vals, _ = builder.NamedQuery(`Select u.img ,u.username ,o.order_id,o.order_time,o.question,o.status,o.service_type 
		from orders as o left join user as u on o.user_id=u.id 
		where o.service_type={{service_type}} and o.adviser_id={{adviser_id}} `,
			map[string]interface{}{
				"service_type": service_type,
				"adviser_id":   adviser_id,
			})
	} else {
		cond, vals, _ = builder.NamedQuery(`Select u.img,u.username,o.order_id,o.order_time,o.question,o.status,o.service_type  
		from orders as o left join user as u on o.user_id=u.id 
		where o.adviser_id={{adviser_id}} `,
			map[string]interface{}{
				"adviser_id": adviser_id,
			})
	}
	row, err := model.Db.Query(cond, vals...)
	if nil != err || nil == row {
		response.SendResponse(c, errmsg.ERROR_DATABASE)
		return
	}
	defer row.Close()
	var res []*model.OrderList
	if err = scanner.Scan(row, &res); err != nil {
		response.SendResponse(c, errmsg.ERROR)
		return
	}
	response.SendResponse(c, errmsg.SUCCSE, res)
}

//订单详情
func OrderInfo(c *gin.Context) {
	//1.绑定参数
	order_id, _ := c.GetQuery("order_id")
	//2.查询订单
	row, err := model.Db.Query(`select o.order_id,o.status,o.service_type,o.order_time,o.delivery_time,u.username,u.birth,u.gender,o.situation,o.question
	from orders as o left join user as u on o.user_id=u.id 
	where o.order_id=? `, order_id)
	if nil != err || nil == row {
		response.SendResponse(c, errmsg.ERROR_DATABASE)
		return
	}
	defer row.Close()
	var res *model.OrderInfo
	if err = scanner.Scan(row, &res); err != nil {
		response.SendResponse(c, errmsg.ERROR)
		return
	}
	response.SendResponse(c, errmsg.SUCCSE, res)

}

//回复订单
func OrderReply(c *gin.Context) {
	var reply model.OrderReply
	//1.绑定参数
	adviser_id := c.GetInt("id")
	if err := c.ShouldBind(&reply); err != nil {
		response.SendResponse(c, errmsg.INVALID_PARAMS)
		return
	}
	//2.查询订单
	o, _ := model.GetOneOrder(model.Db, map[string]interface{}{"order_id": reply.OrderID})
	if o == nil {
		response.SendResponse(c, errmsg.ERROR_ORDER_NOT_EXIST)
		return
	} else {
		if o.Status == 2 {
			response.SendResponse(c, errmsg.ERROR_ORDER_TIMEOUT)
			return
		}
	}
	//3.更新订单状态,回复内容及完成时间并给顾问增加金币
	o.Status = 1
	o.Reply = reply.Reply
	o.DeliveryTime = time.Now()
	conn, _ := model.Db.Begin()
	if _, err := conn.Exec("update orders set status=1,reply=?,delivery_time=? where order_id=?",
		reply.Reply, o.DeliveryTime, reply.OrderID); err != nil {
		conn.Rollback()
		response.SendResponse(c, errmsg.ERROR_DATABASE)
		return
	}
	if _, err := conn.Exec("update adviser set coins=coins+? where id=?", o.Cost, adviser_id); err != nil {
		conn.Rollback()
		response.SendResponse(c, errmsg.ERROR_DATABASE)
		return
	}
	conn.Commit()
	response.SendResponse(c, errmsg.SUCCSE, o)
}

//订单加急
func OrderUrgent(c *gin.Context) {
	//1.绑定参数
	user_id := c.GetInt("id")
	order_id, _ := c.GetQuery("order_id")
	//2.查询订单
	order, _ := model.GetOneOrder(model.Db, map[string]interface{}{"order_id": order_id})
	if order == nil {
		response.SendResponse(c, errmsg.ERROR_ORDER_NOT_EXIST)
		return
	} else {
		if order.Status != 0 {
			response.SendResponse(c, errmsg.ERROR_ORDER_STATUS_WRONG)
			return
		}
	}
	extra_cost := order.Cost / 2
	//3.检查用户是否有足够的金币
	u, _ := c.Get("user")
	user, _ := u.(*model.User)
	if user.Coins < extra_cost {
		response.SendResponse(c, errmsg.ERROR_COINS_NOT_ENOUGH)
		return
	}
	//4.更新订单状态,加急时间并给顾问增加金币
	order.Status = 3
	order.Cost += extra_cost
	conn, _ := model.Db.Begin()
	if _, err := conn.Exec("update orders set status=3,cost=cost+? where order_id=?", extra_cost, order_id); err != nil {
		fmt.Println(err)
		conn.Rollback()
		response.SendResponse(c, errmsg.ERROR_DATABASE)
		return
	}
	if _, err := conn.Exec("update user set coins=coins-? where id=?", extra_cost, user_id); err != nil {
		fmt.Println(err)
		conn.Rollback()
		response.SendResponse(c, errmsg.ERROR_DATABASE)
		return
	}
	conn.Commit()
	//5.开启定时任务，若1小时未回复，则订单变为普通状态，金币归还用户
	exp_time := time.Now().Add(30 * time.Second)
	spec := utils.GetCronSpec(exp_time)
	var entry_id cron.EntryID
	entry_id, _ = model.Cron.AddFunc(spec, func() {
		var err error
		var o *model.Order
		if o, err = model.GetOneOrder(model.Db, map[string]interface{}{"order_id": order_id}); err != nil {
			return
		}
		if o.Status != 1 {
			//开启事务，修改订单状态及金额，并归还用户金币
			conn, _ := model.Db.Begin()
			if _, err := conn.Exec("update orders set status=0,cost=cost-? where order_id=?", extra_cost, order_id); err != nil {
				fmt.Println(err)
				conn.Rollback()
				return
			}
			if _, err := conn.Exec("update user set coins=coins+? where id=?", extra_cost, user_id); err != nil {
				fmt.Println(err)
				conn.Rollback()
				return
			}
			conn.Commit()
		}
		model.Cron.Remove(entry_id)
	})
	response.SendResponse(c, errmsg.SUCCSE, order)
}
