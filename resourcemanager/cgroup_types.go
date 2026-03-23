package resourcemanager

type CgroupHandle struct {
	path string
}

type MemoryEvents struct {
	Low     uint64
	High    uint64
	Max     uint64
	OOM     uint64
	OOMKill uint64
}
