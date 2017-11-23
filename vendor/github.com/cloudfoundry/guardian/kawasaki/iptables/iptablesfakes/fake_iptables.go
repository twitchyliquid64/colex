// Code generated by counterfeiter. DO NOT EDIT.
package iptablesfakes

import (
	"sync"

	"code.cloudfoundry.org/guardian/kawasaki/iptables"
)

type FakeIPTables struct {
	CreateChainStub        func(table, chain string) error
	createChainMutex       sync.RWMutex
	createChainArgsForCall []struct {
		table string
		chain string
	}
	createChainReturns struct {
		result1 error
	}
	createChainReturnsOnCall map[int]struct {
		result1 error
	}
	DeleteChainStub        func(table, chain string) error
	deleteChainMutex       sync.RWMutex
	deleteChainArgsForCall []struct {
		table string
		chain string
	}
	deleteChainReturns struct {
		result1 error
	}
	deleteChainReturnsOnCall map[int]struct {
		result1 error
	}
	FlushChainStub        func(table, chain string) error
	flushChainMutex       sync.RWMutex
	flushChainArgsForCall []struct {
		table string
		chain string
	}
	flushChainReturns struct {
		result1 error
	}
	flushChainReturnsOnCall map[int]struct {
		result1 error
	}
	DeleteChainReferencesStub        func(table, targetChain, referencedChain string) error
	deleteChainReferencesMutex       sync.RWMutex
	deleteChainReferencesArgsForCall []struct {
		table           string
		targetChain     string
		referencedChain string
	}
	deleteChainReferencesReturns struct {
		result1 error
	}
	deleteChainReferencesReturnsOnCall map[int]struct {
		result1 error
	}
	PrependRuleStub        func(chain string, rule iptables.Rule) error
	prependRuleMutex       sync.RWMutex
	prependRuleArgsForCall []struct {
		chain string
		rule  iptables.Rule
	}
	prependRuleReturns struct {
		result1 error
	}
	prependRuleReturnsOnCall map[int]struct {
		result1 error
	}
	BulkPrependRulesStub        func(chain string, rules []iptables.Rule) error
	bulkPrependRulesMutex       sync.RWMutex
	bulkPrependRulesArgsForCall []struct {
		chain string
		rules []iptables.Rule
	}
	bulkPrependRulesReturns struct {
		result1 error
	}
	bulkPrependRulesReturnsOnCall map[int]struct {
		result1 error
	}
	InstanceChainStub        func(instanceId string) string
	instanceChainMutex       sync.RWMutex
	instanceChainArgsForCall []struct {
		instanceId string
	}
	instanceChainReturns struct {
		result1 string
	}
	instanceChainReturnsOnCall map[int]struct {
		result1 string
	}
	invocations      map[string][][]interface{}
	invocationsMutex sync.RWMutex
}

func (fake *FakeIPTables) CreateChain(table string, chain string) error {
	fake.createChainMutex.Lock()
	ret, specificReturn := fake.createChainReturnsOnCall[len(fake.createChainArgsForCall)]
	fake.createChainArgsForCall = append(fake.createChainArgsForCall, struct {
		table string
		chain string
	}{table, chain})
	fake.recordInvocation("CreateChain", []interface{}{table, chain})
	fake.createChainMutex.Unlock()
	if fake.CreateChainStub != nil {
		return fake.CreateChainStub(table, chain)
	}
	if specificReturn {
		return ret.result1
	}
	return fake.createChainReturns.result1
}

func (fake *FakeIPTables) CreateChainCallCount() int {
	fake.createChainMutex.RLock()
	defer fake.createChainMutex.RUnlock()
	return len(fake.createChainArgsForCall)
}

func (fake *FakeIPTables) CreateChainArgsForCall(i int) (string, string) {
	fake.createChainMutex.RLock()
	defer fake.createChainMutex.RUnlock()
	return fake.createChainArgsForCall[i].table, fake.createChainArgsForCall[i].chain
}

func (fake *FakeIPTables) CreateChainReturns(result1 error) {
	fake.CreateChainStub = nil
	fake.createChainReturns = struct {
		result1 error
	}{result1}
}

func (fake *FakeIPTables) CreateChainReturnsOnCall(i int, result1 error) {
	fake.CreateChainStub = nil
	if fake.createChainReturnsOnCall == nil {
		fake.createChainReturnsOnCall = make(map[int]struct {
			result1 error
		})
	}
	fake.createChainReturnsOnCall[i] = struct {
		result1 error
	}{result1}
}

func (fake *FakeIPTables) DeleteChain(table string, chain string) error {
	fake.deleteChainMutex.Lock()
	ret, specificReturn := fake.deleteChainReturnsOnCall[len(fake.deleteChainArgsForCall)]
	fake.deleteChainArgsForCall = append(fake.deleteChainArgsForCall, struct {
		table string
		chain string
	}{table, chain})
	fake.recordInvocation("DeleteChain", []interface{}{table, chain})
	fake.deleteChainMutex.Unlock()
	if fake.DeleteChainStub != nil {
		return fake.DeleteChainStub(table, chain)
	}
	if specificReturn {
		return ret.result1
	}
	return fake.deleteChainReturns.result1
}

func (fake *FakeIPTables) DeleteChainCallCount() int {
	fake.deleteChainMutex.RLock()
	defer fake.deleteChainMutex.RUnlock()
	return len(fake.deleteChainArgsForCall)
}

func (fake *FakeIPTables) DeleteChainArgsForCall(i int) (string, string) {
	fake.deleteChainMutex.RLock()
	defer fake.deleteChainMutex.RUnlock()
	return fake.deleteChainArgsForCall[i].table, fake.deleteChainArgsForCall[i].chain
}

func (fake *FakeIPTables) DeleteChainReturns(result1 error) {
	fake.DeleteChainStub = nil
	fake.deleteChainReturns = struct {
		result1 error
	}{result1}
}

func (fake *FakeIPTables) DeleteChainReturnsOnCall(i int, result1 error) {
	fake.DeleteChainStub = nil
	if fake.deleteChainReturnsOnCall == nil {
		fake.deleteChainReturnsOnCall = make(map[int]struct {
			result1 error
		})
	}
	fake.deleteChainReturnsOnCall[i] = struct {
		result1 error
	}{result1}
}

func (fake *FakeIPTables) FlushChain(table string, chain string) error {
	fake.flushChainMutex.Lock()
	ret, specificReturn := fake.flushChainReturnsOnCall[len(fake.flushChainArgsForCall)]
	fake.flushChainArgsForCall = append(fake.flushChainArgsForCall, struct {
		table string
		chain string
	}{table, chain})
	fake.recordInvocation("FlushChain", []interface{}{table, chain})
	fake.flushChainMutex.Unlock()
	if fake.FlushChainStub != nil {
		return fake.FlushChainStub(table, chain)
	}
	if specificReturn {
		return ret.result1
	}
	return fake.flushChainReturns.result1
}

func (fake *FakeIPTables) FlushChainCallCount() int {
	fake.flushChainMutex.RLock()
	defer fake.flushChainMutex.RUnlock()
	return len(fake.flushChainArgsForCall)
}

func (fake *FakeIPTables) FlushChainArgsForCall(i int) (string, string) {
	fake.flushChainMutex.RLock()
	defer fake.flushChainMutex.RUnlock()
	return fake.flushChainArgsForCall[i].table, fake.flushChainArgsForCall[i].chain
}

func (fake *FakeIPTables) FlushChainReturns(result1 error) {
	fake.FlushChainStub = nil
	fake.flushChainReturns = struct {
		result1 error
	}{result1}
}

func (fake *FakeIPTables) FlushChainReturnsOnCall(i int, result1 error) {
	fake.FlushChainStub = nil
	if fake.flushChainReturnsOnCall == nil {
		fake.flushChainReturnsOnCall = make(map[int]struct {
			result1 error
		})
	}
	fake.flushChainReturnsOnCall[i] = struct {
		result1 error
	}{result1}
}

func (fake *FakeIPTables) DeleteChainReferences(table string, targetChain string, referencedChain string) error {
	fake.deleteChainReferencesMutex.Lock()
	ret, specificReturn := fake.deleteChainReferencesReturnsOnCall[len(fake.deleteChainReferencesArgsForCall)]
	fake.deleteChainReferencesArgsForCall = append(fake.deleteChainReferencesArgsForCall, struct {
		table           string
		targetChain     string
		referencedChain string
	}{table, targetChain, referencedChain})
	fake.recordInvocation("DeleteChainReferences", []interface{}{table, targetChain, referencedChain})
	fake.deleteChainReferencesMutex.Unlock()
	if fake.DeleteChainReferencesStub != nil {
		return fake.DeleteChainReferencesStub(table, targetChain, referencedChain)
	}
	if specificReturn {
		return ret.result1
	}
	return fake.deleteChainReferencesReturns.result1
}

func (fake *FakeIPTables) DeleteChainReferencesCallCount() int {
	fake.deleteChainReferencesMutex.RLock()
	defer fake.deleteChainReferencesMutex.RUnlock()
	return len(fake.deleteChainReferencesArgsForCall)
}

func (fake *FakeIPTables) DeleteChainReferencesArgsForCall(i int) (string, string, string) {
	fake.deleteChainReferencesMutex.RLock()
	defer fake.deleteChainReferencesMutex.RUnlock()
	return fake.deleteChainReferencesArgsForCall[i].table, fake.deleteChainReferencesArgsForCall[i].targetChain, fake.deleteChainReferencesArgsForCall[i].referencedChain
}

func (fake *FakeIPTables) DeleteChainReferencesReturns(result1 error) {
	fake.DeleteChainReferencesStub = nil
	fake.deleteChainReferencesReturns = struct {
		result1 error
	}{result1}
}

func (fake *FakeIPTables) DeleteChainReferencesReturnsOnCall(i int, result1 error) {
	fake.DeleteChainReferencesStub = nil
	if fake.deleteChainReferencesReturnsOnCall == nil {
		fake.deleteChainReferencesReturnsOnCall = make(map[int]struct {
			result1 error
		})
	}
	fake.deleteChainReferencesReturnsOnCall[i] = struct {
		result1 error
	}{result1}
}

func (fake *FakeIPTables) PrependRule(chain string, rule iptables.Rule) error {
	fake.prependRuleMutex.Lock()
	ret, specificReturn := fake.prependRuleReturnsOnCall[len(fake.prependRuleArgsForCall)]
	fake.prependRuleArgsForCall = append(fake.prependRuleArgsForCall, struct {
		chain string
		rule  iptables.Rule
	}{chain, rule})
	fake.recordInvocation("PrependRule", []interface{}{chain, rule})
	fake.prependRuleMutex.Unlock()
	if fake.PrependRuleStub != nil {
		return fake.PrependRuleStub(chain, rule)
	}
	if specificReturn {
		return ret.result1
	}
	return fake.prependRuleReturns.result1
}

func (fake *FakeIPTables) PrependRuleCallCount() int {
	fake.prependRuleMutex.RLock()
	defer fake.prependRuleMutex.RUnlock()
	return len(fake.prependRuleArgsForCall)
}

func (fake *FakeIPTables) PrependRuleArgsForCall(i int) (string, iptables.Rule) {
	fake.prependRuleMutex.RLock()
	defer fake.prependRuleMutex.RUnlock()
	return fake.prependRuleArgsForCall[i].chain, fake.prependRuleArgsForCall[i].rule
}

func (fake *FakeIPTables) PrependRuleReturns(result1 error) {
	fake.PrependRuleStub = nil
	fake.prependRuleReturns = struct {
		result1 error
	}{result1}
}

func (fake *FakeIPTables) PrependRuleReturnsOnCall(i int, result1 error) {
	fake.PrependRuleStub = nil
	if fake.prependRuleReturnsOnCall == nil {
		fake.prependRuleReturnsOnCall = make(map[int]struct {
			result1 error
		})
	}
	fake.prependRuleReturnsOnCall[i] = struct {
		result1 error
	}{result1}
}

func (fake *FakeIPTables) BulkPrependRules(chain string, rules []iptables.Rule) error {
	var rulesCopy []iptables.Rule
	if rules != nil {
		rulesCopy = make([]iptables.Rule, len(rules))
		copy(rulesCopy, rules)
	}
	fake.bulkPrependRulesMutex.Lock()
	ret, specificReturn := fake.bulkPrependRulesReturnsOnCall[len(fake.bulkPrependRulesArgsForCall)]
	fake.bulkPrependRulesArgsForCall = append(fake.bulkPrependRulesArgsForCall, struct {
		chain string
		rules []iptables.Rule
	}{chain, rulesCopy})
	fake.recordInvocation("BulkPrependRules", []interface{}{chain, rulesCopy})
	fake.bulkPrependRulesMutex.Unlock()
	if fake.BulkPrependRulesStub != nil {
		return fake.BulkPrependRulesStub(chain, rules)
	}
	if specificReturn {
		return ret.result1
	}
	return fake.bulkPrependRulesReturns.result1
}

func (fake *FakeIPTables) BulkPrependRulesCallCount() int {
	fake.bulkPrependRulesMutex.RLock()
	defer fake.bulkPrependRulesMutex.RUnlock()
	return len(fake.bulkPrependRulesArgsForCall)
}

func (fake *FakeIPTables) BulkPrependRulesArgsForCall(i int) (string, []iptables.Rule) {
	fake.bulkPrependRulesMutex.RLock()
	defer fake.bulkPrependRulesMutex.RUnlock()
	return fake.bulkPrependRulesArgsForCall[i].chain, fake.bulkPrependRulesArgsForCall[i].rules
}

func (fake *FakeIPTables) BulkPrependRulesReturns(result1 error) {
	fake.BulkPrependRulesStub = nil
	fake.bulkPrependRulesReturns = struct {
		result1 error
	}{result1}
}

func (fake *FakeIPTables) BulkPrependRulesReturnsOnCall(i int, result1 error) {
	fake.BulkPrependRulesStub = nil
	if fake.bulkPrependRulesReturnsOnCall == nil {
		fake.bulkPrependRulesReturnsOnCall = make(map[int]struct {
			result1 error
		})
	}
	fake.bulkPrependRulesReturnsOnCall[i] = struct {
		result1 error
	}{result1}
}

func (fake *FakeIPTables) InstanceChain(instanceId string) string {
	fake.instanceChainMutex.Lock()
	ret, specificReturn := fake.instanceChainReturnsOnCall[len(fake.instanceChainArgsForCall)]
	fake.instanceChainArgsForCall = append(fake.instanceChainArgsForCall, struct {
		instanceId string
	}{instanceId})
	fake.recordInvocation("InstanceChain", []interface{}{instanceId})
	fake.instanceChainMutex.Unlock()
	if fake.InstanceChainStub != nil {
		return fake.InstanceChainStub(instanceId)
	}
	if specificReturn {
		return ret.result1
	}
	return fake.instanceChainReturns.result1
}

func (fake *FakeIPTables) InstanceChainCallCount() int {
	fake.instanceChainMutex.RLock()
	defer fake.instanceChainMutex.RUnlock()
	return len(fake.instanceChainArgsForCall)
}

func (fake *FakeIPTables) InstanceChainArgsForCall(i int) string {
	fake.instanceChainMutex.RLock()
	defer fake.instanceChainMutex.RUnlock()
	return fake.instanceChainArgsForCall[i].instanceId
}

func (fake *FakeIPTables) InstanceChainReturns(result1 string) {
	fake.InstanceChainStub = nil
	fake.instanceChainReturns = struct {
		result1 string
	}{result1}
}

func (fake *FakeIPTables) InstanceChainReturnsOnCall(i int, result1 string) {
	fake.InstanceChainStub = nil
	if fake.instanceChainReturnsOnCall == nil {
		fake.instanceChainReturnsOnCall = make(map[int]struct {
			result1 string
		})
	}
	fake.instanceChainReturnsOnCall[i] = struct {
		result1 string
	}{result1}
}

func (fake *FakeIPTables) Invocations() map[string][][]interface{} {
	fake.invocationsMutex.RLock()
	defer fake.invocationsMutex.RUnlock()
	fake.createChainMutex.RLock()
	defer fake.createChainMutex.RUnlock()
	fake.deleteChainMutex.RLock()
	defer fake.deleteChainMutex.RUnlock()
	fake.flushChainMutex.RLock()
	defer fake.flushChainMutex.RUnlock()
	fake.deleteChainReferencesMutex.RLock()
	defer fake.deleteChainReferencesMutex.RUnlock()
	fake.prependRuleMutex.RLock()
	defer fake.prependRuleMutex.RUnlock()
	fake.bulkPrependRulesMutex.RLock()
	defer fake.bulkPrependRulesMutex.RUnlock()
	fake.instanceChainMutex.RLock()
	defer fake.instanceChainMutex.RUnlock()
	copiedInvocations := map[string][][]interface{}{}
	for key, value := range fake.invocations {
		copiedInvocations[key] = value
	}
	return copiedInvocations
}

func (fake *FakeIPTables) recordInvocation(key string, args []interface{}) {
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

var _ iptables.IPTables = new(FakeIPTables)
