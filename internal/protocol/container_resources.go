package protocol

// Container limits are ceilings, not host reservations or Vere loom sizes.
// Zero on historical records means the original 4 GiB / 2 CPU contract.
const (
	ContainerResourcesFeature       = "container-resources-v1"
	DefaultMemoryGiB          uint8 = 4
	DefaultCPUCores           uint8 = 2
	MaxMemoryGiB              uint8 = 16
	MaxCPUCores               uint8 = 16
)

func EffectiveMemoryGiB(n uint8) uint8 {
	if n == 0 {
		return DefaultMemoryGiB
	}
	return n
}
func EffectiveCPUCores(n uint8) uint8 {
	if n == 0 {
		return DefaultCPUCores
	}
	return n
}
func ValidContainerResources(memory, cpu uint8) bool {
	return memory <= MaxMemoryGiB && cpu <= MaxCPUCores
}
func NondefaultContainerResources(memory, cpu uint8) bool {
	return EffectiveMemoryGiB(memory) != DefaultMemoryGiB || EffectiveCPUCores(cpu) != DefaultCPUCores
}
