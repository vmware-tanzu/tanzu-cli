// Code generated by counterfeiter. DO NOT EDIT.
package fakes

import (
	"sync"

	"github.com/vmware-tanzu/tanzu-cli/pkg/centralconfig"
)

type CentralConfig struct {
	GetCentralConfigEntryStub        func(string, interface{}) error
	getCentralConfigEntryMutex       sync.RWMutex
	getCentralConfigEntryArgsForCall []struct {
		arg1 string
		arg2 interface{}
	}
	getCentralConfigEntryReturns struct {
		result1 error
	}
	getCentralConfigEntryReturnsOnCall map[int]struct {
		result1 error
	}
	GetDefaultTanzuEndpointStub        func() (string, error)
	getDefaultTanzuEndpointMutex       sync.RWMutex
	getDefaultTanzuEndpointArgsForCall []struct {
	}
	getDefaultTanzuEndpointReturns struct {
		result1 string
		result2 error
	}
	getDefaultTanzuEndpointReturnsOnCall map[int]struct {
		result1 string
		result2 error
	}
	GetInventoryRefreshTTLSecondsStub        func() (int, error)
	getInventoryRefreshTTLSecondsMutex       sync.RWMutex
	getInventoryRefreshTTLSecondsArgsForCall []struct {
	}
	getInventoryRefreshTTLSecondsReturns struct {
		result1 int
		result2 error
	}
	getInventoryRefreshTTLSecondsReturnsOnCall map[int]struct {
		result1 int
		result2 error
	}
	GetPluginDBCacheRefreshThresholdSecondsStub        func() (int, error)
	getPluginDBCacheRefreshThresholdSecondsMutex       sync.RWMutex
	getPluginDBCacheRefreshThresholdSecondsArgsForCall []struct {
	}
	getPluginDBCacheRefreshThresholdSecondsReturns struct {
		result1 int
		result2 error
	}
	getPluginDBCacheRefreshThresholdSecondsReturnsOnCall map[int]struct {
		result1 int
		result2 error
	}
	GetTanzuConfigEndpointUpdateMappingStub        func() (map[string]string, error)
	getTanzuConfigEndpointUpdateMappingMutex       sync.RWMutex
	getTanzuConfigEndpointUpdateMappingArgsForCall []struct {
	}
	getTanzuConfigEndpointUpdateMappingReturns struct {
		result1 map[string]string
		result2 error
	}
	getTanzuConfigEndpointUpdateMappingReturnsOnCall map[int]struct {
		result1 map[string]string
		result2 error
	}
	GetTanzuConfigEndpointUpdateVersionStub        func() (string, error)
	getTanzuConfigEndpointUpdateVersionMutex       sync.RWMutex
	getTanzuConfigEndpointUpdateVersionArgsForCall []struct {
	}
	getTanzuConfigEndpointUpdateVersionReturns struct {
		result1 string
		result2 error
	}
	getTanzuConfigEndpointUpdateVersionReturnsOnCall map[int]struct {
		result1 string
		result2 error
	}
	GetTanzuPlatformEndpointToServiceEndpointMapStub        func() (centralconfig.TanzuPlatformEndpointToServiceEndpointMap, error)
	getTanzuPlatformEndpointToServiceEndpointMapMutex       sync.RWMutex
	getTanzuPlatformEndpointToServiceEndpointMapArgsForCall []struct {
	}
	getTanzuPlatformEndpointToServiceEndpointMapReturns struct {
		result1 centralconfig.TanzuPlatformEndpointToServiceEndpointMap
		result2 error
	}
	getTanzuPlatformEndpointToServiceEndpointMapReturnsOnCall map[int]struct {
		result1 centralconfig.TanzuPlatformEndpointToServiceEndpointMap
		result2 error
	}
	GetTanzuPlatformSaaSEndpointListStub        func() []string
	getTanzuPlatformSaaSEndpointListMutex       sync.RWMutex
	getTanzuPlatformSaaSEndpointListArgsForCall []struct {
	}
	getTanzuPlatformSaaSEndpointListReturns struct {
		result1 []string
	}
	getTanzuPlatformSaaSEndpointListReturnsOnCall map[int]struct {
		result1 []string
	}
	invocations      map[string][][]interface{}
	invocationsMutex sync.RWMutex
}

func (fake *CentralConfig) GetCentralConfigEntry(arg1 string, arg2 interface{}) error {
	fake.getCentralConfigEntryMutex.Lock()
	ret, specificReturn := fake.getCentralConfigEntryReturnsOnCall[len(fake.getCentralConfigEntryArgsForCall)]
	fake.getCentralConfigEntryArgsForCall = append(fake.getCentralConfigEntryArgsForCall, struct {
		arg1 string
		arg2 interface{}
	}{arg1, arg2})
	stub := fake.GetCentralConfigEntryStub
	fakeReturns := fake.getCentralConfigEntryReturns
	fake.recordInvocation("GetCentralConfigEntry", []interface{}{arg1, arg2})
	fake.getCentralConfigEntryMutex.Unlock()
	if stub != nil {
		return stub(arg1, arg2)
	}
	if specificReturn {
		return ret.result1
	}
	return fakeReturns.result1
}

func (fake *CentralConfig) GetCentralConfigEntryCallCount() int {
	fake.getCentralConfigEntryMutex.RLock()
	defer fake.getCentralConfigEntryMutex.RUnlock()
	return len(fake.getCentralConfigEntryArgsForCall)
}

func (fake *CentralConfig) GetCentralConfigEntryCalls(stub func(string, interface{}) error) {
	fake.getCentralConfigEntryMutex.Lock()
	defer fake.getCentralConfigEntryMutex.Unlock()
	fake.GetCentralConfigEntryStub = stub
}

func (fake *CentralConfig) GetCentralConfigEntryArgsForCall(i int) (string, interface{}) {
	fake.getCentralConfigEntryMutex.RLock()
	defer fake.getCentralConfigEntryMutex.RUnlock()
	argsForCall := fake.getCentralConfigEntryArgsForCall[i]
	return argsForCall.arg1, argsForCall.arg2
}

func (fake *CentralConfig) GetCentralConfigEntryReturns(result1 error) {
	fake.getCentralConfigEntryMutex.Lock()
	defer fake.getCentralConfigEntryMutex.Unlock()
	fake.GetCentralConfigEntryStub = nil
	fake.getCentralConfigEntryReturns = struct {
		result1 error
	}{result1}
}

func (fake *CentralConfig) GetCentralConfigEntryReturnsOnCall(i int, result1 error) {
	fake.getCentralConfigEntryMutex.Lock()
	defer fake.getCentralConfigEntryMutex.Unlock()
	fake.GetCentralConfigEntryStub = nil
	if fake.getCentralConfigEntryReturnsOnCall == nil {
		fake.getCentralConfigEntryReturnsOnCall = make(map[int]struct {
			result1 error
		})
	}
	fake.getCentralConfigEntryReturnsOnCall[i] = struct {
		result1 error
	}{result1}
}

func (fake *CentralConfig) GetDefaultTanzuEndpoint() (string, error) {
	fake.getDefaultTanzuEndpointMutex.Lock()
	ret, specificReturn := fake.getDefaultTanzuEndpointReturnsOnCall[len(fake.getDefaultTanzuEndpointArgsForCall)]
	fake.getDefaultTanzuEndpointArgsForCall = append(fake.getDefaultTanzuEndpointArgsForCall, struct {
	}{})
	stub := fake.GetDefaultTanzuEndpointStub
	fakeReturns := fake.getDefaultTanzuEndpointReturns
	fake.recordInvocation("GetDefaultTanzuEndpoint", []interface{}{})
	fake.getDefaultTanzuEndpointMutex.Unlock()
	if stub != nil {
		return stub()
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	return fakeReturns.result1, fakeReturns.result2
}

func (fake *CentralConfig) GetDefaultTanzuEndpointCallCount() int {
	fake.getDefaultTanzuEndpointMutex.RLock()
	defer fake.getDefaultTanzuEndpointMutex.RUnlock()
	return len(fake.getDefaultTanzuEndpointArgsForCall)
}

func (fake *CentralConfig) GetDefaultTanzuEndpointCalls(stub func() (string, error)) {
	fake.getDefaultTanzuEndpointMutex.Lock()
	defer fake.getDefaultTanzuEndpointMutex.Unlock()
	fake.GetDefaultTanzuEndpointStub = stub
}

func (fake *CentralConfig) GetDefaultTanzuEndpointReturns(result1 string, result2 error) {
	fake.getDefaultTanzuEndpointMutex.Lock()
	defer fake.getDefaultTanzuEndpointMutex.Unlock()
	fake.GetDefaultTanzuEndpointStub = nil
	fake.getDefaultTanzuEndpointReturns = struct {
		result1 string
		result2 error
	}{result1, result2}
}

func (fake *CentralConfig) GetDefaultTanzuEndpointReturnsOnCall(i int, result1 string, result2 error) {
	fake.getDefaultTanzuEndpointMutex.Lock()
	defer fake.getDefaultTanzuEndpointMutex.Unlock()
	fake.GetDefaultTanzuEndpointStub = nil
	if fake.getDefaultTanzuEndpointReturnsOnCall == nil {
		fake.getDefaultTanzuEndpointReturnsOnCall = make(map[int]struct {
			result1 string
			result2 error
		})
	}
	fake.getDefaultTanzuEndpointReturnsOnCall[i] = struct {
		result1 string
		result2 error
	}{result1, result2}
}

func (fake *CentralConfig) GetInventoryRefreshTTLSeconds() (int, error) {
	fake.getInventoryRefreshTTLSecondsMutex.Lock()
	ret, specificReturn := fake.getInventoryRefreshTTLSecondsReturnsOnCall[len(fake.getInventoryRefreshTTLSecondsArgsForCall)]
	fake.getInventoryRefreshTTLSecondsArgsForCall = append(fake.getInventoryRefreshTTLSecondsArgsForCall, struct {
	}{})
	stub := fake.GetInventoryRefreshTTLSecondsStub
	fakeReturns := fake.getInventoryRefreshTTLSecondsReturns
	fake.recordInvocation("GetInventoryRefreshTTLSeconds", []interface{}{})
	fake.getInventoryRefreshTTLSecondsMutex.Unlock()
	if stub != nil {
		return stub()
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	return fakeReturns.result1, fakeReturns.result2
}

func (fake *CentralConfig) GetInventoryRefreshTTLSecondsCallCount() int {
	fake.getInventoryRefreshTTLSecondsMutex.RLock()
	defer fake.getInventoryRefreshTTLSecondsMutex.RUnlock()
	return len(fake.getInventoryRefreshTTLSecondsArgsForCall)
}

func (fake *CentralConfig) GetInventoryRefreshTTLSecondsCalls(stub func() (int, error)) {
	fake.getInventoryRefreshTTLSecondsMutex.Lock()
	defer fake.getInventoryRefreshTTLSecondsMutex.Unlock()
	fake.GetInventoryRefreshTTLSecondsStub = stub
}

func (fake *CentralConfig) GetInventoryRefreshTTLSecondsReturns(result1 int, result2 error) {
	fake.getInventoryRefreshTTLSecondsMutex.Lock()
	defer fake.getInventoryRefreshTTLSecondsMutex.Unlock()
	fake.GetInventoryRefreshTTLSecondsStub = nil
	fake.getInventoryRefreshTTLSecondsReturns = struct {
		result1 int
		result2 error
	}{result1, result2}
}

func (fake *CentralConfig) GetInventoryRefreshTTLSecondsReturnsOnCall(i int, result1 int, result2 error) {
	fake.getInventoryRefreshTTLSecondsMutex.Lock()
	defer fake.getInventoryRefreshTTLSecondsMutex.Unlock()
	fake.GetInventoryRefreshTTLSecondsStub = nil
	if fake.getInventoryRefreshTTLSecondsReturnsOnCall == nil {
		fake.getInventoryRefreshTTLSecondsReturnsOnCall = make(map[int]struct {
			result1 int
			result2 error
		})
	}
	fake.getInventoryRefreshTTLSecondsReturnsOnCall[i] = struct {
		result1 int
		result2 error
	}{result1, result2}
}

func (fake *CentralConfig) GetPluginDBCacheRefreshThresholdSeconds() (int, error) {
	fake.getPluginDBCacheRefreshThresholdSecondsMutex.Lock()
	ret, specificReturn := fake.getPluginDBCacheRefreshThresholdSecondsReturnsOnCall[len(fake.getPluginDBCacheRefreshThresholdSecondsArgsForCall)]
	fake.getPluginDBCacheRefreshThresholdSecondsArgsForCall = append(fake.getPluginDBCacheRefreshThresholdSecondsArgsForCall, struct {
	}{})
	stub := fake.GetPluginDBCacheRefreshThresholdSecondsStub
	fakeReturns := fake.getPluginDBCacheRefreshThresholdSecondsReturns
	fake.recordInvocation("GetPluginDBCacheRefreshThresholdSeconds", []interface{}{})
	fake.getPluginDBCacheRefreshThresholdSecondsMutex.Unlock()
	if stub != nil {
		return stub()
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	return fakeReturns.result1, fakeReturns.result2
}

func (fake *CentralConfig) GetPluginDBCacheRefreshThresholdSecondsCallCount() int {
	fake.getPluginDBCacheRefreshThresholdSecondsMutex.RLock()
	defer fake.getPluginDBCacheRefreshThresholdSecondsMutex.RUnlock()
	return len(fake.getPluginDBCacheRefreshThresholdSecondsArgsForCall)
}

func (fake *CentralConfig) GetPluginDBCacheRefreshThresholdSecondsCalls(stub func() (int, error)) {
	fake.getPluginDBCacheRefreshThresholdSecondsMutex.Lock()
	defer fake.getPluginDBCacheRefreshThresholdSecondsMutex.Unlock()
	fake.GetPluginDBCacheRefreshThresholdSecondsStub = stub
}

func (fake *CentralConfig) GetPluginDBCacheRefreshThresholdSecondsReturns(result1 int, result2 error) {
	fake.getPluginDBCacheRefreshThresholdSecondsMutex.Lock()
	defer fake.getPluginDBCacheRefreshThresholdSecondsMutex.Unlock()
	fake.GetPluginDBCacheRefreshThresholdSecondsStub = nil
	fake.getPluginDBCacheRefreshThresholdSecondsReturns = struct {
		result1 int
		result2 error
	}{result1, result2}
}

func (fake *CentralConfig) GetPluginDBCacheRefreshThresholdSecondsReturnsOnCall(i int, result1 int, result2 error) {
	fake.getPluginDBCacheRefreshThresholdSecondsMutex.Lock()
	defer fake.getPluginDBCacheRefreshThresholdSecondsMutex.Unlock()
	fake.GetPluginDBCacheRefreshThresholdSecondsStub = nil
	if fake.getPluginDBCacheRefreshThresholdSecondsReturnsOnCall == nil {
		fake.getPluginDBCacheRefreshThresholdSecondsReturnsOnCall = make(map[int]struct {
			result1 int
			result2 error
		})
	}
	fake.getPluginDBCacheRefreshThresholdSecondsReturnsOnCall[i] = struct {
		result1 int
		result2 error
	}{result1, result2}
}

func (fake *CentralConfig) GetTanzuConfigEndpointUpdateMapping() (map[string]string, error) {
	fake.getTanzuConfigEndpointUpdateMappingMutex.Lock()
	ret, specificReturn := fake.getTanzuConfigEndpointUpdateMappingReturnsOnCall[len(fake.getTanzuConfigEndpointUpdateMappingArgsForCall)]
	fake.getTanzuConfigEndpointUpdateMappingArgsForCall = append(fake.getTanzuConfigEndpointUpdateMappingArgsForCall, struct {
	}{})
	stub := fake.GetTanzuConfigEndpointUpdateMappingStub
	fakeReturns := fake.getTanzuConfigEndpointUpdateMappingReturns
	fake.recordInvocation("GetTanzuConfigEndpointUpdateMapping", []interface{}{})
	fake.getTanzuConfigEndpointUpdateMappingMutex.Unlock()
	if stub != nil {
		return stub()
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	return fakeReturns.result1, fakeReturns.result2
}

func (fake *CentralConfig) GetTanzuConfigEndpointUpdateMappingCallCount() int {
	fake.getTanzuConfigEndpointUpdateMappingMutex.RLock()
	defer fake.getTanzuConfigEndpointUpdateMappingMutex.RUnlock()
	return len(fake.getTanzuConfigEndpointUpdateMappingArgsForCall)
}

func (fake *CentralConfig) GetTanzuConfigEndpointUpdateMappingCalls(stub func() (map[string]string, error)) {
	fake.getTanzuConfigEndpointUpdateMappingMutex.Lock()
	defer fake.getTanzuConfigEndpointUpdateMappingMutex.Unlock()
	fake.GetTanzuConfigEndpointUpdateMappingStub = stub
}

func (fake *CentralConfig) GetTanzuConfigEndpointUpdateMappingReturns(result1 map[string]string, result2 error) {
	fake.getTanzuConfigEndpointUpdateMappingMutex.Lock()
	defer fake.getTanzuConfigEndpointUpdateMappingMutex.Unlock()
	fake.GetTanzuConfigEndpointUpdateMappingStub = nil
	fake.getTanzuConfigEndpointUpdateMappingReturns = struct {
		result1 map[string]string
		result2 error
	}{result1, result2}
}

func (fake *CentralConfig) GetTanzuConfigEndpointUpdateMappingReturnsOnCall(i int, result1 map[string]string, result2 error) {
	fake.getTanzuConfigEndpointUpdateMappingMutex.Lock()
	defer fake.getTanzuConfigEndpointUpdateMappingMutex.Unlock()
	fake.GetTanzuConfigEndpointUpdateMappingStub = nil
	if fake.getTanzuConfigEndpointUpdateMappingReturnsOnCall == nil {
		fake.getTanzuConfigEndpointUpdateMappingReturnsOnCall = make(map[int]struct {
			result1 map[string]string
			result2 error
		})
	}
	fake.getTanzuConfigEndpointUpdateMappingReturnsOnCall[i] = struct {
		result1 map[string]string
		result2 error
	}{result1, result2}
}

func (fake *CentralConfig) GetTanzuConfigEndpointUpdateVersion() (string, error) {
	fake.getTanzuConfigEndpointUpdateVersionMutex.Lock()
	ret, specificReturn := fake.getTanzuConfigEndpointUpdateVersionReturnsOnCall[len(fake.getTanzuConfigEndpointUpdateVersionArgsForCall)]
	fake.getTanzuConfigEndpointUpdateVersionArgsForCall = append(fake.getTanzuConfigEndpointUpdateVersionArgsForCall, struct {
	}{})
	stub := fake.GetTanzuConfigEndpointUpdateVersionStub
	fakeReturns := fake.getTanzuConfigEndpointUpdateVersionReturns
	fake.recordInvocation("GetTanzuConfigEndpointUpdateVersion", []interface{}{})
	fake.getTanzuConfigEndpointUpdateVersionMutex.Unlock()
	if stub != nil {
		return stub()
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	return fakeReturns.result1, fakeReturns.result2
}

func (fake *CentralConfig) GetTanzuConfigEndpointUpdateVersionCallCount() int {
	fake.getTanzuConfigEndpointUpdateVersionMutex.RLock()
	defer fake.getTanzuConfigEndpointUpdateVersionMutex.RUnlock()
	return len(fake.getTanzuConfigEndpointUpdateVersionArgsForCall)
}

func (fake *CentralConfig) GetTanzuConfigEndpointUpdateVersionCalls(stub func() (string, error)) {
	fake.getTanzuConfigEndpointUpdateVersionMutex.Lock()
	defer fake.getTanzuConfigEndpointUpdateVersionMutex.Unlock()
	fake.GetTanzuConfigEndpointUpdateVersionStub = stub
}

func (fake *CentralConfig) GetTanzuConfigEndpointUpdateVersionReturns(result1 string, result2 error) {
	fake.getTanzuConfigEndpointUpdateVersionMutex.Lock()
	defer fake.getTanzuConfigEndpointUpdateVersionMutex.Unlock()
	fake.GetTanzuConfigEndpointUpdateVersionStub = nil
	fake.getTanzuConfigEndpointUpdateVersionReturns = struct {
		result1 string
		result2 error
	}{result1, result2}
}

func (fake *CentralConfig) GetTanzuConfigEndpointUpdateVersionReturnsOnCall(i int, result1 string, result2 error) {
	fake.getTanzuConfigEndpointUpdateVersionMutex.Lock()
	defer fake.getTanzuConfigEndpointUpdateVersionMutex.Unlock()
	fake.GetTanzuConfigEndpointUpdateVersionStub = nil
	if fake.getTanzuConfigEndpointUpdateVersionReturnsOnCall == nil {
		fake.getTanzuConfigEndpointUpdateVersionReturnsOnCall = make(map[int]struct {
			result1 string
			result2 error
		})
	}
	fake.getTanzuConfigEndpointUpdateVersionReturnsOnCall[i] = struct {
		result1 string
		result2 error
	}{result1, result2}
}

func (fake *CentralConfig) GetTanzuPlatformEndpointToServiceEndpointMap() (centralconfig.TanzuPlatformEndpointToServiceEndpointMap, error) {
	fake.getTanzuPlatformEndpointToServiceEndpointMapMutex.Lock()
	ret, specificReturn := fake.getTanzuPlatformEndpointToServiceEndpointMapReturnsOnCall[len(fake.getTanzuPlatformEndpointToServiceEndpointMapArgsForCall)]
	fake.getTanzuPlatformEndpointToServiceEndpointMapArgsForCall = append(fake.getTanzuPlatformEndpointToServiceEndpointMapArgsForCall, struct {
	}{})
	stub := fake.GetTanzuPlatformEndpointToServiceEndpointMapStub
	fakeReturns := fake.getTanzuPlatformEndpointToServiceEndpointMapReturns
	fake.recordInvocation("GetTanzuPlatformEndpointToServiceEndpointMap", []interface{}{})
	fake.getTanzuPlatformEndpointToServiceEndpointMapMutex.Unlock()
	if stub != nil {
		return stub()
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	return fakeReturns.result1, fakeReturns.result2
}

func (fake *CentralConfig) GetTanzuPlatformEndpointToServiceEndpointMapCallCount() int {
	fake.getTanzuPlatformEndpointToServiceEndpointMapMutex.RLock()
	defer fake.getTanzuPlatformEndpointToServiceEndpointMapMutex.RUnlock()
	return len(fake.getTanzuPlatformEndpointToServiceEndpointMapArgsForCall)
}

func (fake *CentralConfig) GetTanzuPlatformEndpointToServiceEndpointMapCalls(stub func() (centralconfig.TanzuPlatformEndpointToServiceEndpointMap, error)) {
	fake.getTanzuPlatformEndpointToServiceEndpointMapMutex.Lock()
	defer fake.getTanzuPlatformEndpointToServiceEndpointMapMutex.Unlock()
	fake.GetTanzuPlatformEndpointToServiceEndpointMapStub = stub
}

func (fake *CentralConfig) GetTanzuPlatformEndpointToServiceEndpointMapReturns(result1 centralconfig.TanzuPlatformEndpointToServiceEndpointMap, result2 error) {
	fake.getTanzuPlatformEndpointToServiceEndpointMapMutex.Lock()
	defer fake.getTanzuPlatformEndpointToServiceEndpointMapMutex.Unlock()
	fake.GetTanzuPlatformEndpointToServiceEndpointMapStub = nil
	fake.getTanzuPlatformEndpointToServiceEndpointMapReturns = struct {
		result1 centralconfig.TanzuPlatformEndpointToServiceEndpointMap
		result2 error
	}{result1, result2}
}

func (fake *CentralConfig) GetTanzuPlatformEndpointToServiceEndpointMapReturnsOnCall(i int, result1 centralconfig.TanzuPlatformEndpointToServiceEndpointMap, result2 error) {
	fake.getTanzuPlatformEndpointToServiceEndpointMapMutex.Lock()
	defer fake.getTanzuPlatformEndpointToServiceEndpointMapMutex.Unlock()
	fake.GetTanzuPlatformEndpointToServiceEndpointMapStub = nil
	if fake.getTanzuPlatformEndpointToServiceEndpointMapReturnsOnCall == nil {
		fake.getTanzuPlatformEndpointToServiceEndpointMapReturnsOnCall = make(map[int]struct {
			result1 centralconfig.TanzuPlatformEndpointToServiceEndpointMap
			result2 error
		})
	}
	fake.getTanzuPlatformEndpointToServiceEndpointMapReturnsOnCall[i] = struct {
		result1 centralconfig.TanzuPlatformEndpointToServiceEndpointMap
		result2 error
	}{result1, result2}
}

func (fake *CentralConfig) GetTanzuPlatformSaaSEndpointList() []string {
	fake.getTanzuPlatformSaaSEndpointListMutex.Lock()
	ret, specificReturn := fake.getTanzuPlatformSaaSEndpointListReturnsOnCall[len(fake.getTanzuPlatformSaaSEndpointListArgsForCall)]
	fake.getTanzuPlatformSaaSEndpointListArgsForCall = append(fake.getTanzuPlatformSaaSEndpointListArgsForCall, struct {
	}{})
	stub := fake.GetTanzuPlatformSaaSEndpointListStub
	fakeReturns := fake.getTanzuPlatformSaaSEndpointListReturns
	fake.recordInvocation("GetTanzuPlatformSaaSEndpointList", []interface{}{})
	fake.getTanzuPlatformSaaSEndpointListMutex.Unlock()
	if stub != nil {
		return stub()
	}
	if specificReturn {
		return ret.result1
	}
	return fakeReturns.result1
}

func (fake *CentralConfig) GetTanzuPlatformSaaSEndpointListCallCount() int {
	fake.getTanzuPlatformSaaSEndpointListMutex.RLock()
	defer fake.getTanzuPlatformSaaSEndpointListMutex.RUnlock()
	return len(fake.getTanzuPlatformSaaSEndpointListArgsForCall)
}

func (fake *CentralConfig) GetTanzuPlatformSaaSEndpointListCalls(stub func() []string) {
	fake.getTanzuPlatformSaaSEndpointListMutex.Lock()
	defer fake.getTanzuPlatformSaaSEndpointListMutex.Unlock()
	fake.GetTanzuPlatformSaaSEndpointListStub = stub
}

func (fake *CentralConfig) GetTanzuPlatformSaaSEndpointListReturns(result1 []string) {
	fake.getTanzuPlatformSaaSEndpointListMutex.Lock()
	defer fake.getTanzuPlatformSaaSEndpointListMutex.Unlock()
	fake.GetTanzuPlatformSaaSEndpointListStub = nil
	fake.getTanzuPlatformSaaSEndpointListReturns = struct {
		result1 []string
	}{result1}
}

func (fake *CentralConfig) GetTanzuPlatformSaaSEndpointListReturnsOnCall(i int, result1 []string) {
	fake.getTanzuPlatformSaaSEndpointListMutex.Lock()
	defer fake.getTanzuPlatformSaaSEndpointListMutex.Unlock()
	fake.GetTanzuPlatformSaaSEndpointListStub = nil
	if fake.getTanzuPlatformSaaSEndpointListReturnsOnCall == nil {
		fake.getTanzuPlatformSaaSEndpointListReturnsOnCall = make(map[int]struct {
			result1 []string
		})
	}
	fake.getTanzuPlatformSaaSEndpointListReturnsOnCall[i] = struct {
		result1 []string
	}{result1}
}

func (fake *CentralConfig) Invocations() map[string][][]interface{} {
	fake.invocationsMutex.RLock()
	defer fake.invocationsMutex.RUnlock()
	fake.getCentralConfigEntryMutex.RLock()
	defer fake.getCentralConfigEntryMutex.RUnlock()
	fake.getDefaultTanzuEndpointMutex.RLock()
	defer fake.getDefaultTanzuEndpointMutex.RUnlock()
	fake.getInventoryRefreshTTLSecondsMutex.RLock()
	defer fake.getInventoryRefreshTTLSecondsMutex.RUnlock()
	fake.getPluginDBCacheRefreshThresholdSecondsMutex.RLock()
	defer fake.getPluginDBCacheRefreshThresholdSecondsMutex.RUnlock()
	fake.getTanzuConfigEndpointUpdateMappingMutex.RLock()
	defer fake.getTanzuConfigEndpointUpdateMappingMutex.RUnlock()
	fake.getTanzuConfigEndpointUpdateVersionMutex.RLock()
	defer fake.getTanzuConfigEndpointUpdateVersionMutex.RUnlock()
	fake.getTanzuPlatformEndpointToServiceEndpointMapMutex.RLock()
	defer fake.getTanzuPlatformEndpointToServiceEndpointMapMutex.RUnlock()
	fake.getTanzuPlatformSaaSEndpointListMutex.RLock()
	defer fake.getTanzuPlatformSaaSEndpointListMutex.RUnlock()
	copiedInvocations := map[string][][]interface{}{}
	for key, value := range fake.invocations {
		copiedInvocations[key] = value
	}
	return copiedInvocations
}

func (fake *CentralConfig) recordInvocation(key string, args []interface{}) {
	fake.invocationsMutex.Lock()
	defer fake.invocationsMutex.Unlock()
	if fake.invocations == nil {
		fake.invocations = map[string][][]interface{}{}
	}
	if fake.invocations[key] == nil {
		fake.invocations[key] = [][]interface{}{}
	}
	fake.invocations[key] = append(fake.invocations[key], args)
}

var _ centralconfig.CentralConfig = new(CentralConfig)
