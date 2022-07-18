package fakes

import "sync"

type ConfigWriter struct {
	WriteCall struct {
		mutex     sync.Mutex
		CallCount int
		Receives  struct {
			WorkingDir string
		}
		Returns struct {
			String string
			Error  error
		}
		Stub func(string) (string, error)
	}
}

func (f *ConfigWriter) Write(param1 string) (string, error) {
	f.WriteCall.mutex.Lock()
	defer f.WriteCall.mutex.Unlock()
	f.WriteCall.CallCount++
	f.WriteCall.Receives.WorkingDir = param1
	if f.WriteCall.Stub != nil {
		return f.WriteCall.Stub(param1)
	}
	return f.WriteCall.Returns.String, f.WriteCall.Returns.Error
}
