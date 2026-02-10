package command

import (
	"go-redis/data"
	"go-redis/resp"
	"math"
	"sort"
	"strconv"
	"strings"
)

// parseFloat 解析浮点数，支持 "inf", "-inf", "+inf"
func parseFloat(s string) (float64, bool) {
	s = strings.ToLower(s)
	switch s {
	case "inf", "+inf":
		return math.Inf(1), true
	case "-inf":
		return math.Inf(-1), true
	default:
		val, err := strconv.ParseFloat(s, 64)
		return val, err == nil
	}
}

// ZAddCommand 向有序集合添加成员
type ZAddCommand struct{}

func (c *ZAddCommand) Name() string  { return "ZADD" }
func (c *ZAddCommand) ArgCount() int { return 3 }
func (c *ZAddCommand) Execute(db *data.DB, args []string, writer *resp.Writer) error {
	key := args[0]

	var zset *data.ZSet
	existing, ok := db.Get(key)
	if ok {
		var typeOk bool
		zset, typeOk = existing.(*data.ZSet)
		if !typeOk {
			return writer.WriteError("WRONGTYPE Operation against a key holding the wrong kind of value")
		}
	} else {
		zset = &data.ZSet{Members: make(map[string]float64)}
	}

	// 解析 score-member 对
	if len(args[1:])%2 != 0 {
		return writer.WriteError("ERR syntax error")
	}

	added := 0
	for i := 1; i < len(args); i += 2 {
		score, err := strconv.ParseFloat(args[i], 64)
		if err != nil {
			return writer.WriteError("ERR value is not a valid float")
		}
		member := args[i+1]

		if _, exists := zset.Members[member]; !exists {
			added++
		}
		zset.Members[member] = score
	}

	db.Set(key, zset)
	return writer.WriteInteger(int64(added))
}

// ZRemCommand 移除有序集合中的成员
type ZRemCommand struct{}

func (c *ZRemCommand) Name() string  { return "ZREM" }
func (c *ZRemCommand) ArgCount() int { return 2 }
func (c *ZRemCommand) Execute(db *data.DB, args []string, writer *resp.Writer) error {
	key := args[0]
	members := args[1:]

	existing, ok := db.Get(key)
	if !ok {
		return writer.WriteInteger(0)
	}

	zset, ok := existing.(*data.ZSet)
	if !ok {
		return writer.WriteError("WRONGTYPE Operation against a key holding the wrong kind of value")
	}

	removed := 0
	for _, member := range members {
		if _, exists := zset.Members[member]; exists {
			delete(zset.Members, member)
			removed++
		}
	}

	if len(zset.Members) == 0 {
		db.Delete(key)
	} else {
		db.Set(key, zset)
	}

	return writer.WriteInteger(int64(removed))
}

// ZScoreCommand 返回成员的分数
type ZScoreCommand struct{}

func (c *ZScoreCommand) Name() string  { return "ZSCORE" }
func (c *ZScoreCommand) ArgCount() int { return 2 }
func (c *ZScoreCommand) Execute(db *data.DB, args []string, writer *resp.Writer) error {
	key, member := args[0], args[1]

	existing, ok := db.Get(key)
	if !ok {
		return writer.WriteNullBulkString()
	}

	zset, ok := existing.(*data.ZSet)
	if !ok {
		return writer.WriteError("WRONGTYPE Operation against a key holding the wrong kind of value")
	}

	score, exists := zset.Members[member]
	if !exists {
		return writer.WriteNullBulkString()
	}

	return writer.WriteBulkString(strconv.FormatFloat(score, 'f', -1, 64))
}

// ZCardCommand 获取有序集合的成员数
type ZCardCommand struct{}

func (c *ZCardCommand) Name() string  { return "ZCARD" }
func (c *ZCardCommand) ArgCount() int { return 1 }
func (c *ZCardCommand) Execute(db *data.DB, args []string, writer *resp.Writer) error {
	existing, ok := db.Get(args[0])
	if !ok {
		return writer.WriteInteger(0)
	}

	zset, ok := existing.(*data.ZSet)
	if !ok {
		return writer.WriteError("WRONGTYPE Operation against a key holding the wrong kind of value")
	}

	return writer.WriteInteger(int64(len(zset.Members)))
}

// getSortedMembers 获取排序后的成员列表
func getSortedMembers(zset *data.ZSet, reverse bool) []data.ZSetItem {
	items := make([]data.ZSetItem, 0, len(zset.Members))
	for member, score := range zset.Members {
		items = append(items, data.ZSetItem{Member: member, Score: score})
	}

	if reverse {
		sort.Slice(items, func(i, j int) bool {
			if items[i].Score == items[j].Score {
				return items[i].Member > items[j].Member
			}
			return items[i].Score > items[j].Score
		})
	} else {
		sort.Slice(items, func(i, j int) bool {
			if items[i].Score == items[j].Score {
				return items[i].Member < items[j].Member
			}
			return items[i].Score < items[j].Score
		})
	}

	return items
}

// ZRangeCommand 返回指定区间内的成员
type ZRangeCommand struct{}

func (c *ZRangeCommand) Name() string  { return "ZRANGE" }
func (c *ZRangeCommand) ArgCount() int { return 3 }
func (c *ZRangeCommand) Execute(db *data.DB, args []string, writer *resp.Writer) error {
	return zrange(db, args, writer, false)
}

// ZRevRangeCommand 返回指定区间内的成员（逆序）
type ZRevRangeCommand struct{}

func (c *ZRevRangeCommand) Name() string  { return "ZREVRANGE" }
func (c *ZRevRangeCommand) ArgCount() int { return 3 }
func (c *ZRevRangeCommand) Execute(db *data.DB, args []string, writer *resp.Writer) error {
	return zrange(db, args, writer, true)
}

func zrange(db *data.DB, args []string, writer *resp.Writer, reverse bool) error {
	key, startStr, stopStr := args[0], args[1], args[2]

	withScores := false
	if len(args) > 3 && strings.ToUpper(args[3]) == "WITHSCORES" {
		withScores = true
	}

	start, err := strconv.Atoi(startStr)
	if err != nil {
		return writer.WriteError("ERR value is not an integer or out of range")
	}

	stop, err := strconv.Atoi(stopStr)
	if err != nil {
		return writer.WriteError("ERR value is not an integer or out of range")
	}

	existing, ok := db.Get(key)
	if !ok {
		return writer.WriteStringArray([]string{})
	}

	zset, ok := existing.(*data.ZSet)
	if !ok {
		return writer.WriteError("WRONGTYPE Operation against a key holding the wrong kind of value")
	}

	items := getSortedMembers(zset, reverse)
	length := len(items)

	// 处理负数索引
	if start < 0 {
		start = length + start
	}
	if stop < 0 {
		stop = length + stop
	}

	// 边界检查
	if start < 0 {
		start = 0
	}
	if stop >= length {
		stop = length - 1
	}
	if start > stop || start >= length {
		return writer.WriteStringArray([]string{})
	}

	result := items[start : stop+1]

	if withScores {
		// 返回 member score member score ...
		output := make([]string, 0, len(result)*2)
		for _, item := range result {
			output = append(output, item.Member, strconv.FormatFloat(item.Score, 'f', -1, 64))
		}
		return writer.WriteStringArray(output)
	}

	// 只返回 members
	output := make([]string, len(result))
	for i, item := range result {
		output[i] = item.Member
	}
	return writer.WriteStringArray(output)
}

// ZRankCommand 返回成员的排名
type ZRankCommand struct{}

func (c *ZRankCommand) Name() string  { return "ZRANK" }
func (c *ZRankCommand) ArgCount() int { return 2 }
func (c *ZRankCommand) Execute(db *data.DB, args []string, writer *resp.Writer) error {
	return zrank(db, args, writer, false)
}

// ZRevRankCommand 返回成员的排名（逆序）
type ZRevRankCommand struct{}

func (c *ZRevRankCommand) Name() string  { return "ZREVRANK" }
func (c *ZRevRankCommand) ArgCount() int { return 2 }
func (c *ZRevRankCommand) Execute(db *data.DB, args []string, writer *resp.Writer) error {
	return zrank(db, args, writer, true)
}

func zrank(db *data.DB, args []string, writer *resp.Writer, reverse bool) error {
	key, member := args[0], args[1]

	existing, ok := db.Get(key)
	if !ok {
		return writer.WriteNullBulkString()
	}

	zset, ok := existing.(*data.ZSet)
	if !ok {
		return writer.WriteError("WRONGTYPE Operation against a key holding the wrong kind of value")
	}

	if _, exists := zset.Members[member]; !exists {
		return writer.WriteNullBulkString()
	}

	items := getSortedMembers(zset, reverse)
	for i, item := range items {
		if item.Member == member {
			return writer.WriteInteger(int64(i))
		}
	}

	return writer.WriteNullBulkString()
}

// ZIncrByCommand 对成员分数加上增量
type ZIncrByCommand struct{}

func (c *ZIncrByCommand) Name() string  { return "ZINCRBY" }
func (c *ZIncrByCommand) ArgCount() int { return 3 }
func (c *ZIncrByCommand) Execute(db *data.DB, args []string, writer *resp.Writer) error {
	key, incrementStr, member := args[0], args[1], args[2]

	increment, err := strconv.ParseFloat(incrementStr, 64)
	if err != nil {
		return writer.WriteError("ERR value is not a valid float")
	}

	var zset *data.ZSet
	existing, ok := db.Get(key)
	if ok {
		var typeOk bool
		zset, typeOk = existing.(*data.ZSet)
		if !typeOk {
			return writer.WriteError("WRONGTYPE Operation against a key holding the wrong kind of value")
		}
	} else {
		zset = &data.ZSet{Members: make(map[string]float64)}
	}

	score := zset.Members[member]
	score += increment
	zset.Members[member] = score

	db.Set(key, zset)
	return writer.WriteBulkString(strconv.FormatFloat(score, 'f', -1, 64))
}

// ZRangeByScoreCommand 返回指定分数区间的成员
type ZRangeByScoreCommand struct{}

func (c *ZRangeByScoreCommand) Name() string  { return "ZRANGEBYSCORE" }
func (c *ZRangeByScoreCommand) ArgCount() int { return 3 }
func (c *ZRangeByScoreCommand) Execute(db *data.DB, args []string, writer *resp.Writer) error {
	key, minStr, maxStr := args[0], args[1], args[2]

	withScores := false
	withLimit := false
	var limitOffset, limitCount int

	// 解析可选参数
	for i := 3; i < len(args); i++ {
		switch strings.ToUpper(args[i]) {
		case "WITHSCORES":
			withScores = true
		case "LIMIT":
			if i+2 < len(args) {
				withLimit = true
				limitOffset, _ = strconv.Atoi(args[i+1])
				limitCount, _ = strconv.Atoi(args[i+2])
				i += 2
			}
		}
	}

	minScore, ok := parseFloat(minStr)
	if !ok {
		return writer.WriteError("ERR min or max is not a float")
	}

	maxScore, ok := parseFloat(maxStr)
	if !ok {
		return writer.WriteError("ERR min or max is not a float")
	}

	existing, ok := db.Get(key)
	if !ok {
		return writer.WriteStringArray([]string{})
	}

	zset, ok := existing.(*data.ZSet)
	if !ok {
		return writer.WriteError("WRONGTYPE Operation against a key holding the wrong kind of value")
	}

	// 获取排序后的成员并过滤
	items := getSortedMembers(zset, false)
	var result []data.ZSetItem

	for _, item := range items {
		if item.Score >= minScore && item.Score <= maxScore {
			result = append(result, item)
		}
	}

	// 应用 LIMIT
	if withLimit {
		if limitOffset >= len(result) {
			result = []data.ZSetItem{}
		} else {
			end := limitOffset + limitCount
			if end > len(result) {
				end = len(result)
			}
			result = result[limitOffset:end]
		}
	}

	if withScores {
		output := make([]string, 0, len(result)*2)
		for _, item := range result {
			output = append(output, item.Member, strconv.FormatFloat(item.Score, 'f', -1, 64))
		}
		return writer.WriteStringArray(output)
	}

	output := make([]string, len(result))
	for i, item := range result {
		output[i] = item.Member
	}
	return writer.WriteStringArray(output)
}

// ZRemRangeByRankCommand 移除指定排名区间的成员
type ZRemRangeByRankCommand struct{}

func (c *ZRemRangeByRankCommand) Name() string  { return "ZREMRANGEBYRANK" }
func (c *ZRemRangeByRankCommand) ArgCount() int { return 3 }
func (c *ZRemRangeByRankCommand) Execute(db *data.DB, args []string, writer *resp.Writer) error {
	key, startStr, stopStr := args[0], args[1], args[2]

	start, err := strconv.Atoi(startStr)
	if err != nil {
		return writer.WriteError("ERR value is not an integer or out of range")
	}

	stop, err := strconv.Atoi(stopStr)
	if err != nil {
		return writer.WriteError("ERR value is not an integer or out of range")
	}

	existing, ok := db.Get(key)
	if !ok {
		return writer.WriteInteger(0)
	}

	zset, ok := existing.(*data.ZSet)
	if !ok {
		return writer.WriteError("WRONGTYPE Operation against a key holding the wrong kind of value")
	}

	items := getSortedMembers(zset, false)
	length := len(items)

	// 处理负数索引
	if start < 0 {
		start = length + start
	}
	if stop < 0 {
		stop = length + stop
	}

	// 边界检查
	if start < 0 {
		start = 0
	}
	if stop >= length {
		stop = length - 1
	}
	if start > stop || start >= length {
		return writer.WriteInteger(0)
	}

	// 删除指定范围的成员
	toRemove := items[start : stop+1]
	for _, item := range toRemove {
		delete(zset.Members, item.Member)
	}

	if len(zset.Members) == 0 {
		db.Delete(key)
	} else {
		db.Set(key, zset)
	}

	return writer.WriteInteger(int64(len(toRemove)))
}

// ZRemRangeByScoreCommand 移除指定分数区间的成员
type ZRemRangeByScoreCommand struct{}

func (c *ZRemRangeByScoreCommand) Name() string  { return "ZREMRANGEBYSCORE" }
func (c *ZRemRangeByScoreCommand) ArgCount() int { return 3 }
func (c *ZRemRangeByScoreCommand) Execute(db *data.DB, args []string, writer *resp.Writer) error {
	key, minStr, maxStr := args[0], args[1], args[2]

	minScore, ok := parseFloat(minStr)
	if !ok {
		return writer.WriteError("ERR min or max is not a float")
	}

	maxScore, ok := parseFloat(maxStr)
	if !ok {
		return writer.WriteError("ERR min or max is not a float")
	}

	existing, ok := db.Get(key)
	if !ok {
		return writer.WriteInteger(0)
	}

	zset, ok := existing.(*data.ZSet)
	if !ok {
		return writer.WriteError("WRONGTYPE Operation against a key holding the wrong kind of value")
	}

	removed := 0
	for member, score := range zset.Members {
		if score >= minScore && score <= maxScore {
			delete(zset.Members, member)
			removed++
		}
	}

	if len(zset.Members) == 0 {
		db.Delete(key)
	} else {
		db.Set(key, zset)
	}

	return writer.WriteInteger(int64(removed))
}
