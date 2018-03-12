package metal

import (
	"github.com/ka2n/masminer/inspect/metal/gpu"
	"github.com/ka2n/masminer/inspect/metal/gpu/gpustat"
)

// GPUInfo : GPUの情報を取得します
func GPUInfo() ([]gpustat.GPUStat, error) {
	return gpu.Stat()
}
