package gpu

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/NVIDIA/go-nvml/pkg/nvml"
)

const (
	MiB = 1048576
)

type GpuInfoList struct {
	Inner     map[string]*GpuInfo `json:"device"`
	MemorySum Memory              `json:"sum.memory"`
}

func NewGpuInfoList() (*GpuInfoList, error) {
	ret := nvml.Init()
	if ret != nvml.SUCCESS {
		return nil, errors.New(nvml.ErrorString(ret))
	}
	defer nvml.Shutdown()

	count, ret := nvml.DeviceGetCount()
	if ret != nvml.SUCCESS {
		return nil, errors.New(nvml.ErrorString(ret))
	}

	devices := []*nvml.Device{}

	for i := 0; i < count; i++ {
		device, ret := nvml.DeviceGetHandleByIndex(i)
		if ret != nvml.SUCCESS {
			return nil, errors.New(nvml.ErrorString(ret))
		}

		devices = append(devices, &device)
	}

	return newGpuInfoList(devices)
}

func newGpuInfoList(devices []*nvml.Device) (*GpuInfoList, error) {
	inner := map[string]*GpuInfo{}
	for i, device := range devices {
		gpuInfo, err := newGpuInfo(device)
		if err != nil {
			return nil, err
		}

		inner[fmt.Sprint(i)] = gpuInfo
	}

	gpuInfoList := &GpuInfoList{
		Inner:     inner,
		MemorySum: Memory{},
	}

	for _, gpuInfo := range inner {
		gpuInfoList.MemorySum.Total += gpuInfo.Memory.Total
		gpuInfoList.MemorySum.Used += gpuInfo.Memory.Used
		gpuInfoList.MemorySum.Free += gpuInfo.Memory.Free
	}

	return gpuInfoList, nil
}

func flatten(m map[string]interface{}) map[string]interface{} {
	o := make(map[string]interface{})
	for k, v := range m {
		switch child := v.(type) {
		case map[string]interface{}:
			nm := flatten(child)
			for nk, nv := range nm {
				o[k+"."+nk] = nv
			}
		default:
			o[k] = v
		}
	}
	return o
}

func (g *GpuInfoList) ToFlatJson(prefix string) (map[string]string, error) {
	bytes, err := json.Marshal(*g)
	if err != nil {
		return nil, err
	}

	result := make(map[string]interface{})
	if err := json.Unmarshal(bytes, &result); err != nil {
		return nil, err
	}

	output := make(map[string]string)
	for k, v := range flatten(result) {
		output[fmt.Sprintf("%s/%s", prefix, k)] = fmt.Sprintf("%v", v)
	}
	return output, nil
}

type GpuInfo struct {
	UUID   string `json:"uuid"`
	Name   string `json:"name"`
	Memory Memory `json:"memory"`
}

func newGpuInfo(device *nvml.Device) (*GpuInfo, error) {
	uuid, ret := device.GetUUID()
	if ret != nvml.SUCCESS {
		return nil, errors.New(nvml.ErrorString(ret))
	}

	name, ret := device.GetName()
	if ret != nvml.SUCCESS {
		return nil, errors.New(nvml.ErrorString(ret))
	}

	memory, ret := device.GetMemoryInfo()
	if ret != nvml.SUCCESS {
		return nil, errors.New(nvml.ErrorString(ret))
	}

	gpuInfo := &GpuInfo{
		UUID: uuid,
		Name: name,
		Memory: Memory{
			Total: memory.Total / MiB,
			Used:  memory.Used / MiB,
			Free:  memory.Free / MiB,
		},
	}
	return gpuInfo, nil
}

type Memory struct {
	Total uint64 `json:"total"` // MiB
	Used  uint64 `json:"used"`  // MiB
	Free  uint64 `json:"free"`  // MiB
}
