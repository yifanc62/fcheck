package main

type FCheckList struct {
	Time  int64         `json:"time"`
	Files []*FCheckFile `json:"files"`
}

type FCheckFile struct {
	Path string `json:"path"`
	SHA1 string `json:"sha1"`
	Size int64  `json:"size"`
}

type FCheckDiffList struct {
	Time        int64    `json:"time"`
	Mismatching []string `json:"mismatching"`
	Missing     []string `json:"missing"`
	Redundant   []string `json:"redundant"`
}
