package command

import (
	"go-redis/data"
	"go-redis/resp"
	"math/rand"
	"strconv"
)

// SAddCommand 向集合添加成员
type SAddCommand struct{}

func (c *SAddCommand) Name() string  { return "SADD" }
func (c *SAddCommand) ArgCount() int { return 2 }
func (c *SAddCommand) Execute(db *data.DB, args []string, writer *resp.Writer) error {
	key := args[0]
	members := args[1:]

	var set *data.Set
	existing, ok := db.Get(key)
	if ok {
		var typeOk bool
		set, typeOk = existing.(*data.Set)
		if !typeOk {
			return writer.WriteError("WRONGTYPE Operation against a key holding the wrong kind of value")
		}
	} else {
		set = &data.Set{Members: make(map[string]struct{})}
	}

	added := 0
	for _, member := range members {
		if _, exists := set.Members[member]; !exists {
			set.Members[member] = struct{}{}
			added++
		}
	}

	db.Set(key, set)
	return writer.WriteInteger(int64(added))
}

// SRemCommand 移除集合成员
type SRemCommand struct{}

func (c *SRemCommand) Name() string  { return "SREM" }
func (c *SRemCommand) ArgCount() int { return 2 }
func (c *SRemCommand) Execute(db *data.DB, args []string, writer *resp.Writer) error {
	key := args[0]
	members := args[1:]

	existing, ok := db.Get(key)
	if !ok {
		return writer.WriteInteger(0)
	}

	set, ok := existing.(*data.Set)
	if !ok {
		return writer.WriteError("WRONGTYPE Operation against a key holding the wrong kind of value")
	}

	removed := 0
	for _, member := range members {
		if _, exists := set.Members[member]; exists {
			delete(set.Members, member)
			removed++
		}
	}

	if len(set.Members) == 0 {
		db.Delete(key)
	} else {
		db.Set(key, set)
	}

	return writer.WriteInteger(int64(removed))
}

// SIsMemberCommand 判断成员是否在集合中
type SIsMemberCommand struct{}

func (c *SIsMemberCommand) Name() string  { return "SISMEMBER" }
func (c *SIsMemberCommand) ArgCount() int { return 2 }
func (c *SIsMemberCommand) Execute(db *data.DB, args []string, writer *resp.Writer) error {
	key, member := args[0], args[1]

	existing, ok := db.Get(key)
	if !ok {
		return writer.WriteInteger(0)
	}

	set, ok := existing.(*data.Set)
	if !ok {
		return writer.WriteError("WRONGTYPE Operation against a key holding the wrong kind of value")
	}

	if _, exists := set.Members[member]; exists {
		return writer.WriteInteger(1)
	}
	return writer.WriteInteger(0)
}

// SCardCommand 获取集合成员数
type SCardCommand struct{}

func (c *SCardCommand) Name() string  { return "SCARD" }
func (c *SCardCommand) ArgCount() int { return 1 }
func (c *SCardCommand) Execute(db *data.DB, args []string, writer *resp.Writer) error {
	existing, ok := db.Get(args[0])
	if !ok {
		return writer.WriteInteger(0)
	}

	set, ok := existing.(*data.Set)
	if !ok {
		return writer.WriteError("WRONGTYPE Operation against a key holding the wrong kind of value")
	}

	return writer.WriteInteger(int64(len(set.Members)))
}

// SMembersCommand 返回集合中的所有成员
type SMembersCommand struct{}

func (c *SMembersCommand) Name() string  { return "SMEMBERS" }
func (c *SMembersCommand) ArgCount() int { return 1 }
func (c *SMembersCommand) Execute(db *data.DB, args []string, writer *resp.Writer) error {
	existing, ok := db.Get(args[0])
	if !ok {
		return writer.WriteStringArray([]string{})
	}

	set, ok := existing.(*data.Set)
	if !ok {
		return writer.WriteError("WRONGTYPE Operation against a key holding the wrong kind of value")
	}

	members := make([]string, 0, len(set.Members))
	for member := range set.Members {
		members = append(members, member)
	}
	return writer.WriteStringArray(members)
}

// SPopCommand 移除并返回集合中的一个随机元素
type SPopCommand struct{}

func (c *SPopCommand) Name() string  { return "SPOP" }
func (c *SPopCommand) ArgCount() int { return 1 }
func (c *SPopCommand) Execute(db *data.DB, args []string, writer *resp.Writer) error {
	key := args[0]
	count := 1
	if len(args) > 1 {
		var err error
		count, err = strconv.Atoi(args[1])
		if err != nil {
			return writer.WriteError("ERR value is not an integer or out of range")
		}
	}

	existing, ok := db.Get(key)
	if !ok {
		if count == 1 {
			return writer.WriteNullBulkString()
		}
		return writer.WriteStringArray([]string{})
	}

	set, ok := existing.(*data.Set)
	if !ok {
		return writer.WriteError("WRONGTYPE Operation against a key holding the wrong kind of value")
	}

	if len(set.Members) == 0 {
		db.Delete(key)
		if count == 1 {
			return writer.WriteNullBulkString()
		}
		return writer.WriteStringArray([]string{})
	}

	// 获取所有成员
	members := make([]string, 0, len(set.Members))
	for member := range set.Members {
		members = append(members, member)
	}

	if count == 1 {
		// 随机选择一个
		idx := rand.Intn(len(members))
		delete(set.Members, members[idx])
		if len(set.Members) == 0 {
			db.Delete(key)
		} else {
			db.Set(key, set)
		}
		return writer.WriteBulkString(members[idx])
	}

	// 返回多个随机元素
	if count > len(members) {
		count = len(members)
	}

	result := make([]string, count)
	for i := 0; i < count; i++ {
		idx := rand.Intn(len(members))
		result[i] = members[idx]
		delete(set.Members, members[idx])
		members = append(members[:idx], members[idx+1:]...)
	}

	if len(set.Members) == 0 {
		db.Delete(key)
	} else {
		db.Set(key, set)
	}

	return writer.WriteStringArray(result)
}

// SRandMemberCommand 返回集合中一个或多个随机数
type SRandMemberCommand struct{}

func (c *SRandMemberCommand) Name() string  { return "SRANDMEMBER" }
func (c *SRandMemberCommand) ArgCount() int { return 1 }
func (c *SRandMemberCommand) Execute(db *data.DB, args []string, writer *resp.Writer) error {
	key := args[0]
	count := 1
	if len(args) > 1 {
		var err error
		count, err = strconv.Atoi(args[1])
		if err != nil {
			return writer.WriteError("ERR value is not an integer or out of range")
		}
	}

	existing, ok := db.Get(key)
	if !ok {
		if count == 1 {
			return writer.WriteNullBulkString()
		}
		return writer.WriteStringArray([]string{})
	}

	set, ok := existing.(*data.Set)
	if !ok {
		return writer.WriteError("WRONGTYPE Operation against a key holding the wrong kind of value")
	}

	members := make([]string, 0, len(set.Members))
	for member := range set.Members {
		members = append(members, member)
	}

	if count == 1 {
		idx := rand.Intn(len(members))
		return writer.WriteBulkString(members[idx])
	}

	// 返回多个随机元素（不删除）
	allowDuplicates := count < 0
	if count < 0 {
		count = -count
	}

	result := make([]string, 0, count)
	for i := 0; i < count && len(members) > 0; i++ {
		idx := rand.Intn(len(members))
		result = append(result, members[idx])
		if !allowDuplicates {
			members = append(members[:idx], members[idx+1:]...)
		}
	}

	return writer.WriteStringArray(result)
}

// SUnionCommand 返回所有给定集合的并集
type SUnionCommand struct{}

func (c *SUnionCommand) Name() string  { return "SUNION" }
func (c *SUnionCommand) ArgCount() int { return 1 }
func (c *SUnionCommand) Execute(db *data.DB, args []string, writer *resp.Writer) error {
	result := make(map[string]struct{})

	for _, key := range args {
		existing, ok := db.Get(key)
		if !ok {
			continue
		}

		set, ok := existing.(*data.Set)
		if !ok {
			return writer.WriteError("WRONGTYPE Operation against a key holding the wrong kind of value")
		}

		for member := range set.Members {
			result[member] = struct{}{}
		}
	}

	members := make([]string, 0, len(result))
	for member := range result {
		members = append(members, member)
	}
	return writer.WriteStringArray(members)
}

// SInterCommand 返回所有给定集合的交集
type SInterCommand struct{}

func (c *SInterCommand) Name() string  { return "SINTER" }
func (c *SInterCommand) ArgCount() int { return 1 }
func (c *SInterCommand) Execute(db *data.DB, args []string, writer *resp.Writer) error {
	if len(args) == 0 {
		return writer.WriteStringArray([]string{})
	}

	// 获取第一个集合
	existing, ok := db.Get(args[0])
	if !ok {
		return writer.WriteStringArray([]string{})
	}

	firstSet, ok := existing.(*data.Set)
	if !ok {
		return writer.WriteError("WRONGTYPE Operation against a key holding the wrong kind of value")
	}

	// 复制第一个集合
	result := make(map[string]struct{})
	for member := range firstSet.Members {
		result[member] = struct{}{}
	}

	// 与其他集合求交集
	for i := 1; i < len(args); i++ {
		existing, ok := db.Get(args[i])
		if !ok {
			return writer.WriteStringArray([]string{})
		}

		set, ok := existing.(*data.Set)
		if !ok {
			return writer.WriteError("WRONGTYPE Operation against a key holding the wrong kind of value")
		}

		// 移除不在当前集合中的成员
		for member := range result {
			if _, exists := set.Members[member]; !exists {
				delete(result, member)
			}
		}

		if len(result) == 0 {
			break
		}
	}

	members := make([]string, 0, len(result))
	for member := range result {
		members = append(members, member)
	}
	return writer.WriteStringArray(members)
}

// SDiffCommand 返回第一个集合与其他集合的差集
type SDiffCommand struct{}

func (c *SDiffCommand) Name() string  { return "SDIFF" }
func (c *SDiffCommand) ArgCount() int { return 1 }
func (c *SDiffCommand) Execute(db *data.DB, args []string, writer *resp.Writer) error {
	if len(args) == 0 {
		return writer.WriteStringArray([]string{})
	}

	// 获取第一个集合
	existing, ok := db.Get(args[0])
	if !ok {
		return writer.WriteStringArray([]string{})
	}

	firstSet, ok := existing.(*data.Set)
	if !ok {
		return writer.WriteError("WRONGTYPE Operation against a key holding the wrong kind of value")
	}

	// 复制第一个集合
	result := make(map[string]struct{})
	for member := range firstSet.Members {
		result[member] = struct{}{}
	}

	// 移除在其他集合中的成员
	for i := 1; i < len(args); i++ {
		existing, ok := db.Get(args[i])
		if !ok {
			continue
		}

		set, ok := existing.(*data.Set)
		if !ok {
			return writer.WriteError("WRONGTYPE Operation against a key holding the wrong kind of value")
		}

		for member := range set.Members {
			delete(result, member)
		}

		if len(result) == 0 {
			break
		}
	}

	members := make([]string, 0, len(result))
	for member := range result {
		members = append(members, member)
	}
	return writer.WriteStringArray(members)
}
