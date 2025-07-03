package qqwife

import (
	"errors"
	"math/rand"
	"sort"
	"strconv"
	"strings"

	"github.com/FloatTech/floatbox/math"
	"github.com/FloatTech/imgfactory"
	sql "github.com/FloatTech/sqlite"
	control "github.com/FloatTech/zbputils/control"
	"github.com/FloatTech/zbputils/ctxext"
	zero "github.com/wdvxdr1123/ZeroBot"
	"github.com/wdvxdr1123/ZeroBot/message"

	// 画图
	"github.com/FloatTech/floatbox/file"
	"github.com/FloatTech/gg"
	"github.com/FloatTech/zbputils/img/text"

	// 货币系统
	"github.com/FloatTech/AnimeAPI/wallet"
)

// 好感度系统
type favorability struct {
	Userinfo string // 记录用户
	Favor    int    // 好感度
}

func init() {
	// 好感度系统
	engine.OnRegex(`^查好感度\s*(\[CQ:at,qq=)?(\d+)`, zero.OnlyGroup, getdb).SetBlock(true).Limit(ctxext.LimitByUser).
		Handle(func(ctx *zero.Ctx) {
			fiancee, _ := strconv.ParseInt(ctx.State["regex_matched"].([]string)[2], 10, 64)

			uid := ctx.Event.UserID
			favor, err := 民政局.查好感度(uid, fiancee)
			if err != nil {
				ctx.SendChain(message.Text("[ERROR]:", err))
				return
			}
			// 输出结果
			ctx.SendChain(
				message.At(uid),
				message.Text("\n当前你们好感度为", favor),
			)
		})
	// 礼物系统
	engine.OnRegex(`^买礼物给\s?(\[CQ:at,(?:\S*,)?qq=(\d+)(?:,\S*)?\]|(\d+))`, getdb).SetBlock(true).Limit(ctxext.LimitByUser).
		Handle(func(ctx *zero.Ctx) {
			gid := ctx.Event.GroupID
			uid := ctx.Event.UserID
			fiancee := ctx.State["regex_matched"].([]string)
			gay, _ := strconv.ParseInt(fiancee[2]+fiancee[3], 10, 64)
			if gay == uid {
				ctx.Send(message.ReplyWithMessage(ctx.Event.MessageID, message.At(uid), message.Text("你想给自己买什么礼物呢?")))
				return
			}

			// 定义礼物名称数组
			type Gift struct {
				Name        string
				MeasureWord string
			}
			var gifts = []Gift{
				{"鲜花", "束"},
				{"巧克力", "盒"},
				{"女装", "件"},
				{"香水", "瓶"},
				{"手链", "条"},
				{"项链", "条"},
				{"手表", "块"},
				{"玩偶", "个"},
				{"手机壳", "个"},
				{"游戏机", "台"},
				{"耳机", "副"},
				{"围巾", "条"},
				{"书籍", "本"},
				{"水杯", "个"},
			}
			// 选择随机礼物
			giftIndex := rand.Intn(len(gifts))
			selectedGift := gifts[giftIndex]

			// 获取CD
			groupInfo, err := 民政局.查看设置(gid)
			if err != nil {
				ctx.SendChain(message.Text("[ERROR]:", err))
				return
			}
			ok, err := 民政局.判断CD(gid, uid, "买礼物", groupInfo.CDtime)
			if err != nil {
				ctx.SendChain(message.Text("[ERROR]:", err))
				return
			}
			if !ok {
				ctx.SendChain(message.Text("舔狗，今天你已经送过礼物了。"))
				return
			}
			// 获取好感度
			favor, err := 民政局.查好感度(uid, gay)
			if err != nil {
				ctx.SendChain(message.Text("[ERROR]:好感度库发生问题力\n", err))
				return
			}
			// 对接小熊饼干
			walletinfo := wallet.GetWalletOf(uid)
			if walletinfo < 5 {
				ctx.SendChain(message.Text("你钱包没钱啦！"))
				return
			}

			// 计算礼物价格的最小值和最大值
			minValue := 4
			maxValue := math.Min(walletinfo, 25) // 钱包里的钱和25中取最小值

			// 生成礼物价格随机数
			moneyToFavor := rand.Intn(int(maxValue-minValue+1)) + minValue

			newFavor := 1
			moodMax := 4 // 心情上限
			// 计算钱对应的好感值
			if favor > 90 {
				moodMax = 2
				newFavor = int(float64(moneyToFavor) * 0.5) //当前好感大于90，1/2概率降低好感，增加好感时会 x0.5倍
			} else if favor > 66 {
				moodMax = 3
				newFavor = int(float64(moneyToFavor) * 0.6) //当前好感大于66，1/3概率降低好感，增加好感时会 x0.6倍
			} else if favor > 36 {
				moodMax = 4
				newFavor = moneyToFavor //当前好感大于36，1/4概率降低好感，正常增加好感（钱和好感度1:1）
			} else {
				moodMax = 5
				newFavor = int(float64(moneyToFavor) * 1.33) //初始为1/5概率降低好感，增加好感时会x1.33倍
			}

			// 随机对方心情
			mood := rand.Intn(moodMax)
			if mood == 0 { // 心情为0则减少好感度
				if favor > 90 {
					newFavor = -newFavor * 2 //当前好感大于90，双倍扣除好感度
				} else if favor > 50 {
					newFavor = -newFavor //当前好感大于50，正常扣除好感度
				} else {
					newFavor = -newFavor / 2 //当前好感小于50，扣除的好感度减少一半
				}
			}
			// 记录结果
			err = wallet.InsertWalletOf(uid, -moneyToFavor)
			if err != nil {
				ctx.SendChain(message.Text("[ERROR]:钱包坏掉力:\n", err))
				return
			}
			lastfavor, err := 民政局.更新好感度(uid, gay, newFavor)
			if err != nil {
				ctx.SendChain(message.Text("[ERROR]:好感度数据库发生问题力\n", err))
				return
			}
			// 写入CD
			err = 民政局.记录CD(gid, uid, "买礼物")
			if err != nil {
				ctx.SendChain(message.At(uid), message.Text("[ERROR]:你的技能CD记录失败\n", err))
			}
			// 输出结果
			if mood == 0 {
				ctx.SendChain(message.Text("你花了", moneyToFavor, wallet.GetWalletName(), "买了1", selectedGift.MeasureWord, selectedGift.Name, "送给了ta,ta很不喜欢,你们的好感度降低至", lastfavor, " (", newFavor, ")"))
			} else {
				ctx.SendChain(message.Text("你花了", moneyToFavor, wallet.GetWalletName(), "买了1", selectedGift.MeasureWord, selectedGift.Name, "送给了ta,ta很喜欢,你们的好感度升至", lastfavor, " (+", newFavor, ")"))
			}
		})
	engine.OnFullMatch("好感度列表", zero.OnlyGroup, getdb).SetBlock(true).Limit(ctxext.LimitByUser).
		Handle(func(ctx *zero.Ctx) {
			uid := ctx.Event.UserID
			fianceeInfo, err := 民政局.getGroupFavorability(uid)
			if err != nil {
				ctx.SendChain(message.Text("[ERROR]:ERROR: ", err))
				return
			}
			/***********设置图片的大小和底色***********/
			number := len(fianceeInfo)
			if number > 10 {
				number = 10
			}
			fontSize := 50.0
			canvas := gg.NewContext(1150, int(170+(50+70)*float64(number)))
			canvas.SetRGB(1, 1, 1) // 白色
			canvas.Clear()
			/***********下载字体***********/
			data, err := file.GetLazyData(text.BoldFontFile, control.Md5File, true)
			if err != nil {
				ctx.SendChain(message.Text("[ERROR]:ERROR: ", err))
			}
			/***********设置字体颜色为黑色***********/
			canvas.SetRGB(0, 0, 0)
			/***********设置字体大小,并获取字体高度用来定位***********/
			if err = canvas.ParseFontFace(data, fontSize*2); err != nil {
				ctx.SendChain(message.Text("[ERROR]:ERROR: ", err))
				return
			}
			sl, h := canvas.MeasureString("你的好感度排行列表")
			/***********绘制标题***********/
			canvas.DrawString("你的好感度排行列表", (1100-sl)/2, 100) // 放置在中间位置
			canvas.DrawString("————————————————————", 0, 160)
			/***********设置字体大小,并获取字体高度用来定位***********/
			if err = canvas.ParseFontFace(data, fontSize); err != nil {
				ctx.SendChain(message.Text("[ERROR]:ERROR: ", err))
				return
			}
			i := 0
			for _, info := range fianceeInfo {
				if i > 9 {
					break
				}
				if info.Userinfo == "" {
					continue
				}
				fianceID, err := strconv.ParseInt(info.Userinfo, 10, 64)
				if err != nil {
					ctx.SendChain(message.Text("[ERROR]:ERROR: ", err))
					return
				}
				if fianceID == 0 {
					continue
				}
				userName := ctx.CardOrNickName(fianceID)
				canvas.SetRGB255(0, 0, 0)
				canvas.DrawString(userName+"("+info.Userinfo+")", 10, float64(180+(50+70)*i))
				canvas.DrawString(strconv.Itoa(info.Favor), 1020, float64(180+60+(50+70)*i))
				canvas.DrawRectangle(10, float64(180+60+(50+70)*i)-h/2, 1000, 50)
				canvas.SetRGB255(150, 150, 150)
				canvas.Fill()
				canvas.SetRGB255(0, 0, 0)
				canvas.DrawRectangle(10, float64(180+60+(50+70)*i)-h/2, float64(info.Favor)*10, 50)
				canvas.SetRGB255(231, 27, 100)
				canvas.Fill()
				i++
			}
			data, err = imgfactory.ToBytes(canvas.Image())
			if err != nil {
				ctx.SendChain(message.Text("[qqwife]ERROR: ", err))
				return
			}
			ctx.SendChain(message.ImageBytes(data))
		})

	engine.OnFullMatch("好感度数据整理", zero.SuperUserPermission, getdb).SetBlock(true).Limit(ctxext.LimitByUser).
		Handle(func(ctx *zero.Ctx) {
			ctx.SendChain(message.Text("开始整理力，请稍等"))
			民政局.Lock()
			defer 民政局.Unlock()
			count, err := 民政局.db.Count("favorability")
			if err != nil {
				ctx.SendChain(message.Text("[ERROR]: ", err))
				return
			}
			if count == 0 {
				ctx.SendChain(message.Text("[ERROR]: 不存在好感度数据."))
				return
			}
			favor := favorability{}
			delInfo := make([]string, 0, count*2)
			favorInfo := make(map[string]int, count*2)
			_ = 民政局.db.FindFor("favorability", &favor, "GROUP BY Userinfo", func() error {
				delInfo = append(delInfo, favor.Userinfo)
				// 解析旧数据
				userList := strings.Split(favor.Userinfo, "+")
				maxQQ, _ := strconv.ParseInt(userList[0], 10, 64)
				minQQ, _ := strconv.ParseInt(userList[1], 10, 64)
				if maxQQ > minQQ {
					favor.Userinfo = userList[0] + "+" + userList[1]
				} else {
					favor.Userinfo = userList[1] + "+" + userList[0]
				}
				// 判断是否是重复的
				score, ok := favorInfo[favor.Userinfo]
				if ok {
					if score < favor.Favor {
						favorInfo[favor.Userinfo] = favor.Favor
					}
				} else {
					favorInfo[favor.Userinfo] = favor.Favor
				}
				return nil
			})
			// 删除旧数据
			q, s := sql.QuerySet("WHERE Userinfo", "IN", delInfo)
			err = 民政局.db.Del("favorability", q, s...)
			if err != nil {
				ctx.SendChain(message.Text("[ERROR]: 删除好感度时发生了错误。\n错误信息:", err))
			}
			for userInfo, favor := range favorInfo {
				favorInfo := favorability{
					Userinfo: userInfo,
					Favor:    favor,
				}
				err = 民政局.db.Insert("favorability", &favorInfo)
				if err != nil {
					userList := strings.Split(userInfo, "+")
					uid1, _ := strconv.ParseInt(userList[0], 10, 64)
					uid2, _ := strconv.ParseInt(userList[1], 10, 64)
					ctx.SendChain(message.Text("[ERROR]: 更新", ctx.CardOrNickName(uid1), "和", ctx.CardOrNickName(uid2), "的好感度时发生了错误。\n错误信息:", err))
				}
			}
			ctx.SendChain(message.Text("清理好了哦"))
		})
}

func (sql *婚姻登记) 查好感度(uid, target int64) (int, error) {
	sql.Lock()
	defer sql.Unlock()
	err := sql.db.Create("favorability", &favorability{})
	if err != nil {
		return 0, err
	}
	info := favorability{}
	if uid > target {
		userinfo := strconv.FormatInt(uid, 10) + "+" + strconv.FormatInt(target, 10)
		err = sql.db.Find("favorability", &info, "WHERE Userinfo = ?", userinfo)
		if err != nil {
			_ = sql.db.Find("favorability", &info, "WHERE Userinfo glob ?", "*"+userinfo+"*")
		}
	} else {
		userinfo := strconv.FormatInt(target, 10) + "+" + strconv.FormatInt(uid, 10)
		err = sql.db.Find("favorability", &info, "WHERE Userinfo = ?", userinfo)
		if err != nil {
			_ = sql.db.Find("favorability", &info, "WHERE Userinfo glob ?", "*"+userinfo+"*")
		}
	}
	return info.Favor, nil
}

// 获取好感度数据组
type favorList []favorability

func (s favorList) Len() int {
	return len(s)
}
func (s favorList) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
func (s favorList) Less(i, j int) bool {
	return s[i].Favor > s[j].Favor
}
func (sql *婚姻登记) getGroupFavorability(uid int64) (list favorList, err error) {
	uidStr := strconv.FormatInt(uid, 10)
	sql.RLock()
	defer sql.RUnlock()
	info := favorability{}
	err = sql.db.FindFor("favorability", &info, "WHERE Userinfo glob ?", func() error {
		var target string
		userList := strings.Split(info.Userinfo, "+")
		switch {
		case len(userList) == 0:
			return errors.New("好感度系统数据存在错误")
		case userList[0] == uidStr:
			target = userList[1]
		default:
			target = userList[0]
		}
		list = append(list, favorability{
			Userinfo: target,
			Favor:    info.Favor,
		})
		return nil
	}, "*"+uidStr+"*")
	sort.Sort(list)
	return
}

// 设置好感度 正增负减
func (sql *婚姻登记) 更新好感度(uid, target int64, score int) (favor int, err error) {
	sql.Lock()
	defer sql.Unlock()
	err = sql.db.Create("favorability", &favorability{})
	if err != nil {
		return
	}
	info := favorability{}
	uidstr := strconv.FormatInt(uid, 10)
	targstr := strconv.FormatInt(target, 10)
	if uid > target {
		info.Userinfo = uidstr + "+" + targstr
		err = sql.db.Find("favorability", &info, "WHERE Userinfo = ?", info.Userinfo)
	} else {
		info.Userinfo = targstr + "+" + uidstr
		err = sql.db.Find("favorability", &info, "WHERE Userinfo = ?", info.Userinfo)
	}
	if err != nil {
		err = sql.db.Find("favorability", &info, "WHERE Userinfo glob ?", "*"+targstr+"+"+uidstr+"*")
		if err == nil { // 如果旧数据存在就删除旧数据
			err = 民政局.db.Del("favorability", "WHERE Userinfo = ?", info.Userinfo)
		}
	}
	info.Favor += score
	if info.Favor > 100 {
		info.Favor = 100
	} else if info.Favor < 0 {
		info.Favor = 0
	}
	err = sql.db.Insert("favorability", &info)
	return info.Favor, err
}
