package copier

import (
	"reflect"
	"slices"
	"strings"
	"sync"
)

// planPath 标识缓存 plan 对应的拷贝路径。
type planPath uint8

const (
	pathStructStruct planPath = iota
	pathStructToMap
	pathMapToStruct
)

// planOpts 进键的标量选项指纹：caseSensitive/tagName/must 影响字段匹配 plan 内容。
// 其余选项：nameConverter/fieldNameMapping/skipFields 为内容型（第三步再做），
// 非空时不走缓存；ignoreEmpty/maxDepth/converters/valueConverter/methodMapping
// 是运行时值/闭包处理，不影响字段匹配 plan，不进键。
type planOpts struct {
	caseSensitive bool
	tagName       string
	must          bool
}

// planKey 缓存键：类型组合 + 路径 + 标量选项指纹。
type planKey struct {
	srcType reflect.Type
	dstType reflect.Type
	path    planPath
	opts    planOpts
}

// fieldMapping 一条 src 字段 → dst 字段的匹配记录（struct→struct）。
// dstIdx 为 dst 字段的 FieldByIndex 索引链（支持嵌入字段提升匹配）；
// 为空表示无匹配字段。name 为 toName 转换后的目标名（setter 降级使用），
// srcName 为 src 字段原始名（valueConverter 回调使用）。
type fieldMapping struct {
	srcIdx  int    // src 字段索引
	srcName string // src 字段原始名（valueConverter 回调使用）
	name    string // toName 转换后的目标名（setter 降级使用）
	dstIdx  []int  // dst 字段索引链；空表示无匹配字段
}

// structPlan 预计算的 struct→struct 字段映射（按 src 字段序）。
type structPlan struct {
	fields []fieldMapping
}

// structToMapEntry 一条 src 字段 → map key 的记录（struct→map）。
type structToMapEntry struct {
	srcIdx  int    // src 字段索引
	srcName string // src 字段原始名（TypeConvert/valueConverter 使用）
	name    string // toName 转换后的 map key
	expand  bool   // 匿名 struct 字段：运行时递归展开
}

// structToMapPlan 预计算的 struct→map 字段映射（按 src 字段序）。
type structToMapPlan struct {
	entries []structToMapEntry
}

// mapToStructPlan 预计算的 map key → dst 字段映射（map→struct）。
// lookup 键：大小写不敏感时为字段名小写化，敏感时为字段原名；
// 值为 dst 字段 FieldByIndex 索引链（含嵌入提升）。
type mapToStructPlan struct {
	lookup map[string][]int
}

// planCache 全局 plan 缓存，键为 planKey，值按 path 分
// *structPlan / *structToMapPlan / *mapToStructPlan。
// 类型元数据不可变，plan 无需失效；sync.Map 并发安全。
var planCache sync.Map

// planEligible 判断 opt 的字段匹配配置是否可走缓存：
// 要求 nameConverter==nil、fieldNameMapping 为空、skipFields 为空
// （这三类内容型选项第三步才做，暂不缓存）。
// caseSensitive/tagName/must 已进键，不参与判断；
// ignoreEmpty、maxDepth、converters、valueConverter、methodMapping 是
// 运行时值/闭包处理，不影响字段匹配 plan，不参与判断。
func planEligible(opt *options) bool {
	return opt.nameConverter == nil &&
		len(opt.fieldNameMapping) == 0 &&
		len(opt.skipFields) == 0
}

func planKeyOf(srcType, dstType reflect.Type, path planPath, opt *options) planKey {
	return planKey{
		srcType: srcType,
		dstType: dstType,
		path:    path,
		opts: planOpts{
			caseSensitive: opt.caseSensitive,
			tagName:       opt.tagName,
			must:          opt.must,
		},
	}
}

// buildStructPlan 用与 cpyStruct 现有逐字段逻辑完全一致的规则构建 plan：
// 遍历 src 字段，逐条执行：PkgPath 过滤（与现有一致，含匿名未导出 struct 字段保留）、
// isSkipField、parseTag、must 过滤、toName、getFieldByName →dstIdx
// （caseSensitive 关闭时 EqualFold 不敏感匹配，开启后精确匹配）。
// 无法预计算的运行时检查（ignoreEmpty、TypeConvert、valueConverter、深度、visited）不进入 plan。
func buildStructPlan(srcType, dstType reflect.Type, opt *options) *structPlan {
	plan := &structPlan{}

	for i, n := 0, srcType.NumField(); i < n; i++ {
		sf := srcType.Field(i)
		if sf.PkgPath != "" && !sf.Anonymous {
			continue
		}

		// 与现有 cpyStruct 一致：跳过列表过滤
		if opt.isSkipField(sf.Name) {
			continue
		}

		tag := parseTag(sf.Tag.Get(opt.tagName))
		if tag.Contains(tagSkip) {
			continue
		}

		// must 模式：只拷贝带 must 标签的字段
		if opt.must && !tag.Contains(tagRequired) {
			continue
		}

		name := toName(sf.Name, tag, opt)

		m := fieldMapping{
			srcIdx:  i,
			srcName: sf.Name,
			name:    name,
		}

		// 与运行时 getFieldByName 完全一致的匹配：
		// Value.FieldByName(Func) 内部即 Type.FieldByName(Func) + FieldByIndex(f.Index)
		if opt.caseSensitive {
			if f, ok := dstType.FieldByName(name); ok {
				m.dstIdx = f.Index
			}
		} else if f, ok := dstType.FieldByNameFunc(func(s string) bool {
			return strings.EqualFold(s, name)
		}); ok {
			m.dstIdx = f.Index
		}

		plan.fields = append(plan.fields, m)
	}

	return plan
}

// getStructPlan 缓存读取（struct→struct）：planEligible(opt) 为 true 时查/建缓存；
// 非 eligible 时返回 nil，调用方走原路径。
func getStructPlan(srcType, dstType reflect.Type, opt *options) *structPlan {
	if !planEligible(opt) {
		return nil
	}

	key := planKeyOf(srcType, dstType, pathStructStruct, opt)
	if v, ok := planCache.Load(key); ok {
		return v.(*structPlan)
	}

	plan := buildStructPlan(srcType, dstType, opt)
	actual, _ := planCache.LoadOrStore(key, plan)
	return actual.(*structPlan)
}

// buildStructToMapPlan 用与 deepCopyInner Map 分支 struct→map 循环完全一致的规则构建：
// PkgPath 过滤 → isSkipField → parseTag/ignore → must 过滤 →
// 匿名 struct 字段标记 expand（顺序与现有一致：tag/must 检查之后才判断匿名展开；
// 匿名但非 struct 类型如嵌入 interface 不标记 expand）→ 其余 toName 得 name。
func buildStructToMapPlan(srcType reflect.Type, opt *options) *structToMapPlan {
	plan := &structToMapPlan{}

	for i, n := 0, srcType.NumField(); i < n; i++ {
		sf := srcType.Field(i)
		if sf.PkgPath != "" && !sf.Anonymous {
			continue
		}

		// 与现有循环一致：跳过列表过滤
		if opt.isSkipField(sf.Name) {
			continue
		}

		tag := parseTag(sf.Tag.Get(opt.tagName))
		if tag.Contains(tagSkip) {
			continue
		}

		// must 模式：只拷贝带 must 标签的字段
		if opt.must && !tag.Contains(tagRequired) {
			continue
		}

		if sf.Anonymous && sf.Type.Kind() == reflect.Struct {
			plan.entries = append(plan.entries, structToMapEntry{
				srcIdx:  i,
				srcName: sf.Name,
				expand:  true,
			})
			continue
		}

		plan.entries = append(plan.entries, structToMapEntry{
			srcIdx:  i,
			srcName: sf.Name,
			name:    toName(sf.Name, tag, opt),
		})
	}

	return plan
}

// getStructToMapPlan 缓存读取（struct→map）：plan 内容只依赖 src 类型，
// dstType 在键中占位为 srcType。
func getStructToMapPlan(srcType reflect.Type, opt *options) *structToMapPlan {
	if !planEligible(opt) {
		return nil
	}

	key := planKeyOf(srcType, srcType, pathStructToMap, opt)
	if v, ok := planCache.Load(key); ok {
		return v.(*structToMapPlan)
	}

	plan := buildStructToMapPlan(srcType, opt)
	actual, _ := planCache.LoadOrStore(key, plan)
	return actual.(*structToMapPlan)
}

// buildMapToStructPlan 用与 getFieldByName 完全一致的匹配规则反向构建
// map key → dst 字段索引链的查找表。
// 遍历 dst 类型字段（含嵌入提升，仅导出匿名 struct 字段递归）；
// 每个候选字段用 dstType.FieldByName/Func 验证确实是匹配目标（处理歧义：
// 同小写键多个字段时 FieldByNameFunc 返回第一个、提升歧义返回 false）。
// 注意：map→struct 的 NameConvert 在运行时作用在 map key 上、不可反转，
// 故 planEligible 已要求 nameConverter==nil 且 fieldNameMapping 为空。
func buildMapToStructPlan(dstType reflect.Type, opt *options) *mapToStructPlan {
	lookup := make(map[string][]int)

	collect := func(name string, index []int) {
		var f reflect.StructField
		var ok bool
		if opt.caseSensitive {
			f, ok = dstType.FieldByName(name)
		} else {
			f, ok = dstType.FieldByNameFunc(func(s string) bool {
				return strings.EqualFold(s, name)
			})
		}
		if !ok || !slices.Equal(f.Index, index) {
			return
		}

		key := name
		if !opt.caseSensitive {
			key = strings.ToLower(name)
		}
		if _, exists := lookup[key]; !exists {
			lookup[key] = append([]int(nil), index...)
		}
	}

	var walk func(t reflect.Type, prefix []int)
	walk = func(t reflect.Type, prefix []int) {
		for i := 0; i < t.NumField(); i++ {
			sf := t.Field(i)
			idx := append(append([]int(nil), prefix...), i)
			collect(sf.Name, idx)

			// 提升：匿名嵌入 struct / *struct 字段（与 reflect FieldByNameFunc 一致，
			// BFS 下探所有嵌入，不做导出性过滤——未导出嵌入字段的运行时可见性
			// 由调用方 CanSet 检查兜底，行为与 getFieldByName 等价）
			if sf.Anonymous {
				ntyp := sf.Type
				if ntyp.Kind() == reflect.Pointer {
					ntyp = ntyp.Elem()
				}
				if ntyp.Kind() == reflect.Struct {
					walk(ntyp, idx)
				}
			}
		}
	}
	walk(dstType, nil)

	return &mapToStructPlan{lookup: lookup}
}

// getMapToStructPlan 缓存读取（map→struct）：plan 内容只依赖 dst 类型，
// srcType 在键中占位为 dstType。
func getMapToStructPlan(dstType reflect.Type, opt *options) *mapToStructPlan {
	if !planEligible(opt) {
		return nil
	}

	key := planKeyOf(dstType, dstType, pathMapToStruct, opt)
	if v, ok := planCache.Load(key); ok {
		return v.(*mapToStructPlan)
	}

	plan := buildMapToStructPlan(dstType, opt)
	actual, _ := planCache.LoadOrStore(key, plan)
	return actual.(*mapToStructPlan)
}
