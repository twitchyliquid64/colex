// Code generated by counterfeiter. DO NOT EDIT.
package runruncfakes

import (
	"sync"

	"code.cloudfoundry.org/guardian/rundmc/runrunc"
	"code.cloudfoundry.org/lager"
)

type FakeRuncCmdRunner struct {
	RunAndLogStub        func(log lager.Logger, cmd runrunc.LoggingCmd) error
	runAndLogMutex       sync.RWMutex
	runAndLogArgsForCall []struct {
		log lager.Logger
		cmd runrunc.LoggingCmd
	}
	runAndLogReturns struct {
		result1 error
	}
	runAndLogReturnsOnCall map[int]struct {
		result1 error
	}
	invocations      map[string][][]interface{}
	invocationsMutex sync.RWMutex
}

func (fake *FakeRuncCmdRunner) RunAndLog(log lager.Logger, cmd runrunc.LoggingCmd) error {
	fake.runAndLogMutex.Lock()
	ret, specificReturn := fake.runAndLogReturnsOnCall[len(fake.runAndLogArgsForCall)]
	fake.runAndLogArgsForCall = append(fake.runAndLogArgsForCall, struct {
		log lager.Logger
		cmd runrunc.LoggingCmd
	}{log, cmd})
	fake.recordInvocation("RunAndLog", []interface{}{log, cmd})
	fake.runAndLogMutex.Unlock()
	if fake.RunAndLogStub != nil {
		return fake.RunAndLogStub(log, cmd)
	}
	if specificReturn {
		return ret.result1
	}
	return fake.runAndLogReturns.result1
}

func (fake *FakeRuncCmdRunner) RunAndLogCallCount() int {
	fake.runAndLogMutex.RLock()
	defer fake.runAndLogMutex.RUnlock()
	return len(fake.runAndLogArgsForCall)
}

func (fake *FakeRuncCmdRunner) RunAndLogArgsForCall(i int) (lager.Logger, runrunc.LoggingCmd) {
	fake.runAndLogMutex.RLock()
	defer fake.runAndLogMutex.RUnlock()
	return fake.runAndLogArgsForCall[i].log, fake.runAndLogArgsForCall[i].cmd
}

func (fake *FakeRuncCmdRunner) RunAndLogReturns(result1 error) {
	fake.RunAndLogStub = nil
	fake.runAndLogReturns = struct {
		result1 error
	}{result1}
}

func (fake *FakeRuncCmdRunner) RunAndLogReturnsOnCall(i int, result1 error) {
	fake.RunAndLogStub = nil
	if fake.runAndLogReturnsOnCall == nil {
		fake.runAndLogReturnsOnCall = make(map[int]struct {
			result1 error
		})
	}
	fake.runAndLogReturnsOnCall[i] = struct {
		result1 error
	}{result1}
}

func (fake *FakeRuncCmdRunner) Invocations() map[string][][]interface{} {
	fake.invocationsMutex.RLock()
	defer fake.invocationsMutex.RUnlock()
	fake.runAndLogMutex.RLock()
	defer fake.runAndLogMutex.RUnlock()
	copiedInvocations := map[string][][]interface{}{}
	for key, value := range fake.invocations {
		copiedInvocations[key] = value
	}
	return copiedInvocations
}

func (fake *FakeRuncCmdRunner) recordInvocation(key string, args []interface{}) {
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

var _ runrunc.RuncCmdRunner = new(FakeRuncCmdRunner)
