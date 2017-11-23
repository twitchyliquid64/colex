// Code generated by counterfeiter. DO NOT EDIT.
package kawasakifakes

import (
	"sync"

	"code.cloudfoundry.org/guardian/kawasaki"
	"code.cloudfoundry.org/lager"
)

type FakeDnsResolvConfigurer struct {
	ConfigureStub        func(log lager.Logger, cfg kawasaki.NetworkConfig, pid int) error
	configureMutex       sync.RWMutex
	configureArgsForCall []struct {
		log lager.Logger
		cfg kawasaki.NetworkConfig
		pid int
	}
	configureReturns struct {
		result1 error
	}
	configureReturnsOnCall map[int]struct {
		result1 error
	}
	invocations      map[string][][]interface{}
	invocationsMutex sync.RWMutex
}

func (fake *FakeDnsResolvConfigurer) Configure(log lager.Logger, cfg kawasaki.NetworkConfig, pid int) error {
	fake.configureMutex.Lock()
	ret, specificReturn := fake.configureReturnsOnCall[len(fake.configureArgsForCall)]
	fake.configureArgsForCall = append(fake.configureArgsForCall, struct {
		log lager.Logger
		cfg kawasaki.NetworkConfig
		pid int
	}{log, cfg, pid})
	fake.recordInvocation("Configure", []interface{}{log, cfg, pid})
	fake.configureMutex.Unlock()
	if fake.ConfigureStub != nil {
		return fake.ConfigureStub(log, cfg, pid)
	}
	if specificReturn {
		return ret.result1
	}
	return fake.configureReturns.result1
}

func (fake *FakeDnsResolvConfigurer) ConfigureCallCount() int {
	fake.configureMutex.RLock()
	defer fake.configureMutex.RUnlock()
	return len(fake.configureArgsForCall)
}

func (fake *FakeDnsResolvConfigurer) ConfigureArgsForCall(i int) (lager.Logger, kawasaki.NetworkConfig, int) {
	fake.configureMutex.RLock()
	defer fake.configureMutex.RUnlock()
	return fake.configureArgsForCall[i].log, fake.configureArgsForCall[i].cfg, fake.configureArgsForCall[i].pid
}

func (fake *FakeDnsResolvConfigurer) ConfigureReturns(result1 error) {
	fake.ConfigureStub = nil
	fake.configureReturns = struct {
		result1 error
	}{result1}
}

func (fake *FakeDnsResolvConfigurer) ConfigureReturnsOnCall(i int, result1 error) {
	fake.ConfigureStub = nil
	if fake.configureReturnsOnCall == nil {
		fake.configureReturnsOnCall = make(map[int]struct {
			result1 error
		})
	}
	fake.configureReturnsOnCall[i] = struct {
		result1 error
	}{result1}
}

func (fake *FakeDnsResolvConfigurer) Invocations() map[string][][]interface{} {
	fake.invocationsMutex.RLock()
	defer fake.invocationsMutex.RUnlock()
	fake.configureMutex.RLock()
	defer fake.configureMutex.RUnlock()
	copiedInvocations := map[string][][]interface{}{}
	for key, value := range fake.invocations {
		copiedInvocations[key] = value
	}
	return copiedInvocations
}

func (fake *FakeDnsResolvConfigurer) recordInvocation(key string, args []interface{}) {
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

var _ kawasaki.DnsResolvConfigurer = new(FakeDnsResolvConfigurer)
