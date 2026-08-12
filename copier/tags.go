package copier

import "strings"

const (
	defaultTag = "copier"
)

type tagt uint8

const (
	tagMust tagt = 1 << iota
	tagIgnore
	tagToName
)

// 标签 copier 取值 must、ignore、toname 等
// copier:"must,toname=xxx" 必须复制，并重命名为 xxx

type tagOption struct {
	flg    tagt
	toname string
}

func parseTag(tag string) *tagOption {
	opt := tagOption{}

	flg := tagt(0)
	for t := range strings.SplitSeq(tag, ",") {
		tag, value, found := strings.Cut(t, "=")
		switch tag {
		case "-", "ignore":
			flg |= tagIgnore
		case "must":
			flg |= tagMust
		case "toname":
			flg |= tagToName
			if found {
				opt.toname = value
			}
		}
	}

	opt.flg = flg

	return &opt
}

func (o *tagOption) Contains(tag tagt) bool {
	if o.flg == 0 {
		return false
	}

	return o.flg&tag == tag
}

func (o *tagOption) ToName() string {
	if o.flg&tagToName == 0 {
		return ""
	}

	return o.toname
}
